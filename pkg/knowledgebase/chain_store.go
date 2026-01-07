package knowledgebase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// SQLiteStore SQLite 存储实现
type SQLiteStore struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewSQLiteStore 创建 SQLite 存储
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	store := &SQLiteStore{
		db:     db,
		logger: zap.NewNop(), // 使用默认 logger，实际使用时会被替换
	}

	// 初始化表结构
	if err := store.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化表失败: %w", err)
	}

	return store, nil
}

// initTables 初始化数据库表
func (s *SQLiteStore) initTables() error {
	queries := []string{
		// 思维链主表
		`CREATE TABLE IF NOT EXISTS chains (
			id TEXT PRIMARY KEY,
			question TEXT NOT NULL,
			question_id TEXT,
			domain TEXT,
			complexity TEXT,
			keywords TEXT,  -- JSON array
			model TEXT,
			reasoning TEXT,
			answer TEXT,
			confidence REAL,
			metadata TEXT,  -- JSON object
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		// 思维链步骤表
		`CREATE TABLE IF NOT EXISTS chain_steps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chain_id TEXT NOT NULL,
			step_id INTEGER NOT NULL,
			description TEXT,
			action TEXT,
			content TEXT,
			result TEXT,
			FOREIGN KEY (chain_id) REFERENCES chains(id) ON DELETE CASCADE
		)`,
		// 创建索引
		`CREATE INDEX IF NOT EXISTS idx_chains_domain ON chains(domain)`,
		`CREATE INDEX IF NOT EXISTS idx_chains_complexity ON chains(complexity)`,
		`CREATE INDEX IF NOT EXISTS idx_chains_question_id ON chains(question_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chain_steps_chain_id ON chain_steps(chain_id)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("执行 SQL 失败: %w, SQL: %s", err, query)
		}
	}

	return nil
}

// SaveChain 保存思维链
func (s *SQLiteStore) SaveChain(ctx context.Context, chain *ChainOfThought) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 序列化关键词和元数据
	keywordsJSON, _ := json.Marshal(chain.Keywords)
	metadataJSON, _ := json.Marshal(chain.Metadata)

	// 插入或更新主表
	query := `INSERT OR REPLACE INTO chains 
		(id, question, question_id, domain, complexity, keywords, model, reasoning, answer, confidence, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.ExecContext(ctx, query,
		chain.ID,
		chain.Question,
		chain.QuestionID,
		chain.Domain,
		chain.Complexity,
		string(keywordsJSON),
		chain.Model,
		chain.Reasoning,
		chain.Answer,
		chain.Confidence,
		string(metadataJSON),
		chain.CreatedAt,
		chain.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("插入思维链失败: %w", err)
	}

	// 删除旧步骤
	_, err = tx.ExecContext(ctx, "DELETE FROM chain_steps WHERE chain_id = ?", chain.ID)
	if err != nil {
		return fmt.Errorf("删除旧步骤失败: %w", err)
	}

	// 插入新步骤
	for _, step := range chain.Steps {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO chain_steps (chain_id, step_id, description, action, content, result) VALUES (?, ?, ?, ?, ?, ?)",
			chain.ID, step.StepID, step.Description, step.Action, step.Content, step.Result)
		if err != nil {
			return fmt.Errorf("插入步骤失败: %w", err)
		}
	}

	return tx.Commit()
}

// RetrieveSimilar 检索相似思维链
func (s *SQLiteStore) RetrieveSimilar(ctx context.Context, question string, domain string, maxResults int, config *KnowledgeBaseConfig) ([]*ChainOfThought, error) {
	// 查询所有思维链
	query := "SELECT id, question, question_id, domain, complexity, keywords, model, reasoning, answer, confidence, metadata, created_at, updated_at FROM chains"

	// 如果指定了领域，添加过滤
	args := []interface{}{}
	if domain != "" {
		query += " WHERE domain = ?"
		args = append(args, domain)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	// 计算相似度并排序
	type scoredChain struct {
		chain      *ChainOfThought
		similarity float64
	}

	scoredChains := make([]scoredChain, 0)

	for rows.Next() {
		chain, err := s.scanChain(rows)
		if err != nil {
			continue
		}

		// 计算相似度
		similarity := s.calculateSimilarity(question, domain, chain, config)
		if similarity >= config.MinSimilarity {
			scoredChains = append(scoredChains, scoredChain{
				chain:      chain,
				similarity: similarity,
			})
		}
	}

	// 按相似度排序
	sort.Slice(scoredChains, func(i, j int) bool {
		return scoredChains[i].similarity > scoredChains[j].similarity
	})

	// 取 Top K
	results := make([]*ChainOfThought, 0, maxResults)
	for i := 0; i < len(scoredChains) && i < maxResults; i++ {
		results = append(results, scoredChains[i].chain)
	}

	// 加载步骤
	for _, chain := range results {
		steps, err := s.loadSteps(ctx, chain.ID)
		if err == nil {
			chain.Steps = steps
		}
	}

	return results, nil
}

// scanChain 从行扫描思维链
func (s *SQLiteStore) scanChain(rows *sql.Rows) (*ChainOfThought, error) {
	var chain ChainOfThought
	var keywordsJSON, metadataJSON string
	var createdAt, updatedAt string

	err := rows.Scan(
		&chain.ID,
		&chain.Question,
		&chain.QuestionID,
		&chain.Domain,
		&chain.Complexity,
		&keywordsJSON,
		&chain.Model,
		&chain.Reasoning,
		&chain.Answer,
		&chain.Confidence,
		&metadataJSON,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	// 解析 JSON
	json.Unmarshal([]byte(keywordsJSON), &chain.Keywords)
	json.Unmarshal([]byte(metadataJSON), &chain.Metadata)

	// 解析时间
	chain.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	chain.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &chain, nil
}

// loadSteps 加载思维链步骤
func (s *SQLiteStore) loadSteps(ctx context.Context, chainID string) ([]ReasoningStep, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT step_id, description, action, content, result FROM chain_steps WHERE chain_id = ? ORDER BY step_id",
		chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	steps := make([]ReasoningStep, 0)
	for rows.Next() {
		var step ReasoningStep
		err := rows.Scan(&step.StepID, &step.Description, &step.Action, &step.Content, &step.Result)
		if err != nil {
			continue
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// calculateSimilarity 计算相似度（复用 retriever 的逻辑）
func (s *SQLiteStore) calculateSimilarity(question string, domain string, chain *ChainOfThought, config *KnowledgeBaseConfig) float64 {
	similarity := 0.0

	// 提取问题关键词（简单实现：按空格分割）
	questionKeywords := extractKeywords(question)

	// 关键词相似度（Jaccard）
	if len(questionKeywords) > 0 && len(chain.Keywords) > 0 {
		keywordSim := calculateKeywordSimilarity(questionKeywords, chain.Keywords)
		similarity += keywordSim * config.WeightKeywords
	}

	// 领域相似度
	if domain != "" && chain.Domain != "" {
		if domain == chain.Domain {
			similarity += config.WeightDomain
		} else if strings.Contains(chain.Domain, strings.Split(domain, "_")[0]) {
			similarity += config.WeightDomain * 0.5
		}
	}

	// 复杂度相似度
	complexity := "" // 可以从问题分析中获取，这里简化处理
	if complexity != "" && chain.Complexity != "" {
		if complexity == chain.Complexity {
			similarity += config.WeightComplexity
		} else {
			complexityOrder := map[string]int{"low": 1, "medium": 2, "high": 3}
			if c1, c2 := complexityOrder[complexity], complexityOrder[chain.Complexity]; c1 > 0 && c2 > 0 {
				if math.Abs(float64(c1-c2)) == 1 {
					similarity += config.WeightComplexity * 0.5
				}
			}
		}
	}

	// 归一化
	maxWeight := config.WeightKeywords + config.WeightDomain + config.WeightComplexity
	if maxWeight > 0 {
		similarity = similarity / maxWeight
	}

	return similarity
}

// GetChainCount 获取思维链数量
func (s *SQLiteStore) GetChainCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chains").Scan(&count)
	return count, err
}

// Close 关闭数据库连接
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// 辅助函数

// extractKeywords 从问题中提取关键词（简单实现）
func extractKeywords(question string) []string {
	// 移除标点符号
	question = strings.ToLower(question)
	words := strings.Fields(question)

	// 过滤停用词（简化版）
	stopWords := map[string]bool{
		"的": true, "是": true, "在": true, "有": true, "和": true,
		"the": true, "is": true, "a": true, "an": true, "and": true,
	}

	keywords := make([]string, 0)
	for _, word := range words {
		if !stopWords[word] && len(word) > 1 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// calculateKeywordSimilarity 计算关键词相似度（Jaccard）
func calculateKeywordSimilarity(keywords1, keywords2 []string) float64 {
	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, kw := range keywords1 {
		set1[strings.ToLower(kw)] = true
	}
	for _, kw := range keywords2 {
		set2[strings.ToLower(kw)] = true
	}

	intersection := 0
	for kw := range set1 {
		if set2[kw] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

package knowledgebase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// ChainOfThought 表示一个完整的解题思维链
type ChainOfThought struct {
	ID         string   `json:"id"`
	Question   string   `json:"question"`    // 原始问题
	QuestionID string   `json:"question_id"` // HLE 测试集问题ID
	Domain     string   `json:"domain"`      // 领域（数学、编程等）
	Complexity string   `json:"complexity"`  // 难度（low/medium/high）
	Keywords   []string `json:"keywords"`    // 关键词
	Model      string   `json:"model"`       // 来源模型（gpt-4, claude等）

	// 思维链内容
	Reasoning  string          `json:"reasoning"`  // 完整推理过程
	Steps      []ReasoningStep `json:"steps"`      // 结构化步骤
	Answer     string          `json:"answer"`     // 最终答案
	Confidence float64         `json:"confidence"` // 置信度

	// 元数据
	Metadata  map[string]interface{} `json:"metadata"` // 扩展元数据
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ReasoningStep 表示思维链中的一个步骤
type ReasoningStep struct {
	StepID      int    `json:"step_id"`
	Description string `json:"description"` // 步骤描述
	Action      string `json:"action"`      // 操作类型（think, calculate, reason等）
	Content     string `json:"content"`     // 步骤内容
	Result      string `json:"result"`      // 步骤结果
}

// KnowledgeBase 知识库接口
type KnowledgeBase interface {
	// 存储思维链
	AddChain(ctx context.Context, chain *ChainOfThought) error

	// 检索相似思维链
	RetrieveSimilar(ctx context.Context, question string, domain string, maxResults int) ([]*ChainOfThought, error)

	// 格式化思维链为 LLM 上下文
	FormatChainsForLLM(chains []*ChainOfThought) string

	// 批量导入
	ImportChains(ctx context.Context, chains []*ChainOfThought) error

	// 获取思维链数量
	GetChainCount(ctx context.Context) (int, error)

	// 关闭连接
	Close() error
}

// KnowledgeBaseConfig 知识库配置
type KnowledgeBaseConfig struct {
	StoragePath      string  `yaml:"storage_path"`
	MaxResults       int     `yaml:"max_results"`
	MinSimilarity    float64 `yaml:"min_similarity"`
	WeightKeywords   float64 `yaml:"weight_keywords"`
	WeightDomain     float64 `yaml:"weight_domain"`
	WeightComplexity float64 `yaml:"weight_complexity"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *KnowledgeBaseConfig {
	return &KnowledgeBaseConfig{
		StoragePath:      "./data/knowledge_base.db",
		MaxResults:       3,
		MinSimilarity:    0.5,
		WeightKeywords:   0.4,
		WeightDomain:     0.4,
		WeightComplexity: 0.2,
	}
}

// NewKnowledgeBase 创建知识库实例
func NewKnowledgeBase(cfg *KnowledgeBaseConfig) (KnowledgeBase, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	logger := logging.WithComponent("KnowledgeBase")
	logger.Info("初始化知识库",
		zap.String("storage_path", cfg.StoragePath))

	// 创建 SQLite 存储实现
	store, err := NewSQLiteStore(cfg.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("创建存储失败: %w", err)
	}

	kb := &knowledgeBaseImpl{
		logger: logger,
		config: cfg,
		store:  store,
	}

	return kb, nil
}

// knowledgeBaseImpl 知识库实现
type knowledgeBaseImpl struct {
	logger *zap.Logger
	config *KnowledgeBaseConfig
	store  *SQLiteStore
}

// AddChain 添加思维链
func (kb *knowledgeBaseImpl) AddChain(ctx context.Context, chain *ChainOfThought) error {
	if chain.ID == "" {
		chain.ID = generateChainID()
	}
	if chain.CreatedAt.IsZero() {
		chain.CreatedAt = time.Now()
	}
	chain.UpdatedAt = time.Now()

	err := kb.store.SaveChain(ctx, chain)
	if err != nil {
		kb.logger.Error("保存思维链失败", zap.Error(err), zap.String("chain_id", chain.ID))
		return err
	}

	kb.logger.Debug("保存思维链成功",
		zap.String("chain_id", chain.ID),
		zap.String("question_id", chain.QuestionID),
		zap.String("domain", chain.Domain))

	return nil
}

// RetrieveSimilar 检索相似思维链
func (kb *knowledgeBaseImpl) RetrieveSimilar(ctx context.Context, question string, domain string, maxResults int) ([]*ChainOfThought, error) {
	if maxResults <= 0 {
		maxResults = kb.config.MaxResults
	}

	// 从存储中检索
	chains, err := kb.store.RetrieveSimilar(ctx, question, domain, maxResults, kb.config)
	if err != nil {
		kb.logger.Error("检索思维链失败", zap.Error(err))
		return nil, err
	}

	kb.logger.Debug("检索思维链完成",
		zap.Int("count", len(chains)),
		zap.String("domain", domain))

	return chains, nil
}

// FormatChainsForLLM 格式化思维链为 LLM 上下文
func (kb *knowledgeBaseImpl) FormatChainsForLLM(chains []*ChainOfThought) string {
	if len(chains) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("=== 参考解题思路 ===\n\n")

	for i, chain := range chains {
		builder.WriteString(fmt.Sprintf("========== 参考示例 %d ==========\n", i+1))
		builder.WriteString(fmt.Sprintf("问题：%s\n\n", chain.Question))

		// 格式化推理过程
		if chain.Reasoning != "" {
			builder.WriteString("推理过程：\n")
			builder.WriteString(chain.Reasoning)
			builder.WriteString("\n\n")
		}

		// 格式化结构化步骤
		if len(chain.Steps) > 0 {
			builder.WriteString("解题步骤：\n")
			for j, step := range chain.Steps {
				builder.WriteString(fmt.Sprintf("步骤 %d: %s\n", j+1, step.Description))
				if step.Content != "" {
					builder.WriteString(fmt.Sprintf("  内容: %s\n", step.Content))
				}
				if step.Result != "" {
					builder.WriteString(fmt.Sprintf("  结果: %s\n", step.Result))
				}
			}
			builder.WriteString("\n")
		}

		builder.WriteString(fmt.Sprintf("最终答案：%s\n", chain.Answer))
		if chain.Model != "" {
			builder.WriteString(fmt.Sprintf("来源模型：%s\n", chain.Model))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// ImportChains 批量导入思维链
func (kb *knowledgeBaseImpl) ImportChains(ctx context.Context, chains []*ChainOfThought) error {
	kb.logger.Info("开始批量导入思维链", zap.Int("count", len(chains)))

	for i, chain := range chains {
		if err := kb.AddChain(ctx, chain); err != nil {
			kb.logger.Error("导入思维链失败",
				zap.Int("index", i),
				zap.String("chain_id", chain.ID),
				zap.Error(err))
			return fmt.Errorf("导入第 %d 条思维链失败: %w", i+1, err)
		}
	}

	kb.logger.Info("批量导入完成", zap.Int("count", len(chains)))
	return nil
}

// GetChainCount 获取思维链数量
func (kb *knowledgeBaseImpl) GetChainCount(ctx context.Context) (int, error) {
	return kb.store.GetChainCount(ctx)
}

// Close 关闭连接
func (kb *knowledgeBaseImpl) Close() error {
	return kb.store.Close()
}

// generateChainID 生成思维链ID
func generateChainID() string {
	return fmt.Sprintf("chain_%d", time.Now().UnixNano())
}

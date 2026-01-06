package retriever

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/hle-agent/hle-agent/pkg/logging"
	"github.com/hle-agent/hle-agent/pkg/memory"
	"go.uber.org/zap"
)

// Retriever retrieves similar knowledge from memory
type Retriever struct {
	logger *zap.Logger
	memory *memory.ShortTermMemory
	config *RetrieverConfig
}

// RetrieverConfig represents retriever configuration
type RetrieverConfig struct {
	MaxResults        int     `yaml:"max_results"`
	MinSimilarity     float64 `yaml:"min_similarity"`
	WeightKeywords    float64 `yaml:"weight_keywords"`
	WeightDomain      float64 `yaml:"weight_domain"`
	WeightComplexity  float64 `yaml:"weight_complexity"`
}

// NewRetriever creates a new Retriever
func NewRetriever(mem *memory.ShortTermMemory, cfg *RetrieverConfig) *Retriever {
	if cfg == nil {
		cfg = &RetrieverConfig{
			MaxResults:       3,
			MinSimilarity:    0.3,
			WeightKeywords:   0.4,
			WeightDomain:     0.4,
			WeightComplexity: 0.2,
		}
	}

	return &Retriever{
		logger: logging.WithComponent("Retriever"),
		memory: mem,
		config: cfg,
	}
}

// RetrievedKnowledge represents retrieved knowledge
type RetrievedKnowledge struct {
	Similarity float64           `json:"similarity"`
	Question   string            `json:"question"`
	Domain     string            `json:"domain"`
	Steps      []RetrievedStep   `json:"steps"`
	Answer     string            `json:"answer"`
	Confidence float64           `json:"confidence"`
	Reasoning  string            `json:"reasoning"`
}

// RetrievedStep represents a step from retrieved knowledge
type RetrievedStep struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Action      string `json:"action"`
	ToolName    string `json:"tool_name,omitempty"`
	Success     bool   `json:"success"`
}

// scoredEntry represents a scored memory entry
type scoredEntry struct {
	entry     memory.MemoryEntry
	similarity float64
}

// Retrieve retrieves similar knowledge for the given question
func (r *Retriever) Retrieve(question string, domain string, keywords []string, complexity string) []*RetrievedKnowledge {
	history := r.memory.GetRecentHistory(20)

	if len(history) == 0 {
		r.logger.Debug("没有历史记录可供检索")
		return nil
	}

	scoredEntries := make([]scoredEntry, 0)

	for _, entry := range history {
		if entry.Type != memory.EntryQuestion {
			continue
		}

		similarity := r.calculateSimilarity(entry, domain, keywords, complexity)
		if similarity >= r.config.MinSimilarity {
			scoredEntries = append(scoredEntries, scoredEntry{
				entry:     entry,
				similarity: similarity,
			})
		}
	}

	// Sort by similarity descending
	sort.Slice(scoredEntries, func(i, j int) bool {
		return scoredEntries[i].similarity > scoredEntries[j].similarity
	})

	// Take top K results
	results := make([]*RetrievedKnowledge, 0, r.config.MaxResults)
	for i := 0; i < len(scoredEntries) && i < r.config.MaxResults; i++ {
		knowledge := r.buildRetrievedKnowledge(scoredEntries[i])
		results = append(results, knowledge)
	}

	r.logger.Debug("知识检索完成",
		zap.Int("history_count", len(history)),
		zap.Int("retrieved_count", len(results)))

	return results
}

// calculateSimilarity calculates similarity between current question and historical question
func (r *Retriever) calculateSimilarity(entry memory.MemoryEntry, domain string, keywords []string, complexity string) float64 {
	similarity := 0.0

	// Get keywords from metadata
	entryKeywords := extractKeywordsFromMetadata(entry.Metadata)

	// Keyword similarity (Jaccard index)
	if len(keywords) > 0 && len(entryKeywords) > 0 {
		keywordSimilarity := r.calculateKeywordSimilarity(keywords, entryKeywords)
		similarity += keywordSimilarity * r.config.WeightKeywords
	}

	// Get domain and complexity from metadata
	var entryDomain, entryComplexity string
	if metadata, ok := entry.Metadata.(map[string]interface{}); ok {
		if domainStr, ok := metadata["domain"].(string); ok {
			entryDomain = domainStr
		}
		if complexityStr, ok := metadata["complexity"].(string); ok {
			entryComplexity = complexityStr
		}
	}

	// Domain similarity
	if domain != "" && entryDomain != "" {
		if domain == entryDomain {
			similarity += r.config.WeightDomain
		} else {
			// Partial match for sub-domain
			if strings.Contains(entryDomain, strings.Split(domain, "_")[0]) {
				similarity += r.config.WeightDomain * 0.5
			}
		}
	}

	// Complexity similarity
	if complexity != "" && entryComplexity != "" {
		if complexity == entryComplexity {
			similarity += r.config.WeightComplexity
		} else {
			// Adjacent complexity levels
			complexityOrder := map[string]int{"low": 1, "medium": 2, "high": 3}
			if c1, c2 := complexityOrder[complexity], complexityOrder[entryComplexity]; c1 > 0 && c2 > 0 {
				if math.Abs(float64(c1-c2)) == 1 {
					similarity += r.config.WeightComplexity * 0.5
				}
			}
		}
	}

	// Normalize similarity
	maxWeight := r.config.WeightKeywords + r.config.WeightDomain + r.config.WeightComplexity
	if maxWeight > 0 {
		similarity = similarity / maxWeight
	}

	return similarity
}

// extractKeywordsFromMetadata extracts keywords from metadata
func extractKeywordsFromMetadata(metadata interface{}) []string {
	if metadata == nil {
		return nil
	}
	if metadataMap, ok := metadata.(map[string]interface{}); ok {
		if keywordsRaw, ok := metadataMap["keywords"].([]interface{}); ok {
			keywords := make([]string, 0, len(keywordsRaw))
			for _, kw := range keywordsRaw {
				if kwStr, ok := kw.(string); ok {
					keywords = append(keywords, kwStr)
				}
			}
			return keywords
		}
	}
	return nil
}

// calculateKeywordSimilarity calculates Jaccard similarity between two keyword lists
func (r *Retriever) calculateKeywordSimilarity(keywords1, keywords2 []string) float64 {
	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0
	}

	// Convert to lowercase for comparison
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, kw := range keywords1 {
		set1[strings.ToLower(kw)] = true
	}
	for _, kw := range keywords2 {
		set2[strings.ToLower(kw)] = true
	}

	// Calculate intersection and union
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

// buildRetrievedKnowledge builds RetrievedKnowledge from historical entry
func (r *Retriever) buildRetrievedKnowledge(scored scoredEntry) *RetrievedKnowledge {
	entry := scored.entry

	knowledge := &RetrievedKnowledge{
		Similarity: scored.similarity,
		Question:   entry.Content,
		Confidence: entry.Confidence,
		Steps:      make([]RetrievedStep, 0),
	}

	// Get domain from metadata
	var entryDomain string
	if metadata, ok := entry.Metadata.(map[string]interface{}); ok {
		if domainStr, ok := metadata["domain"].(string); ok {
			knowledge.Domain = domainStr
			entryDomain = domainStr
		}
	}

	// Get step results for this question
	stepResults := r.memory.GetStepResults()
	for _, sr := range stepResults {
		if sr.StepID > 0 {
			knowledge.Steps = append(knowledge.Steps, RetrievedStep{
				ID:      sr.StepID,
				Success: sr.Success,
			})
		}
	}

	// Get the final answer from memory history
	recentHistory := r.memory.GetRecentHistory(5)
	for _, h := range recentHistory {
		if h.Type == memory.EntryAnswer {
			knowledge.Answer = h.Content
			break
		}
	}

	_ = entryDomain // suppress unused variable warning
	return knowledge
}

// BuildContextForPlanner builds retrieval context for the Planner
func (r *Retriever) BuildContextForPlanner(question string, domain string, keywords []string, complexity string) string {
	retrieved := r.Retrieve(question, domain, keywords, complexity)

	if len(retrieved) == 0 {
		return ""
	}

	var context strings.Builder
	context.WriteString("=== 相关历史问题参考 ===\n\n")

	for i, item := range retrieved {
		context.WriteString("==========\n")
		context.WriteString(fmt.Sprintf("%d. 相似度: %.1f%%\n\n", i+1, item.Similarity*100))

		context.WriteString("问题：")
		context.WriteString(truncateString(item.Question, 200))
		context.WriteString("\n\n")

		if item.Answer != "" {
			context.WriteString("答案：")
			context.WriteString(truncateString(item.Answer, 300))
			context.WriteString("\n\n")
		}

		if len(item.Steps) > 0 {
			context.WriteString(fmt.Sprintf("步骤数：%d\n", len(item.Steps)))
		}

		context.WriteString("\n")
	}

	return context.String()
}

// truncateString truncates a string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

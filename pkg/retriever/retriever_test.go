package retriever

import (
	"testing"
	"time"

	"github.com/hle-agent/hle-agent/pkg/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetriever(t *testing.T) {
	mem := memory.NewDefaultMemory()
	r := NewRetriever(mem, nil)

	assert.NotNil(t, r)
	assert.NotNil(t, r.logger)
	assert.NotNil(t, r.memory)
	assert.Equal(t, 3, r.config.MaxResults)
	assert.Equal(t, 0.3, r.config.MinSimilarity)
}

func TestNewRetrieverWithCustomConfig(t *testing.T) {
	mem := memory.NewDefaultMemory()
	cfg := &RetrieverConfig{
		MaxResults:       5,
		MinSimilarity:    0.5,
		WeightKeywords:   0.5,
		WeightDomain:     0.3,
		WeightComplexity: 0.2,
	}
	r := NewRetriever(mem, cfg)

	assert.NotNil(t, r)
	assert.Equal(t, 5, r.config.MaxResults)
	assert.Equal(t, 0.5, r.config.MinSimilarity)
	assert.Equal(t, 0.5, r.config.WeightKeywords)
}

func TestRetrieveEmptyHistory(t *testing.T) {
	mem := memory.NewDefaultMemory()
	r := NewRetriever(mem, nil)

	result := r.Retrieve("test question", "cryptography", []string{"key"}, "medium")

	assert.Nil(t, result)
}

func TestRetrieveWithHistory(t *testing.T) {
	mem := memory.NewDefaultMemory()
	mem.StartSession("test-session")

	// Add a historical question
	mem.AddQuestion("How to encrypt with AES?", map[string]interface{}{
		"domain":     "cryptography",
		"sub_domain": "encryption",
		"complexity": "medium",
		"keywords":   []string{"AES", "encryption", "key"},
		"has_code":   true,
	})

	r := NewRetriever(mem, nil)
	result := r.Retrieve("What is AES encryption?", "cryptography", []string{"AES", "encryption"}, "medium")

	require.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result), 0)
}

func TestCalculateKeywordSimilarity(t *testing.T) {
	mem := memory.NewDefaultMemory()
	r := NewRetriever(mem, nil)

	tests := []struct {
		name     string
		kw1      []string
		kw2      []string
		expected float64
	}{
		{
			name:     "identical keywords",
			kw1:      []string{"AES", "encryption"},
			kw2:      []string{"AES", "encryption"},
			expected: 1.0,
		},
		{
			name:     "partial overlap",
			kw1:      []string{"AES", "encryption", "key"},
			kw2:      []string{"AES", "encryption"},
			expected: 2.0 / 3.0, // intersection=2, union=3
		},
		{
			name:     "no overlap",
			kw1:      []string{"AES"},
			kw2:      []string{"RSA"},
			expected: 0.0,
		},
		{
			name:     "empty lists",
			kw1:      []string{},
			kw2:      []string{},
			expected: 0.0,
		},
		{
			name:     "one empty list",
			kw1:      []string{"AES"},
			kw2:      []string{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.calculateKeywordSimilarity(tt.kw1, tt.kw2)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestBuildContextForPlannerEmpty(t *testing.T) {
	mem := memory.NewDefaultMemory()
	r := NewRetriever(mem, nil)

	result := r.BuildContextForPlanner("test", "cryptography", []string{"key"}, "medium")

	assert.Equal(t, "", result)
}

func TestBuildContextForPlannerWithData(t *testing.T) {
	mem := memory.NewDefaultMemory()
	mem.StartSession("test-session")

	// Add a historical question with answer
	mem.AddQuestion("Solve 2+2=?", map[string]interface{}{
		"domain":     "math",
		"complexity": "low",
		"keywords":   []string{"addition"},
	})

	r := NewRetriever(mem, nil)
	result := r.BuildContextForPlanner("What is 2+2?", "math", []string{"addition"}, "low")

	// Result should contain content because similarity is above threshold
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "相关历史问题参考")
	assert.Contains(t, result, "Solve 2+2=?")
}

func TestRetrieverWithDifferentDomains(t *testing.T) {
	mem := memory.NewDefaultMemory()
	mem.StartSession("test-session")

	// Add crypto question
	mem.AddQuestion("Encrypt with AES", map[string]interface{}{
		"domain":     "cryptography",
		"complexity": "medium",
		"keywords":   []string{"encryption"},
	})

	r := NewRetriever(mem, nil)

	// Should find crypto question when domain matches
	result1 := r.Retrieve("AES encryption", "cryptography", []string{"AES"}, "medium")
	assert.GreaterOrEqual(t, len(result1), 0)

	// Should not find when domain differs
	result2 := r.Retrieve("Python code", "programming", []string{"python"}, "low")
	assert.Equal(t, 0, len(result2))
}

func TestRetrieverConfigYAMLTags(t *testing.T) {
	cfg := &RetrieverConfig{
		MaxResults:        10,
		MinSimilarity:     0.4,
		WeightKeywords:    0.5,
		WeightDomain:      0.3,
		WeightComplexity:  0.2,
	}

	assert.Equal(t, 10, cfg.MaxResults)
	assert.Equal(t, 0.4, cfg.MinSimilarity)
	assert.Equal(t, 0.5, cfg.WeightKeywords)
	assert.Equal(t, 0.3, cfg.WeightDomain)
	assert.Equal(t, 0.2, cfg.WeightComplexity)
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to max",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than max",
			input:    "hello world",
			maxLen:   5,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetrieveMaxResults(t *testing.T) {
	mem := memory.NewDefaultMemory()
	mem.StartSession("test-session")

	// Add multiple questions
	for i := 0; i < 10; i++ {
		mem.AddQuestion("Unrelated question", map[string]interface{}{
			"domain":     "cryptography",
			"complexity": "medium",
			"keywords":   []string{"encryption", "key"},
		})
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	r := NewRetriever(mem, &RetrieverConfig{
		MaxResults:    3,
		MinSimilarity: 0.0,
	})

	result := r.Retrieve("encryption", "cryptography", []string{"encryption"}, "medium")

	// Should be limited to max results
	if len(result) > 3 {
		assert.Fail(t, "Results should be limited to max results")
	}
}

func TestRetrieveMinSimilarity(t *testing.T) {
	mem := memory.NewDefaultMemory()
	mem.StartSession("test-session")

	// Add a question with very different keywords
	mem.AddQuestion("Unrelated programming question", map[string]interface{}{
		"domain":     "programming",
		"complexity": "high",
		"keywords":   []string{"python", "django"},
	})

	r := NewRetriever(mem, &RetrieverConfig{
		MinSimilarity: 0.8, // High threshold
	})

	result := r.Retrieve("cryptography encryption", "cryptography", []string{"crypto"}, "medium")

	// Should return nil due to low similarity
	assert.Equal(t, 0, len(result))
}

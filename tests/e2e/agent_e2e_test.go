package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hle-agent/hle-agent/internal/agent"
	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_ProcessSimpleQuestion tests the complete flow with a simple question
// This test uses the real MiniMax API from config.yaml
func TestE2E_ProcessSimpleQuestion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if API key is available
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		// Try to load from config
		cfg, err := config.Load("../../config.yaml")
		if err != nil || cfg.Model.APIKey == "" {
			t.Skip("API key not available for E2E testing")
			return
		}
		apiKey = cfg.Model.APIKey
	}

	// Load config
	cfg, err := config.Load("../../config.yaml")
	require.NoError(t, err)

	// Override API key if provided via environment
	if apiKey != "" {
		cfg.Model.APIKey = apiKey
	}

	// Create agent
	hleAgent, err := agent.NewHLEAgent(cfg)
	require.NoError(t, err)

	// Test with a simple question
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := hleAgent.Process(ctx, "What is 2+2?")

	require.NoError(t, err, "Process should not return error")
	require.NotNil(t, result, "Result should not be nil")

	assert.NotEmpty(t, result.Answer, "Answer should not be empty")
	assert.Greater(t, result.Confidence, 0.0, "Confidence should be greater than 0")
	assert.Greater(t, result.TotalDuration, 0.0, "Duration should be greater than 0")

	t.Logf("Answer: %s", result.Answer)
	t.Logf("Confidence: %.2f%%", result.Confidence*100)
	t.Logf("Duration: %.2fs", result.TotalDuration)
	if result.Explanation != "" {
		t.Logf("Explanation: %s", result.Explanation)
	}
}

// TestE2E_ProcessMathQuestion tests with a math question
func TestE2E_ProcessMathQuestion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil || cfg.Model.APIKey == "" {
		t.Skip("Config or API key not available for E2E testing")
		return
	}

	hleAgent, err := agent.NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := hleAgent.Process(ctx, "Calculate 15 * 23")

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Answer)
	assert.Contains(t, result.Answer, "345", "Answer should contain 345")

	t.Logf("Answer: %s", result.Answer)
}

// TestE2E_ProcessBatch tests batch processing
func TestE2E_ProcessBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil || cfg.Model.APIKey == "" {
		t.Skip("Config or API key not available for E2E testing")
		return
	}

	hleAgent, err := agent.NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	questions := []string{
		"What is 2+2?",
		"What is 3*4?",
	}

	results, err := hleAgent.ProcessBatch(ctx, questions)

	require.NoError(t, err)
	require.Len(t, results, 2)

	for i, result := range results {
		assert.NotNil(t, result, "Result %d should not be nil", i)
		if result != nil {
			assert.NotEmpty(t, result.Answer, "Answer %d should not be empty", i)
			t.Logf("Question %d Answer: %s", i+1, result.Answer)
		}
	}
}

// TestE2E_ProcessWithContextCancellation tests graceful cancellation
func TestE2E_ProcessWithContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil || cfg.Model.APIKey == "" {
		t.Skip("Config or API key not available for E2E testing")
		return
	}

	hleAgent, err := agent.NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait a bit to ensure context is cancelled
	time.Sleep(10 * time.Millisecond)

	result, err := hleAgent.Process(ctx, "What is 2+2?")

	// Should handle cancellation gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "context", "Error should be related to context cancellation")
	} else {
		// If no error, result might be nil or partial
		_ = result
	}
}

// TestE2E_ProcessComplexQuestion tests with a more complex question
func TestE2E_ProcessComplexQuestion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil || cfg.Model.APIKey == "" {
		t.Skip("Config or API key not available for E2E testing")
		return
	}

	hleAgent, err := agent.NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	question := "Explain the difference between symmetric and asymmetric encryption, and give an example of each."

	result, err := hleAgent.Process(ctx, question)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Answer)
	assert.Greater(t, len(result.Answer), 50, "Answer should be detailed")

	t.Logf("Answer length: %d", len(result.Answer))
	t.Logf("Answer preview: %.100s...", result.Answer)
}

// TestE2E_ProcessWithKnowledgeBase tests with knowledge base enabled
func TestE2E_ProcessWithKnowledgeBase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil || cfg.Model.APIKey == "" {
		t.Skip("Config or API key not available for E2E testing")
		return
	}

	// Enable knowledge base if not already enabled
	if !cfg.KnowledgeBase.Enabled {
		t.Skip("Knowledge base not enabled in config")
		return
	}

	hleAgent, err := agent.NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := hleAgent.Process(ctx, "What is 2+2?")

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Answer)

	t.Logf("Answer with KB: %s", result.Answer)
}

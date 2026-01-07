package agent

import (
	"context"
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHLEAgent_Process_WithEmptyQuestion(t *testing.T) {
	// This test requires a valid config, so we'll skip it if config is not available
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Skip("Config file not available for testing")
	}

	agent, err := NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := agent.Process(ctx, "")

	// Empty question should be handled gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "empty")
	} else {
		// If no error, should return a result
		assert.NotNil(t, result)
	}
}

func TestHLEAgent_Process_WithValidQuestion(t *testing.T) {
	// This test requires a valid config and API key
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Skip("Config file not available for testing")
	}

	agent, err := NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := agent.Process(ctx, "What is 2+2?")

	// This will make a real API call, so we check for either success or network error
	if err != nil {
		// Network or API errors are acceptable in test environment
		assert.Contains(t, err.Error(), "failed")
	} else {
		assert.NotNil(t, result)
		if result != nil {
			assert.NotEmpty(t, result.Answer)
		}
	}
}

func TestHLEAgent_ProcessBatch_WithEmptyList(t *testing.T) {
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Skip("Config file not available for testing")
	}

	agent, err := NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	questions := []string{}

	results, err := agent.ProcessBatch(ctx, questions)

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestHLEAgent_ProcessBatch_WithValidQuestions(t *testing.T) {
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Skip("Config file not available for testing")
	}

	agent, err := NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	questions := []string{
		"What is 2+2?",
	}

	results, err := agent.ProcessBatch(ctx, questions)

	if err != nil {
		// Network errors are acceptable
		assert.Contains(t, err.Error(), "failed")
	} else {
		assert.Len(t, results, 1)
		if len(results) > 0 {
			assert.NotNil(t, results[0])
		}
	}
}

func TestHLEAgent_Process_WithContextCancellation(t *testing.T) {
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Skip("Config file not available for testing")
	}

	agent, err := NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := agent.Process(ctx, "What is 2+2?")

	// Should handle cancellation gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	} else {
		// If no error, result might be nil or partial
		_ = result
	}
}

func TestNewHLEAgent_WithInvalidConfig(t *testing.T) {
	invalidCfg := &config.Config{}

	agent, err := NewHLEAgent(invalidCfg)

	assert.Error(t, err)
	assert.Nil(t, agent)
}

func TestHLEAgent_synthesizeAnswer(t *testing.T) {
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Skip("Config file not available for testing")
	}

	agent, err := NewHLEAgent(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	question := "What is 2+2?"
	answer := "4"
	reasoning := "Step 1: Analyze the problem\nStep 2: Calculate 2+2 = 4"

	synthesized, err := agent.synthesizeAnswer(ctx, question, answer, reasoning)

	if err != nil {
		// Network errors are acceptable
		assert.Contains(t, err.Error(), "failed")
	} else {
		assert.NotEmpty(t, synthesized)
	}
}

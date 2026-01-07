package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_Execute_WithNilStep(t *testing.T) {
	registry := NewToolRegistry()
	executor := NewExecutor(registry, nil, nil)

	plan := &HLEPlan{ID: "plan_001"}
	result, err := executor.Execute(context.Background(), plan, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "nil")
}

func TestExecutor_Execute_WithLLMAction(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		BaseURL:  "http://localhost:9999",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	executor := NewTestExecutor(client, nil)

	plan := &HLEPlan{ID: "plan_001"}
	step := &HLEStep{
		ID:          "step_1",
		Name:        "Calculate",
		Description: "What is 2+2?",
		Action:      "llm",
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, plan, step)

	// Will fail due to network, but tests the structure
	if err != nil {
		// Expected due to network error
		assert.Contains(t, err.Error(), "LLM")
	} else {
		assert.NotNil(t, result)
		if result != nil {
			assert.Equal(t, "step_1", result.StepID)
		}
	}
}

func TestExecutor_Execute_WithToolAction(t *testing.T) {
	mockTool := &testTool{
		name:        "test_tool",
		description: "Test tool",
		result:      "tool output",
	}

	registry := NewToolRegistry()
	registry.Register(mockTool)

	executor := NewExecutor(registry, nil, nil)

	plan := &HLEPlan{ID: "plan_001"}
	step := &HLEStep{
		ID:          "step_1",
		Description: "Execute tool",
		Action:      "tool",
		ToolName:    "test_tool",
		Params:      "test params",
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, plan, step)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "tool output", result.Output)
}

func TestExecutor_Execute_WithToolNotFound(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		BaseURL:  "http://localhost:9999",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	registry := NewToolRegistry()
	executor := NewExecutor(registry, client, nil)

	plan := &HLEPlan{ID: "plan_001"}
	step := &HLEStep{
		ID:          "step_1",
		Description: "Execute tool",
		Action:      "tool",
		ToolName:    "non_existent_tool",
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, plan, step)

	// Should fallback to LLM
	if err != nil {
		// Network error expected
		assert.Contains(t, err.Error(), "LLM")
	} else {
		assert.NotNil(t, result)
	}
}

// RetryableMockTool is a mock tool that can simulate retry behavior
type RetryableMockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, params string) (string, error)
}

func (m *RetryableMockTool) Name() string {
	return m.name
}

func (m *RetryableMockTool) Description() string {
	return m.description
}

func (m *RetryableMockTool) Execute(ctx context.Context, params string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return "", errors.New("not implemented")
}

func TestExecutor_Execute_WithToolRetry(t *testing.T) {
	attemptCount := 0
	mockTool := &RetryableMockTool{
		name: "test_tool",
		executeFunc: func(ctx context.Context, params string) (string, error) {
			attemptCount++
			if attemptCount < 2 {
				return "", errors.New("tool error")
			}
			return "success after retry", nil
		},
	}

	registry := NewToolRegistry()
	registry.Register(mockTool)

	cfg := &ExecutorConfig{
		MaxRetries:     3,
		RetryBackoff:   10 * time.Millisecond,
		EnableFallback: false,
	}

	executor := NewExecutor(registry, nil, cfg)

	plan := &HLEPlan{ID: "plan_001"}
	step := &HLEStep{
		ID:          "step_1",
		Description: "Execute tool",
		Action:      "tool",
		ToolName:    "test_tool",
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, plan, step)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "success after retry", result.Output)
	assert.Equal(t, 2, attemptCount)
}

// ErrorTool is a tool that always returns an error
type ErrorTool struct {
	name        string
	description string
}

func (e *ErrorTool) Name() string {
	return e.name
}

func (e *ErrorTool) Description() string {
	return e.description
}

func (e *ErrorTool) Execute(ctx context.Context, params string) (string, error) {
	return "", errors.New("tool error")
}

func TestExecutor_Execute_WithToolFailureAndFallback(t *testing.T) {
	mockTool := &ErrorTool{
		name:        "test_tool",
		description: "Test tool that fails",
	}

	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		BaseURL:  "http://localhost:9999",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	registry := NewToolRegistry()
	registry.Register(mockTool)

	executorCfg := &ExecutorConfig{
		MaxRetries:     1,
		RetryBackoff:   10 * time.Millisecond,
		EnableFallback: true,
	}

	executor := NewExecutor(registry, client, executorCfg)

	plan := &HLEPlan{ID: "plan_001"}
	step := &HLEStep{
		ID:          "step_1",
		Description: "Execute tool",
		Action:      "tool",
		ToolName:    "test_tool",
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, plan, step)

	// Should fallback to LLM (will fail due to network)
	if err != nil {
		// Network error expected
		assert.Contains(t, err.Error(), "LLM")
	} else {
		assert.NotNil(t, result)
	}
}

func TestExecutor_executeWithLLM_WithNilClient(t *testing.T) {
	registry := NewToolRegistry()
	executor := NewExecutor(registry, nil, nil)

	step := &HLEStep{
		ID:          "step_1",
		Description: "Test step",
	}

	result, err := executor.executeWithLLM(context.Background(), step)

	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "未初始化")
}

func TestExecutor_executeToolWithRetry_WithSuccess(t *testing.T) {
	mockTool := &testTool{
		name:        "test_tool",
		description: "Test tool",
		result:      "success",
	}

	registry := NewToolRegistry()
	executor := NewExecutor(registry, nil, nil)

	ctx := context.Background()
	result, err := executor.executeToolWithRetry(ctx, mockTool, "test params")

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestExecutor_executeToolWithRetry_WithFailure(t *testing.T) {
	mockTool := &ErrorTool{
		name:        "test_tool",
		description: "Test tool that fails",
	}

	registry := NewToolRegistry()
	cfg := &ExecutorConfig{
		MaxRetries:     2,
		RetryBackoff:   10 * time.Millisecond,
		EnableFallback: false,
	}
	executor := NewExecutor(registry, nil, cfg)

	ctx := context.Background()
	result, err := executor.executeToolWithRetry(ctx, mockTool, "test params")

	assert.Error(t, err)
	assert.Empty(t, result)
}


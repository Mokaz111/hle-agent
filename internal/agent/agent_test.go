package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.tools)
}

func TestToolRegistryGet(t *testing.T) {
	registry := NewToolRegistry()

	// Initially empty
	_, exists := registry.Get("python_executor")
	assert.False(t, exists)
}

func TestToolRegistryListEmpty(t *testing.T) {
	registry := NewToolRegistry()

	tools := registry.List()
	assert.Empty(t, tools)
}

func TestAnswerResultStruct(t *testing.T) {
	result := &AnswerResult{
		Answer:     "42",
		Confidence: 0.95,
	}

	assert.Equal(t, "42", result.Answer)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestExecutorCreation(t *testing.T) {
	registry := NewToolRegistry()
	executor := NewExecutor(registry)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.toolRegistry)
}

func TestAgentStructTypes(t *testing.T) {
	// Test that we can create basic structs
	assert.NotNil(t, &Planner{})
	assert.NotNil(t, &Executor{})
	assert.NotNil(t, &Replanner{})
	assert.NotNil(t, &ToolRegistry{})
}

func TestAnswerResultWithAllFields(t *testing.T) {
	result := &AnswerResult{
		Answer:      "The answer is 42",
		Explanation: "This is the explanation",
		Confidence:  0.95,
	}

	assert.Equal(t, "The answer is 42", result.Answer)
	assert.Equal(t, "This is the explanation", result.Explanation)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestToolRegistryWithMultipleTools(t *testing.T) {
	registry := NewToolRegistry()

	// Register a tool
	tool := &testTool{name: "test_tool", description: "Test tool"}
	registry.Register(tool)

	// Verify it was registered
	retrieved, exists := registry.Get("test_tool")
	assert.True(t, exists)
	assert.Equal(t, "test_tool", retrieved.Name())
}

func TestToolRegistryOverwrite(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := &testTool{name: "my_tool", description: "First"}
	tool2 := &testTool{name: "my_tool", description: "Second"}

	registry.Register(tool1)
	registry.Register(tool2)

	retrieved, _ := registry.Get("my_tool")
	assert.Equal(t, "Second", retrieved.Description())
}

// testTool is a simple test implementation of Tool
type testTool struct {
	name        string
	description string
	result      string
}

func (t *testTool) Name() string        { return t.name }
func (t *testTool) Description() string { return t.description }
func (t *testTool) Execute(ctx context.Context, params string) (string, error) {
	if t.result != "" {
		return t.result, nil
	}
	return "default", nil
}

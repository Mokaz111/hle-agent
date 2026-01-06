package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockTool is a mock implementation of the Tool interface for testing
type MockTool struct {
	name        string
	description string
	result      interface{}
	err         error
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return m.result, m.err
}

func TestNewExecutor(t *testing.T) {
	registry := NewToolRegistry()
	executor := NewExecutor(registry)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.toolRegistry)
	assert.NotNil(t, executor.logger)
	assert.Equal(t, 3, executor.maxRetries)
}

func TestExecutorWithCustomRegistry(t *testing.T) {
	registry := NewToolRegistry()
	executor := NewExecutor(registry)

	assert.Equal(t, registry, executor.toolRegistry)
}

func TestToolRegistryRegisterWithTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := &MockTool{
		name:        "test_tool",
		description: "A test tool",
		result:      "test result",
	}

	registry.Register(tool)

	retrieved, exists := registry.Get("test_tool")
	assert.True(t, exists)
	assert.Equal(t, "test_tool", retrieved.Name())
}

func TestToolRegistryListWithTools(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := &MockTool{name: "tool1", description: "Tool 1"}
	tool2 := &MockTool{name: "tool2", description: "Tool 2"}

	registry.Register(tool1)
	registry.Register(tool2)

	tools := registry.List()
	assert.Len(t, tools, 2)
	assert.Contains(t, tools, "tool1")
	assert.Contains(t, tools, "tool2")
}

func TestToolRegistryGetNonExistentTool(t *testing.T) {
	registry := NewToolRegistry()

	_, exists := registry.Get("non_existent")
	assert.False(t, exists)
}

func TestToolRegistryEmptyList(t *testing.T) {
	registry := NewToolRegistry()

	tools := registry.List()
	assert.Empty(t, tools)
}

func TestMockToolExecuteSuccess(t *testing.T) {
	tool := &MockTool{
		name:    "test",
		result:  "success",
		err:     nil,
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	assert.Equal(t, "success", result)
	assert.NoError(t, err)
}

func TestMockToolExecuteWithError(t *testing.T) {
	tool := &MockTool{
		name:   "test",
		result: nil,
		err:    assert.AnError,
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestMockToolNameAndDescription(t *testing.T) {
	tool := &MockTool{
		name:        "my_tool",
		description: "My custom tool description",
	}

	assert.Equal(t, "my_tool", tool.Name())
	assert.Equal(t, "My custom tool description", tool.Description())
}

func TestExecutorLogger(t *testing.T) {
	registry := NewToolRegistry()
	executor := NewExecutor(registry)

	assert.NotNil(t, executor.logger)
}

func TestToolRegistryLogger(t *testing.T) {
	registry := NewToolRegistry()

	assert.NotNil(t, registry.logger)
}

func TestExecutorMaxRetries(t *testing.T) {
	registry := NewToolRegistry()
	executor := NewExecutor(registry)

	assert.Equal(t, 3, executor.maxRetries)
}

func TestMockToolWithParams(t *testing.T) {
	tool := &MockTool{
		name:    "test",
		result:  "processed",
	}

	params := map[string]interface{}{
		"code":     "print('hello')",
		"language": "python",
	}

	result, err := tool.Execute(context.Background(), params)

	assert.Equal(t, "processed", result)
	assert.NoError(t, err)
}

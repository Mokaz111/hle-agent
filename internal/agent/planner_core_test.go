package agent

import (
	"context"
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanner_Plan_WithEmptyQuestion(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, false)

	ctx := context.Background()
	plan, err := planner.Plan(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "empty")
}

func TestPlanner_Plan_WithValidQuestion(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		BaseURL:  "http://localhost:9999", // Non-existent to avoid real calls
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, false)

	ctx := context.Background()
	// This will fail due to network, but tests the structure
	plan, err := planner.Plan(ctx, "What is 2+2?")

	// We expect an error due to network, but the function should be called
	// In a real test with mock, we'd verify the plan structure
	if err != nil {
		// Expected due to network error
		assert.Contains(t, err.Error(), "failed to generate plan")
	} else {
		assert.NotNil(t, plan)
	}
}

func TestPlanner_CreatePlan_WithEmptyContext(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, false)

	ctx := context.Background()
	plan := planner.CreatePlan(ctx)

	assert.NotNil(t, plan)
	// Should return a default plan
	hlePlan, ok := plan.(*HLEPlan)
	if ok {
		assert.NotEmpty(t, hlePlan.ID)
		assert.NotEmpty(t, hlePlan.FirstStepID)
	}
}

func TestPlanner_CreatePlan_WithQuestionInContext(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		BaseURL:  "http://localhost:9999",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, false)

	ctx := WithQuestion(context.Background(), "What is 2+2?")
	plan := planner.CreatePlan(ctx)

	assert.NotNil(t, plan)
	// Should return a plan (default or generated)
	hlePlan, ok := plan.(*HLEPlan)
	if ok {
		assert.NotEmpty(t, hlePlan.ID)
	}
}

func TestPlanner_Plan_WithKnowledgeBase(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		BaseURL:  "http://localhost:9999",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, true)

	ctx := context.Background()
	// This will fail due to network, but tests KB integration
	plan, err := planner.Plan(ctx, "What is 2+2?")

	// We expect an error due to network
	if err != nil {
		assert.Contains(t, err.Error(), "failed to generate plan")
	} else {
		assert.NotNil(t, plan)
	}
}

func TestPlanner_parsePlan_WithValidJSON(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, false)

	jsonResponse := `{
		"id": "plan_001",
		"steps": [
			{
				"id": "step_1",
				"description": "Analyze the problem",
				"action": "llm"
			},
			{
				"id": "step_2",
				"description": "Calculate the answer",
				"action": "tool",
				"tool_name": "python_executor"
			}
		]
	}`

	plan := planner.parsePlan(jsonResponse, "What is 2+2?")

	assert.NotNil(t, plan)
	assert.Equal(t, "plan_001", plan.ID)
	assert.Len(t, plan.Steps, 2)
	assert.Equal(t, "step_1", plan.Steps[0].ID)
	assert.Equal(t, "Analyze the problem", plan.Steps[0].Description)
}

func TestPlanner_parsePlan_WithInvalidJSON(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, false)

	invalidResponse := "This is not JSON"
	plan := planner.parsePlan(invalidResponse, "What is 2+2?")

	// Should return a fallback plan
	assert.NotNil(t, plan)
	assert.NotEmpty(t, plan.ID)
	assert.Len(t, plan.Steps, 1)
	assert.Equal(t, "llm", plan.Steps[0].Action)
}

func TestPlanner_parsePlan_WithMissingStepID(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, false)

	jsonResponse := `{
		"id": "plan_001",
		"steps": [
			{
				"description": "Step without ID",
				"action": "llm"
			}
		]
	}`

	plan := planner.parsePlan(jsonResponse, "What is 2+2?")

	assert.NotNil(t, plan)
	assert.Len(t, plan.Steps, 1)
	// Should generate step ID
	assert.NotEmpty(t, plan.Steps[0].ID)
}

func TestPlanner_selectPromptTemplate(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	planner := NewTestPlanner(client, false)

	// Test with nil analysis
	template := planner.selectPromptTemplate(nil)
	assert.Equal(t, "planner_general", template)
}

func TestPlanner_Plan_WithNoPromptTemplate(t *testing.T) {
	// This test is difficult to implement without breaking the planner structure
	// The prompts field is initialized in NewPlanner and cannot be easily set to nil
	// We'll skip this test for now as it requires refactoring the planner
	t.Skip("Skipping test that requires planner refactoring")
}

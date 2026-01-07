package agent

import (
	"context"
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplanner_analyzeStepResults_WithEmptyResults(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	replanner := NewTestReplanner(client)

	analysis := replanner.analyzeStepResults([]*HLEStepResult{})

	assert.True(t, analysis.NeedReplan)
	assert.Equal(t, 0.0, analysis.AvgConfidence)
	assert.False(t, analysis.HasErrors)
	// AllSuccessful is initialized as true, but with no results it's technically true
	// (no failures means all successful, even if there are no results)
	assert.True(t, analysis.AllSuccessful)
}

func TestReplanner_analyzeStepResults_WithAllSuccessful(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	replanner := NewTestReplanner(client)

	stepResults := []*HLEStepResult{
		{StepID: "step_1", Success: true, Confidence: 0.9},
		{StepID: "step_2", Success: true, Confidence: 0.95},
		{StepID: "step_3", Success: true, Confidence: 0.85},
	}

	analysis := replanner.analyzeStepResults(stepResults)

	assert.False(t, analysis.NeedReplan)
	assert.InDelta(t, 0.9, analysis.AvgConfidence, 0.01)
	assert.False(t, analysis.HasErrors)
	assert.True(t, analysis.AllSuccessful)
}

func TestReplanner_analyzeStepResults_WithSomeFailed(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	replanner := NewTestReplanner(client)

	stepResults := []*HLEStepResult{
		{StepID: "step_1", Success: true, Confidence: 0.9},
		{StepID: "step_2", Success: false, Confidence: 0.0},
		{StepID: "step_3", Success: true, Confidence: 0.8},
	}

	analysis := replanner.analyzeStepResults(stepResults)

	assert.True(t, analysis.HasErrors)
	assert.False(t, analysis.AllSuccessful)
	assert.Contains(t, analysis.FailedSteps, "step_2")
}

func TestReplanner_analyzeStepResults_WithLowConfidence(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	replanner := NewTestReplanner(client)

	stepResults := []*HLEStepResult{
		{StepID: "step_1", Success: true, Confidence: 0.5},
		{StepID: "step_2", Success: true, Confidence: 0.6},
	}

	analysis := replanner.analyzeStepResults(stepResults)

	assert.True(t, analysis.NeedReplan) // Average confidence < 0.7
}

func TestReplanner_analyzeStepResults_WithAllFailed(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	replanner := NewTestReplanner(client)

	stepResults := []*HLEStepResult{
		{StepID: "step_1", Success: false, Confidence: 0.0},
		{StepID: "step_2", Success: false, Confidence: 0.0},
	}

	analysis := replanner.analyzeStepResults(stepResults)

	assert.True(t, analysis.NeedReplan)
	assert.True(t, analysis.HasErrors)
	assert.False(t, analysis.AllSuccessful)
	assert.Len(t, analysis.FailedSteps, 2)
}

func TestReplanner_Replan_WithEmptyResults(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		BaseURL:  "http://localhost:9999",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	replanner := NewTestReplanner(client)

	plan := &HLEPlan{
		ID:          "plan_001",
		FirstStepID: "step_1",
		Steps: []HLEStep{
			{ID: "step_1", Description: "Step 1"},
		},
	}

	ctx := context.Background()
	newPlan, err := replanner.Replan(ctx, plan, []*HLEStepResult{}, "test question")

	// Will fail due to network, but tests the structure
	if err != nil {
		assert.Contains(t, err.Error(), "failed to generate new plan")
	} else {
		assert.NotNil(t, newPlan)
	}
}

func TestReplanner_Replan_WithAllSuccessfulSteps(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	replanner := NewTestReplanner(client)

	plan := &HLEPlan{
		ID:          "plan_001",
		FirstStepID: "step_1",
		Steps: []HLEStep{
			{ID: "step_1", Description: "Step 1"},
			{ID: "step_2", Description: "Step 2"},
		},
	}

	stepResults := []*HLEStepResult{
		{StepID: "step_1", Success: true, Confidence: 0.9},
		{StepID: "step_2", Success: true, Confidence: 0.95},
	}

	ctx := context.Background()
	newPlan, err := replanner.Replan(ctx, plan, stepResults, "test question")

	assert.NoError(t, err)
	// When all steps are successful, should return original plan
	assert.Equal(t, plan, newPlan)
}

func TestReplanner_CreatePlan(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	require.NoError(t, err)

	replanner := NewTestReplanner(client)

	ctx := WithQuestion(context.Background(), "test question")
	plan := replanner.CreatePlan(ctx)

	assert.NotNil(t, plan)
	hlePlan, ok := plan.(*HLEPlan)
	if ok {
		assert.NotEmpty(t, hlePlan.ID)
	}
}


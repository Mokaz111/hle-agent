package agent

import (
	"context"
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/stretchr/testify/assert"
)

func TestNewReplanner(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client := llm.NewClient(cfg)

	replanner := NewReplanner(client)

	assert.NotNil(t, replanner)
	assert.NotNil(t, replanner.llmClient)
	assert.NotNil(t, replanner.prompts)
	assert.NotNil(t, replanner.logger)
}

func TestReplanWithEmptyResults(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	replanner := NewReplanner(llm.NewClient(cfg))

	plan := &models.Plan{
		ID:         "plan-001",
		TotalSteps: 2,
		Steps: []models.Step{
			{ID: 1, Description: "Step 1"},
			{ID: 2, Description: "Step 2"},
		},
	}

	result, err := replanner.Replan(context.Background(), plan, []*models.StepResult{}, "Test question")

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReplanWithNilResults(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	replanner := NewReplanner(llm.NewClient(cfg))

	plan := &models.Plan{
		ID:         "plan-001",
		TotalSteps: 1,
		Steps: []models.Step{
			{ID: 1, Description: "Step 1"},
		},
	}

	result, err := replanner.Replan(context.Background(), plan, nil, "Test question")

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReplanWithContext(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	replanner := NewReplanner(llm.NewClient(cfg))

	ctx := context.Background()
	plan := &models.Plan{
		ID:         "plan-001",
		TotalSteps: 1,
		Steps: []models.Step{
			{ID: 1, Description: "Step 1"},
		},
	}

	result, err := replanner.Replan(ctx, plan, nil, "Test question")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, ctx)
}

func TestReplanWithDifferentQuestionTypes(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	replanner := NewReplanner(llm.NewClient(cfg))

	questions := []string{
		"Explain RSA encryption",
		"Write Python code to sort a list",
		"Solve x^2 + 2x + 1 = 0",
		"What is the capital of France?",
	}

	for _, question := range questions {
		plan := &models.Plan{
			ID:         "plan-001",
			TotalSteps: 1,
			Steps: []models.Step{
				{ID: 1, Description: "Step 1"},
			},
		}

		result, err := replanner.Replan(context.Background(), plan, nil, question)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	}
}

func TestReplanLogger(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	replanner := NewReplanner(llm.NewClient(cfg))

	assert.NotNil(t, replanner.logger)
}

func TestReplanWithSingleStepPlan(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	replanner := NewReplanner(llm.NewClient(cfg))

	plan := &models.Plan{
		ID:         "plan-001",
		TotalSteps: 1,
		Steps: []models.Step{
			{ID: 1, Description: "Single step"},
		},
	}

	result, err := replanner.Replan(context.Background(), plan, nil, "Simple question")

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReplanWithMultipleSteps(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	replanner := NewReplanner(llm.NewClient(cfg))

	plan := &models.Plan{
		ID:         "plan-001",
		TotalSteps: 5,
		Steps: []models.Step{
			{ID: 1, Description: "Step 1"},
			{ID: 2, Description: "Step 2"},
			{ID: 3, Description: "Step 3"},
			{ID: 4, Description: "Step 4"},
			{ID: 5, Description: "Step 5"},
		},
	}

	result, err := replanner.Replan(context.Background(), plan, nil, "Complex question")

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

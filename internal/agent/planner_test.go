package agent

import (
	"context"
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/stretchr/testify/assert"
)

func TestNewPlanner(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client := llm.NewClient(cfg)

	planner := NewPlanner(client)

	assert.NotNil(t, planner)
	assert.NotNil(t, planner.llmClient)
	assert.NotNil(t, planner.prompts)
	assert.NotNil(t, planner.logger)
}

func TestPlannerWithContext(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	planner := NewPlanner(llm.NewClient(cfg))

	ctx := context.Background()
	assert.NotNil(t, ctx)

	// Test that we can create a plan with context
	plan := planner.createDefaultPlan("test question")
	assert.NotNil(t, plan)
	assert.NotEmpty(t, plan.ID)
}

func TestPlannerCreateDefaultPlanStructure(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	planner := NewPlanner(llm.NewClient(cfg))

	question := "Test question for planning"
	plan := planner.createDefaultPlan(question)

	assert.NotNil(t, plan)
	assert.NotEmpty(t, plan.ID)
	assert.Contains(t, plan.ID, "plan_")
	assert.GreaterOrEqual(t, plan.TotalSteps, 1)
	assert.Len(t, plan.Steps, plan.TotalSteps)
}

func TestPlannerWithDifferentProviders(t *testing.T) {
	providers := []string{"openai", "oneapi", "deepseek"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			cfg := &config.ModelConfig{
				Provider: provider,
				Model:    "test-model",
				APIKey:   "test-key",
			}
			planner := NewPlanner(llm.NewClient(cfg))

			assert.NotNil(t, planner)
		})
	}
}

func TestPlannerLogFields(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	planner := NewPlanner(llm.NewClient(cfg))

	assert.NotNil(t, planner.logger)
}

func TestPlannerPromptsManager(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	planner := NewPlanner(llm.NewClient(cfg))

	assert.NotNil(t, planner.prompts)
	// Verify we can get templates
	template := planner.prompts.GetTemplate("planner_general")
	assert.NotEmpty(t, template)
}

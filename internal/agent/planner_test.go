package agent

import (
	"context"
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/hle-agent/hle-agent/pkg/memory"
	"github.com/hle-agent/hle-agent/pkg/retriever"
	"github.com/stretchr/testify/assert"
)

// createTestPlanner creates a Planner with a mock retriever for testing
func createTestPlanner(t *testing.T, cfg *config.ModelConfig) *Planner {
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)

	mem := memory.NewDefaultMemory()
	mem.StartSession("test-session")
	r := retriever.NewRetriever(mem, nil)

	return NewPlanner(client, r)
}

func TestNewPlanner(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}

	planner := createTestPlanner(t, cfg)

	assert.NotNil(t, planner)
	assert.NotNil(t, planner.llmClient)
	assert.NotNil(t, planner.prompts)
	assert.NotNil(t, planner.logger)
}

func TestPlannerWithContext(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	planner := createTestPlanner(t, cfg)

	ctx := context.Background()
	assert.NotNil(t, ctx)

	// Test that the planner has the required components
	assert.NotNil(t, planner.llmClient)
	assert.NotNil(t, planner.retriever)
}

func TestPlannerCreateDefaultPlanStructure(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	planner := createTestPlanner(t, cfg)

	// Test that the planner has the required components
	assert.NotNil(t, planner)
	assert.NotNil(t, planner.llmClient)
	assert.NotNil(t, planner.prompts)
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
			planner := createTestPlanner(t, cfg)

			assert.NotNil(t, planner)
		})
	}
}

func TestPlannerLogFields(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	planner := createTestPlanner(t, cfg)

	assert.NotNil(t, planner.logger)
}

func TestPlannerPromptsManager(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	planner := createTestPlanner(t, cfg)

	assert.NotNil(t, planner.prompts)
	// Verify we can get templates
	template := planner.prompts.GetTemplate("planner_general")
	assert.NotEmpty(t, template)
}

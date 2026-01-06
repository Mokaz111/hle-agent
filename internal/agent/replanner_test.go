package agent

import (
	"context"
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/stretchr/testify/assert"
)

func TestNewReplanner(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
	}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)

	replanner := NewReplanner(client)

	assert.NotNil(t, replanner)
	assert.NotNil(t, replanner.llmClient)
	assert.NotNil(t, replanner.prompts)
	assert.NotNil(t, replanner.logger)
}

func TestReplanWithEmptyResults(t *testing.T) {
	// 需要模拟 LLM 客户端，跳过真实 API 调用
	t.Skip("需要模拟 LLM 客户端")

	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)
	replanner := NewReplanner(client)

	plan := &HLEPlan{
		ID:          "plan-001",
		FirstStepID: "step_1",
	}

	ctx := context.Background()
	newPlan, err := replanner.Replan(ctx, plan, []*HLEStepResult{}, "test question")
	assert.NoError(t, err)
	assert.NotNil(t, newPlan)
}

func TestReplanWithNilResults(t *testing.T) {
	// 需要模拟 LLM 客户端，跳过真实 API 调用
	t.Skip("需要模拟 LLM 客户端")

	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)
	replanner := NewReplanner(client)

	plan := &HLEPlan{
		ID:          "plan-001",
		FirstStepID: "step_1",
	}

	ctx := context.Background()
	newPlan, err := replanner.Replan(ctx, plan, nil, "test question")
	assert.NoError(t, err)
	assert.NotNil(t, newPlan)
}

func TestReplanWithContext(t *testing.T) {
	// 需要模拟 LLM 客户端，跳过真实 API 调用
	t.Skip("需要模拟 LLM 客户端")

	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)
	replanner := NewReplanner(client)

	plan := &HLEPlan{
		ID:          "plan-001",
		FirstStepID: "step_1",
	}

	ctx := context.Background()
	newPlan, err := replanner.Replan(ctx, plan, []*HLEStepResult{}, "test question")
	assert.NoError(t, err)
	assert.NotNil(t, newPlan)
}

func TestReplanWithDifferentQuestionTypes(t *testing.T) {
	// 需要模拟 LLM 客户端，跳过真实 API 调用
	t.Skip("需要模拟 LLM 客户端")

	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)
	replanner := NewReplanner(client)

	questions := []string{
		"Explain RSA encryption",
		"Calculate 2+2",
		"Write a Python function",
	}

	for _, question := range questions {
		t.Run(question, func(t *testing.T) {
			plan := &HLEPlan{
				ID:          "plan-001",
				FirstStepID: "step_1",
			}

			ctx := context.Background()
			newPlan, err := replanner.Replan(ctx, plan, []*HLEStepResult{}, question)
			assert.NoError(t, err)
			assert.NotNil(t, newPlan)
		})
	}
}

func TestReplanLogger(t *testing.T) {
	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)
	replanner := NewReplanner(client)

	assert.NotNil(t, replanner.logger)
}

func TestReplanWithSingleStepPlan(t *testing.T) {
	// 需要模拟 LLM 客户端，跳过真实 API 调用
	t.Skip("需要模拟 LLM 客户端")

	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)
	replanner := NewReplanner(client)

	plan := &HLEPlan{
		ID:          "plan-001",
		FirstStepID: "step_1",
	}

	ctx := context.Background()
	newPlan, err := replanner.Replan(ctx, plan, []*HLEStepResult{}, "test question")
	assert.NoError(t, err)
	assert.NotNil(t, newPlan)
}

func TestReplanWithMultipleSteps(t *testing.T) {
	// 需要模拟 LLM 客户端，跳过真实 API 调用
	t.Skip("需要模拟 LLM 客户端")

	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)
	replanner := NewReplanner(client)

	plan := &HLEPlan{
		ID:          "plan-001",
		FirstStepID: "step_1",
	}

	ctx := context.Background()
	newPlan, err := replanner.Replan(ctx, plan, []*HLEStepResult{}, "test question")
	assert.NoError(t, err)
	assert.NotNil(t, newPlan)
}

func TestReplanWithAllSuccessfulSteps(t *testing.T) {
	// 需要模拟 LLM 客户端，跳过真实 API 调用
	t.Skip("需要模拟 LLM 客户端")

	cfg := &config.ModelConfig{Provider: "openai", Model: "gpt-4", APIKey: "test-key"}
	client, err := llm.NewClient(cfg)
	assert.NoError(t, err)
	replanner := NewReplanner(client)

	plan := &HLEPlan{
		ID:          "plan-001",
		FirstStepID: "step_1",
	}

	stepResults := []*HLEStepResult{
		{StepID: "1", Success: true, Confidence: 0.9},
		{StepID: "2", Success: true, Confidence: 0.95},
		{StepID: "3", Success: true, Confidence: 0.85},
	}

	ctx := context.Background()
	_, err = replanner.Replan(ctx, plan, stepResults, "test question")
	// When all steps are successful, Replan might return nil plan
	// This is expected behavior when no replanning is needed
	assert.NoError(t, err)
}

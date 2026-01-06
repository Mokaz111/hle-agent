package agent

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/hle-agent/hle-agent/pkg/logging"
)

// Tool defines the interface for tools that can be executed by the Executor
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params string) (string, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools  map[string]Tool
	logger *zap.Logger
}

// NewToolRegistry creates a new ToolRegistry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:  make(map[string]Tool),
		logger: logging.WithComponent("ToolRegistry"),
	}
}

// Register registers a tool
func (r *ToolRegistry) Register(tool Tool) {
	if tool == nil {
		return
	}
	r.tools[tool.Name()] = tool
	r.logger.Debug("工具已注册", zap.String("tool_name", tool.Name()))
}

// Get gets a tool by name
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List lists all registered tools
func (r *ToolRegistry) List() []string {
	tools := make([]string, 0, len(r.tools))
	for name := range r.tools {
		tools = append(tools, name)
	}
	return tools
}

// Executor executes plan steps
type Executor struct {
	toolRegistry *ToolRegistry
	logger       *zap.Logger
}

// NewExecutor creates a new Executor instance
func NewExecutor(toolRegistry *ToolRegistry) *Executor {
	logger := logging.WithComponent("Executor")

	return &Executor{
		toolRegistry: toolRegistry,
		logger:       logger,
	}
}

// Execute executes a single step of the plan
func (e *Executor) Execute(ctx context.Context, plan *HLEPlan, step *HLEStep) (*HLEStepResult, error) {
	if step == nil {
		return nil, fmt.Errorf("step is nil")
	}

	logger := e.logger.With(
		zap.String("action", "Execute"),
		zap.String("plan_id", plan.ID),
		zap.String("step_id", step.ID),
		zap.String("step_name", step.Name),
	)

	logger.Info("开始执行步骤",
		zap.String("description", step.Description),
		zap.String("tool", step.ToolName))

	// Default action: use LLM to reason
	action := strings.ToLower(step.Action)
	if action == "" || action == "llm" {
		// Use LLM for reasoning
		result, err := e.executeWithLLM(ctx, step)
		if err != nil {
			logger.Error("LLM 执行失败", zap.Error(err))
			return &HLEStepResult{
				StepID:     step.ID,
				Success:    false,
				Error:      err.Error(),
				Output:     "",
				Confidence: 0.0,
			}, nil
		}

		logger.Info("步骤执行成功")
		return &HLEStepResult{
			StepID:     step.ID,
			Success:    true,
			Error:      "",
			Output:     result,
			Confidence: 0.9,
		}, nil
	}

	// Use registered tool
	if step.ToolName != "" {
		tool, ok := e.toolRegistry.Get(step.ToolName)
		if !ok {
			logger.Warn("工具未找到，尝试使用 LLM", zap.String("tool_name", step.ToolName))
			result, err := e.executeWithLLM(ctx, step)
			if err != nil {
				return &HLEStepResult{
					StepID:     step.ID,
					Success:    false,
					Error:      err.Error(),
					Output:     "",
					Confidence: 0.0,
				}, nil
			}
			return &HLEStepResult{
				StepID:     step.ID,
				Success:    true,
				Error:      "",
				Output:     result,
				Confidence: 0.9,
			}, nil
		}

		// Execute tool
		params := step.Params
		if params == "" {
			params = step.Description
		}

		result, err := tool.Execute(ctx, params)
		if err != nil {
			logger.Error("工具执行失败", zap.Error(err))
			return &HLEStepResult{
				StepID:     step.ID,
				Success:    false,
				Error:      err.Error(),
				Output:     "",
				Confidence: 0.0,
			}, nil
		}

		logger.Info("工具执行成功")
		return &HLEStepResult{
			StepID:     step.ID,
			Success:    true,
			Error:      "",
			Output:     result,
			Confidence: 0.95,
		}, nil
	}

	// Fallback: use LLM
	result, err := e.executeWithLLM(ctx, step)
	if err != nil {
		logger.Error("执行失败", zap.Error(err))
		return &HLEStepResult{
			StepID:     step.ID,
			Success:    false,
			Error:      err.Error(),
			Output:     "",
			Confidence: 0.0,
		}, nil
	}

	return &HLEStepResult{
		StepID:     step.ID,
		Success:    true,
		Error:      "",
		Output:     result,
		Confidence: 0.85,
	}, nil
}

// executeWithLLM uses the LLM to execute a step
func (e *Executor) executeWithLLM(ctx context.Context, step *HLEStep) (string, error) {
	// This is a placeholder - in a real implementation, we would use the LLM client
	// For now, return the step description as the "result"
	return step.Description, nil
}

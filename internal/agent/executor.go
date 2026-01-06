package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
	logger *zap.Logger
}

// Tool defines the interface for execution tools
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:  make(map[string]Tool),
		logger: logging.WithComponent("ToolRegistry"),
	}
}

// Register registers a tool
func (r *ToolRegistry) Register(tool Tool) {
	r.logger.Info("注册工具",
		zap.String("tool_name", tool.Name()),
		zap.String("tool_description", tool.Description()))
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *ToolRegistry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Executor implements the Eino ADK Executor interface
type Executor struct {
	toolRegistry *ToolRegistry
	logger       *zap.Logger
	maxRetries   int
}

// NewExecutor creates a new Executor
func NewExecutor(registry *ToolRegistry) *Executor {
	executor := &Executor{
		toolRegistry: registry,
		logger:       logging.WithComponent("Executor"),
		maxRetries:   3,
	}

	// Register default tools
	executor.registerDefaultTools()

	return executor
}

// registerDefaultTools registers the default tools
func (e *Executor) registerDefaultTools() {
	e.logger.Debug("初始化默认工具注册表")
	// Tools are registered externally via ToolRegistry
	// The Python and SageMath executors will be registered here
}

// Execute executes a single step of the plan
func (e *Executor) Execute(ctx context.Context, step *models.Step, input map[string]interface{}) (*models.StepResult, error) {
	logger := e.logger.With(
		zap.String("action", "Execute"),
		zap.Int("step_id", step.ID),
		zap.String("description", step.Description),
		zap.String("action_type", step.Action),
		zap.String("tool", step.ToolName),
	)

	startTime := time.Now()

	result := &models.StepResult{
		StepID:     step.ID,
		Success:    false,
		Timestamp:  time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}

	logger.Debug("开始执行步骤")

	// Execute based on action type
	switch step.Action {
	case "llm":
		// LLM reasoning step
		output, err := e.executeLLM(ctx, step, input)
		if err != nil {
			logger.Error("LLM 执行失败",
				zap.Error(err),
				zap.Int("step_id", step.ID))
			result.Error = err.Error()
			result.Duration = time.Since(startTime).Seconds()
			return result, nil
		}
		result.Output = output
		result.Success = true
		result.Confidence = 0.85
		logger.Info("LLM 步骤执行成功",
			zap.Float64("confidence", result.Confidence))

	case "tool":
		// Tool execution step
		output, err := e.executeTool(ctx, step, input)
		if err != nil {
			logger.Warn("工具执行失败，尝试重试",
				zap.Error(err),
				zap.Int("max_retries", e.maxRetries))
			// Check if we should retry
			if step.RetryOnFail {
				output, err = e.retryTool(ctx, step, input, logger)
				if err != nil {
					logger.Error("工具执行失败，已达最大重试次数",
						zap.Error(err),
						zap.Int("step_id", step.ID))
					result.Error = err.Error()
					result.Duration = time.Since(startTime).Seconds()
					return result, nil
				}
			} else {
				logger.Error("工具执行失败（不重试）",
					zap.Error(err),
					zap.Int("step_id", step.ID))
				result.Error = err.Error()
				result.Duration = time.Since(startTime).Seconds()
				return result, nil
			}
		}
		result.Output = output
		result.Success = true
		result.Confidence = 0.9
		logger.Info("工具执行成功",
			zap.Float64("confidence", result.Confidence))

	default:
		logger.Warn("未知的操作类型，使用默认处理",
			zap.String("action", step.Action))
		result.Output = fmt.Sprintf("Unknown action type: %s", step.Action)
		result.Success = true
		result.Confidence = 0.5
	}

	result.Duration = time.Since(startTime).Seconds()

	logger.Debug("步骤执行完成",
		zap.Bool("success", result.Success),
		zap.Float64("duration_seconds", result.Duration))

	return result, nil
}

// executeLLM performs LLM reasoning
func (e *Executor) executeLLM(ctx context.Context, step *models.Step, input map[string]interface{}) (interface{}, error) {
	logger := e.logger.With(
		zap.String("action", "executeLLM"),
		zap.Int("step_id", step.ID),
	)

	// In a full implementation, this would call the LLM
	// For now, we return a placeholder
	logger.Debug("执行 LLM 推理步骤",
		zap.String("description", step.Description))

	// Simulate LLM reasoning
	result := map[string]interface{}{
		"reasoning":  step.Description,
		"conclusion": fmt.Sprintf("基于推理得出结论：%s", step.Description),
		"step_id":    step.ID,
	}

	return result, nil
}

// executeTool executes a tool
func (e *Executor) executeTool(ctx context.Context, step *models.Step, input map[string]interface{}) (interface{}, error) {
	logger := e.logger.With(
		zap.String("action", "executeTool"),
		zap.Int("step_id", step.ID),
		zap.String("tool_name", step.ToolName),
	)

	// Get tool from registry
	tool, ok := e.toolRegistry.Get(step.ToolName)
	if !ok {
		logger.Error("工具未找到",
			zap.String("tool_name", step.ToolName),
			zap.Strings("available_tools", e.toolRegistry.List()))
		return nil, fmt.Errorf("tool not found: %s", step.ToolName)
	}

	logger.Debug("获取工具成功",
		zap.String("tool_name", step.ToolName),
		zap.String("tool_description", tool.Description()))

	// Build parameters
	params := step.Parameters
	if params == nil {
		params = make(map[string]interface{})
	}

	// Merge input data
	if inputData, ok := input["input_data"].(map[string]interface{}); ok {
		for k, v := range inputData {
			params[k] = v
		}
	}

	// Add context information
	params["step_id"] = step.ID
	params["timestamp"] = time.Now().Format(time.RFC3339)

	logger.Debug("执行工具",
		zap.String("tool_name", step.ToolName),
		zap.Any("parameters", params))

	// Execute the tool
	output, err := tool.Execute(ctx, params)
	if err != nil {
		logger.Error("工具执行失败",
			zap.Error(err),
			zap.String("tool_name", step.ToolName))
		return nil, err
	}

	logger.Info("工具执行成功",
		zap.String("tool_name", step.ToolName),
		zap.Any("output_type", fmt.Sprintf("%T", output)))

	return output, nil
}

// retryTool retries a failed tool execution
func (e *Executor) retryTool(ctx context.Context, step *models.Step, input map[string]interface{}, logger *zap.Logger) (interface{}, error) {
	for i := 0; i < e.maxRetries; i++ {
		waitTime := time.Duration(i+1) * time.Second
		logger.Debug("重试工具执行",
			zap.Int("attempt", i+1),
			zap.Int("max_retries", e.maxRetries),
			zap.Duration("wait_time", waitTime))

		time.Sleep(waitTime)

		output, err := e.executeTool(ctx, step, input)
		if err == nil {
			logger.Info("重试成功",
				zap.Int("attempt", i+1))
			return output, nil
		}

		logger.Warn("重试失败",
			zap.Int("attempt", i+1),
			zap.Error(err))
	}

	return nil, fmt.Errorf("tool execution failed after %d retries", e.maxRetries)
}

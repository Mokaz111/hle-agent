package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/hle-agent/hle-agent/pkg/logging"
	"github.com/hle-agent/hle-agent/pkg/llm"
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
	toolRegistry   *ToolRegistry
	llmClient      *llm.Client
	logger         *zap.Logger
	maxRetries     int
	retryBackoff   time.Duration
	enableFallback bool
}

// ExecutorConfig represents executor configuration
type ExecutorConfig struct {
	MaxRetries     int
	RetryBackoff   time.Duration
	EnableFallback bool
}

// NewExecutor creates a new Executor instance
func NewExecutor(toolRegistry *ToolRegistry, llmClient *llm.Client, cfg *ExecutorConfig) *Executor {
	logger := logging.WithComponent("Executor")

	// 设置默认值
	maxRetries := 3
	retryBackoff := 1 * time.Second
	enableFallback := true

	if cfg != nil {
		if cfg.MaxRetries > 0 {
			maxRetries = cfg.MaxRetries
		}
		if cfg.RetryBackoff > 0 {
			retryBackoff = cfg.RetryBackoff
		}
		enableFallback = cfg.EnableFallback
	}

	return &Executor{
		toolRegistry:   toolRegistry,
		llmClient:      llmClient,
		logger:         logger,
		maxRetries:     maxRetries,
		retryBackoff:   retryBackoff,
		enableFallback: enableFallback,
	}
}

// Execute executes a single step of the plan
func (e *Executor) Execute(ctx context.Context, plan *HLEPlan, step *HLEStep) (*HLEStepResult, error) {
	if step == nil {
		return nil, fmt.Errorf("step is nil")
	}

	startTime := time.Now()
	logger := e.logger.With(
		zap.String("action", "Execute"),
		zap.String("plan_id", plan.ID),
		zap.String("step_id", step.ID),
		zap.String("step_name", step.Name),
		zap.String("step_action", step.Action),
	)

	logger.Info("开始执行步骤",
		zap.String("description", step.Description),
		zap.String("tool", step.ToolName),
		zap.String("decision_point", "step_execution_start"))

	// Default action: use LLM to reason
	action := strings.ToLower(step.Action)
	if action == "" || action == "llm" {
		// Use LLM for reasoning
		llmStartTime := time.Now()
		result, err := e.executeWithLLM(ctx, step)
		llmDuration := time.Since(llmStartTime)
		
		if err != nil {
			totalDuration := time.Since(startTime)
			agentErr := NewLLMInvocationError("LLM 执行步骤失败", err)
			logger.Error("LLM 执行失败",
				zap.Error(agentErr),
				zap.Duration("llm_duration", llmDuration),
				zap.Duration("total_duration", totalDuration),
				zap.String("recovery_suggestion", agentErr.GetRecoverySuggestion()),
				zap.String("decision_point", "llm_execution_failed"))
			return &HLEStepResult{
				StepID:     step.ID,
				Success:    false,
				Error:      agentErr.Error(),
				Output:     "",
				Confidence: 0.0,
			}, nil
		}

		totalDuration := time.Since(startTime)
		logger.Info("步骤执行成功（LLM）",
			zap.Duration("total_duration", totalDuration),
			zap.Float64("total_duration_seconds", totalDuration.Seconds()),
			zap.Duration("llm_duration", llmDuration),
			zap.Float64("llm_duration_seconds", llmDuration.Seconds()),
			zap.String("decision_point", "step_execution_success_llm"),
			zap.Int("output_length", len(result)),
			zap.String("step_type", "llm_reasoning"))
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
			agentErr := NewToolNotFoundError(step.ToolName)
			logger.Warn("工具未找到，尝试使用 LLM",
				zap.String("tool_name", step.ToolName),
				zap.String("recovery_suggestion", agentErr.GetRecoverySuggestion()))
			result, err := e.executeWithLLM(ctx, step)
			if err != nil {
				llmErr := NewLLMInvocationError("降级到 LLM 失败", err)
				return &HLEStepResult{
					StepID:     step.ID,
					Success:    false,
					Error:      llmErr.Error(),
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

		// Execute tool with retry and fallback
		params := step.Params
		if params == "" {
			params = step.Description
		}

		logger.Debug("准备执行工具",
			zap.String("tool_name", tool.Name()),
			zap.Int("params_length", len(params)),
			zap.Int("max_retries", e.maxRetries),
			zap.Duration("retry_backoff", e.retryBackoff),
			zap.Bool("enable_fallback", e.enableFallback),
			zap.String("decision_point", "tool_execution_prepare"))

		result, err := e.executeToolWithRetry(ctx, tool, params)
		if err != nil {
			totalDuration := time.Since(startTime)
			logger.Error("工具执行失败，已尝试重试和降级",
				zap.Error(err),
				zap.Duration("total_duration", totalDuration),
				zap.String("decision_point", "tool_execution_failed_after_retry"))
			return &HLEStepResult{
				StepID:     step.ID,
				Success:    false,
				Error:      err.Error(),
				Output:     "",
				Confidence: 0.0,
			}, nil
		}

		totalDuration := time.Since(startTime)
		logger.Info("工具执行成功",
			zap.Duration("total_duration", totalDuration),
			zap.Float64("total_duration_seconds", totalDuration.Seconds()),
			zap.String("decision_point", "tool_execution_success"),
			zap.String("tool_name", tool.Name()),
			zap.Int("output_length", len(result)),
			zap.String("step_type", "tool_execution"))
		return &HLEStepResult{
			StepID:     step.ID,
			Success:    true,
			Error:      "",
			Output:     result,
			Confidence: 0.95,
		}, nil
	}

	// Fallback: use LLM
	logger.Debug("使用 LLM 作为降级方案",
		zap.String("decision_point", "fallback_to_llm"))
	llmStartTime := time.Now()
	result, err := e.executeWithLLM(ctx, step)
	llmDuration := time.Since(llmStartTime)
	if err != nil {
		totalDuration := time.Since(startTime)
		logger.Error("执行失败",
			zap.Error(err),
			zap.Duration("llm_duration", llmDuration),
			zap.Duration("total_duration", totalDuration),
			zap.String("decision_point", "fallback_llm_failed"))
		return &HLEStepResult{
			StepID:     step.ID,
			Success:    false,
			Error:      err.Error(),
			Output:     "",
			Confidence: 0.0,
		}, nil
	}

	totalDuration := time.Since(startTime)
	logger.Info("步骤执行完成（LLM降级）",
		zap.Duration("total_duration", totalDuration),
		zap.Float64("total_duration_seconds", totalDuration.Seconds()),
		zap.Duration("llm_duration", llmDuration),
		zap.Float64("llm_duration_seconds", llmDuration.Seconds()),
		zap.String("decision_point", "llm_fallback_success"),
		zap.Int("output_length", len(result)),
		zap.String("step_type", "llm_fallback"))
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
	if e.llmClient == nil {
		return "", NewConfigurationError("LLM client 未初始化", nil)
	}

	llmLogger := e.logger.With(
		zap.String("step_id", step.ID),
		zap.String("decision_point", "llm_invocation_start"))

	// Build prompt from step description
	prompt := fmt.Sprintf("请执行以下步骤：\n%s", step.Description)
	
	// Add step parameters if available
	if step.Params != "" {
		prompt += fmt.Sprintf("\n\n参数：\n%s", step.Params)
	}

	llmLogger.Debug("构建 LLM 提示词",
		zap.Int("prompt_length", len(prompt)),
		zap.Int("description_length", len(step.Description)),
		zap.Int("params_length", len(step.Params)))

	// Call LLM with system prompt
	systemPrompt := "你是一个专业的解题助手。请仔细分析问题并给出准确的答案。"
	
	llmCallStartTime := time.Now()
	result, err := e.llmClient.GenerateWithSystemPrompt(ctx, systemPrompt, prompt)
	llmCallDuration := time.Since(llmCallStartTime)
	
	if err != nil {
		llmLogger.Error("LLM 调用失败",
			zap.Error(err),
			zap.Duration("llm_call_duration", llmCallDuration),
			zap.String("decision_point", "llm_invocation_failed"))
		return "", NewLLMInvocationError("LLM 调用失败", err)
	}

	llmLogger.Debug("LLM 调用成功",
		zap.Duration("llm_call_duration", llmCallDuration),
		zap.Float64("llm_call_duration_seconds", llmCallDuration.Seconds()),
		zap.Int("result_length", len(result)),
		zap.String("decision_point", "llm_invocation_success"))

	return result, nil
}

// executeToolWithRetry executes a tool with retry mechanism and fallback to LLM
func (e *Executor) executeToolWithRetry(ctx context.Context, tool Tool, params string) (string, error) {
	var lastErr error
	startTime := time.Now()
	
	logger := e.logger.With(
		zap.String("tool", tool.Name()),
		zap.String("decision_point", "tool_retry_start"),
	)
	
	// Try executing the tool with retries
	for attempt := 0; attempt < e.maxRetries; attempt++ {
		attemptStartTime := time.Now()
		result, err := tool.Execute(ctx, params)
		attemptDuration := time.Since(attemptStartTime)
		if err == nil {
			totalDuration := time.Since(startTime)
			if attempt > 0 {
				logger.Info("工具执行成功（重试后）",
					zap.Int("attempt", attempt+1),
					zap.Duration("attempt_duration", attemptDuration),
					zap.Duration("total_duration", totalDuration),
					zap.String("decision_point", "tool_retry_success"))
			} else {
				logger.Debug("工具执行成功（首次尝试）",
					zap.Duration("duration", attemptDuration),
					zap.String("decision_point", "tool_first_attempt_success"))
			}
			return result, nil
		}
		
		lastErr = err
		backoff := time.Duration(1<<uint(attempt)) * e.retryBackoff
		e.logger.Warn("工具执行失败，准备重试",
			zap.String("tool", tool.Name()),
			zap.Int("attempt", attempt+1),
			zap.Int("max_retries", e.maxRetries),
			zap.Duration("attempt_duration", attemptDuration),
			zap.Duration("backoff_duration", backoff),
			zap.String("decision_point", "tool_retry_prepare"),
			zap.Error(err))
		
		// Exponential backoff before retry
		if attempt < e.maxRetries-1 {
			backoffStartTime := time.Now()
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
				backoffDuration := time.Since(backoffStartTime)
				logger.Debug("退避等待完成",
					zap.Duration("backoff_duration", backoffDuration),
					zap.String("decision_point", "backoff_complete"))
				// Continue to next retry
			}
		}
	}
	
	// All retries failed, fallback to LLM
	totalDuration := time.Since(startTime)
	logger.Info("工具执行失败，降级到 LLM",
		zap.Int("total_attempts", e.maxRetries),
		zap.Duration("total_duration", totalDuration),
		zap.String("decision_point", "tool_fallback_to_llm"),
		zap.Error(lastErr))
	
	// Create a step for LLM fallback
	fallbackStep := &HLEStep{
		ID:          "fallback",
		Description: fmt.Sprintf("工具 %s 执行失败，请使用推理解决以下问题：\n%s", tool.Name(), params),
		Action:      "llm",
	}
	
	result, err := e.executeWithLLM(ctx, fallbackStep)
	if err != nil {
		toolErr := NewToolExecutionError(tool.Name(), lastErr)
		llmErr := NewLLMInvocationError("LLM 降级失败", err)
		return "", fmt.Errorf("工具执行失败且 LLM 降级也失败: %v, %v", toolErr, llmErr)
	}
	
	return result, nil
}

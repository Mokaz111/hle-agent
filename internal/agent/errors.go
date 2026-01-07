package agent

import "fmt"

// AgentError 是 Agent 相关的错误基类
type AgentError struct {
	Code    string
	Message string
	Err     error
}

func (e *AgentError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AgentError) Unwrap() error {
	return e.Err
}

// 错误代码定义
const (
	ErrCodePlanGeneration    = "PLAN_GENERATION_FAILED"
	ErrCodeStepExecution     = "STEP_EXECUTION_FAILED"
	ErrCodeToolExecution     = "TOOL_EXECUTION_FAILED"
	ErrCodeLLMInvocation     = "LLM_INVOCATION_FAILED"
	ErrCodeReplanning        = "REPLANNING_FAILED"
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeTimeout           = "TIMEOUT"
	ErrCodeToolNotFound      = "TOOL_NOT_FOUND"
	ErrCodeConfiguration     = "CONFIGURATION_ERROR"
)

// NewPlanGenerationError 创建计划生成错误
func NewPlanGenerationError(message string, err error) *AgentError {
	return &AgentError{
		Code:    ErrCodePlanGeneration,
		Message: message,
		Err:     err,
	}
}

// NewStepExecutionError 创建步骤执行错误
func NewStepExecutionError(message string, err error) *AgentError {
	return &AgentError{
		Code:    ErrCodeStepExecution,
		Message: message,
		Err:     err,
	}
}

// NewToolExecutionError 创建工具执行错误
func NewToolExecutionError(toolName string, err error) *AgentError {
	return &AgentError{
		Code:    ErrCodeToolExecution,
		Message: fmt.Sprintf("工具 '%s' 执行失败", toolName),
		Err:     err,
	}
}

// NewLLMInvocationError 创建 LLM 调用错误
func NewLLMInvocationError(message string, err error) *AgentError {
	return &AgentError{
		Code:    ErrCodeLLMInvocation,
		Message: message,
		Err:     err,
	}
}

// NewReplanningError 创建重规划错误
func NewReplanningError(message string, err error) *AgentError {
	return &AgentError{
		Code:    ErrCodeReplanning,
		Message: message,
		Err:     err,
	}
}

// NewInvalidInputError 创建无效输入错误
func NewInvalidInputError(message string) *AgentError {
	return &AgentError{
		Code:    ErrCodeInvalidInput,
		Message: message,
		Err:     nil,
	}
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(operation string) *AgentError {
	return &AgentError{
		Code:    ErrCodeTimeout,
		Message: fmt.Sprintf("操作 '%s' 超时", operation),
		Err:     nil,
	}
}

// NewToolNotFoundError 创建工具未找到错误
func NewToolNotFoundError(toolName string) *AgentError {
	return &AgentError{
		Code:    ErrCodeToolNotFound,
		Message: fmt.Sprintf("工具 '%s' 未找到", toolName),
		Err:     nil,
	}
}

// NewConfigurationError 创建配置错误
func NewConfigurationError(message string, err error) *AgentError {
	return &AgentError{
		Code:    ErrCodeConfiguration,
		Message: message,
		Err:     err,
	}
}

// IsRetryable 判断错误是否可重试
func (e *AgentError) IsRetryable() bool {
	switch e.Code {
	case ErrCodeToolExecution, ErrCodeLLMInvocation, ErrCodeTimeout:
		return true
	default:
		return false
	}
}

// GetRecoverySuggestion 获取错误恢复建议
func (e *AgentError) GetRecoverySuggestion() string {
	switch e.Code {
	case ErrCodePlanGeneration:
		return "建议：检查题目格式是否正确，或尝试简化题目描述"
	case ErrCodeStepExecution:
		return "建议：检查步骤参数是否正确，或尝试重新规划"
	case ErrCodeToolExecution:
		return "建议：检查工具配置是否正确，或尝试使用 LLM 推理替代"
	case ErrCodeLLMInvocation:
		return "建议：检查 LLM 配置和网络连接，或稍后重试"
	case ErrCodeReplanning:
		return "建议：检查执行结果，或尝试使用更简单的解题策略"
	case ErrCodeToolNotFound:
		return "建议：检查工具是否已正确注册"
	case ErrCodeTimeout:
		return "建议：增加超时时间，或简化操作"
	default:
		return "建议：查看详细错误信息，联系技术支持"
	}
}


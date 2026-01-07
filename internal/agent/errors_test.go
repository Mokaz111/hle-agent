package agent

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *AgentError
		wantErr string
	}{
		{
			name: "error with wrapped error",
			err: &AgentError{
				Code:    ErrCodePlanGeneration,
				Message: "test message",
				Err:     errors.New("wrapped error"),
			},
			wantErr: "[PLAN_GENERATION_FAILED] test message: wrapped error",
		},
		{
			name: "error without wrapped error",
			err: &AgentError{
				Code:    ErrCodeStepExecution,
				Message: "test message",
				Err:     nil,
			},
			wantErr: "[STEP_EXECUTION_FAILED] test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantErr, tt.err.Error())
		})
	}
}

func TestAgentError_Unwrap(t *testing.T) {
	wrappedErr := errors.New("wrapped error")
	err := &AgentError{
		Code:    ErrCodePlanGeneration,
		Message: "test",
		Err:     wrappedErr,
	}

	assert.Equal(t, wrappedErr, err.Unwrap())
}

func TestNewPlanGenerationError(t *testing.T) {
	wrappedErr := errors.New("plan error")
	err := NewPlanGenerationError("failed to generate plan", wrappedErr)

	assert.Equal(t, ErrCodePlanGeneration, err.Code)
	assert.Equal(t, "failed to generate plan", err.Message)
	assert.Equal(t, wrappedErr, err.Err)
}

func TestNewStepExecutionError(t *testing.T) {
	wrappedErr := errors.New("step error")
	err := NewStepExecutionError("failed to execute step", wrappedErr)

	assert.Equal(t, ErrCodeStepExecution, err.Code)
	assert.Equal(t, "failed to execute step", err.Message)
	assert.Equal(t, wrappedErr, err.Err)
}

func TestNewToolExecutionError(t *testing.T) {
	wrappedErr := errors.New("tool error")
	err := NewToolExecutionError("python_executor", wrappedErr)

	assert.Equal(t, ErrCodeToolExecution, err.Code)
	assert.Contains(t, err.Message, "python_executor")
	assert.Equal(t, wrappedErr, err.Err)
}

func TestNewLLMInvocationError(t *testing.T) {
	wrappedErr := errors.New("llm error")
	err := NewLLMInvocationError("failed to invoke LLM", wrappedErr)

	assert.Equal(t, ErrCodeLLMInvocation, err.Code)
	assert.Equal(t, "failed to invoke LLM", err.Message)
	assert.Equal(t, wrappedErr, err.Err)
}

func TestNewReplanningError(t *testing.T) {
	wrappedErr := errors.New("replan error")
	err := NewReplanningError("failed to replan", wrappedErr)

	assert.Equal(t, ErrCodeReplanning, err.Code)
	assert.Equal(t, "failed to replan", err.Message)
	assert.Equal(t, wrappedErr, err.Err)
}

func TestNewInvalidInputError(t *testing.T) {
	err := NewInvalidInputError("invalid input")

	assert.Equal(t, ErrCodeInvalidInput, err.Code)
	assert.Equal(t, "invalid input", err.Message)
	assert.Nil(t, err.Err)
}

func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError("operation")

	assert.Equal(t, ErrCodeTimeout, err.Code)
	assert.Contains(t, err.Message, "operation")
	assert.Nil(t, err.Err)
}

func TestNewToolNotFoundError(t *testing.T) {
	err := NewToolNotFoundError("unknown_tool")

	assert.Equal(t, ErrCodeToolNotFound, err.Code)
	assert.Contains(t, err.Message, "unknown_tool")
	assert.Nil(t, err.Err)
}

func TestNewConfigurationError(t *testing.T) {
	wrappedErr := errors.New("config error")
	err := NewConfigurationError("invalid configuration", wrappedErr)

	assert.Equal(t, ErrCodeConfiguration, err.Code)
	assert.Equal(t, "invalid configuration", err.Message)
	assert.Equal(t, wrappedErr, err.Err)
}

func TestAgentError_IsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  *AgentError
		want bool
	}{
		{
			name: "tool execution error is retryable",
			err:  NewToolExecutionError("tool", errors.New("error")),
			want: true,
		},
		{
			name: "LLM invocation error is retryable",
			err:  NewLLMInvocationError("error", errors.New("error")),
			want: true,
		},
		{
			name: "timeout error is retryable",
			err:  NewTimeoutError("operation"),
			want: true,
		},
		{
			name: "plan generation error is not retryable",
			err:  NewPlanGenerationError("error", errors.New("error")),
			want: false,
		},
		{
			name: "invalid input error is not retryable",
			err:  NewInvalidInputError("error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.IsRetryable())
		})
	}
}

func TestAgentError_GetRecoverySuggestion(t *testing.T) {
	tests := []struct {
		name     string
		err      *AgentError
		contains string
	}{
		{
			name:     "plan generation error",
			err:      NewPlanGenerationError("error", nil),
			contains: "检查题目格式",
		},
		{
			name:     "step execution error",
			err:      NewStepExecutionError("error", nil),
			contains: "检查步骤参数",
		},
		{
			name:     "tool execution error",
			err:      NewToolExecutionError("tool", nil),
			contains: "检查工具配置",
		},
		{
			name:     "LLM invocation error",
			err:      NewLLMInvocationError("error", nil),
			contains: "检查 LLM 配置",
		},
		{
			name:     "replanning error",
			err:      NewReplanningError("error", nil),
			contains: "检查执行结果",
		},
		{
			name:     "tool not found error",
			err:      NewToolNotFoundError("tool"),
			contains: "检查工具是否已正确注册",
		},
		{
			name:     "timeout error",
			err:      NewTimeoutError("operation"),
			contains: "增加超时时间",
		},
		{
			name:     "unknown error",
			err:      &AgentError{Code: "UNKNOWN"},
			contains: "联系技术支持",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := tt.err.GetRecoverySuggestion()
			assert.Contains(t, suggestion, tt.contains)
		})
	}
}


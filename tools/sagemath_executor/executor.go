package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SageMathExecutor implements the MCP tool for SageMath execution
type SageMathExecutor struct {
	timeout time.Duration
}

// NewSageMathExecutor creates a new SageMath executor
func NewSageMathExecutor(timeoutSeconds int) *SageMathExecutor {
	return &SageMathExecutor{
		timeout: time.Duration(timeoutSeconds) * time.Second,
	}
}

// Name returns the tool name
func (e *SageMathExecutor) Name() string {
	return "sagemath_executor"
}

// Description returns the tool description
func (e *SageMathExecutor) Description() string {
	return "Executes SageMath code for advanced mathematical computations, symbolic algebra, and number theory"
}

// Execute executes SageMath code
func (e *SageMathExecutor) Execute(ctx context.Context, params string) (string, error) {
	// Extract code from params (JSON format)
	var paramsMap map[string]interface{}
	if err := json.Unmarshal([]byte(params), &paramsMap); err != nil {
		return "", fmt.Errorf("invalid params format: %w", err)
	}

	// Extract code from params
	code, ok := paramsMap["code"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'code' parameter")
	}

	// Build the SageMath command - use sage -c for command line execution
	cmd := exec.CommandContext(ctx, "sage", "-c", code)

	// Set timeout if not already in context
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	// Execute command
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime).Seconds()

	// Parse output
	result := make(map[string]interface{})
	result["code"] = code
	result["duration_seconds"] = duration

	if err != nil {
		result["success"] = false
		result["error"] = string(output)
		return "", fmt.Errorf("SageMath execution failed: %s", string(output))
	}

	result["success"] = true
	result["output"] = strings.TrimSpace(string(output))

	// Return JSON result
	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// ExecuteWithInput executes SageMath code with input data
func (e *SageMathExecutor) ExecuteWithInput(ctx context.Context, code string, input interface{}) (string, error) {
	// Wrap code with input variable
	wrappedCode := fmt.Sprintf(`
# Input data as Python dictionary
input_data = %s

# User code
%s

# Print result
print(result)
`, formatSageInput(input), code)

	return e.Execute(ctx, fmt.Sprintf(`{"code": %s}`, formatSageInput(wrappedCode)))
}

// formatSageInput formats input data for SageMath
func formatSageInput(input interface{}) string {
	// SageMath uses Python syntax
	data, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(data)
}

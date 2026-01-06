package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PythonExecutor implements the MCP tool for Python execution
type PythonExecutor struct {
	timeout time.Duration
}

// NewPythonExecutor creates a new Python executor
func NewPythonExecutor(timeoutSeconds int) *PythonExecutor {
	return &PythonExecutor{
		timeout: time.Duration(timeoutSeconds) * time.Second,
	}
}

// Name returns the tool name
func (e *PythonExecutor) Name() string {
	return "python_executor"
}

// Description returns the tool description
func (e *PythonExecutor) Description() string {
	return "Executes Python code for data analysis, cryptography, and general computation tasks"
}

// Execute executes Python code
func (e *PythonExecutor) Execute(ctx context.Context, params string) (string, error) {
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

	// Build the Python command
	cmd := exec.CommandContext(ctx, "python3", "-c", code)

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
		return "", fmt.Errorf("Python execution failed: %s", string(output))
	}

	result["success"] = true
	result["output"] = strings.TrimSpace(string(output))

	// Return JSON result
	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// ExecuteWithInput executes Python code with input data
func (e *PythonExecutor) ExecuteWithInput(ctx context.Context, code string, input interface{}) (string, error) {
	// Wrap code with input variable
	wrappedCode := fmt.Sprintf(`
import json
input_data = %s

# User code
%s

# Print result
print(json.dumps(result))
`, formatInput(input), code)

	return e.Execute(ctx, fmt.Sprintf(`{"code": %s}`, formatInput(wrappedCode)))
}

// formatInput formats input data as JSON
func formatInput(input interface{}) string {
	data, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ExecuteScript executes a Python script file
func (e *PythonExecutor) ExecuteScript(ctx context.Context, scriptPath string, args []string) (interface{}, error) {
	cmd := exec.CommandContext(ctx, "python3", append([]string{scriptPath}, args...)...)
	
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime).Seconds()
	
	result := make(map[string]interface{})
	result["script_path"] = scriptPath
	result["duration_seconds"] = duration
	
	if err != nil {
		result["success"] = false
		result["error"] = string(output)
		return result, fmt.Errorf("Python script execution failed: %s", string(output))
	}
	
	result["success"] = true
	result["output"] = strings.TrimSpace(string(output))
	
	return result, nil
}

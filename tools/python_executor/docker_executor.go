package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DockerPythonExecutor executes Python code in an isolated Docker container
type DockerPythonExecutor struct {
	timeout    time.Duration
	imageName  string
	dockerPath string
}

// NewDockerPythonExecutor creates a new Docker-based Python executor
func NewDockerPythonExecutor(timeoutSeconds int, imageName string) *DockerPythonExecutor {
	if imageName == "" {
		imageName = "hle-agent-python:latest"
	}

	dockerPath := "docker"
	if path, err := exec.LookPath("docker"); err == nil {
		dockerPath = path
	}

	return &DockerPythonExecutor{
		timeout:    time.Duration(timeoutSeconds) * time.Second,
		imageName:  imageName,
		dockerPath: dockerPath,
	}
}

// Name returns the tool name
func (e *DockerPythonExecutor) Name() string {
	return "python_executor"
}

// Description returns the tool description
func (e *DockerPythonExecutor) Description() string {
	return "Executes Python code in an isolated Docker container for data analysis, cryptography, and general computation tasks"
}

// Execute executes Python code in a Docker container
func (e *DockerPythonExecutor) Execute(ctx context.Context, params string) (string, error) {
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

	// Escape code for shell
	escapedCode := strings.ReplaceAll(code, "'", "'\"'\"'")

	// Build Docker command
	// Use --rm to automatically remove container after execution
	// Use --network none for network isolation
	// Use --memory and --cpus for resource limits
	dockerArgs := []string{
		"run",
		"--rm",
		"--network", "none",
		"--memory", "1g",
		"--cpus", "1.0",
		"--read-only",
		"--tmpfs", "/tmp:noexec,nosuid,size=100m",
		"--user", "1000:1000",
		"--security-opt", "no-new-privileges:true",
		e.imageName,
		"python3", "-c", escapedCode,
	}

	// Set timeout if not already in context
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	// Execute Docker command
	cmd := exec.CommandContext(ctx, e.dockerPath, dockerArgs...)

	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime).Seconds()

	// Parse output
	result := make(map[string]interface{})
	result["code"] = code
	result["duration_seconds"] = duration
	result["execution_mode"] = "docker"

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

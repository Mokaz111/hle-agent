package models

// PlanStatus represents the status of a plan
type PlanStatus string

const (
	PlanStatusPending    PlanStatus = "pending"
	PlanStatusExecuting  PlanStatus = "executing"
	PlanStatusNeedReplan PlanStatus = "need_replan"
	PlanStatusCompleted  PlanStatus = "completed"
	PlanStatusFailed     PlanStatus = "failed"
)

// StepStatus represents the status of a step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// Plan represents a解题 plan
type Plan struct {
	ID          string       `json:"id"`
	QuestionID  string       `json:"question_id"`
	Steps       []Step       `json:"steps"`
	TotalSteps  int          `json:"total_steps"`
	CurrentStep int          `json:"current_step"`
	Status      PlanStatus   `json:"status"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Step represents a single step in the plan
type Step struct {
	ID           int                    `json:"id"`
	Description  string                 `json:"description"`
	Action       string                 `json:"action"`
	ToolName     string                 `json:"tool_name,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RetryOnFail  bool                   `json:"retry_on_fail"`
	IsKeyPoint   bool                   `json:"is_key_point,omitempty"`
	Status       StepStatus             `json:"status"`
}

// StepResult represents the result of a step execution
type StepResult struct {
	StepID     int         `json:"step_id"`
	Success    bool        `json:"success"`
	Output     interface{} `json:"output,omitempty"`
	Error      string      `json:"error,omitempty"`
	Confidence float64     `json:"confidence"`
	Timestamp  string      `json:"timestamp"`
	Duration   float64     `json:"duration_seconds"`
}

// FinalResult represents the final result of answering a question
type FinalResult struct {
	QuestionID    string           `json:"question_id"`
	Answer        string           `json:"answer"`
	Explanation   string           `json:"explanation"`
	Confidence    float64          `json:"confidence"`
	Steps         []*StepResult    `json:"steps"`
	TotalDuration float64          `json:"total_duration"`
	IsCorrect     bool             `json:"is_correct"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Question represents an HLE benchmark question
type Question struct {
	ID         string                 `json:"id"`
	Category   string                 `json:"category"`
	Content    string                 `json:"content"`
	Options    []string               `json:"options,omitempty"`
	CorrectAns string                 `json:"correct_answer"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ToolResult represents the result of tool execution
type ToolResult struct {
	Success   bool        `json:"success"`
	Output    interface{} `json:"output,omitempty"`
	Error     string      `json:"error,omitempty"`
	Duration  float64     `json:"duration_seconds"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EvaluationResult represents the evaluation result
type EvaluationResult struct {
	QuestionID    string  `json:"question_id"`
	PredictedAns  string  `json:"predicted_answer"`
	CorrectAns    string  `json:"correct_answer"`
	IsCorrect     bool    `json:"is_correct"`
	Confidence    float64 `json:"confidence"`
	Reasoning     string  `json:"reasoning,omitempty"`
	Duration      float64 `json:"duration_seconds"`
}

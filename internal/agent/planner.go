package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"go.uber.org/zap"

	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"github.com/hle-agent/hle-agent/pkg/perception"
	"github.com/hle-agent/hle-agent/pkg/prompts"
	"github.com/hle-agent/hle-agent/pkg/retriever"
	"github.com/hle-agent/hle-agent/pkg/utils"
)

// HLEPlan represents a plan for solving a question
// Implements the planexecute.Plan interface
type HLEPlan struct {
	ID          string    `json:"id"`
	QuestionID  string    `json:"question_id,omitempty"`
	Steps       []HLEStep `json:"steps"`
	FirstStepID string    `json:"first_step_id"`
}

// FirstStep returns the first step ID to be executed
func (p *HLEPlan) FirstStep() string {
	if len(p.Steps) > 0 {
		return p.Steps[0].ID
	}
	return ""
}

// MarshalJSON implements json.Marshaler
func (p *HLEPlan) MarshalJSON() ([]byte, error) {
	type Alias HLEPlan
	return json.Marshal(&struct {
		TotalSteps int `json:"total_steps"`
		*Alias
	}{
		TotalSteps: len(p.Steps),
		Alias:      (*Alias)(p),
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (p *HLEPlan) UnmarshalJSON(data []byte) error {
	type Alias HLEPlan
	aux := &struct {
		TotalSteps int `json:"total_steps"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	p.Steps = make([]HLEStep, 0, aux.TotalSteps)
	return nil
}

// HLEStep represents a step in a plan
type HLEStep struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	ToolName    string `json:"tool_name,omitempty"`
	Params      string `json:"parameters,omitempty"`
}

// HLEStepResult represents the result of executing a step
type HLEStepResult struct {
	StepID     string  `json:"id"`
	Success    bool    `json:"success"`
	Error      string  `json:"error,omitempty"`
	Output     string  `json:"output"`
	Confidence float64 `json:"confidence"`
}

// Planner generates a plan for solving a question
// Implements the factory-based pattern
type Planner struct {
	llmClient  *llm.Client
	prompts    *prompts.PromptManager
	retriever  *retriever.Retriever
	perception *perception.Perception
	logger     *zap.Logger
}

// NewPlanner creates a new Planner instance
func NewPlanner(llmClient *llm.Client, retriever *retriever.Retriever) *Planner {
	logger := logging.WithComponent("Planner")

	return &Planner{
		llmClient:  llmClient,
		prompts:    prompts.NewPromptManager(),
		retriever:  retriever,
		perception: perception.NewPerception(),
		logger:     logger,
	}
}

// CreatePlan implements planexecute.NewPlan function type
// This method is used by Eino ADK to create a new plan
func (p *Planner) CreatePlan(ctx context.Context) planexecute.Plan {
	// Extract question from context
	question := extractQuestionFromContext(ctx)

	if question == "" {
		p.logger.Warn("上下文中没有题目信息，创建默认计划")
		// 返回一个带有默认步骤的计划，确保不为空
		return &HLEPlan{
			ID:          fmt.Sprintf("plan_%d", time.Now().UnixNano()),
			FirstStepID: "step_1",
			Steps: []HLEStep{
				{
					ID:          "step_1",
					Name:        "分析问题",
					Description: "请分析并回答这个问题",
					Action:      "llm",
				},
			},
		}
	}

	plan, err := p.Plan(ctx, question)
	if err != nil {
		p.logger.Error("创建计划失败，使用默认计划", zap.Error(err))
		// 返回一个有效的默认计划，确保不为空
		return &HLEPlan{
			ID:          fmt.Sprintf("plan_%d", time.Now().UnixNano()),
			QuestionID:  safeSubstring(question, 0, 20),
			FirstStepID: "step_1",
			Steps: []HLEStep{
				{
					ID:          "step_1",
					Name:        "分析问题",
					Description: question,
					Action:      "llm",
				},
			},
		}
	}

	// 确保 plan 不为空
	if plan == nil {
		p.logger.Warn("计划为空，创建默认计划")
		return &HLEPlan{
			ID:          fmt.Sprintf("plan_%d", time.Now().UnixNano()),
			FirstStepID: "step_1",
			Steps: []HLEStep{
				{
					ID:          "step_1",
					Name:        "分析问题",
					Description: question,
					Action:      "llm",
				},
			},
		}
	}

	// 确保 FirstStepID 有效
	if plan.FirstStepID == "" && len(plan.Steps) > 0 {
		plan.FirstStepID = plan.Steps[0].ID
	}

	return plan
}

// extractQuestionFromContext 从上下文中提取题目信息
func extractQuestionFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// 尝试从 context 值中提取题目
	if question, ok := ctx.Value("question").(string); ok {
		return question
	}

	// 尝试从 context 值中提取用户消息
	if userMsg, ok := ctx.Value("user_message").(string); ok {
		return userMsg
	}

	return ""
}

// Plan generates a plan for solving the given question
func (p *Planner) Plan(ctx context.Context, question string) (*HLEPlan, error) {
	if question == "" {
		return nil, fmt.Errorf("question is empty")
	}

	logger := p.logger.With(
		zap.String("action", "Plan"),
		zap.String("question_preview", utils.TruncateString(question, 100)),
	)

	logger.Info("开始生成解题计划")

	// Get historical context if retriever is available
	var historicalContext string
	if p.retriever != nil {
		// Build context from historical questions
		historicalContext = p.retriever.BuildContextForPlanner(
			question,
			"general",
			[]string{},
			"medium",
		)

		if historicalContext != "" {
			logger.Debug("获取到历史上下文",
				zap.Int("context_length", len(historicalContext)))
		}
	}

	// Analyze question using perception layer
	analysis := p.perception.Analyze(question)

	// Select appropriate prompt template based on question type
	templateName := p.selectPromptTemplate(analysis)
	template := p.prompts.GetTemplate(templateName)
	if template == "" {
		// Fallback to general template
		template = p.prompts.GetTemplate("planner_general")
		if template == "" {
			return nil, fmt.Errorf("no prompt template available")
		}
	}

	// Build prompt with question and context
	var promptBuilder strings.Builder
	promptBuilder.WriteString(template)
	promptBuilder.WriteString("\n\n历史上下文:\n")
	if historicalContext != "" {
		promptBuilder.WriteString(historicalContext)
	} else {
		promptBuilder.WriteString("无历史记录")
	}
	promptBuilder.WriteString("\n\n题目:")
	promptBuilder.WriteString(question)

	fullPrompt := promptBuilder.String()

	logger.Debug("生成计划提示词",
		zap.String("template", templateName),
		zap.Int("prompt_length", len(fullPrompt)))

	// Generate plan using LLM
	response, err := p.llmClient.GenerateWithSystemPrompt(
		ctx,
		"你是一个专业的学术题目解题专家。请根据题目制定详细的解题计划。",
		fullPrompt,
	)
	if err != nil {
		logger.Error("生成计划失败", zap.Error(err))
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// Parse the plan from LLM response
	plan := p.parsePlan(response, question)

	if plan == nil {
		logger.Error("无法解析生成的计划")
		return nil, fmt.Errorf("failed to parse plan from response")
	}

	logger.Info("计划生成成功",
		zap.String("plan_id", plan.ID),
		zap.Int("total_steps", len(plan.Steps)))

	return plan, nil
}

// parsePlan parses the LLM response into a HLEPlan structure
func (p *Planner) parsePlan(response string, question string) *HLEPlan {
	// Try to parse as JSON first
	var planData struct {
		ID         string `json:"id"`
		QuestionID string `json:"question_id,omitempty"`
		Steps      []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
			Action      string `json:"action"`
			ToolName    string `json:"tool_name,omitempty"`
			Parameters  string `json:"parameters,omitempty"`
		} `json:"steps"`
	}

	// Try JSON parsing
	if strings.HasPrefix(strings.TrimSpace(response), "{") {
		if err := json.Unmarshal([]byte(response), &planData); err == nil {
			// Successfully parsed JSON
			steps := make([]HLEStep, len(planData.Steps))
			for i, s := range planData.Steps {
				stepID := s.ID
				if stepID == "" {
					stepID = fmt.Sprintf("step_%d", i+1)
				}
				steps[i] = HLEStep{
					ID:          stepID,
					Name:        fmt.Sprintf("step_%d", i+1),
					Description: s.Description,
					Action:      s.Action,
					ToolName:    s.ToolName,
					Params:      s.Parameters,
				}
			}

			if planData.QuestionID == "" {
				planData.QuestionID = utils.GenerateID("q")
			}

			firstStepID := ""
			if len(steps) > 0 {
				firstStepID = steps[0].ID
			}

			return &HLEPlan{
				ID:          planData.ID,
				QuestionID:  planData.QuestionID,
				Steps:       steps,
				FirstStepID: firstStepID,
			}
		}
	}

	// Fallback: Create a simple plan with one step
	stepID := fmt.Sprintf("step_%d", 1)
	steps := []HLEStep{
		{
			ID:          stepID,
			Name:        "step_1",
			Description: response,
			Action:      "llm",
		},
	}

	planID := utils.GenerateID("plan")

	return &HLEPlan{
		ID:          planID,
		QuestionID:  utils.GenerateID("q"),
		Steps:       steps,
		FirstStepID: stepID,
	}
}

// selectPromptTemplate selects the appropriate prompt template based on question analysis
func (p *Planner) selectPromptTemplate(analysis *perception.AnalysisResult) string {
	if analysis == nil || analysis.QuestionInfo == nil {
		return "planner_general"
	}

	// Check for code-related questions
	if analysis.QuestionInfo.CodePresent {
		if codeLang := strings.ToLower(analysis.QuestionInfo.CodeLanguage); strings.Contains(codeLang, "python") {
			return "planner_programming"
		}
		return "planner_cryptography"
	}

	// Use domain recognition if available
	if analysis.DomainRecognition != nil {
		domain := analysis.DomainRecognition.PrimaryDomain
		switch domain {
		case perception.DomainCryptography:
			return "planner_cryptography"
		case perception.DomainCalculation:
			return "planner_math"
		case perception.DomainProgramming:
			return "planner_programming"
		case perception.DomainRobotics:
			return "planner_robotics"
		default:
			return "planner_general"
		}
	}

	return "planner_general"
}

// safeSubstring returns a substring of s, with length limit
func safeSubstring(s string, start, length int) string {
	if start < 0 {
		start = 0
	}
	if start >= len(s) {
		return ""
	}
	if length <= 0 {
		return ""
	}
	if start+length > len(s) {
		return s[start:]
	}
	return s[start : start+length]
}

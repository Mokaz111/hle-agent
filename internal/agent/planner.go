package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"github.com/hle-agent/hle-agent/pkg/prompts"
	"github.com/hle-agent/hle-agent/pkg/utils"
	"go.uber.org/zap"
)

// Planner implements the Eino ADK Planner interface
type Planner struct {
	llmClient *llm.Client
	prompts   *prompts.PromptManager
	logger    *zap.Logger
}

// NewPlanner creates a new Planner
func NewPlanner(llmClient *llm.Client) *Planner {
	return &Planner{
		llmClient: llmClient,
		prompts:   prompts.NewPromptManager(),
		logger:    logging.WithComponent("Planner"),
	}
}

// Plan generates a解题 plan for the given question
func (p *Planner) Plan(ctx context.Context, question string) (*models.Plan, error) {
	logger := p.logger.With(
		zap.String("action", "Plan"),
		zap.String("question_preview", utils.TruncateString(question, 50)),
	)

	logger.Info("开始生成解题计划")

	// Step 1: Identify question type
	questionType := p.identifyQuestionType(question)
	logger.Debug("题型识别完成", zap.String("type", questionType))

	// Step 2: Select appropriate prompt template
	promptName := p.selectPrompt(questionType)
	logger.Debug("选择 Prompt 模板", zap.String("template", promptName))

	// Step 3: Build prompt
	prompt := p.prompts.GetTemplate(promptName)
	renderedPrompt := prompts.Render(prompt, map[string]string{
		"question": question,
	})

	logger.Debug("构建 Prompt 完成",
		zap.Int("prompt_length", len(renderedPrompt)))

	// Step 4: Call LLM to generate plan
	logger.Debug("调用 LLM 生成计划",
		zap.String("model", "gpt-4"),
		zap.Int("prompt_length", len(renderedPrompt)))

	response, err := p.llmClient.GenerateWithSystemPrompt(
		ctx,
		"你是一个专业的解题规划助手。请根据题目生成结构化的解题计划。",
		renderedPrompt,
	)
	if err != nil {
		logger.Error("LLM 调用失败", zap.Error(err))
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	logger.Debug("LLM 调用成功", zap.Int("response_length", len(response)))

	// Step 5: Parse response into Plan
	plan, err := p.parsePlan(response, question)
	if err != nil {
		logger.Warn("解析 LLM 响应失败，使用默认计划", zap.Error(err))
		plan = p.createDefaultPlan(question)
	}

	logger.Info("解题计划生成成功",
		zap.String("plan_id", plan.ID),
		zap.Int("total_steps", plan.TotalSteps),
		zap.String("first_step", plan.Steps[0].Description))

	return plan, nil
}

// identifyQuestionType identifies the type of question
func (p *Planner) identifyQuestionType(question string) string {
	questionLower := utils.ToLower(question)

	// Define type indicators
	typeIndicators := []struct {
		keywords  []string
		typeName  string
	}{
		{[]string{"cipher", "encrypt", "decrypt", "密码", "密文", "substitution"}, "cryptography"},
		{[]string{"code", "function", "class", "bug", "error", "python", "代码", "程序"}, "programming"},
		{[]string{"calculate", "compute", "sqrt", "integral", "derivative", "计算", "积分", "求导"}, "calculation"},
		{[]string{"robot", "kinematic", "运动学", "动力学"}, "robotics"},
		{[]string{"machine learning", "neural", "模型", "训练", "ML"}, "machine_learning"},
	}

	for _, ind := range typeIndicators {
		for _, keyword := range ind.keywords {
			if utils.Contains(questionLower, keyword) {
				return ind.typeName
			}
		}
	}

	return "general"
}

// selectPrompt selects the appropriate prompt template
func (p *Planner) selectPrompt(questionType string) string {
	switch questionType {
	case "cryptography":
		return "planner_cryptography"
	case "programming":
		return "planner_programming"
	case "calculation":
		return "planner_calculation"
	case "robotics":
		return "planner_robotics"
	case "machine_learning":
		return "planner_ml"
	default:
		return "planner_general"
	}
}

// parsePlan parses the LLM response into a Plan
func (p *Planner) parsePlan(response, question string) (*models.Plan, error) {
	logger := p.logger.With(zap.String("action", "parsePlan"))

	// Try to parse as JSON
	var planData struct {
		ID    string `json:"id"`
		Steps []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			Action      string `json:"action"`
			ToolName    string `json:"tool_name,omitempty"`
			Parameters  struct {
				Code string `json:"code,omitempty"`
			} `json:"parameters,omitempty"`
		} `json:"steps"`
	}

	if err := json.Unmarshal([]byte(response), &planData); err != nil {
		logger.Warn("JSON 解析失败，尝试提取",
			zap.Error(err),
			zap.Int("response_length", len(response)))
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	// Validate parsed data
	if len(planData.Steps) == 0 {
		logger.Warn("解析的计划没有步骤")
		return nil, fmt.Errorf("no steps in parsed plan")
	}

	// Convert to models.Plan
	plan := &models.Plan{
		ID:          planData.ID,
		QuestionID:  generateQuestionID(question),
		Steps:       make([]models.Step, 0, len(planData.Steps)),
		TotalSteps:  len(planData.Steps),
		CurrentStep: 1,
		Status:      models.PlanStatusPending,
	}

	for _, stepData := range planData.Steps {
		step := models.Step{
			ID:          stepData.ID,
			Description: stepData.Description,
			Action:      stepData.Action,
			ToolName:    stepData.ToolName,
			RetryOnFail: true,
		}

		if stepData.Parameters.Code != "" {
			step.Parameters = map[string]interface{}{
				"code": stepData.Parameters.Code,
			}
		}

		plan.Steps = append(plan.Steps, step)
	}

	logger.Debug("计划解析成功",
		zap.String("plan_id", plan.ID),
		zap.Int("steps_count", len(plan.Steps)))

	return plan, nil
}

// createDefaultPlan creates a default plan for the question
func (p *Planner) createDefaultPlan(question string) *models.Plan {
	return &models.Plan{
		ID:          fmt.Sprintf("plan_%d", time.Now().Unix()),
		QuestionID:  generateQuestionID(question),
		Steps: []models.Step{
			{ID: 1, Description: "理解题目要求", Action: "llm", IsKeyPoint: true},
			{ID: 2, Description: "分析关键信息", Action: "llm"},
			{ID: 3, Description: "制定解题策略", Action: "llm"},
			{ID: 4, Description: "执行计算或推理", Action: "tool", ToolName: "python_executor"},
			{ID: 5, Description: "验证答案正确性", Action: "llm", IsKeyPoint: true},
			{ID: 6, Description: "生成最终答案", Action: "llm"},
		},
		TotalSteps:  6,
		CurrentStep: 1,
		Status:      models.PlanStatusPending,
	}
}

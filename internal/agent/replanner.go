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
	"github.com/hle-agent/hle-agent/pkg/prompts"
)

// HLEReplanner evaluates step results and triggers replanning if needed
type Replanner struct {
	llmClient *llm.Client
	prompts   *prompts.PromptManager
	logger    *zap.Logger
}

// NewReplanner creates a new Replanner instance
func NewReplanner(llmClient *llm.Client) *Replanner {
	logger := logging.WithComponent("Replanner")

	return &Replanner{
		llmClient: llmClient,
		prompts:   prompts.NewPromptManager(),
		logger:    logger,
	}
}

// CreatePlan implements planexecute.NewPlan function type for replanning
// This method is used by Eino ADK to create a new plan based on execution results
func (r *Replanner) CreatePlan(ctx context.Context) planexecute.Plan {
	// Extract question from context
	question := extractQuestionFromContext(ctx)
	if question == "" {
		r.logger.Warn("上下文中没有题目信息，无法重新规划")
		return &HLEPlan{}
	}

	// For now, return a simple plan - the actual replanning logic
	// will be triggered by the Replan method when called directly
	return &HLEPlan{
		ID:          fmt.Sprintf("replan_%d", time.Now().UnixNano()),
		QuestionID:  question,
		FirstStepID: "replan_step_1",
	}
}

// Replan evaluates the current plan and step results, and determines if a new plan is needed
func (r *Replanner) Replan(ctx context.Context, plan *HLEPlan, stepResults []*HLEStepResult, question string) (*HLEPlan, error) {
	logger := r.logger.With(
		zap.String("action", "Replan"),
		zap.String("plan_id", plan.ID),
		zap.Int("total_steps", len(plan.Steps)),
		zap.Int("completed_steps", len(stepResults)),
	)

	logger.Info("开始评估执行结果")

	// Analyze step results
	analysis := r.analyzeStepResults(stepResults)

	if analysis.NeedReplan {
		logger.Info("需要重新规划",
			zap.Float64("avg_confidence", analysis.AvgConfidence),
			zap.Bool("has_errors", analysis.HasErrors))

		// Generate new plan using LLM
		newPlan, err := r.generateNewPlan(ctx, question, plan, stepResults, analysis)
		if err != nil {
			logger.Error("重新规划失败", zap.Error(err))
			return nil, fmt.Errorf("failed to generate new plan: %w", err)
		}

		logger.Info("新计划生成成功",
			zap.String("new_plan_id", newPlan.ID),
			zap.Int("new_total_steps", len(newPlan.Steps)))

		return newPlan, nil
	}

	logger.Info("无需重新规划，继续执行原计划")
	return plan, nil
}

// StepAnalysis contains analysis of step execution results
type StepAnalysis struct {
	NeedReplan    bool
	AvgConfidence float64
	HasErrors     bool
	FailedSteps   []string
	AllSuccessful bool
}

// analyzeStepResults analyzes the step results to determine if replanning is needed
func (r *Replanner) analyzeStepResults(stepResults []*HLEStepResult) *StepAnalysis {
	analysis := &StepAnalysis{
		NeedReplan:    false,
		AvgConfidence: 0.0,
		HasErrors:     false,
		FailedSteps:   []string{},
		AllSuccessful: true,
	}

	if len(stepResults) == 0 {
		analysis.NeedReplan = true
		return analysis
	}

	var totalConfidence float64
	var failedCount int

	for _, result := range stepResults {
		totalConfidence += result.Confidence

		if !result.Success {
			analysis.HasErrors = true
			analysis.AllSuccessful = false
			analysis.FailedSteps = append(analysis.FailedSteps, result.StepID)
			failedCount++
		}
	}

	analysis.AvgConfidence = totalConfidence / float64(len(stepResults))

	// Determine if replanning is needed based on confidence threshold
	confidenceThreshold := 0.7
	if analysis.AvgConfidence < confidenceThreshold {
		analysis.NeedReplan = true
	}

	// Replan if all steps failed
	if failedCount == len(stepResults) {
		analysis.NeedReplan = true
	}

	return analysis
}

// generateNewPlan generates a new plan based on the analysis
func (r *Replanner) generateNewPlan(ctx context.Context, question string, oldPlan *HLEPlan, stepResults []*HLEStepResult, analysis *StepAnalysis) (*HLEPlan, error) {
	// Build context from previous attempts
	var contextBuilder strings.Builder
	contextBuilder.WriteString("原始问题:\n")
	contextBuilder.WriteString(question)
	contextBuilder.WriteString("\n\n执行结果分析:\n")
	contextBuilder.WriteString(fmt.Sprintf("- 平均置信度: %.2f\n", analysis.AvgConfidence))
	contextBuilder.WriteString(fmt.Sprintf("- 是否有错误: %v\n", analysis.HasErrors))
	contextBuilder.WriteString(fmt.Sprintf("- 失败步骤: %v\n", analysis.FailedSteps))
	contextBuilder.WriteString("\n各步骤结果:\n")

	for _, result := range stepResults {
		status := "成功"
		if !result.Success {
			status = fmt.Sprintf("失败: %s", result.Error)
		}
		contextBuilder.WriteString(fmt.Sprintf("- 步骤 %s: %s (置信度: %.2f)\n", result.StepID, status, result.Confidence))
		if result.Output != "" {
			contextBuilder.WriteString(fmt.Sprintf("  输出: %s\n", truncateString(result.Output, 200)))
		}
	}

	context := contextBuilder.String()

	// Use LLM to generate a new plan
	systemPrompt := "你是一个专业的解题专家。之前的解题尝试没有成功或置信度较低，请根据执行结果分析，重新制定一个更有效的解题计划。"
	prompt := fmt.Sprintf(`基于以下分析结果，请制定一个新的解题计划：

%s

请生成一个新的解题计划，考虑：
1. 为什么之前的尝试可能失败
2. 是否有更好的解题思路
3. 如何避免之前的错误

请以 JSON 格式输出计划，格式如下：
{
  "id": "新计划ID",
  "steps": [
    {
      "id": "步骤ID",
      "description": "步骤描述",
      "action": "llm 或使用具体工具名",
      "tool_name": "工具名（如果有）",
      "parameters": "参数（如果有）"
    }
  ]
}`, context)

	response, err := r.llmClient.GenerateWithSystemPrompt(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, err
	}

	// Parse the new plan
	newPlan := r.parsePlan(response, question)

	if newPlan == nil {
		// Fallback: return modified old plan
		newPlan = &HLEPlan{
			ID:         fmt.Sprintf("plan_%d", time.Now().UnixNano()),
			QuestionID: oldPlan.QuestionID,
			Steps:      oldPlan.Steps,
		}
		if len(newPlan.Steps) > 0 {
			newPlan.FirstStepID = newPlan.Steps[0].ID
		}
	}

	return newPlan, nil
}

// parsePlan parses the LLM response into a HLEPlan structure
func (r *Replanner) parsePlan(response string, question string) *HLEPlan {
	// Try to parse as JSON first
	var planData struct {
		ID    string `json:"id"`
		Steps []struct {
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

			if planData.ID == "" {
				planData.ID = fmt.Sprintf("plan_%d", time.Now().UnixNano())
			}

			firstStepID := ""
			if len(steps) > 0 {
				firstStepID = steps[0].ID
			}

			return &HLEPlan{
				ID:          planData.ID,
				QuestionID:  fmt.Sprintf("q_%d", time.Now().UnixNano()),
				Steps:       steps,
				FirstStepID: firstStepID,
			}
		}
	}

	// Fallback: return nil to use old plan
	return nil
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

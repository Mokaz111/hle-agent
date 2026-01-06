package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"github.com/hle-agent/hle-agent/pkg/prompts"
	"go.uber.org/zap"
)

// Replanner implements the Eino ADK Replanner interface
type Replanner struct {
	llmClient *llm.Client
	prompts   *prompts.PromptManager
	logger    *zap.Logger
}

// NewReplanner creates a new Replanner
func NewReplanner(llmClient *llm.Client) *Replanner {
	return &Replanner{
		llmClient: llmClient,
		prompts:   prompts.NewPromptManager(),
		logger:    logging.WithComponent("Replanner"),
	}
}

// Replan evaluates the execution results and decides whether to replan
func (r *Replanner) Replan(
	ctx context.Context,
	plan *models.Plan,
	stepResults []*models.StepResult,
	question string,
) (*models.Plan, error) {
	logger := r.logger.With(
		zap.String("action", "Replan"),
		zap.String("plan_id", plan.ID),
		zap.Int("completed_steps", len(stepResults)),
		zap.Int("total_steps", plan.TotalSteps),
	)

	logger.Debug("评估执行结果")

	// Analyze results
	successCount := 0
	failCount := 0

	for _, result := range stepResults {
		if result.Success {
			successCount++
		} else {
			failCount++
		}
	}

	logger.Debug("执行结果统计",
		zap.Int("success_count", successCount),
		zap.Int("fail_count", failCount),
		zap.Float64("success_rate", float64(successCount)/float64(len(stepResults))))

	// Check if all steps completed successfully
	if successCount == plan.TotalSteps {
		logger.Info("所有步骤执行成功，无需重规划")
		return nil, nil // No replanning needed, plan completed
	}

	// Check if more than half the steps failed
	failRate := float64(failCount) / float64(plan.TotalSteps)
	if failRate > 0.5 {
		logger.Warn("超过一半步骤失败，需要重新规划",
			zap.Float64("fail_rate", failRate))
		// Need to generate a new plan
		return r.generateNewPlan(ctx, question, stepResults)
	}

	// Check if only the last step failed
	if len(stepResults) > 0 {
		lastResult := stepResults[len(stepResults)-1]
		if !lastResult.Success && lastResult.Error != "" {
			logger.Warn("最后一步失败，尝试修复",
				zap.String("error", lastResult.Error))
			// Try to fix the current step
			fixedPlan := r.fixCurrentStep(plan, stepResults)
			return fixedPlan, nil
		}
	}

	// Continue with the current plan
	logger.Debug("继续执行当前计划")
	return plan, nil
}

// generateNewPlan generates a new plan based on the failed results
func (r *Replanner) generateNewPlan(
	ctx context.Context,
	question string,
	stepResults []*models.StepResult,
) (*models.Plan, error) {
	logger := r.logger.With(
		zap.String("action", "generateNewPlan"),
	)

	// Format results for the prompt
	formattedResults := r.formatResults(stepResults)

	logger.Debug("格式化执行结果用于重规划",
		zap.Int("results_count", len(stepResults)))

	// Build prompt
	prompt := r.prompts.GetTemplate("replanner")
	renderedPrompt := prompts.Render(prompt, map[string]string{
		"question":    question,
		"old_results": formattedResults,
	})

	// Call LLM to generate new plan
	logger.Debug("调用 LLM 生成新计划")
	response, err := r.llmClient.GenerateWithSystemPrompt(
		ctx,
		"你是一个专业的解题规划专家。请根据失败的原因为题目生成新的解题计划。",
		renderedPrompt,
	)
	if err != nil {
		logger.Error("LLM 调用失败", zap.Error(err))
		return nil, fmt.Errorf("failed to generate new plan: %w", err)
	}

	logger.Debug("LLM 调用成功，解析响应",
		zap.Int("response_length", len(response)))

	// Parse response
	var replanData struct {
		NeedReplan bool   `json:"need_replan"`
		Reason     string `json:"reason"`
		NewSteps   []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			Action      string `json:"action"`
			ToolName    string `json:"tool_name,omitempty"`
		} `json:"new_steps"`
	}

	if err := json.Unmarshal([]byte(response), &replanData); err != nil {
		logger.Warn("解析 LLM 响应失败，使用修复策略",
			zap.Error(err))
		// If parsing fails, return the original plan with modifications
		return r.fixCurrentStep(&models.Plan{}, stepResults), nil
	}

	// Check if replanning is needed
	if !replanData.NeedReplan {
		logger.Info("LLM 建议不需重规划",
			zap.String("reason", replanData.Reason))

		// Create a copy of the original plan
		newPlan := &models.Plan{
			ID:          fmt.Sprintf("replan_%d", len(stepResults)),
			Steps:       make([]models.Step, 0),
			TotalSteps:  len(replanData.NewSteps),
			CurrentStep: len(stepResults) + 1,
			Status:      models.PlanStatusNeedReplan,
		}

		for _, stepData := range replanData.NewSteps {
			newPlan.Steps = append(newPlan.Steps, models.Step{
				ID:          stepData.ID,
				Description: stepData.Description,
				Action:      stepData.Action,
				ToolName:    stepData.ToolName,
				RetryOnFail: true,
			})
		}

		logger.Info("生成新计划",
			zap.String("plan_id", newPlan.ID),
			zap.Int("new_steps", len(newPlan.Steps)))

		return newPlan, nil
	}

	// Return original plan (no change)
	logger.Info("按照 LLM 建议继续当前计划")
	return nil, nil
}

// fixCurrentStep fixes the current failed step
func (r *Replanner) fixCurrentStep(plan *models.Plan, stepResults []*models.StepResult) *models.Plan {
	logger := r.logger.With(
		zap.String("action", "fixCurrentStep"),
		zap.Int("current_step", len(stepResults)+1),
	)

	// Create a copy of the original plan
	newPlan := &models.Plan{
		ID:          fmt.Sprintf("fix_%d", len(stepResults)),
		QuestionID:  plan.QuestionID,
		Steps:       make([]models.Step, len(plan.Steps)),
		TotalSteps:  plan.TotalSteps,
		CurrentStep: len(stepResults) + 1,
		Status:      models.PlanStatusExecuting,
	}

	// Copy existing steps
	if len(plan.Steps) > 0 {
		copy(newPlan.Steps, plan.Steps)
	}

	// Modify the current step to retry
	currentIdx := len(stepResults)
	if currentIdx < len(newPlan.Steps) {
		newPlan.Steps[currentIdx].Description += " (重试)"
		newPlan.Steps[currentIdx].RetryOnFail = true
		logger.Debug("修改当前步骤为重试模式",
			zap.Int("step_id", currentIdx+1))
	}

	logger.Info("生成修复计划",
		zap.String("plan_id", newPlan.ID),
		zap.Int("current_step", newPlan.CurrentStep))

	return newPlan
}

// formatResults formats the step results for the prompt
func (r *Replanner) formatResults(results []*models.StepResult) string {
	var sb strings.Builder

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("步骤 %d:\n", i+1))
		sb.WriteString(fmt.Sprintf("  步骤ID: %d\n", result.StepID))
		sb.WriteString(fmt.Sprintf("  成功: %v\n", result.Success))
		if result.Error != "" {
			sb.WriteString(fmt.Sprintf("  错误: %s\n", result.Error))
		}
		if result.Output != nil {
			sb.WriteString(fmt.Sprintf("  输出类型: %s\n", fmt.Sprintf("%T", result.Output)))
		}
		if result.Duration > 0 {
			sb.WriteString(fmt.Sprintf("  耗时: %.2fs\n", result.Duration))
		}
		sb.WriteString(fmt.Sprintf("  置信度: %.2f%%\n", result.Confidence*100))
		sb.WriteString("\n")
	}

	return sb.String()
}

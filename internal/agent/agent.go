package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/feedback"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"github.com/hle-agent/hle-agent/pkg/memory"
	"github.com/hle-agent/hle-agent/pkg/perception"
	"github.com/hle-agent/hle-agent/pkg/retriever"
	"github.com/hle-agent/hle-agent/pkg/utils"
	"go.uber.org/zap"
)

// HLEAgent represents the HLE Exam Answering Agent
type HLEAgent struct {
	cfg           *config.Config
	llmClient     *llm.Client
	planner       *Planner
	executor      *Executor
	replanner     *Replanner
	toolRegistry  *ToolRegistry
	shortTermMem  *memory.ShortTermMemory
	perception    *perception.Perception
	retriever     *retriever.Retriever
	feedback      *feedback.Feedback
	logger        *zap.Logger
}

// NewHLEAgent creates a new HLE Agent instance
func NewHLEAgent(cfg *config.Config) (*HLEAgent, error) {
	// Initialize logging if not already done
	logger := logging.WithComponent("HLEAgent")

	logger.Info("初始化 HLE Agent",
		zap.String("version", cfg.App.Version),
		zap.Int("max_iterations", cfg.Agent.MaxIterations))

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("配置验证失败", zap.Error(err))
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// Create LLM client
	llmClient := llm.NewClient(&cfg.Model)

	// Create tool registry
	toolRegistry := NewToolRegistry()

	// Create Planner
	planner := NewPlanner(llmClient)

	// Create Executor
	executor := NewExecutor(toolRegistry)

	// Create Replanner
	replanner := NewReplanner(llmClient)

	// Create short-term memory
	shortTermMem := memory.NewDefaultMemory()
	shortTermMem.StartSession(generateSessionID())

	// Create perception layer
	perceptionLayer := perception.NewPerception()

	// Create feedback layer
	feedbackLayer := feedback.NewFeedback()

	// Create retriever
	retrieverLayer := retriever.NewRetriever(shortTermMem, nil)

	agent := &HLEAgent{
		cfg:           cfg,
		llmClient:     llmClient,
		planner:       planner,
		executor:      executor,
		replanner:     replanner,
		toolRegistry:  toolRegistry,
		shortTermMem:  shortTermMem,
		perception:    perceptionLayer,
		retriever:     retrieverLayer,
		feedback:      feedbackLayer,
		logger:        logger,
	}

	logger.Info("HLE Agent 初始化完成")
	return agent, nil
}

// Process processes a single question and returns the answer
func (a *HLEAgent) Process(ctx context.Context, question string) (*AnswerResult, error) {
	logger := a.logger.With(
		zap.String("action", "Process"),
		zap.String("question_preview", utils.TruncateString(question, 100)),
	)

	logger.Info("开始处理问题")

	startTime := time.Now()

	// Step 0: Perception layer - analyze the question
	preprocessedQuestion := a.perception.Preprocess(question)
	analysisResult := a.perception.Analyze(preprocessedQuestion)

	logger.Info("问题感知分析完成",
		zap.String("domain", string(analysisResult.Domain)),
		zap.String("sub_domain", analysisResult.SubDomain),
		zap.String("complexity", string(analysisResult.Complexity)),
		zap.Bool("has_code", analysisResult.CodePresent),
		zap.Strings("keywords", analysisResult.Keywords))

	// Step 0.5: Add question to short-term memory with perception metadata
	a.shortTermMem.AddQuestion(preprocessedQuestion, map[string]interface{}{
		"domain":     analysisResult.Domain,
		"sub_domain": analysisResult.SubDomain,
		"complexity": analysisResult.Complexity,
		"has_code":   analysisResult.CodePresent,
		"code_lang":  analysisResult.CodeLanguage,
		"keywords":   analysisResult.Keywords,
	})

	logger.Debug("问题已添加到短期记忆",
		zap.Int("history_size", len(a.shortTermMem.GetRecentHistory(10))))

	// Step 0.7: Retrieve similar historical knowledge
	retrievedContext := a.retriever.BuildContextForPlanner(
		preprocessedQuestion,
		string(analysisResult.Domain),
		analysisResult.Keywords,
		string(analysisResult.Complexity),
	)

	if retrievedContext != "" {
		logger.Info("检索到相关历史知识",
			zap.Int("context_length", len(retrievedContext)))
	}

	// Step 1: Generate plan using Planner (can use perception info and retrieved context)
	plan, err := a.planner.Plan(ctx, question)
	if err != nil {
		logger.Error("生成计划失败", zap.Error(err))
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	logger.Info("计划生成成功",
		zap.Int("total_steps", plan.TotalSteps),
		zap.String("plan_id", plan.ID))

	// Store plan in memory
	a.shortTermMem.AddPlan(plan)

	// Step 2: Execute the plan using Executor
	stepResults, err := a.executePlanWithMemory(ctx, plan)
	if err != nil {
		logger.Error("执行计划失败", zap.Error(err))
		return nil, fmt.Errorf("failed to execute plan: %w", err)
	}

	// Store step results in memory
	for _, result := range stepResults {
		a.shortTermMem.AddStepResult(result)
	}

	// Step 3: Evaluate results using Replanner and iterate if needed
	var finalResult *models.FinalResult
	for iteration := 0; iteration < a.cfg.Agent.MaxIterations; iteration++ {
		logger.Debug("开始重规划迭代",
			zap.Int("iteration", iteration),
			zap.Int("completed_steps", len(stepResults)))

		// Check if we need to replan
		newPlan, err := a.replanner.Replan(ctx, plan, stepResults, question)
		if err != nil {
			logger.Error("重规划失败", zap.Error(err))
			return nil, fmt.Errorf("failed to replan: %w", err)
		}

		// If no replanning needed and plan is complete, finish
		if newPlan == nil && isPlanComplete(stepResults, plan) {
			finalResult = buildFinalResult(question, stepResults)
			logger.Info("计划执行完成，无需重规划")
			break
		}

		// If we need to replan, update the plan
		if newPlan != nil {
			plan = newPlan
			logger.Info("执行重规划",
				zap.Int("new_plan_steps", len(plan.Steps)))

			// Store updated plan
			a.shortTermMem.AddPlan(plan)

			stepResults, err = a.executePlanWithMemory(ctx, plan)
			if err != nil {
				logger.Error("执行重规划步骤失败", zap.Error(err))
				return nil, fmt.Errorf("failed to execute replanned steps: %w", err)
			}

			// Store new step results
			for _, result := range stepResults {
				a.shortTermMem.AddStepResult(result)
			}
		}
	}

	// If we exited the loop without a result, use what we have
	if finalResult == nil {
		finalResult = buildFinalResult(question, stepResults)
		logger.Warn("达到最大迭代次数，使用当前结果")
	}

	// Store answer in memory
	a.shortTermMem.AddAnswer(finalResult.Answer, finalResult.Confidence)

	// Feedback layer - validate and classify the result
	feedbackRecord := a.feedback.Process(
		generateSessionID(),
		question,
		finalResult.Answer,
		stepResults,
	)

	logger.Info("反馈处理完成",
		zap.Bool("is_acceptable", feedbackRecord.IsAcceptable),
		zap.Float64("final_confidence", feedbackRecord.FinalConfidence),
		zap.Int("issues_count", len(feedbackRecord.ValidateResult.Issues)))

	// Log session statistics
	stats := a.shortTermMem.GetStatistics()
	logger.Info("问题处理完成",
		zap.Float64("duration_seconds", time.Since(startTime).Seconds()),
		zap.Float64("confidence", finalResult.Confidence),
		zap.Bool("is_correct", finalResult.IsCorrect),
		zap.Int("total_steps", stats.TotalSteps),
		zap.Float64("avg_confidence", stats.AvgConfidence))
	duration := time.Since(startTime).Seconds()
	result := &AnswerResult{
		Answer:        finalResult.Answer,
		Explanation:   finalResult.Explanation,
		Confidence:    finalResult.Confidence,
		TotalDuration: duration,
		Steps:         stepResults,
	}

	logger.Info("问题处理完成",
		zap.Float64("duration_seconds", duration),
		zap.Float64("confidence", finalResult.Confidence),
		zap.Bool("is_correct", finalResult.IsCorrect),
		zap.Int("total_steps", len(stepResults)))

	return result, nil
}

// executePlan executes the plan and returns step results
func (a *HLEAgent) executePlan(ctx context.Context, plan *models.Plan) ([]*models.StepResult, error) {
	logger := a.logger.With(
		zap.String("action", "executePlan"),
		zap.String("plan_id", plan.ID),
		zap.Int("total_steps", plan.TotalSteps),
	)

	results := make([]*models.StepResult, 0)

	for _, step := range plan.Steps {
		// Skip already completed steps
		if step.ID <= len(results) {
			continue
		}

		logger.Debug("执行步骤",
			zap.Int("step_id", step.ID),
			zap.String("description", step.Description),
			zap.String("action", step.Action),
			zap.String("tool", step.ToolName))

		// Execute the step
		result, err := a.executor.Execute(ctx, &step, map[string]interface{}{
			"question":   plan.QuestionID,
			"plan_steps": len(plan.Steps),
		})
		if err != nil {
			logger.Error("步骤执行失败",
				zap.Int("step_id", step.ID),
				zap.Error(err))
			return nil, fmt.Errorf("step %d execution failed: %w", step.ID, err)
		}

		results = append(results, result)

		logger.Debug("步骤执行完成",
			zap.Int("step_id", step.ID),
			zap.Bool("success", result.Success),
			zap.Float64("confidence", result.Confidence),
			zap.Float64("duration_seconds", result.Duration))

		// Check for critical failure
		if !result.Success && step.IsKeyPoint {
			logger.Warn("关键步骤执行失败",
				zap.Int("step_id", step.ID),
				zap.String("description", step.Description))
			break
		}
	}

	return results, nil
}

// executePlanWithMemory executes the plan and stores results in memory
func (a *HLEAgent) executePlanWithMemory(ctx context.Context, plan *models.Plan) ([]*models.StepResult, error) {
	results, err := a.executePlan(ctx, plan)
	if err != nil {
		return nil, err
	}

	// Store reasoning from step results
	for _, result := range results {
		if result.Success && result.Output != nil {
			if output, ok := result.Output.(map[string]interface{}); ok {
				if reasoning, ok := output["reasoning"].(string); ok {
					a.shortTermMem.AddReasoning(reasoning, result.Confidence)
				}
			}
		}
	}

	return results, nil
}

// ProcessBatch processes multiple questions
func (a *HLEAgent) ProcessBatch(ctx context.Context, questions []string) ([]*AnswerResult, error) {
	logger := a.logger.With(
		zap.String("action", "ProcessBatch"),
		zap.Int("question_count", len(questions)),
	)

	logger.Info("开始批量处理问题")

	results := make([]*AnswerResult, 0, len(questions))

	for i, question := range questions {
		logger.Debug("处理问题",
			zap.Int("index", i+1),
			zap.Int("total", len(questions)))

		result, err := a.Process(ctx, question)
		if err != nil {
			logger.Error("处理问题失败",
				zap.Int("index", i+1),
				zap.Error(err))
			return nil, fmt.Errorf("failed to process question at index %d: %w", i, err)
		}
		results = append(results, result)
	}

	logger.Info("批量处理完成",
		zap.Int("total_processed", len(results)),
		zap.Float64("success_rate", float64(len(results))/float64(len(questions))))

	return results, nil
}

// RegisterTool registers a tool with the agent
func (a *HLEAgent) RegisterTool(tool Tool) {
	a.logger.Info("注册工具",
		zap.String("tool_name", tool.Name()),
		zap.String("tool_description", tool.Description()))

	a.toolRegistry.Register(tool)
}

// GetConfig returns the agent configuration
func (a *HLEAgent) GetConfig() *config.Config {
	return a.cfg
}

// AnswerResult represents the result of answering a question
type AnswerResult struct {
	Answer        string               `json:"answer"`
	Explanation   string               `json:"explanation"`
	Confidence    float64              `json:"confidence"`
	TotalDuration float64              `json:"total_duration"`
	Steps         []*models.StepResult `json:"steps"`
}

// isPlanComplete checks if the plan execution is complete
func isPlanComplete(results []*models.StepResult, plan *models.Plan) bool {
	return len(results) >= plan.TotalSteps
}

// buildFinalResult builds the final result from step results
func buildFinalResult(question string, stepResults []*models.StepResult) *models.FinalResult {
	// Calculate average confidence
	var totalConf float64
	for _, result := range stepResults {
		totalConf += result.Confidence
	}
	avgConf := float64(0)
	if len(stepResults) > 0 {
		avgConf = totalConf / float64(len(stepResults))
	}

	// Extract answer from the last successful step
	answer := "No answer generated"
	for i := len(stepResults) - 1; i >= 0; i-- {
		if stepResults[i].Success && stepResults[i].Output != nil {
			if output, ok := stepResults[i].Output.(map[string]interface{}); ok {
				if conclusion, ok := output["conclusion"].(string); ok {
					answer = conclusion
				}
			}
			break
		}
	}

	return &models.FinalResult{
		QuestionID:    generateQuestionID(question),
		Answer:        answer,
		Explanation:   generateExplanation(stepResults),
		Confidence:    avgConf,
		Steps:         stepResults,
		TotalDuration: calculateTotalDuration(stepResults),
		IsCorrect:     avgConf > 0.7,
	}
}

// generateQuestionID generates a unique ID for the question
func generateQuestionID(question string) string {
	return fmt.Sprintf("q_%d", time.Now().UnixNano()%1000000)
}

// generateExplanation generates an explanation from step results
func generateExplanation(results []*models.StepResult) string {
	var sb strings.Builder
	sb.WriteString("解题过程：\n")
	for i, result := range results {
		sb.WriteString(fmt.Sprintf("%d. Step %d\n", i+1, result.StepID))
		if result.Success {
			sb.WriteString(fmt.Sprintf("   ✓ 成功 (置信度: %.0f%%)\n", result.Confidence*100))
		} else {
			sb.WriteString(fmt.Sprintf("   ✗ 失败: %s\n", result.Error))
		}
	}
	return sb.String()
}

// calculateTotalDuration calculates total duration from step results
func calculateTotalDuration(results []*models.StepResult) float64 {
	var total float64
	for _, result := range results {
		total += result.Duration
	}
	return total
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano()%1000000)
}

// GetMemory returns the short-term memory instance
func (a *HLEAgent) GetMemory() *memory.ShortTermMemory {
	return a.shortTermMem
}

// ClearMemory clears the short-term memory
func (a *HLEAgent) ClearMemory() {
	a.shortTermMem.Clear()
	a.shortTermMem.StartSession(generateSessionID())
}

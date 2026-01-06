package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"go.uber.org/zap"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/feedback"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"github.com/hle-agent/hle-agent/pkg/memory"
	"github.com/hle-agent/hle-agent/pkg/perception"
	"github.com/hle-agent/hle-agent/pkg/retriever"
	"github.com/hle-agent/hle-agent/pkg/utils"
	pythonTool "github.com/hle-agent/hle-agent/tools/python_executor"
	sageTool "github.com/hle-agent/hle-agent/tools/sagemath_executor"
)

// HLEAgent represents the HLE Exam Answering Agent
type HLEAgent struct {
	cfg          *config.Config
	llmClient    *llm.Client
	planner      *Planner
	executor     *Executor
	replanner    *Replanner
	agent        adk.Agent
	toolRegistry *ToolRegistry
	shortTermMem *memory.ShortTermMemory
	perception   *perception.Perception
	retriever    *retriever.Retriever
	feedback     *feedback.Feedback
	logger       *zap.Logger
	agentName    string
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
	llmClient, err := llm.NewClient(&cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}

	// Create tool registry
	toolRegistry := NewToolRegistry()

	// Register available tools
	pythonExecutor := pythonTool.NewPythonExecutor(300) // 5分钟超时
	sageExecutor := sageTool.NewSageMathExecutor(300)

	toolRegistry.Register(pythonExecutor)
	toolRegistry.Register(sageExecutor)

	logger.Info("已注册工具",
		zap.String("python_executor", pythonExecutor.Name()),
		zap.String("sagemath_executor", sageExecutor.Name()))

	// Create retriever
	shortTermMem := memory.NewDefaultMemory()
	shortTermMem.StartSession(generateSessionID())
	retrieverLayer := retriever.NewRetriever(shortTermMem, nil)

	// Create Planner
	planner := NewPlanner(llmClient, retrieverLayer)

	// Create Executor
	executor := NewExecutor(toolRegistry)

	// Create Replanner
	replanner := NewReplanner(llmClient)

	// Create Eino ADK Plan-Execute Agent using factory functions
	ctx := context.Background()

	// 获取 LLM 客户端的 chatModel
	chatModel := llmClient.GetChatModel().(*openai.ChatModel)

	// Create planner agent with our custom planner
	// 使用 ChatModelWithFormattedOutput 避免 Eino ADK 内部 panic
	plannerAgent, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ChatModelWithFormattedOutput: chatModel,
		NewPlan:                      planner.CreatePlan,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Planner 失败: %w", err)
	}

	// Create executor agent with our custom executor
	executorAgent, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: chatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Executor 失败: %w", err)
	}

	// Create replanner agent with our custom replanner
	replannerAgent, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: chatModel,
		NewPlan:   replanner.CreatePlan,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Replanner 失败: %w", err)
	}

	// Create the main plan-execute agent
	einoAgent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:   plannerAgent,
		Executor:  executorAgent,
		Replanner: replannerAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Eino Agent 失败: %w", err)
	}

	// Create perception layer
	perceptionLayer := perception.NewPerception()

	// Create feedback layer
	feedbackLayer := feedback.NewFeedback()

	agent := &HLEAgent{
		cfg:          cfg,
		llmClient:    llmClient,
		planner:      planner,
		executor:     executor,
		replanner:    replanner,
		agent:        einoAgent,
		toolRegistry: toolRegistry,
		shortTermMem: shortTermMem,
		perception:   perceptionLayer,
		retriever:    retrieverLayer,
		feedback:     feedbackLayer,
		logger:       logger,
		agentName:    "hle-agent",
	}

	logger.Info("HLE Agent 初始化完成")
	return agent, nil
}

// GetName returns the agent name
func (a *HLEAgent) GetName() string {
	return a.agentName
}

// Run executes the agent with the given input
// Implements the Eino ADK Agent interface
func (a *HLEAgent) Run(ctx context.Context, input *adk.AgentInput, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	return a.agent.Run(ctx, input, opts...)
}

// Process processes a single question and returns the answer (legacy method)
func (a *HLEAgent) Process(ctx context.Context, question string) (*AnswerResult, error) {
	logger := a.logger.With(
		zap.String("action", "Process"),
		zap.String("question_preview", utils.TruncateString(question, 100)),
	)

	logger.Info("开始处理问题")

	startTime := time.Now()

	// Start new session
	sessionID := generateSessionID()
	a.shortTermMem.StartSession(sessionID)

	// Step 0: Perception layer - analyze the question
	analysis := a.perception.Analyze(question)

	logger.Debug("感知层分析完成",
		zap.String("domain", string(analysis.Domain)),
		zap.Bool("has_code", analysis.QuestionInfo.CodePresent))

	// Store question in short-term memory
	a.shortTermMem.AddQuestion(question, analysis)

	// Step 1: Retrieve similar historical problems (if configured)
	var historicalContext string
	if a.retriever != nil {
		recommendedTool := a.perception.GetRecommendedTool(question)
		tools := []string{}
		if recommendedTool != "" {
			tools = append(tools, recommendedTool)
		}

		complexity := "medium"
		if analysis.QuestionInfo != nil {
			if analysis.QuestionInfo.Complexity == "high" {
				complexity = "high"
			} else if analysis.QuestionInfo.Complexity == "low" {
				complexity = "low"
			}
		}

		historicalContext = a.retriever.BuildContextForPlanner(
			question,
			string(analysis.Domain),
			tools,
			complexity,
		)

		if historicalContext != "" {
			logger.Debug("获取到历史相似问题",
				zap.Int("context_length", len(historicalContext)))
		}
	}

	// Step 2: Generate plan using custom Planner
	plan, err := a.planner.Plan(ctx, question)
	if err != nil {
		logger.Error("计划生成失败", zap.Error(err))
		return nil, fmt.Errorf("计划生成失败: %w", err)
	}

	// Store plan in memory
	a.shortTermMem.AddPlan(&models.Plan{
		ID:         plan.ID,
		QuestionID: plan.QuestionID,
		Steps:      convertSteps(plan.Steps),
		TotalSteps: len(plan.Steps),
	})

	logger.Info("计划生成成功",
		zap.String("plan_id", plan.ID),
		zap.Int("total_steps", len(plan.Steps)))

	// Execute plan steps using custom Executor
	stepResults := make([]*HLEStepResult, 0, len(plan.Steps))
	maxIterations := a.cfg.Agent.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	var finalAnswer string
	var finalConfidence float64

	for i := 0; i < len(plan.Steps) && i < maxIterations; i++ {
		step := &plan.Steps[i]

		logger.Info("开始执行步骤",
			zap.Int("step_index", i+1),
			zap.Int("total_steps", len(plan.Steps)),
			zap.String("step_description", step.Description))

		// Execute step
		stepResult, err := a.executor.Execute(ctx, plan, step)
		if err != nil {
			logger.Error("步骤执行出错", zap.Error(err))
			stepResult = &HLEStepResult{
				StepID:     step.ID,
				Success:    false,
				Error:      err.Error(),
				Output:     "",
				Confidence: 0.0,
			}
		}

		stepResults = append(stepResults, stepResult)

		// Store step result in memory
		stepID, _ := strconv.Atoi(step.ID)
		a.shortTermMem.AddStepResult(&models.StepResult{
			StepID:     stepID,
			Success:    stepResult.Success,
			Output:     stepResult.Output,
			Confidence: stepResult.Confidence,
			Timestamp:  time.Now().Format(time.RFC3339),
		})

		// Check if this is the final step and generate answer
		if i == len(plan.Steps)-1 || i == maxIterations-1 {
			finalAnswer = stepResult.Output
			finalConfidence = stepResult.Confidence
		}

		// Check if replanning is needed
		if i < len(plan.Steps)-1 {
			analysis := a.replanner.analyzeStepResults(stepResults)
			if analysis.NeedReplan {
				logger.Info("触发重新规划",
					zap.Float64("avg_confidence", analysis.AvgConfidence))

				// Replan
				newPlan, err := a.replanner.Replan(ctx, plan, stepResults, question)
				if err != nil {
					logger.Error("重新规划失败", zap.Error(err))
				} else if newPlan != nil {
					plan = newPlan
					logger.Info("新计划生成成功",
						zap.String("new_plan_id", plan.ID),
						zap.Int("new_total_steps", len(plan.Steps)))
				}
			}
		}
	}

	// Generate final answer using reasoning if available
	allStepResults := a.shortTermMem.GetStepResults()
	if len(allStepResults) > 0 {
		var reasoningParts []string
		for _, sr := range allStepResults {
			if sr.Output != nil {
				reasoningParts = append(reasoningParts, fmt.Sprintf("步骤 %d: %v", sr.StepID, sr.Output))
			}
		}
		if len(reasoningParts) > 0 {
			combinedReasoning := strings.Join(reasoningParts, "\n\n")
			// Use LLM to synthesize final answer
			synthesizedAnswer, err := a.synthesizeAnswer(ctx, question, finalAnswer, combinedReasoning)
			if err == nil {
				finalAnswer = synthesizedAnswer
			}
		}
	}

	// Store answer in memory
	a.shortTermMem.AddAnswer(finalAnswer, finalConfidence)

	// Process feedback
	questionID := generateQuestionID()
	feedbackRecord := a.feedback.Process(
		questionID,
		question,
		finalAnswer,
		allStepResults,
	)

	// Get explanation from feedback
	explanation := ""
	if feedbackRecord != nil && feedbackRecord.ValidateResult != nil {
		if msg, ok := feedbackRecord.ValidateResult.Details["message"].(string); ok {
			explanation = msg
		} else if len(feedbackRecord.ValidateResult.Issues) > 0 {
			// Build message from issues
			var issueMessages []string
			for _, issue := range feedbackRecord.ValidateResult.Issues {
				issueMessages = append(issueMessages, issue.Message)
			}
			explanation = strings.Join(issueMessages, "; ")
		}
	}

	logger.Info("问题处理完成",
		zap.String("answer_preview", utils.TruncateString(finalAnswer, 100)),
		zap.Float64("confidence", finalConfidence))

	duration := time.Since(startTime).Seconds()

	// Convert step results to models.StepResult
	modelStepResults := make([]*models.StepResult, len(stepResults))
	for i, sr := range stepResults {
		stepID, _ := strconv.Atoi(sr.StepID)
		modelStepResults[i] = &models.StepResult{
			StepID:     stepID,
			Success:    sr.Success,
			Output:     sr.Output,
			Confidence: sr.Confidence,
			Timestamp:  time.Now().Format(time.RFC3339),
		}
	}

	return &AnswerResult{
		Answer:        finalAnswer,
		Explanation:   explanation,
		Confidence:    finalConfidence,
		TotalDuration: duration,
		Steps:         modelStepResults,
	}, nil
}

// synthesizeAnswer uses LLM to synthesize a final answer from reasoning trace
func (a *HLEAgent) synthesizeAnswer(ctx context.Context, question, answer, reasoning string) (string, error) {
	prompt := fmt.Sprintf(`基于以下推理过程，请给出最终答案：

问题: %s

推理过程:
%s

初步答案:
%s

请综合推理过程，给出最终答案。如果推理过程已经完整且正确，请直接返回初步答案。如果需要修正，请给出修正后的答案。`, question, reasoning, answer)

	return a.llmClient.GenerateWithSystemPrompt(ctx,
		"你是一个专业的学术解题专家。请综合推理过程给出最终答案。",
		prompt)
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
		logger.Info("处理问题",
			zap.Int("current", i+1),
			zap.Int("total", len(questions)),
			zap.String("preview", utils.TruncateString(question, 50)))

		result, err := a.Process(ctx, question)
		if err != nil {
			logger.Error("问题处理失败",
				zap.Int("index", i),
				zap.Error(err))

			// Add failed result
			results = append(results, &AnswerResult{
				Answer:     "",
				Confidence: 0,
				Steps:      []*models.StepResult{},
			})
			continue
		}

		results = append(results, result)
	}

	logger.Info("批量处理完成",
		zap.Int("total", len(questions)),
		zap.Int("success", len(results)))

	return results, nil
}

// AnswerResult represents the result of answering a question
type AnswerResult struct {
	Answer        string               `json:"answer"`
	Explanation   string               `json:"explanation"`
	Confidence    float64              `json:"confidence"`
	TotalDuration float64              `json:"total_duration"`
	Steps         []*models.StepResult `json:"steps"`
}

// Helper functions
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

func generateQuestionID() string {
	return fmt.Sprintf("q_%d", time.Now().UnixNano())
}

// convertSteps converts HLEStep to models.Step
func convertSteps(einoSteps []HLEStep) []models.Step {
	steps := make([]models.Step, len(einoSteps))
	for i, s := range einoSteps {
		// Convert string ID to int
		stepID, _ := strconv.Atoi(s.ID)
		steps[i] = models.Step{
			ID:          stepID,
			Description: s.Description,
			Action:      s.Action,
			ToolName:    s.ToolName,
		}
	}
	return steps
}

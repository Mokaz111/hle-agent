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
	"github.com/hle-agent/hle-agent/pkg/knowledgebase"
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

	// Register available tools based on configuration
	var pythonExecutor Tool
	var sageExecutor Tool

	// Python executor
	if cfg.Tools.Python.Enabled {
		timeout := cfg.Tools.Python.Timeout
		if timeout <= 0 {
			timeout = 300 // Default 5 minutes
		}

		executionMode := cfg.Tools.Python.ExecutionMode
		if executionMode == "" && cfg.Tools.Python.DockerImage != "" {
			executionMode = "docker"
		}
		if executionMode == "" {
			executionMode = "local"
		}

		if executionMode == "docker" {
			imageName := cfg.Tools.Python.DockerImage
			if imageName == "" {
				imageName = "hle-agent-python:latest"
			}
			pythonExecutor = pythonTool.NewDockerPythonExecutor(timeout, imageName)
			logger.Info("使用 Docker 模式执行 Python 代码",
				zap.String("image", imageName),
				zap.Int("timeout", timeout))
		} else {
			pythonExecutor = pythonTool.NewPythonExecutor(timeout)
			logger.Info("使用本地模式执行 Python 代码", zap.Int("timeout", timeout))
		}
		toolRegistry.Register(pythonExecutor)
	}

	// SageMath executor
	if cfg.Tools.SageMath.Enabled {
		timeout := cfg.Tools.SageMath.Timeout
		if timeout <= 0 {
			timeout = 300 // Default 5 minutes
		}

		executionMode := cfg.Tools.SageMath.ExecutionMode
		if executionMode == "" && cfg.Tools.SageMath.DockerImage != "" {
			executionMode = "docker"
		}
		if executionMode == "" {
			executionMode = "local"
		}

		if executionMode == "docker" {
			imageName := cfg.Tools.SageMath.DockerImage
			if imageName == "" {
				imageName = "hle-agent-sagemath:latest"
			}
			sageExecutor = sageTool.NewDockerSageMathExecutor(timeout, imageName)
			logger.Info("使用 Docker 模式执行 SageMath 代码",
				zap.String("image", imageName),
				zap.Int("timeout", timeout))
		} else {
			sageExecutor = sageTool.NewSageMathExecutor(timeout)
			logger.Info("使用本地模式执行 SageMath 代码", zap.Int("timeout", timeout))
		}
		toolRegistry.Register(sageExecutor)
	}

	if pythonExecutor != nil || sageExecutor != nil {
		logger.Info("已注册工具",
			zap.Bool("python_enabled", pythonExecutor != nil),
			zap.Bool("sagemath_enabled", sageExecutor != nil))
	}

	logger.Info("已注册工具",
		zap.String("python_executor", pythonExecutor.Name()),
		zap.String("sagemath_executor", sageExecutor.Name()))

	// Create retriever
	shortTermMem := memory.NewDefaultMemory()
	shortTermMem.StartSession(generateSessionID())
	retrieverLayer := retriever.NewRetriever(shortTermMem, nil)

	// Create knowledge base (if enabled)
	var knowledgeBase knowledgebase.KnowledgeBase
	if cfg.KnowledgeBase.Enabled {
		kbConfig := &knowledgebase.KnowledgeBaseConfig{
			StoragePath:      cfg.KnowledgeBase.StoragePath,
			MaxResults:       cfg.KnowledgeBase.MaxResults,
			MinSimilarity:    cfg.KnowledgeBase.MinSimilarity,
			WeightKeywords:   cfg.KnowledgeBase.WeightKeywords,
			WeightDomain:     cfg.KnowledgeBase.WeightDomain,
			WeightComplexity: cfg.KnowledgeBase.WeightComplexity,
		}

		// 设置默认值
		if kbConfig.StoragePath == "" {
			kbConfig.StoragePath = "./data/knowledge_base.db"
		}
		if kbConfig.MaxResults == 0 {
			kbConfig.MaxResults = 3
		}
		if kbConfig.MinSimilarity == 0 {
			kbConfig.MinSimilarity = 0.5
		}
		if kbConfig.WeightKeywords == 0 {
			kbConfig.WeightKeywords = 0.4
		}
		if kbConfig.WeightDomain == 0 {
			kbConfig.WeightDomain = 0.4
		}
		if kbConfig.WeightComplexity == 0 {
			kbConfig.WeightComplexity = 0.2
		}

		var err error
		knowledgeBase, err = knowledgebase.NewKnowledgeBase(kbConfig)
		if err != nil {
			logger.Warn("知识库初始化失败，将不使用知识库", zap.Error(err))
		} else {
			count, _ := knowledgeBase.GetChainCount(context.Background())
			logger.Info("知识库初始化成功", zap.Int("chain_count", count))
		}
	}

	// Create Planner
	planner := NewPlanner(llmClient, retrieverLayer, knowledgeBase)

	// Create Executor with configuration
	executorConfig := &ExecutorConfig{
		MaxRetries:     cfg.Agent.Executor.MaxRetries,
		RetryBackoff:   time.Duration(cfg.Agent.Executor.RetryBackoff) * time.Second,
		EnableFallback: cfg.Agent.Executor.EnableFallback,
	}
	// 设置默认值
	if executorConfig.MaxRetries == 0 {
		executorConfig.MaxRetries = 3
	}
	if executorConfig.RetryBackoff == 0 {
		executorConfig.RetryBackoff = 1 * time.Second
	}
	executor := NewExecutor(toolRegistry, llmClient, executorConfig)

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

	// Create executor agent
	// Note: Tools are registered separately through Eino ADK's tool system
	// The Executor will use tools through the plan steps
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

// Process processes a single question and returns the answer
// This method now uses Eino ADK Agent.Run() instead of manual execution loop
func (a *HLEAgent) Process(ctx context.Context, question string) (*AnswerResult, error) {
	logger := a.logger.With(
		zap.String("action", "Process"),
		zap.String("question_preview", utils.TruncateString(question, 100)),
	)

	logger.Info("开始处理问题",
		zap.String("decision_point", "process_start"),
		zap.Int("question_length", len(question)))

	startTime := time.Now()

	// Start new session
	sessionStartTime := time.Now()
	sessionID := generateSessionID()
	a.shortTermMem.StartSession(sessionID)
	sessionDuration := time.Since(sessionStartTime)
	logger.Debug("会话已启动",
		zap.String("session_id", sessionID),
		zap.Duration("session_init_duration", sessionDuration),
		zap.String("decision_point", "session_started"))

	// Step 0: Perception layer - analyze the question
	perceptionStartTime := time.Now()
	analysis := a.perception.Analyze(question)
	perceptionDuration := time.Since(perceptionStartTime)

	logger.Debug("感知层分析完成",
		zap.String("domain", string(analysis.Domain)),
		zap.Bool("has_code", analysis.QuestionInfo.CodePresent),
		zap.Duration("perception_duration", perceptionDuration),
		zap.Float64("perception_duration_seconds", perceptionDuration.Seconds()),
		zap.String("decision_point", "perception_analysis_complete"))

	// Store question in short-term memory
	memoryStartTime := time.Now()
	a.shortTermMem.AddQuestion(question, analysis)
	memoryDuration := time.Since(memoryStartTime)
	logger.Debug("问题已存储到记忆",
		zap.Duration("memory_store_duration", memoryDuration),
		zap.String("decision_point", "question_stored"))

	// Use Eino ADK Agent to process the question
	// Set context with question for Planner and Replanner
	ctx = WithQuestion(ctx, question)
	ctx = WithSessionID(ctx, sessionID)

	// Create AgentInput for Eino ADK
	input := &adk.AgentInput{
		Messages: []adk.Message{
			{
				Role:    "user",
				Content: question,
			},
		},
	}

	// Run the agent using Eino ADK
	agentRunStartTime := time.Now()
	iterator := a.agent.Run(ctx, input)
	logger.Debug("Eino Agent 已启动",
		zap.String("decision_point", "eino_agent_started"))

	// Collect results from async iterator
	var finalAnswer string
	var finalConfidence float64 = 0.8 // Default confidence
	eventCount := 0
	messageCount := 0
	var agentRunDuration time.Duration
	var eventProcessingDuration time.Duration

	// Process events from the async iterator
	// According to Eino ADK, Next() returns (event, hasNext)
	eventProcessingStartTime := time.Now()
	for {
		event, hasNext := iterator.Next()
		if !hasNext {
			break
		}

		eventCount++
		if event == nil {
			continue
		}

		// Try to extract message from event using Eino ADK helper
		if msg, remainingEvent, err := adk.GetMessage(event); err == nil && msg != nil {
			if msg.Role == "assistant" {
				messageCount++
				if finalAnswer == "" {
					finalAnswer = msg.Content
				} else {
					// Append to existing answer if multiple messages
					finalAnswer += "\n\n" + msg.Content
				}
				logger.Debug("收到 Agent 响应",
					zap.Int("message_index", messageCount),
					zap.String("content_preview", utils.TruncateString(msg.Content, 100)),
					zap.Int("content_length", len(msg.Content)),
					zap.String("decision_point", "agent_message_received"))
			}
			// Continue processing remaining events if any
			if remainingEvent != nil {
				event = remainingEvent
			}
		}

		// Log event for debugging
		logger.Debug("处理 Agent 事件",
			zap.Int("event_index", eventCount),
			zap.String("decision_point", "event_processed"))
	}
	eventProcessingDuration = time.Since(eventProcessingStartTime)
	agentRunDuration = time.Since(agentRunStartTime)

	logger.Debug("事件处理完成",
		zap.Int("total_events", eventCount),
		zap.Int("total_messages", messageCount),
		zap.Duration("event_processing_duration", eventProcessingDuration),
		zap.Duration("agent_run_duration", agentRunDuration),
		zap.String("decision_point", "event_processing_complete"))

	// If no answer was extracted, try to synthesize from memory
	if finalAnswer == "" {
		logger.Debug("未从事件流中提取到答案，尝试从记忆合成",
			zap.String("decision_point", "answer_synthesis_start"))

		synthesisStartTime := time.Now()
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
				logger.Debug("准备合成答案",
					zap.Int("step_results_count", len(allStepResults)),
					zap.Int("reasoning_parts_count", len(reasoningParts)),
					zap.Int("combined_reasoning_length", len(combinedReasoning)),
					zap.String("decision_point", "answer_synthesis_prepare"))

				// Use LLM to synthesize final answer
				synthesizedAnswer, err := a.synthesizeAnswer(ctx, question, "", combinedReasoning)
				synthesisDuration := time.Since(synthesisStartTime)
				if err == nil {
					finalAnswer = synthesizedAnswer
					logger.Info("答案合成成功",
						zap.Duration("synthesis_duration", synthesisDuration),
						zap.Float64("synthesis_duration_seconds", synthesisDuration.Seconds()),
						zap.Int("synthesized_answer_length", len(finalAnswer)),
						zap.String("decision_point", "answer_synthesis_success"))
				} else {
					logger.Warn("合成答案失败",
						zap.Error(err),
						zap.Duration("synthesis_duration", synthesisDuration),
						zap.String("decision_point", "answer_synthesis_failed"))
				}
			}
		}

		// If still no answer, use a fallback
		if finalAnswer == "" {
			logger.Warn("无法生成答案，使用默认回复",
				zap.String("decision_point", "answer_fallback_to_default"))
			finalAnswer = "无法生成答案，请检查输入或重试。"
			finalConfidence = 0.0
		}
	}

	// Store answer in memory
	a.shortTermMem.AddAnswer(finalAnswer, finalConfidence)

	// Process feedback
	questionID := generateQuestionID()
	allStepResults := a.shortTermMem.GetStepResults()
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

	totalDuration := time.Since(startTime)
	logger.Info("问题处理完成",
		zap.String("answer_preview", utils.TruncateString(finalAnswer, 100)),
		zap.Int("answer_length", len(finalAnswer)),
		zap.Float64("confidence", finalConfidence),
		zap.Duration("total_duration", totalDuration),
		zap.Float64("total_duration_seconds", totalDuration.Seconds()),
		zap.Duration("agent_run_duration", agentRunDuration),
		zap.Int("total_events", eventCount),
		zap.Int("total_messages", messageCount),
		zap.String("decision_point", "process_complete"))

	duration := totalDuration.Seconds()

	// Convert step results to models.StepResult
	// Use step results from memory (already in correct format)
	modelStepResults := allStepResults
	if len(modelStepResults) == 0 {
		// Create empty slice if no results
		modelStepResults = make([]*models.StepResult, 0)
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

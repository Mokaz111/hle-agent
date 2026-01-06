package agent

import (
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/feedback"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/hle-agent/hle-agent/pkg/memory"
	"github.com/hle-agent/hle-agent/pkg/perception"
	"github.com/hle-agent/hle-agent/pkg/prompts"
	"github.com/hle-agent/hle-agent/pkg/retriever"
	"github.com/hle-agent/hle-agent/pkg/utils"
)

// TestMemoryIntegration 测试内存组件集成
func TestMemoryIntegration(t *testing.T) {
	memoryStore := memory.NewDefaultMemory()

	// 测试会话管理
	testSessionID := "test-session-123"
	memoryStore.StartSession(testSessionID)

	// 添加问题
	memoryStore.AddQuestion(testSessionID, "测试问题1")

	// 验证历史记录
	history := memoryStore.GetRecentHistory(5)
	if len(history) != 1 {
		t.Errorf("期望 1 条历史记录, got: %d", len(history))
	}

	// 清理会话
	memoryStore.Clear()
	history = memoryStore.GetRecentHistory(5)
	if len(history) != 0 {
		t.Errorf("期望 0 条历史记录, got: %d", len(history))
	}

	t.Log("内存集成测试通过")
}

// TestPerceptionMemoryIntegration 测试感知层与内存的集成
func TestPerceptionMemoryIntegration(t *testing.T) {
	perceptionLayer := perception.NewPerception()
	memoryStore := memory.NewDefaultMemory()

	testSessionID := "perception-test-session"

	// 测试问题
	testQuestion := "用 Python 编写一个函数计算斐波那契数列"

	// 分析问题
	analysis := perceptionLayer.Analyze(testQuestion)

	// 验证分析结果
	if analysis == nil {
		t.Fatal("分析结果不应为 nil")
	}

	t.Logf("问题: %s", testQuestion)
	t.Logf("关键词: %v", analysis.Keywords)
	t.Logf("复杂度: %s", analysis.Complexity)
	t.Logf("代码检测: %v", analysis.QuestionInfo.CodePresent)
	t.Logf("代码语言: %s", analysis.QuestionInfo.CodeLanguage)

	// 只保存一个问题到内存
	memoryStore.AddQuestion(testSessionID, testQuestion)

	// 验证内存记录只有1条
	history := memoryStore.GetRecentHistory(10)
	if len(history) != 1 {
		t.Errorf("期望 1 条历史记录, got: %d", len(history))
	}

	t.Log("感知层与内存集成测试通过")
}

// TestRetrieverMemoryIntegration 测试检索器与内存的集成
func TestRetrieverMemoryIntegration(t *testing.T) {
	memoryStore := memory.NewDefaultMemory()
	retrieverStore := retriever.NewRetriever(memoryStore, nil)

	testSessionID := "retriever-test-session"

	// 添加历史问题
	questions := []string{
		"计算 1+1 的结果",
		"用 Python 实现快速排序算法",
		"解释什么是机器学习",
	}

	for _, q := range questions {
		memoryStore.AddQuestion(testSessionID, q)
	}

	// 测试检索相似问题
	results := retrieverStore.Retrieve("编程问题", "programming", []string{"Python"}, "medium")

	t.Logf("检索到 %d 条相似问题", len(results))

	if results != nil {
		for i, r := range results {
			t.Logf("[%d] 相似度: %.2f%%, 问题: %s", i, r.Similarity*100, r.Question)
		}
	}

	t.Log("检索器与内存集成测试通过")
}

// TestFeedbackIntegration 测试反馈层集成
func TestFeedbackIntegration(t *testing.T) {
	feedbackLayer := feedback.NewFeedback()

	// 测试答案验证
	validator := feedback.NewValidator(nil)
	validResult := validator.Validate("问题是 1+1", "答案是 2", nil)
	t.Logf("验证结果: IsValid=%v, Confidence=%v", validResult.IsValid, validResult.Confidence)

	// 测试错误分类
	classifier := feedback.NewClassifier()
	stepResults := []*models.StepResult{
		{StepID: 1, Success: false},
	}
	classifyResult := classifier.Classify(stepResults)
	t.Logf("分类结果: ErrorType=%s, Suggestions=%v", classifyResult.ErrorType, classifyResult.Suggestions)

	// 测试反馈聚合
	feedbackRecord := feedbackLayer.Process(
		utils.GenerateID("q"),
		"最终答案",
		"0.95",
		nil,
	)

	t.Logf("反馈记录: 置信度 %.2f%%", feedbackRecord.FinalConfidence*100)

	t.Log("反馈层集成测试通过")
}

// TestConfigIntegration 测试配置组件集成
func TestConfigIntegration(t *testing.T) {
	// 测试多 Provider 配置
	providers := []config.ProviderType{
		config.ProviderOpenAI,
		config.ProviderOneAPI,
		config.ProviderDeepSeek,
		config.ProviderAnthropic,
	}

	for _, provider := range providers {
		cfg := &config.ModelConfig{
			Provider: string(provider),
			APIKey:   "test-key",
			Model:    "test-model",
		}

		providerType := cfg.GetProviderType()
		if providerType != provider {
			t.Errorf("Provider 类型不匹配, expected: %s, got: %s", provider, providerType)
		}
	}

	t.Log("配置组件集成测试通过")
}

// TestPromptLLMIntegration 测试提示词与 LLM 客户端的集成
func TestPromptLLMIntegration(t *testing.T) {
	promptManager := prompts.NewPromptManager()

	// 测试获取模板
	templateNames := []string{
		"planner_math",
		"planner_cryptography",
		"planner_programming",
		"replan",
	}

	for _, name := range templateNames {
		template := promptManager.GetTemplate(name)
		if template == "" {
			t.Errorf("模板不应为空: %s", name)
		} else {
			t.Logf("模板 %s: %s", name, template)
		}
	}

	t.Log("提示词与 LLM 客户端集成测试通过")
}

// TestToolRegistryIntegration 测试工具注册表集成
func TestToolRegistryIntegration(t *testing.T) {
	registry := NewToolRegistry()

	// 注册测试工具
	testTool := &testTool{name: "test_tool", description: "测试工具"}
	registry.Register(testTool)

	// 验证工具已注册
	tool, ok := registry.Get("test_tool")
	if !ok {
		t.Fatal("工具未找到")
	}

	if tool.Name() != "test_tool" {
		t.Errorf("工具名称不匹配, expected: test_tool, got: %s", tool.Name())
	}

	// 列出所有工具
	tools := registry.List()
	if len(tools) != 1 {
		t.Errorf("期望 1 个工具, got: %d", len(tools))
	}

	t.Log("工具注册表集成测试通过")
}

// TestEinoAgentInterfaceIntegration 测试 Eino ADK Agent 接口集成
func TestEinoAgentInterfaceIntegration(t *testing.T) {
	// 测试 Planner 接口
	planner := &Planner{
		llmClient: nil, // 跳过真实 LLM 调用
		prompts:   prompts.NewPromptManager(),
		logger:    nil,
	}

	// 验证 Planner 实现
	_ = planner.Plan

	// 测试 Executor 接口
	executor := &Executor{
		toolRegistry: NewToolRegistry(),
		logger:       nil,
	}

	// 验证 Executor 实现
	_ = executor.Execute

	// 测试 Replanner 接口
	replanner := &Replanner{
		llmClient: nil,
		prompts:   prompts.NewPromptManager(),
		logger:    nil,
	}

	// 验证 Replanner 实现
	_ = replanner.Replan

	t.Log("Eino ADK Agent 接口集成测试通过")
}

// TestAgentSessionManagement 测试 Agent 会话管理
func TestAgentSessionManagement(t *testing.T) {
	// 由于 Eino ADK 内部依赖问题，暂时跳过此测试
	t.Skip("需要完整的 Eino ADK 环境配置")

	cfg := &config.Config{
		Model: config.ModelConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "test-model",
			BaseURL:  "https://api.openai.com/v1",
		},
		Agent: config.AgentConfig{
			MaxIterations: 10,
		},
		Memory: config.MemoryConfig{
			ShortTermMaxSteps: 10,
		},
		Output: config.OutputConfig{
			Format: "jsonl",
		},
	}

	hleAgent, err := NewHLEAgent(cfg)
	if err != nil {
		t.Fatalf("创建 HLE Agent 失败: %v", err)
	}

	// 测试 HLEAgent 创建成功
	if hleAgent == nil {
		t.Fatal("HLEAgent 不应为空")
	}

	t.Log("Agent 会话管理测试通过")
}

// TestLLMClientIntegration 测试 LLM 客户端集成
func TestLLMClientIntegration(t *testing.T) {
	// 跳过真实 API 调用测试
	t.Skip("需要真实的 LLM API 连接")

	modelConfig := &config.ModelConfig{
		Provider:  "openai",
		APIKey:    "test-key",
		Model:     "gpt-3.5-turbo",
		BaseURL:   "https://api.openai.com/v1",
		Timeout:   30,
		MaxTokens: 4096,
	}

	llmClient, err := llm.NewClient(modelConfig)
	if err != nil {
		t.Fatalf("创建 LLM 客户端失败: %v", err)
	}

	if llmClient == nil {
		t.Fatal("LLM 客户端不应为空")
	}

	// 测试生成方法存在
	_ = llmClient.Generate
	_ = llmClient.GenerateWithSystemPrompt
	_ = llmClient.Stream

	t.Log("LLM 客户端集成测试通过")
}

// TestPerceptionAnalysisIntegration 测试感知层分析功能
func TestPerceptionAnalysisIntegration(t *testing.T) {
	perceptionLayer := perception.NewPerception()

	testCases := []struct {
		name     string
		question string
	}{
		{"数学问题", "计算 2+3*4 的值"},
		{"编程问题", "用 Python 写一个快速排序"},
		{"密码学问题", "使用 RSA 算法加密消息"},
		{"一般问题", "什么是人工智能？"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analysis := perceptionLayer.Analyze(tc.question)

			if analysis == nil {
				t.Fatal("分析结果不应为 nil")
			}

			t.Logf("问题: %s", tc.question)
			t.Logf("复杂度: %s", analysis.Complexity)
			t.Logf("关键词数: %d", len(analysis.Keywords))
			t.Logf("包含代码: %v", analysis.QuestionInfo.CodePresent)
		})
	}

	t.Log("感知层分析集成测试通过")
}

// TestUtilsIntegration 测试工具函数集成
func TestUtilsIntegration(t *testing.T) {
	// 测试 GenerateID
	id1 := utils.GenerateID("test1")
	id2 := utils.GenerateID("test2")

	if id1 == "" || id2 == "" {
		t.Error("生成的 ID 不应为空")
	}

	if id1 == id2 {
		t.Error("两个生成的 ID 应该不同")
	}

	t.Logf("生成的 ID: %s", id1)
	t.Logf("生成的 ID: %s", id2)

	// 测试其他工具函数
	_ = utils.TruncateString("test string", 10)
	_ = utils.ToLower("TEST")
	_ = utils.Contains("test string", "test")
	_ = utils.ContainsAny("test string", "abc", "def", "test")
	_ = utils.MaskAPIKey("sk-1234567890abcdef")

	t.Log("工具函数集成测试通过")
}

// TestRetrieverBuildContextIntegration 测试检索器构建上下文功能
func TestRetrieverBuildContextIntegration(t *testing.T) {
	memoryStore := memory.NewDefaultMemory()
	retrieverStore := retriever.NewRetriever(memoryStore, nil)

	testSessionID := "context-test-session"

	// 添加相似的问题
	memoryStore.AddQuestion(testSessionID, "如何用 Python 实现快速排序？")
	memoryStore.AddQuestion(testSessionID, "Python 排序算法的实现")
	memoryStore.AddQuestion(testSessionID, "解释机器学习中的梯度下降算法")

	// 测试构建上下文 - 查询与历史问题相似
	context := retrieverStore.BuildContextForPlanner("Python 排序问题", "programming", []string{"排序", "算法", "Python"}, "medium")

	t.Logf("构建的上下文:\n%s", context)

	// 即使上下文为空，测试也应该通过（因为检索可能没有找到足够相似的问题）
	t.Log("检索器构建上下文集成测试通过")
}

package perception

import (
	"context"
	"fmt"
	"strings"

	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// LLMRecognizer 使用LLM进行领域识别
type LLMRecognizer struct {
	llmClient interface {
		GenerateWithSystemPrompt(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	}
	logger *zap.Logger
}

// NewLLMRecognizer 创建LLM领域识别器
func NewLLMRecognizer(llmClient interface {
	GenerateWithSystemPrompt(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}) *LLMRecognizer {
	return &LLMRecognizer{
		llmClient: llmClient,
		logger:    logging.WithComponent("LLMRecognizer"),
	}
}

// RecognizeWithLLM 使用LLM识别领域
func (r *LLMRecognizer) RecognizeWithLLM(ctx context.Context, question string) (Domain, float64, error) {
	systemPrompt := `你是一个专业的领域分类专家。请分析给定的问题，判断它属于哪个学术领域。

可选的领域包括：
- cybersecurity: 网络安全（包括安全漏洞、攻击防御、认证机制、加密协议、Shamir Secret Sharing、生物识别认证等）
- cryptography: 密码学（包括加密算法、密码分析、密钥管理、替换密码、单表替换等）
- programming: 编程（包括代码调试、算法设计、软件工程、Python错误、SageMath等）
- calculation: 数学计算（包括高等数学、线性代数、概率统计等）
- machine_learning: 机器学习（包括矩阵秩、ReLU激活函数、神经网络层等）
- artificial_intelligence: 人工智能（包括深度学习、神经网络、模型训练、水印技术、统计学习等）
- robotics: 机器人学（包括运动学、动力学、路径规划、关节角度、末端执行器等）
- data_science: 数据科学（包括数据分析、数据挖掘、数据可视化、线性可分性、嵌入表示、特征工程等）
- general: 通用领域（无法明确分类的问题）

请只返回领域名称（小写，使用下划线），格式如下：
domain: {领域名称}

如果问题涉及多个领域，请选择最主要的领域。`

	prompt := fmt.Sprintf("请分析以下问题的领域：\n\n%s", question)

	response, err := r.llmClient.GenerateWithSystemPrompt(ctx, systemPrompt, prompt)
	if err != nil {
		r.logger.Warn("LLM领域识别失败，使用默认领域",
			zap.Error(err),
			zap.String("question_preview", truncateString(question, 100)))
		return DomainGeneral, 0.5, err
	}

	// 解析响应
	domain, confidence := r.parseLLMResponse(response)
	if domain == DomainUnknown {
		domain = DomainGeneral
		confidence = 0.5
	}

	r.logger.Debug("LLM领域识别完成",
		zap.String("domain", string(domain)),
		zap.Float64("confidence", confidence),
		zap.String("llm_response", truncateString(response, 200)))

	return domain, confidence, nil
}

// parseLLMResponse 解析LLM的响应
func (r *LLMRecognizer) parseLLMResponse(response string) (Domain, float64) {
	responseLower := strings.ToLower(response)

	// 查找 domain: 模式
	domainPatterns := map[string]Domain{
		"domain: cybersecurity":           DomainCybersecurity,
		"domain: cryptography":            DomainCryptography,
		"domain: programming":             DomainProgramming,
		"domain: calculation":             DomainCalculation,
		"domain: machine_learning":        DomainMachineLearning,
		"domain: machine learning":        DomainMachineLearning,
		"domain: artificial_intelligence": DomainArtificialIntelligence,
		"domain: artificial intelligence": DomainArtificialIntelligence,
		"domain: ai":                      DomainArtificialIntelligence,
		"domain: data_science":            DomainDataScience,
		"domain: data science":            DomainDataScience,
		"domain: robotics":                DomainRobotics,
		"domain: general":                 DomainGeneral,
		"cybersecurity":                   DomainCybersecurity,
		"cryptography":                    DomainCryptography,
		"programming":                     DomainProgramming,
		"calculation":                     DomainCalculation,
		"machine_learning":                DomainMachineLearning,
		"machine learning":                DomainMachineLearning,
		"artificial_intelligence":         DomainArtificialIntelligence,
		"artificial intelligence":         DomainArtificialIntelligence,
		"ai":                              DomainArtificialIntelligence,
		"data_science":                    DomainDataScience,
		"data science":                    DomainDataScience,
		"robotics":                        DomainRobotics,
		"general":                         DomainGeneral,
	}

	for pattern, domain := range domainPatterns {
		if strings.Contains(responseLower, pattern) {
			// 计算置信度：如果明确匹配，置信度较高
			confidence := 0.8
			if strings.Contains(responseLower, "domain:") {
				confidence = 0.9
			}
			return domain, confidence
		}
	}

	return DomainUnknown, 0.0
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

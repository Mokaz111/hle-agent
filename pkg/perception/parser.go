package perception

import (
	"strings"

	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// Domain represents the domain/category of a question
type Domain string

const (
	DomainCryptography        Domain = "cryptography"         // 密码学
	DomainCybersecurity      Domain = "cybersecurity"        // 网络安全
	DomainProgramming         Domain = "programming"         // 编程
	DomainCalculation         Domain = "calculation"         // 数学计算
	DomainRobotics            Domain = "robotics"            // 机器人学
	DomainMachineLearning     Domain = "machine_learning"     // 机器学习
	DomainArtificialIntelligence Domain = "artificial_intelligence" // 人工智能
	DomainDataScience         Domain = "data_science"         // 数据科学
	DomainGeneral             Domain = "general"              // 通用
	DomainUnknown             Domain = "unknown"              // 未知
)

// QuestionInfo represents parsed question information
type QuestionInfo struct {
	Domain           Domain                  `json:"domain"`
	SubDomain        string                  `json:"sub_domain,omitempty"`
	Language         string                  `json:"language,omitempty"`
	Keywords         []string                `json:"keywords"`
	CodePresent      bool                    `json:"code_present"`
	CodeLanguage     string                  `json:"code_language,omitempty"`
	Complexity       Complexity              `json:"complexity"`
	Metadata         map[string]interface{}  `json:"metadata,omitempty"`
}

// Complexity represents the complexity level of a question
type Complexity string

const (
	ComplexityLow    Complexity = "low"
	ComplexityMedium Complexity = "medium"
	ComplexityHigh   Complexity = "high"
	ComplexityUnknown Complexity = "unknown"
)

// Parser parses input data and extracts structured information
type Parser struct {
	logger *zap.Logger
}

// NewParser creates a new Parser
func NewParser() *Parser {
	return &Parser{
		logger: logging.WithComponent("Parser"),
	}
}

// ParseQuestion parses a question string and returns structured information
func (p *Parser) ParseQuestion(question string) *QuestionInfo {
	info := &QuestionInfo{
		Domain:     DomainUnknown,
		Complexity: ComplexityUnknown,
		Keywords:   make([]string, 0),
		Metadata:   make(map[string]interface{}),
	}

	// Extract keywords
	info.Keywords = p.extractKeywords(question)

	// Detect code
	info.CodePresent, info.CodeLanguage = p.detectCode(question)

	// Estimate complexity
	info.Complexity = p.estimateComplexity(question, info.Keywords)

	// Store original length for metadata
	info.Metadata["original_length"] = len(question)
	info.Metadata["word_count"] = len(strings.Fields(question))

	p.logger.Debug("问题解析完成",
		zap.String("domain", string(info.Domain)),
		zap.String("complexity", string(info.Complexity)),
		zap.Bool("has_code", info.CodePresent),
		zap.Strings("keywords", info.Keywords))

	return info
}

// extractKeywords extracts key terms from the question
func (p *Parser) extractKeywords(question string) []string {
	questionLower := strings.ToLower(question)
	keywords := make([]string, 0)

	// Keyword patterns for different domains
	patterns := map[Domain][]string{
		DomainCryptography: {
			"cipher", "encrypt", "decrypt", "密码", "密文", "substitution",
			"vigenere", "caesar", "aes", "rsa", "破解", "解密", "加密",
		},
		DomainProgramming: {
			"code", "function", "class", "bug", "error", "python", "java",
			"javascript", "c++", "程序", "代码", "算法", "数据结构",
			"debug", "compile", "runtime", "exception",
		},
		DomainCalculation: {
			"calculate", "compute", "sqrt", "integral", "derivative",
			"limit", "矩阵", "方程", "求导", "积分", "极限", "计算",
			"probability", "statistics", "概率", "统计",
		},
		DomainRobotics: {
			"robot", "kinematic", "dynamics", "运动学", "动力学",
			"trajectory", "路径", "关节", "manipulator", "末端执行器",
		},
		DomainMachineLearning: {
			"machine learning", "neural", "deep learning", "模型", "训练",
			"dataset", "gradient", "loss", "accuracy", "分类", "回归",
			"classification", "regression", "clustering",
		},
	}

	// Extract keywords based on patterns
	for _, domainKeywords := range patterns {
		for _, keyword := range domainKeywords {
			if strings.Contains(questionLower, keyword) {
				if !contains(keywords, keyword) {
					keywords = append(keywords, keyword)
				}
			}
		}
	}

	return keywords
}

// detectCode detects if the question contains code and returns the language
func (p *Parser) detectCode(question string) (bool, string) {
	questionLower := strings.ToLower(question)

	// Code indicators
	codeIndicators := map[string]string{
		"python":     "python",
		"py ":        "python",
		"def ":       "python",
		"class ":     "python",
		"java":       "java",
		"javascript": "javascript",
		"js ":        "javascript",
		"c++":        "c++",
		"cpp":        "c++",
		"c ":         "c",
		"func ":      "go",
		"func main":  "go",
		"sage":       "sagemath",
		"#":          "python", // Comment indicator
		"import ":    "python",
		"package ":   "go",
	}

	for indicator, language := range codeIndicators {
		if strings.Contains(questionLower, indicator) {
			return true, language
		}
	}

	return false, ""
}

// estimateComplexity estimates the complexity of a question
func (p *Parser) estimateComplexity(question string, keywords []string) Complexity {
	// Simple heuristics based on question length and keyword count
	wordCount := len(strings.Fields(question))

	// Multi-step indicators
	multiStepKeywords := []string{
		"steps", "步骤", "first", "then", "finally",
		"分析", "计算", "验证", "结果",
	}
	multiStepCount := 0
	for _, kw := range multiStepKeywords {
		if strings.Contains(strings.ToLower(question), kw) {
			multiStepCount++
		}
	}

	// Estimate complexity
	if wordCount > 100 || (len(keywords) > 5 && multiStepCount > 2) {
		return ComplexityHigh
	} else if wordCount > 50 || len(keywords) > 3 {
		return ComplexityMedium
	}
	return ComplexityLow
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

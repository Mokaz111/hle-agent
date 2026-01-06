package perception

import (
	"regexp"
	"strings"

	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// Recognizer recognizes the domain/category of a question
type Recognizer struct {
	logger       *zap.Logger
	domainRules  []DomainRule
	regexPatterns map[string]*regexp.Regexp
}

// DomainRule defines a rule for domain recognition
type DomainRule struct {
	Domain      Domain
	Patterns    []string
	Weight      float64
	Required    bool
}

// NewRecognizer creates a new Recognizer
func NewRecognizer() *Recognizer {
	r := &Recognizer{
		logger:       logging.WithComponent("Recognizer"),
		domainRules:  make([]DomainRule, 0),
		regexPatterns: make(map[string]*regexp.Regexp),
	}

	// Initialize domain recognition rules
	r.initDomainRules()
	r.initRegexPatterns()

	return r
}

// initDomainRules initializes the domain recognition rules
func (r *Recognizer) initDomainRules() {
	r.domainRules = []DomainRule{
		// Cryptography domain - highest priority for specific keywords
		{
			Domain: DomainCryptography,
			Patterns: []string{
				`cipher`, `encrypt`, `decrypt`, `密码`, `密文`,
				`substitution`, `vigenere`, `caesar`, `aes`, `rsa`,
				`破解`, `解密`, `加密`, `密钥`, `密匙`,
			},
			Weight:   2.0,
			Required: true,
		},
		// Robotics domain
		{
			Domain: DomainRobotics,
			Patterns: []string{
				`robot`, `kinematic`, `dynamics`, `运动学`, `动力学`,
				`trajectory`, `路径`, `关节`, `manipulator`,
				`末端执行器`, `逆运动学`, `正运动学`,
				`DH参数`, `雅可比`, ` Jacobian`,
			},
			Weight:   2.0,
			Required: true,
		},
		// Machine Learning domain
		{
			Domain: DomainMachineLearning,
			Patterns: []string{
				`machine learning`, `neural`, `deep learning`,
				`模型`, `训练`, `dataset`, `gradient`,
				`loss`, `accuracy`, `分类`, `回归`,
				`classification`, `regression`, `clustering`,
				`神经网络`, `深度学习`, `激活函数`,
			},
			Weight:   2.0,
			Required: true,
		},
		// Programming domain
		{
			Domain: DomainProgramming,
			Patterns: []string{
				`code`, `function`, `class`, `bug`, `error`,
				`程序`, `代码`, `算法`, `数据结构`,
				`debug`, `compile`, `runtime`, `exception`,
				`error`, `stack`, `heap`, `pointer`,
			},
			Weight:   1.5,
			Required: false,
		},
		// Calculation domain
		{
			Domain: DomainCalculation,
			Patterns: []string{
				`calculate`, `compute`, `sqrt`, `integral`,
				`derivative`, `limit`, `矩阵`, `方程`,
				`求导`, `积分`, `极限`, `计算`,
				`probability`, `statistics`, `概率`, `统计`,
				`微分`, `偏导`, `泰勒`, `傅里叶`,
			},
			Weight:   1.5,
			Required: false,
		},
	}
}

// initRegexPatterns initializes regex patterns for complex recognition
func (r *Recognizer) initRegexPatterns() {
	patterns := map[string]string{
		// Code block patterns
		"python_code":   `(?i)(def |class |import |from\s+\w+\s+import)`,
		"math_expression": `(?i)(\d+\s*[\+\-\*/\^]\s*\d+|sqrt\(|integral\(|sum\()`,
		"matrix":        `(?i)(\[\s*\[\s*\d+.*\d+\s*\]\s*\]|矩阵)`,
		"equation":      `(?i)(=\s*\w+\s*\(|方程|equation)`,
		"cipher_text":   `(?i)([A-Z]{2,}\s*){5,}|密文|ciphertext`,
	}

	for name, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			r.regexPatterns[name] = re
		}
	}
}

// Recognize recognizes the domain of a question
func (r *Recognizer) Recognize(question string) *DomainRecognition {
	recognition := &DomainRecognition{
		QuestionInfo: &QuestionInfo{
			Domain:     DomainUnknown,
			Keywords:   make([]string, 0),
			Metadata:   make(map[string]interface{}),
		},
		Scores: make(map[Domain]float64),
	}

	questionLower := strings.ToLower(question)

	// Calculate scores for each domain
	for _, rule := range r.domainRules {
		score := r.calculateScore(questionLower, rule)
		if score > 0 {
			recognition.Scores[rule.Domain] = score
			recognition.QuestionInfo.Keywords = append(
				recognition.QuestionInfo.Keywords,
				r.extractMatchedKeywords(questionLower, rule.Patterns)...,
			)
		}
	}

	// Apply regex-based recognition
	r.applyRegexRecognition(question, recognition)

	// Determine primary domain
	recognition.PrimaryDomain = r.determinePrimaryDomain(recognition.Scores)

	// Set domain
	if recognition.PrimaryDomain != DomainUnknown {
		recognition.QuestionInfo.Domain = recognition.PrimaryDomain
		recognition.Confidence = recognition.Scores[recognition.PrimaryDomain]
	}

	// Set sub-domain if applicable
	recognition.QuestionInfo.SubDomain = r.determineSubDomain(question, recognition.PrimaryDomain)

	r.logger.Debug("领域识别完成",
		zap.String("primary_domain", string(recognition.PrimaryDomain)),
		zap.Float64("confidence", recognition.Confidence),
		zap.Any("scores", recognition.Scores))

	return recognition
}

// DomainRecognition represents the result of domain recognition
type DomainRecognition struct {
	*QuestionInfo
	PrimaryDomain Domain           `json:"primary_domain"`
	Scores        map[Domain]float64 `json:"scores"`
	Confidence    float64          `json:"confidence"`
}

// calculateScore calculates the score for a domain based on matched patterns
func (r *Recognizer) calculateScore(question string, rule DomainRule) float64 {
	score := 0.0
	matchCount := 0

	for _, pattern := range rule.Patterns {
		if strings.Contains(question, strings.ToLower(pattern)) {
			matchCount++
			if rule.Required {
				score += rule.Weight * 2
			} else {
				score += rule.Weight
			}
		}
	}

	// Normalize score based on number of patterns
	if len(rule.Patterns) > 0 {
		score = score / float64(len(rule.Patterns)) * float64(matchCount+1)
	}

	return score
}

// extractMatchedKeywords extracts matched keywords from the question
func (r *Recognizer) extractMatchedKeywords(question string, patterns []string) []string {
	matched := make([]string, 0)
	for _, pattern := range patterns {
		if strings.Contains(question, strings.ToLower(pattern)) {
			matched = append(matched, pattern)
		}
	}
	return matched
}

// applyRegexRecognition applies regex-based recognition
func (r *Recognizer) applyRegexRecognition(question string, recognition *DomainRecognition) {
	// Check for code patterns
	for name, re := range r.regexPatterns {
		if re.MatchString(question) {
			switch name {
			case "python_code":
				recognition.QuestionInfo.CodePresent = true
				recognition.QuestionInfo.CodeLanguage = "python"
				recognition.Scores[DomainProgramming] += 0.5
			case "math_expression":
				recognition.Scores[DomainCalculation] += 0.3
			case "matrix":
				recognition.Scores[DomainCalculation] += 0.4
			case "cipher_text":
				recognition.Scores[DomainCryptography] += 0.5
			}
		}
	}
}

// determinePrimaryDomain determines the primary domain based on scores
func (r *Recognizer) determinePrimaryDomain(scores map[Domain]float64) Domain {
	var primary Domain
	var maxScore float64 = 0

	for domain, score := range scores {
		if score > maxScore {
			maxScore = score
			primary = domain
		}
	}

	// Threshold for confidence
	if maxScore < 0.5 {
		return DomainGeneral
	}

	return primary
}

// determineSubDomain determines the sub-domain if applicable
func (r *Recognizer) determineSubDomain(question string, domain Domain) string {
	questionLower := strings.ToLower(question)

	switch domain {
	case DomainCryptography:
		if strings.Contains(questionLower, "caesar") {
			return "caesar_cipher"
		} else if strings.Contains(questionLower, "vigenere") {
			return "vigenere_cipher"
		} else if strings.Contains(questionLower, "substitution") {
			return "substitution_cipher"
		} else if strings.Contains(questionLower, "rsa") {
			return "rsa"
		}
	case DomainCalculation:
		if strings.Contains(questionLower, "integral") || strings.Contains(questionLower, "积分") {
			return "integration"
		} else if strings.Contains(questionLower, "derivative") || strings.Contains(questionLower, "求导") {
			return "differentiation"
		} else if strings.Contains(questionLower, "matrix") || strings.Contains(questionLower, "矩阵") {
			return "linear_algebra"
		}
	case DomainRobotics:
		if strings.Contains(questionLower, "inverse") || strings.Contains(questionLower, "逆") {
			return "inverse_kinematics"
		} else if strings.Contains(questionLower, "forward") || strings.Contains(questionLower, "正") {
			return "forward_kinematics"
		}
	case DomainMachineLearning:
		if strings.Contains(questionLower, "classification") || strings.Contains(questionLower, "分类") {
			return "classification"
		} else if strings.Contains(questionLower, "regression") || strings.Contains(questionLower, "回归") {
			return "regression"
		} else if strings.Contains(questionLower, "clustering") || strings.Contains(questionLower, "聚类") {
			return "clustering"
		}
	}

	return ""
}

// GetRecommendedTools returns recommended tools based on domain
func (r *Recognizer) GetRecommendedTools(domain Domain) []string {
	tools := map[Domain][]string{
		DomainCryptography:   {"python_executor", "sage_math_executor"},
		DomainProgramming:    {"python_executor"},
		DomainCalculation:    {"sage_math_executor", "python_executor"},
		DomainRobotics:       {"python_executor", "sage_math_executor"},
		DomainMachineLearning: {"python_executor"},
		DomainGeneral:        {"python_executor"},
	}

	if t, ok := tools[domain]; ok {
		return t
	}
	return []string{"python_executor"}
}

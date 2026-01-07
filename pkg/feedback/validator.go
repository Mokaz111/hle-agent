package feedback

import (
	"regexp"
	"strings"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// Confidence represents confidence level
type Confidence float64

const (
	ConfidenceVeryLow  Confidence = 0.2
	ConfidenceLow      Confidence = 0.4
	ConfidenceMedium   Confidence = 0.6
	ConfidenceHigh     Confidence = 0.8
	ConfidenceVeryHigh Confidence = 0.95
)

// Validator validates answers and provides confidence assessment
type Validator struct {
	logger *zap.Logger
	config *ValidatorConfig
}

// ValidatorConfig represents validator configuration
type ValidatorConfig struct {
	StrictMode        bool    `yaml:"strict_mode"`
	MinConfidence     float64 `yaml:"min_confidence"`
	MaxAnswerLength   int     `yaml:"max_answer_length"`
	EnableFormatCheck bool    `yaml:"enable_format_check"`
}

// NewValidator creates a new Validator
func NewValidator(cfg *ValidatorConfig) *Validator {
	if cfg == nil {
		cfg = &ValidatorConfig{
			StrictMode:        false,
			MinConfidence:     0.5,
			MaxAnswerLength:   10000,
			EnableFormatCheck: true,
		}
	}

	return &Validator{
		logger: logging.WithComponent("Validator"),
		config: cfg,
	}
}

// ValidateResult represents the validation result
type ValidateResult struct {
	IsValid       bool              `json:"is_valid"`
	Confidence    Confidence        `json:"confidence"`
	Issues        []ValidationIssue `json:"issues"`
	Score         float64           `json:"score"`
	Details       map[string]interface{} `json:"details"`
}

// ValidationIssue represents a validation issue
type ValidationIssue struct {
	Type    IssueType `json:"type"`
	Severity Severity  `json:"severity"`
	Message string    `json:"message"`
	Field   string    `json:"field,omitempty"`
}

// IssueType represents the type of validation issue
type IssueType string

const (
	IssueFormat      IssueType = "format"
	IssueLogic       IssueType = "logic"
	IssueCompleteness IssueType = "completeness"
	IssueConsistency IssueType = "consistency"
	IssueAccuracy    IssueType = "accuracy"
)

// Severity represents the severity of an issue
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Validate validates an answer and returns the validation result
// answer 参数应该是已经提取后的纯答案（不包含 Explanation 和 Confidence）
func (v *Validator) Validate(question string, answer string, stepResults []*models.StepResult) *ValidateResult {
	result := &ValidateResult{
		IsValid:    true,
		Confidence: ConfidenceMedium,
		Issues:     make([]ValidationIssue, 0),
		Details:    make(map[string]interface{}),
	}

	// Check answer length
	if len(answer) == 0 {
		result.IsValid = false
		result.Confidence = ConfidenceVeryLow
		result.Issues = append(result.Issues, ValidationIssue{
			Type:    IssueCompleteness,
			Severity: SeverityCritical,
			Message: "答案为空",
			Field:   "answer",
		})
	} else if len(answer) > v.config.MaxAnswerLength {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:    IssueFormat,
			Severity: SeverityWarning,
			Message: "答案过长，可能包含冗余信息",
			Field:   "answer",
		})
	}

	// 检查答案是否包含格式化标记（说明可能没有正确提取）
	if strings.Contains(answer, "Explanation:") || strings.Contains(answer, "Answer:") || strings.Contains(answer, "Confidence:") {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:    IssueFormat,
			Severity: SeverityWarning,
			Message: "答案可能包含格式化标记，建议检查答案提取逻辑",
			Field:   "answer",
		})
	}

	// Format validation
	if v.config.EnableFormatCheck {
		v.checkFormat(answer, result)
	}

	// Consistency check with step results
	v.checkConsistency(stepResults, result)

	// Calculate confidence based on step results
	v.assessConfidence(stepResults, answer, result)

	// Calculate overall score
	result.Score = v.calculateScore(result)

	v.logger.Debug("答案验证完成",
		zap.Bool("is_valid", result.IsValid),
		zap.Float64("confidence", float64(result.Confidence)),
		zap.Float64("score", result.Score),
		zap.Int("issues_count", len(result.Issues)))

	return result
}

// checkFormat checks the format of the answer
func (v *Validator) checkFormat(answer string, result *ValidateResult) {
	// Check for placeholders
	if strings.Contains(answer, "TODO") || strings.Contains(answer, "待") {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:    IssueCompleteness,
			Severity: SeverityError,
			Message: "答案包含未完成标记",
			Field:   "answer",
		})
	}

	// Check for reasonable answer patterns
	// For math answers, check if it contains numbers
	if v.isMathQuestion(answer) {
		if !v.containsNumbers(answer) {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:    IssueFormat,
				Severity: SeverityWarning,
				Message: "数学问题答案未包含数值",
				Field:   "answer",
			})
		}
	}

	// Check for proper punctuation
	if len(answer) > 10 && !strings.ContainsAny(answer, "。！？,.!?") {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:    IssueFormat,
			Severity: SeverityInfo,
			Message: "答案可能缺少标点符号",
			Field:   "answer",
		})
	}
}

// checkConsistency checks consistency with step results
func (v *Validator) checkConsistency(stepResults []*models.StepResult, result *ValidateResult) {
	if len(stepResults) == 0 {
		return
	}

	// Count successful steps
	successCount := 0
	totalConfidence := float64(0)
	for _, sr := range stepResults {
		if sr.Success {
			successCount++
			totalConfidence += sr.Confidence
		}
	}

	successRate := float64(successCount) / float64(len(stepResults))
	result.Details["step_success_rate"] = successRate
	result.Details["successful_steps"] = successCount
	result.Details["total_steps"] = len(stepResults)

	// If success rate is low, add a warning
	if successRate < 0.5 {
		result.Issues = append(result.Issues, ValidationIssue{
			Type:    IssueLogic,
			Severity: SeverityWarning,
			Message: "部分步骤执行失败，答案可能不可靠",
			Field:   "steps",
		})
	}

	// Check for failed key steps
	for _, sr := range stepResults {
		if !sr.Success && sr.Confidence < 0.3 {
			result.Issues = append(result.Issues, ValidationIssue{
				Type:    IssueLogic,
				Severity: SeverityWarning,
				Message: "存在置信度很低的步骤",
				Field:   "step",
			})
			break
		}
	}

	// Update confidence based on success rate
	if successRate >= 0.9 {
		result.Confidence = ConfidenceVeryHigh
	} else if successRate >= 0.7 {
		result.Confidence = ConfidenceHigh
	} else if successRate >= 0.5 {
		result.Confidence = ConfidenceMedium
	} else {
		result.Confidence = ConfidenceLow
	}
}

// assessConfidence performs detailed confidence assessment
func (v *Validator) assessConfidence(stepResults []*models.StepResult, answer string, result *ValidateResult) {
	// Base confidence from step results
	if len(stepResults) > 0 {
		var totalConf float64
		for _, sr := range stepResults {
			totalConf += sr.Confidence
		}
		avgStepConf := totalConf / float64(len(stepResults))

		// Adjust confidence based on step results
		if avgStepConf > 0.9 {
			result.Confidence = ConfidenceVeryHigh
		} else if avgStepConf > 0.7 {
			result.Confidence = ConfidenceHigh
		} else if avgStepConf > 0.5 {
			result.Confidence = ConfidenceMedium
		} else {
			result.Confidence = ConfidenceLow
		}
	}

	// Adjust based on answer characteristics
	if len(answer) > 0 {
		// Short but complete answers are often more confident
		if len(answer) < 50 && len(answer) > 5 {
			if result.Confidence < ConfidenceHigh {
				result.Confidence += 0.1
			}
		}

		// Very long answers might indicate uncertainty
		if len(answer) > 500 {
			result.Confidence -= 0.05
		}
	}

	// Cap confidence
	if result.Confidence > 1.0 {
		result.Confidence = 1.0
	}
	if result.Confidence < 0 {
		result.Confidence = 0
	}

	result.Details["final_confidence"] = float64(result.Confidence)
}

// calculateScore calculates the overall score
func (v *Validator) calculateScore(result *ValidateResult) float64 {
	score := float64(result.Confidence)

	// Deduct for errors
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityCritical:
			score -= 0.3
		case SeverityError:
			score -= 0.2
		case SeverityWarning:
			score -= 0.1
		case SeverityInfo:
			score -= 0.05
		}
	}

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// Helper functions

func (v *Validator) isMathQuestion(answer string) bool {
	mathIndicators := []string{"计算", "求", "解", "证明", "面积", "体积", "长度", "角度", "根", "积分"}
	for _, indicator := range mathIndicators {
		if strings.Contains(answer, indicator) {
			return true
		}
	}
	return false
}

func (v *Validator) containsNumbers(answer string) bool {
	numbers := regexp.MustCompile(`\d`)
	return numbers.MatchString(answer)
}

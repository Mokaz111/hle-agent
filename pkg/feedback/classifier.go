package feedback

import (
	"strings"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// Classifier classifies errors and provides suggestions
type Classifier struct {
	logger *zap.Logger
	rules  []ClassificationRule
}

// ClassificationRule defines a rule for error classification
type ClassificationRule struct {
	Pattern     string
	ErrorType   ErrorType
	Severity    Severity
	Category    string
	Suggestion  string
}

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeSyntax       ErrorType = "syntax"
	ErrorTypeLogic        ErrorType = "logic"
	ErrorTypeDomain       ErrorType = "domain"
	ErrorTypeTimeout      ErrorType = "timeout"
	ErrorTypeResource     ErrorType = "resource"
	ErrorTypeAPI          ErrorType = "api"
	ErrorTypeUnknown      ErrorType = "unknown"
)

// ClassificationResult represents the classification result
type ClassificationResult struct {
	ErrorType     ErrorType           `json:"error_type"`
	Category      string              `json:"category"`
	Severity      Severity            `json:"severity"`
	Suggestions   []string            `json:"suggestions"`
	RootCause     string              `json:"root_cause"`
	RelatedSteps  []int               `json:"related_steps"`
	RecoveryPlan  string              `json:"recovery_plan"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewClassifier creates a new Classifier
func NewClassifier() *Classifier {
	c := &Classifier{
		logger: logging.WithComponent("Classifier"),
		rules:  make([]ClassificationRule, 0),
	}

	c.initRules()
	return c
}

// initRules initializes classification rules
func (c *Classifier) initRules() {
	c.rules = []ClassificationRule{
		// Syntax errors
		{
			Pattern:    `syntax|语法|syntax error`,
			ErrorType:  ErrorTypeSyntax,
			Severity:   SeverityError,
			Category:   "语法错误",
			Suggestion: "检查代码语法，确保括号、引号等配对正确",
		},
		{
			Pattern:    `indentation|缩进`,
			ErrorType:  ErrorTypeSyntax,
			Severity:   SeverityError,
			Category:   "缩进错误",
			Suggestion: "检查代码缩进，确保使用一致的缩进风格",
		},

		// Logic errors
		{
			Pattern:    `division by zero|除以零`,
			ErrorType:  ErrorTypeLogic,
			Severity:   SeverityError,
			Category:   "逻辑错误",
			Suggestion: "添加除零检查，在除法操作前验证除数不为零",
		},
		{
			Pattern:    `index out of range|索引越界`,
			ErrorType:  ErrorTypeLogic,
			Severity:   SeverityError,
			Category:   "索引错误",
			Suggestion: "检查数组/列表边界，确保索引在有效范围内",
		},
		{
			Pattern:    `null|nil|None`,
			ErrorType:  ErrorTypeLogic,
			Severity:   SeverityError,
			Category:   "空值错误",
			Suggestion: "添加空值检查，在使用变量前验证其不为空",
		},

		// Domain errors
		{
			Pattern:    `math domain|数学定义域`,
			ErrorType:  ErrorTypeDomain,
			Severity:   SeverityWarning,
			Category:   "数学定义域错误",
			Suggestion: "检查数学表达式的定义域，如对数参数大于0",
		},
		{
			Pattern:    `singular matrix|奇异矩阵`,
			ErrorType:  ErrorTypeDomain,
			Severity:   SeverityWarning,
			Category:   "矩阵错误",
			Suggestion: "检查矩阵是否可逆，考虑使用伪逆或正则化",
		},

		// Timeout errors
		{
			Pattern:    `timeout|超时`,
			ErrorType:  ErrorTypeTimeout,
			Severity:   SeverityWarning,
			Category:   "超时错误",
			Suggestion: "优化算法复杂度，或增加超时时间限制",
		},

		// API errors
		{
			Pattern:    `rate limit|速率限制`,
			ErrorType:  ErrorTypeAPI,
			Severity:   SeverityWarning,
			Category:   "API速率限制",
			Suggestion: "添加请求间隔，避免触发速率限制",
		},
		{
			Pattern:    `authentication|认证`,
			ErrorType:  ErrorTypeAPI,
			Severity:   SeverityError,
			Category:   "认证错误",
			Suggestion: "检查API密钥和认证信息",
		},
	}
}

// Classify classifies an error and returns the classification result
func (c *Classifier) Classify(stepResults []*models.StepResult) *ClassificationResult {
	result := &ClassificationResult{
		ErrorType:     ErrorTypeUnknown,
		Category:      "未知错误",
		Severity:      SeverityInfo,
		Suggestions:   make([]string, 0),
		RootCause:     "无法确定错误原因",
		RelatedSteps:  make([]int, 0),
		RecoveryPlan:  "建议重试或手动检查",
		Metadata:      make(map[string]interface{}),
	}

	// Analyze failed steps
	var failedSteps []*models.StepResult
	for _, sr := range stepResults {
		if !sr.Success {
			failedSteps = append(failedSteps, sr)
			result.RelatedSteps = append(result.RelatedSteps, sr.StepID)
		}
	}

	if len(failedSteps) == 0 {
		result.RootCause = "没有失败的步骤"
		result.RecoveryPlan = "答案验证通过，无需恢复"
		return result
	}

	// Analyze the first failed step in detail
	if len(failedSteps) > 0 {
		firstFail := failedSteps[0]
		result.RootCause = c.analyzeError(firstFail.Error)

		// Apply classification rules
		for _, rule := range c.rules {
			if strings.Contains(strings.ToLower(firstFail.Error), strings.ToLower(rule.Pattern)) {
				result.ErrorType = rule.ErrorType
				result.Category = rule.Category
				result.Severity = rule.Severity
				result.Suggestions = append(result.Suggestions, rule.Suggestion)
				break
			}
		}

		// Add default suggestions based on error type
		c.addDefaultSuggestions(result, firstFail)
	}

	// Generate recovery plan
	result.RecoveryPlan = c.generateRecoveryPlan(result)

	// Count errors by type
	errorCounts := c.countErrors(failedSteps)
	result.Metadata["error_counts"] = errorCounts

	c.logger.Debug("错误分类完成",
		zap.String("error_type", string(result.ErrorType)),
		zap.String("category", result.Category),
		zap.Strings("suggestions", result.Suggestions))

	return result
}

// analyzeError analyzes the error message
func (c *Classifier) analyzeError(errorMsg string) string {
	if errorMsg == "" {
		return "未知错误，没有错误信息"
	}

	// Simple error analysis
	if strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "network") {
		return "网络连接问题"
	}
	if strings.Contains(errorMsg, "memory") || strings.Contains(errorMsg, "out of memory") {
		return "内存不足"
	}
	if strings.Contains(errorMsg, "permission") || strings.Contains(errorMsg, "access") {
		return "权限不足"
	}

	return errorMsg
}

// addDefaultSuggestions adds default suggestions based on error type
func (c *Classifier) addDefaultSuggestions(result *ClassificationResult, stepResult *models.StepResult) {
	// Add suggestion based on confidence
	if stepResult.Confidence < 0.5 {
		result.Suggestions = append(result.Suggestions, "步骤置信度较低，建议重新分析问题")
	}

	// Add suggestion based on step ID (approximate step type)
	if stepResult.StepID <= 2 {
		result.Suggestions = append(result.Suggestions, "问题理解阶段可能有误，建议重新阅读题目")
	}
	if stepResult.StepID >= 3 && stepResult.StepID <= 4 {
		result.Suggestions = append(result.Suggestions, "计算或执行阶段可能有误，建议检查计算步骤")
	}
}

// generateRecoveryPlan generates a recovery plan
func (c *Classifier) generateRecoveryPlan(result *ClassificationResult) string {
	switch result.ErrorType {
	case ErrorTypeSyntax:
		return "1. 检查代码语法\n2. 使用代码检查工具\n3. 参考示例代码"
	case ErrorTypeLogic:
		return "1. 添加边界检查\n2. 验证输入参数\n3. 添加调试输出"
	case ErrorTypeDomain:
		return "1. 检查问题定义\n2. 验证数学前提\n3. 考虑特殊情况"
	case ErrorTypeTimeout:
		return "1. 优化算法复杂度\n2. 减少计算量\n3. 增加超时时间"
	case ErrorTypeAPI:
		return "1. 检查API配置\n2. 添加重试机制\n3. 联系API提供商"
	default:
		return "1. 重新执行步骤\n2. 检查输入数据\n3. 查看详细日志"
	}
}

// countErrors counts errors by type
func (c *Classifier) countErrors(failedSteps []*models.StepResult) map[ErrorType]int {
	counts := make(map[ErrorType]int)

	for _, fs := range failedSteps {
		errorType := c.identifyErrorType(fs.Error)
		counts[errorType]++
	}

	return counts
}

// identifyErrorType identifies the error type from error message
func (c *Classifier) identifyErrorType(errorMsg string) ErrorType {
	for _, rule := range c.rules {
		if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(rule.Pattern)) {
			return rule.ErrorType
		}
	}
	return ErrorTypeUnknown
}

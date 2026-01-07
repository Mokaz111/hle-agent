package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// FormattedAnswer 表示格式化后的LLM响应
type FormattedAnswer struct {
	Explanation string  // 解释
	Answer      string  // 最终答案
	Confidence  float64 // 置信度 (0-1)
	RawResponse string  // 原始响应
}

// ExtractFormattedAnswer 从LLM响应中提取格式化的答案
// 支持格式：
// Explanation: {解释}
// Answer: {答案}
// Confidence: {置信度，0%-100%}
func ExtractFormattedAnswer(response string) *FormattedAnswer {
	result := &FormattedAnswer{
		RawResponse: response,
		Confidence:  -1, // -1 表示未提取到置信度
	}

	if response == "" {
		return result
	}

	// 移除 <think> 标签
	cleanedResponse := removeThinkTags(response)

	// 提取 Explanation
	explanationPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:^|\n)\s*Explanation\s*:\s*(.+?)(?:\n\s*(?:Answer|Confidence)|$)`),
		regexp.MustCompile(`(?i)(?:^|\n)\s*解释\s*[：:]\s*(.+?)(?:\n\s*(?:答案|置信度)|$)`),
	}
	for _, pattern := range explanationPatterns {
		matches := pattern.FindStringSubmatch(cleanedResponse)
		if len(matches) > 1 {
			result.Explanation = strings.TrimSpace(matches[1])
			break
		}
	}

	// 提取 Answer
	// 使用多行匹配，直到遇到 Confidence 或 Explanation 或文件结尾
	answerPatterns := []*regexp.Regexp{
		// Answer: ... (直到遇到 Confidence 或 Explanation 或文件结尾，支持多行)
		regexp.MustCompile(`(?is)(?:^|\n)\s*Answer\s*:\s*(.+?)(?:\n\s*(?:Confidence|Explanation)\s*:|$)`),
		// 答案：... (直到遇到置信度或解释或文件结尾，支持多行)
		regexp.MustCompile(`(?is)(?:^|\n)\s*答案\s*[：:]\s*(.+?)(?:\n\s*(?:置信度|解释)\s*[：:]|$)`),
		// 最终答案：... (直到换行或文件结尾，支持多行)
		regexp.MustCompile(`(?is)(?:^|\n)\s*最终答案\s*[：:]\s*(.+?)(?:\n\s*(?:置信度|解释|Confidence|Explanation)\s*[：:]|$)`),
		// The answer is ... (直到换行或文件结尾)
		regexp.MustCompile(`(?is)(?:^|\n)\s*The\s+answer\s+is\s+(.+?)(?:\n\s*(?:Confidence|Explanation)\s*:|$)`),
		// Final answer: ... (直到换行或文件结尾)
		regexp.MustCompile(`(?is)(?:^|\n)\s*Final\s+answer\s*:\s*(.+?)(?:\n\s*(?:Confidence|Explanation)\s*:|$)`),
	}
	for _, pattern := range answerPatterns {
		matches := pattern.FindStringSubmatch(cleanedResponse)
		if len(matches) > 1 {
			result.Answer = strings.TrimSpace(matches[1])
			// 清理答案（移除可能的前缀和多余的空白）
			result.Answer = cleanAnswer(result.Answer)
			// 如果答案不为空，使用它
			if result.Answer != "" {
				break
			}
		}
	}

	// 提取 Confidence
	confidencePatterns := []*regexp.Regexp{
		// Confidence: 85% 或 Confidence: 85
		regexp.MustCompile(`(?i)(?:^|\n)\s*Confidence\s*:\s*(\d+(?:\.\d+)?)\s*%?`),
		// 置信度：85% 或 置信度：85
		regexp.MustCompile(`(?i)(?:^|\n)\s*置信度\s*[：:]\s*(\d+(?:\.\d+)?)\s*%?`),
	}
	for _, pattern := range confidencePatterns {
		matches := pattern.FindStringSubmatch(cleanedResponse)
		if len(matches) > 1 {
			if conf, err := strconv.ParseFloat(matches[1], 64); err == nil {
				// 如果是百分比格式（0-100），转换为0-1
				if conf > 1.0 {
					result.Confidence = conf / 100.0
				} else {
					result.Confidence = conf
				}
				// 确保在0-1范围内
				if result.Confidence > 1.0 {
					result.Confidence = 1.0
				}
				if result.Confidence < 0 {
					result.Confidence = 0
				}
				break
			}
		}
	}

	// 如果没有提取到答案，尝试使用原来的ExtractFinalAnswer
	if result.Answer == "" {
		result.Answer = ExtractFinalAnswer(response)
	}

	// 如果提取到的答案太短（可能是错误提取），尝试从原始响应中重新提取
	if result.Answer != "" && len(result.Answer) < 10 {
		// 尝试使用更宽松的提取策略
		fallbackAnswer := ExtractFinalAnswer(cleanedResponse)
		if len(fallbackAnswer) > len(result.Answer) {
			result.Answer = fallbackAnswer
		}
	}

	return result
}

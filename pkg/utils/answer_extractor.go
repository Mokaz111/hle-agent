package utils

import (
	"regexp"
	"strings"
)

// ExtractFinalAnswer 从 LLM 响应中提取最终答案
// 支持多种格式：
// 1. Answer: xxx
// 2. 最终答案：xxx
// 3. The answer is xxx
// 4. 移除 <think> 标签
func ExtractFinalAnswer(response string) string {
	if response == "" {
		return ""
	}

	// 移除 <think> 标签及其内容
	response = removeThinkTags(response)

	// 尝试多种模式提取答案
	patterns := []*regexp.Regexp{
		// Answer: xxx 或 Answer:xxx
		regexp.MustCompile(`(?i)(?:^|\n)\s*Answer\s*:\s*(.+?)(?:\n|$)`),
		// 最终答案：xxx
		regexp.MustCompile(`(?i)(?:^|\n)\s*最终答案\s*[：:]\s*(.+?)(?:\n|$)`),
		// The answer is xxx
		regexp.MustCompile(`(?i)(?:^|\n)\s*The\s+answer\s+is\s+(.+?)(?:\n|$)`),
		// 答案是 xxx
		regexp.MustCompile(`(?i)(?:^|\n)\s*答案是\s*(.+?)(?:\n|$)`),
		// Final answer: xxx
		regexp.MustCompile(`(?i)(?:^|\n)\s*Final\s+answer\s*:\s*(.+?)(?:\n|$)`),
		// 提取最后一个句子的答案（如果前面有明确的答案标记）
		regexp.MustCompile(`(?i)(?:^|\n)\s*(?:Therefore|Thus|So|Hence|结论|因此|所以)[，,.\s]*(.+?)(?:\.|$)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(response)
		if len(matches) > 1 {
			answer := strings.TrimSpace(matches[1])
			if answer != "" && len(answer) < 2000 { // 允许更长的答案（如完整句子），但避免提取到整个推理过程
				return cleanAnswer(answer)
			}
		}
	}

	// 如果没有找到明确的答案标记，尝试从文本中提取可能的答案
	// 策略1: 查找包含引号的句子（可能是最终答案）
	quotePattern := regexp.MustCompile(`["'` + "`" + `]([^"'` + "`" + `]{5,200})["'` + "`" + `]`)
	quoteMatches := quotePattern.FindAllStringSubmatch(response, -1)
	if len(quoteMatches) > 0 {
		// 取最后一个引号内容（可能是最终答案）
		lastQuote := quoteMatches[len(quoteMatches)-1][1]
		if len(lastQuote) > 5 && len(lastQuote) < 200 {
			return cleanAnswer(lastQuote)
		}
	}

	// 策略2: 查找最后一段较短的文本（可能是答案）
	lines := strings.Split(response, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.HasPrefix(strings.ToLower(line), "think") &&
			!strings.Contains(line, "<think") && !strings.Contains(line, "think>") &&
			len(line) > 5 && len(line) < 200 {
			// 检查是否像是一个答案（不是问题或推理过程）
			lowerLine := strings.ToLower(line)
			if !strings.HasSuffix(line, "?") &&
				!strings.HasPrefix(lowerLine, "step") &&
				!strings.HasPrefix(lowerLine, "plan:") &&
				!strings.HasPrefix(lowerLine, "therefore") &&
				!strings.HasPrefix(lowerLine, "thus") {
				return cleanAnswer(line)
			}
		}
	}

	// 策略3: 如果答案很长，尝试提取前200个字符（可能是答案的开头）
	cleanedResponse := cleanAnswer(response)
	if len(cleanedResponse) > 500 {
		// 尝试找到第一个句号后的内容，或者截取前200字符
		if idx := strings.Index(cleanedResponse, "."); idx > 50 && idx < 300 {
			// 如果第一个句号在合理位置，取到第一个句号
			cleanedResponse = cleanedResponse[:idx+1]
		} else {
			// 否则截取前200字符
			if len(cleanedResponse) > 200 {
				cleanedResponse = cleanedResponse[:200] + "..."
			}
		}
	}

	return cleanedResponse
}

// removeThinkTags 移除 <think> 标签及其内容
func removeThinkTags(text string) string {
	// 移除 <think>...</think> 标签（包括转义的版本 \u003cthink\u003e）
	// 使用非贪婪匹配，匹配到 </think> 为止
	thinkPattern := regexp.MustCompile(`(?i)(?:<think[^>]*>|&lt;think[^>]*&gt;|\\u003cthink[^>]*\\u003e).*?(?:</think>|&lt;/think&gt;|\\u003c/think\\u003e)`)
	text = thinkPattern.ReplaceAllString(text, "")

	// 移除单独的 <think> 或 </think> 标签（包括转义版本）
	text = regexp.MustCompile(`(?i)(?:</?think[^>]*>|&lt;/?think[^>]*&gt;|\\u003c/?think[^>]*\\u003e)`).ReplaceAllString(text, "")

	// 移除 <think> 标签（如果存在）
	text = regexp.MustCompile(`(?i)<think>.*?</think>`).ReplaceAllString(text, "")

	return text
}

// cleanAnswer 清理答案文本
func cleanAnswer(answer string) string {
	// 移除首尾空白
	answer = strings.TrimSpace(answer)

	// 移除常见的答案前缀
	prefixes := []string{
		"The answer is",
		"Answer:",
		"答案是",
		"最终答案：",
		"最终答案:",
		"Final answer:",
		"Therefore,",
		"Thus,",
		"So,",
		"Hence,",
		"结论：",
		"因此，",
		"所以，",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(answer), strings.ToLower(prefix)) {
			answer = strings.TrimSpace(answer[len(prefix):])
			// 移除可能的前导标点
			answer = strings.TrimLeft(answer, "，,：: ")
		}
	}

	// 移除多余的空白
	answer = regexp.MustCompile(`\s+`).ReplaceAllString(answer, " ")

	// 如果答案太长（超过1000字符），尝试截取前一部分（可能是答案）
	// 但对于正常长度的答案（如完整句子），不要截断
	if len(answer) > 1000 {
		// 尝试找到第一个句号或换行
		if idx := strings.Index(answer, "."); idx > 0 && idx < 500 {
			answer = answer[:idx+1]
		} else if idx := strings.Index(answer, "\n"); idx > 0 && idx < 500 {
			answer = answer[:idx]
		} else {
			// 如果都找不到，截取前 500 个字符（允许更长的答案）
			answer = answer[:500] + "..."
		}
	}

	return strings.TrimSpace(answer)
}


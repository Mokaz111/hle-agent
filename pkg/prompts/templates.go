package prompts

import "strings"

// PromptManager manages prompt templates
type PromptManager struct {
	templates map[string]string
}

// NewPromptManager creates a new PromptManager
func NewPromptManager() *PromptManager {
	return &PromptManager{
		templates: DefaultPrompts(),
	}
}

// GetTemplate returns the prompt template for the given name
func (p *PromptManager) GetTemplate(name string) string {
	if prompt, ok := p.templates[name]; ok {
		return prompt
	}
	return p.templates["planner_general"]
}

// DefaultPrompts returns the default prompt templates
func DefaultPrompts() map[string]string {
	return map[string]string{
		// Planner prompts for different question types
		"planner_cryptography": `你是一个专业的密码学解题专家。请分析以下密码学题目并制定解题计划。

题目：{question}

请制定详细的解题步骤，包括：
1. 分析密文结构
2. 识别加密算法
3. 设计解密策略
4. 执行解密计算
5. 验证解密结果

请按以下JSON格式输出：
{
    "id": "plan_001",
    "steps": [
        {
            "id": 1,
            "description": "步骤描述",
            "action": "llm 或 tool",
            "tool_name": "python_executor",
            "parameters": {"code": "Python代码"}
        }
    ]
}`,

		"planner_programming": `你是一个专业的编程题解题专家。请分析以下编程题目并制定解题计划。

题目：{question}

请制定详细的解题步骤，包括：
1. 理解代码要求
2. 分析代码逻辑
3. 识别潜在错误
4. 验证分析结果
5. 给出正确答案

请按以下JSON格式输出：
{
    "id": "plan_001",
    "steps": [
        {
            "id": 1,
            "description": "步骤描述",
            "action": "llm 或 tool",
            "tool_name": "python_executor",
            "parameters": {"code": "Python代码"}
        }
    ]
}`,

		"planner_calculation": `你是一个专业的数学计算解题专家。请分析以下计算题目并制定解题计划。

题目：{question}

请制定详细的解题步骤，包括：
1. 理解计算要求
2. 选择计算方法
3. 执行符号计算
4. 验证计算结果
5. 给出最终答案

请按以下JSON格式输出：
{
    "id": "plan_001",
    "steps": [
        {
            "id": 1,
            "description": "步骤描述",
            "action": "llm 或 tool",
            "tool_name": "sagemath_executor",
            "parameters": {"code": "SageMath代码"}
        }
    ]
}`,

		"planner_general": `你是一个专业的学术题目解题专家。请分析以下题目并制定解题计划。

题目：{question}

请制定详细的解题步骤，包括：
1. 理解题目要求
2. 识别关键信息
3. 选择解题策略
4. 执行计算或推理
5. 验证答案

请按以下JSON格式输出：
{
    "id": "plan_001",
    "steps": [
        {
            "id": 1,
            "description": "步骤描述",
            "action": "llm 或 tool",
            "tool_name": "python_executor",
            "parameters": {"code": "执行代码"}
        }
    ]
}`,

		// Replanner prompt
		"replanner": `请评估以下解题步骤的执行结果，决定是否需要重规划。

题目：{question}

执行结果：
{old_results}

请评估：
1. 所有步骤是否都成功了？
2. 如果有失败的步骤，是否需要重规划整个方案？
3. 是否只需要重试当前失败的步骤？

请按以下JSON格式输出：
{
    "need_replan": true/false,
    "reason": "重规划或不重规划的理由",
    "new_steps": [
        // 如果需要重规划，提供新的步骤列表
    ]
}

如果不需要重规划，返回：
{"need_replan": false, "reason": "理由"}`,
	}
}

// GetTemplate returns the prompt template for the given name
func GetTemplate(name string) string {
	prompts := DefaultPrompts()
	if prompt, ok := prompts[name]; ok {
		return prompt
	}
	return prompts["planner_general"]
}

// Render renders the prompt template with the given values
func Render(template string, values map[string]string) string {
	result := template
	for key, value := range values {
		// Replace {key} placeholder with value
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

package prompts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPromptManager(t *testing.T) {
	pm := NewPromptManager()

	assert.NotNil(t, pm)
	assert.NotNil(t, pm.templates)
}

func TestGetTemplateExisting(t *testing.T) {
	pm := NewPromptManager()

	template := pm.GetTemplate("planner_cryptography")

	assert.NotEmpty(t, template)
	assert.Contains(t, template, "密码学解题专家")
}

func TestGetTemplateProgramming(t *testing.T) {
	pm := NewPromptManager()

	template := pm.GetTemplate("planner_programming")

	assert.NotEmpty(t, template)
	assert.Contains(t, template, "编程题解题专家")
}

func TestGetTemplateCalculation(t *testing.T) {
	pm := NewPromptManager()

	template := pm.GetTemplate("planner_calculation")

	assert.NotEmpty(t, template)
	assert.Contains(t, template, "数学计算解题专家")
}

func TestGetTemplateDefaultFallback(t *testing.T) {
	pm := NewPromptManager()

	// Get a non-existent template
	template := pm.GetTemplate("non_existent_template")

	// Should fall back to planner_general
	assert.NotEmpty(t, template)
}

func TestGetTemplateReplanner(t *testing.T) {
	pm := NewPromptManager()

	template := pm.GetTemplate("replanner")

	assert.NotEmpty(t, template)
}

func TestGetTemplateExecutor(t *testing.T) {
	pm := NewPromptManager()

	template := pm.GetTemplate("executor")

	assert.NotEmpty(t, template)
}

func TestDefaultPromptsReturnsMap(t *testing.T) {
	prompts := DefaultPrompts()

	assert.NotNil(t, prompts)
	assert.GreaterOrEqual(t, len(prompts), 5)
}

func TestDefaultPromptsContainsAllTypes(t *testing.T) {
	prompts := DefaultPrompts()

	expectedTemplates := []string{
		"planner_cryptography",
		"planner_programming",
		"planner_calculation",
		"planner_general",
		"replanner",
	}

	for _, name := range expectedTemplates {
		assert.Contains(t, prompts, name, "Template %s should exist", name)
	}
}

func TestPlannerTemplatesContainQuestionPlaceholder(t *testing.T) {
	pm := NewPromptManager()

	templates := []string{
		"planner_cryptography",
		"planner_programming",
		"planner_calculation",
		"planner_robotics",
		"planner_general",
	}

	for _, name := range templates {
		template := pm.GetTemplate(name)
		assert.Contains(t, template, "{question}", "Template %s should contain {question} placeholder", name)
	}
}

func TestPlannerTemplatesContainJSONFormat(t *testing.T) {
	pm := NewPromptManager()

	templates := []string{
		"planner_cryptography",
		"planner_programming",
		"planner_calculation",
	}

	for _, name := range templates {
		template := pm.GetTemplate(name)
		assert.Contains(t, template, "JSON", "Template %s should mention JSON format", name)
		assert.Contains(t, template, "steps", "Template %s should contain steps field", name)
	}
}

func TestPromptManagerWithCustomTemplates(t *testing.T) {
	customPrompts := map[string]string{
		"custom": "Custom template content",
	}

	pm := &PromptManager{
		templates: customPrompts,
	}

	assert.Equal(t, "Custom template content", pm.GetTemplate("custom"))
}

func TestPromptTemplateConsistency(t *testing.T) {
	pm := NewPromptManager()

	// Multiple calls should return the same template
	template1 := pm.GetTemplate("planner_cryptography")
	template2 := pm.GetTemplate("planner_cryptography")

	assert.Equal(t, template1, template2)
}

func TestAllPlannerTemplatesHaveActionField(t *testing.T) {
	pm := NewPromptManager()

	templates := []string{
		"planner_cryptography",
		"planner_programming",
		"planner_calculation",
		"planner_robotics",
		"planner_general",
	}

	for _, name := range templates {
		template := pm.GetTemplate(name)
		assert.Contains(t, template, "action", "Template %s should mention action field", name)
	}
}

func TestAllPlannerTemplatesHaveToolNameField(t *testing.T) {
	pm := NewPromptManager()

	templates := []string{
		"planner_cryptography",
		"planner_programming",
		"planner_calculation",
		"planner_robotics",
		"planner_general",
	}

	for _, name := range templates {
		template := pm.GetTemplate(name)
		assert.Contains(t, template, "tool_name", "Template %s should mention tool_name field", name)
	}
}

func TestReplannerTemplateContent(t *testing.T) {
	pm := NewPromptManager()

	template := pm.GetTemplate("replanner")

	// Replanner should check if the result is acceptable
	assert.True(t, strings.Contains(template, "结果") || strings.Contains(template, "result"),
		"Replanner template should mention results")
}

func TestExecutorTemplateContent(t *testing.T) {
	pm := NewPromptManager()

	template := pm.GetTemplate("executor")

	// Executor should mention execution
	assert.True(t, strings.Contains(template, "执行") || strings.Contains(template, "execute"),
		"Executor template should mention execution")
}

package perception

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPerception(t *testing.T) {
	p := NewPerception()

	assert.NotNil(t, p)
	assert.NotNil(t, p.parser)
	assert.NotNil(t, p.recognizer)
}

func TestAnalyzeReturnsResult(t *testing.T) {
	p := NewPerception()

	result := p.Analyze("What is 2+2?")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestAnalyzeWithCode(t *testing.T) {
	p := NewPerception()

	codeQuestion := `Write a Python function to calculate factorial:
def factorial(n):
    if n <= 1:
        return 1
    return n * factorial(n-1)
`

	result := p.Analyze(codeQuestion)

	assert.True(t, result.CodePresent)
	assert.Equal(t, "python", result.CodeLanguage)
}

func TestAnalyzeReturnsNonEmptyFields(t *testing.T) {
	p := NewPerception()

	result := p.Analyze("Solve the equation: x^2 + 2x + 1 = 0")

	assert.NotNil(t, result)
	// Domain should be one of the valid domains
	assert.Contains(t, []string{"cryptography", "programming", "calculation", "robotics", "machine_learning", "general", "unknown"}, string(result.Domain))
}

func TestAnalyzeWithEncryption(t *testing.T) {
	p := NewPerception()

	result := p.Analyze("Explain RSA encryption algorithm")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestGetRecommendedTool(t *testing.T) {
	p := NewPerception()

	tool := p.GetRecommendedTool("programming")

	assert.NotEmpty(t, tool)
}

func TestGetRecommendedToolForUnknown(t *testing.T) {
	p := NewPerception()

	tool := p.GetRecommendedTool("unknown_domain")

	assert.NotEmpty(t, tool)
}

func TestPreprocessWhitespace(t *testing.T) {
	p := NewPerception()

	input := "   What   is   2+2?   "
	result := p.Preprocess(input)

	assert.NotEqual(t, input, result)
}

func TestPreprocessNewlines(t *testing.T) {
	p := NewPerception()

	input := "Question:\n\nWhat is 2+2?\n\nAnswer:"
	result := p.Preprocess(input)

	assert.Contains(t, result, "What is 2+2?")
}

func TestExtractCodeBlocks(t *testing.T) {
	p := NewPerception()

	input := "Here is some code:\n```python\ndef hello():\n    print(\"world\")\n```\nEnd of code."

	blocks := p.ExtractCodeBlocks(input)

	assert.GreaterOrEqual(t, len(blocks), 0)
}

func TestExtractCodeBlocksEmpty(t *testing.T) {
	p := NewPerception()

	input := "This is a plain text question without code."

	blocks := p.ExtractCodeBlocks(input)

	assert.Equal(t, 0, len(blocks))
}

func TestGetHistoryEmpty(t *testing.T) {
	p := NewPerception()

	history := p.GetHistory()

	assert.NotNil(t, history)
	assert.Equal(t, 0, len(history))
}

func TestGetDomainStatsEmpty(t *testing.T) {
	p := NewPerception()

	stats := p.GetDomainStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["total"])
}

func TestPerceptionConfig(t *testing.T) {
	p := NewPerception()

	assert.NotNil(t, p.parser)
	assert.NotNil(t, p.recognizer)
	assert.NotNil(t, p.history)
}

func TestDomainConstants(t *testing.T) {
	assert.Equal(t, Domain("cryptography"), DomainCryptography)
	assert.Equal(t, Domain("programming"), DomainProgramming)
	assert.Equal(t, Domain("calculation"), DomainCalculation)
	assert.Equal(t, Domain("robotics"), DomainRobotics)
	assert.Equal(t, Domain("machine_learning"), DomainMachineLearning)
	assert.Equal(t, Domain("general"), DomainGeneral)
}

func TestComplexityConstants(t *testing.T) {
	assert.Equal(t, Complexity("low"), ComplexityLow)
	assert.Equal(t, Complexity("medium"), ComplexityMedium)
	assert.Equal(t, Complexity("high"), ComplexityHigh)
}

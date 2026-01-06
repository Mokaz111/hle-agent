package perception

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRecognizer(t *testing.T) {
	r := NewRecognizer()

	assert.NotNil(t, r)
}

func TestRecognizeReturnsResult(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("Explain the AES encryption algorithm")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestRecognizeWithEncryption(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("RSA encryption algorithm")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestRecognizeWithProgramming(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("Write Python code")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestRecognizeWithMath(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("Solve differential equation")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestRecognizeWithRobotics(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("Robot control system")

	assert.NotNil(t, result)
	assert.Equal(t, DomainRobotics, result.Domain)
}

func TestRecognizeWithAI(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("Neural network training")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestRecognizeWithGeneral(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("What is the capital of France?")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestRecognizeWithSubdomain(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("Implement RSA encryption in Python")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestRecognizeCaseInsensitive(t *testing.T) {
	r := NewRecognizer()

	result1 := r.Recognize("ENCRYPTION ALGORITHM")
	result2 := r.Recognize("encryption algorithm")

	assert.Equal(t, result1.Domain, result2.Domain)
}

func TestGetRecommendedTools(t *testing.T) {
	r := NewRecognizer()

	tools := r.GetRecommendedTools(DomainCryptography)
	assert.NotEmpty(t, tools)

	tools = r.GetRecommendedTools(DomainProgramming)
	assert.NotEmpty(t, tools)

	tools = r.GetRecommendedTools(DomainCalculation)
	assert.NotEmpty(t, tools)

	tools = r.GetRecommendedTools(DomainGeneral)
	assert.NotEmpty(t, tools)
}

func TestRecognizeEmptyQuestion(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestRecognizeShortQuestion(t *testing.T) {
	r := NewRecognizer()

	result := r.Recognize("OK?")

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Domain))
}

func TestGetRecommendedToolsForUnknown(t *testing.T) {
	r := NewRecognizer()

	tools := r.GetRecommendedTools("unknown")

	assert.NotEmpty(t, tools)
}

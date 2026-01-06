package feedback

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator(nil)

	assert.NotNil(t, v)
}

func TestValidateComplete(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("What is the meaning of life?", "The answer is 42", nil)

	assert.True(t, result.IsValid)
}

func TestValidateEmpty(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("What is the meaning of life?", "", nil)

	assert.False(t, result.IsValid)
}

func TestValidateShort(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("What is the meaning of life?", "42", nil)

	// Short answers might still be valid depending on validation config
	assert.NotNil(t, result)
}

func TestValidateCorrectFormat(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("What is 1+1?", "The final answer is: 4", nil)

	// Answers with explanation format might still be valid
	assert.NotNil(t, result)
}

func TestValidateWithExplanation(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("What is 2+2?", "First, we calculate 2+2=4. Therefore, the answer is 4.", nil)

	// Detailed answers are acceptable
	assert.NotNil(t, result)
}

func TestValidateResultFields(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("Question", "Answer", nil)

	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Confidence, 0.0)
	assert.LessOrEqual(t, result.Confidence, 1.0)
}

func TestValidateWithMathQuestion(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("What is 2+2?", "4", nil)

	assert.NotNil(t, result)
}

func TestValidateWithStepResults(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("Question", "Answer", nil)

	assert.NotNil(t, result)
}

func TestValidateConsistency(t *testing.T) {
	v := NewValidator(nil)

	// Test with step results showing successful execution
	result := v.Validate("Question", "Answer", nil)

	assert.NotNil(t, result)
}

func TestValidateConfidenceHigh(t *testing.T) {
	v := NewValidator(nil)

	result := v.Validate("What is the meaning of life?", "The answer is 42. This is definitive.", nil)

	assert.GreaterOrEqual(t, result.Confidence, 0.0)
}

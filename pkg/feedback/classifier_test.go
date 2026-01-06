package feedback

import (
	"testing"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewClassifier(t *testing.T) {
	c := NewClassifier()

	assert.NotNil(t, c)
}

func TestClassifySuccessfulSteps(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: true, Confidence: 1.0},
		{StepID: 2, Success: true, Confidence: 1.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.ErrorType))
}

func TestClassifyWithFailedStep(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: true, Confidence: 1.0},
		{StepID: 2, Success: false, Error: "SyntaxError: invalid syntax", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.ErrorType))
}

func TestClassifyWithTimeout(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "context deadline exceeded", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
}

func TestClassifyWithAPIError(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "API rate limit exceeded", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
}

func TestClassifyWithDomainError(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "division by zero", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
}

func TestClassifyWithLogicError(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "incorrect algorithm result", Confidence: 0.5},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
}

func TestClassifyEmptyResults(t *testing.T) {
	c := NewClassifier()

	result := c.Classify([]*models.StepResult{})

	assert.NotNil(t, result)
}

func TestClassifyNilResults(t *testing.T) {
	c := NewClassifier()

	result := c.Classify(nil)

	assert.NotNil(t, result)
}

func TestClassifyResultHasSuggestions(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "SyntaxError: invalid syntax", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Suggestions)
}

func TestClassifyResultHasRecoveryPlan(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "connection refused", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
	assert.NotEmpty(t, result.RecoveryPlan)
}

func TestClassifyRelatedSteps(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "test error", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
	assert.Contains(t, result.RelatedSteps, 1)
}

func TestClassifyRootCause(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "test error", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
	assert.NotEmpty(t, result.RootCause)
}

func TestClassifySeverity(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "test error", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.Severity))
}

func TestClassifierResultFields(t *testing.T) {
	c := NewClassifier()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "test error", Confidence: 0.0},
	}

	result := c.Classify(stepResults)

	assert.NotNil(t, result)
	assert.NotEmpty(t, string(result.ErrorType))
	assert.NotEmpty(t, result.Category)
	assert.NotEmpty(t, string(result.Severity))
}

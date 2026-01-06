package feedback

import (
	"testing"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewFeedback(t *testing.T) {
	f := NewFeedback()

	assert.NotNil(t, f)
}

func TestProcessBasic(t *testing.T) {
	f := NewFeedback()

	result := f.Process("q1", "Question", "Answer", nil)

	assert.NotNil(t, result)
}

func TestProcessWithValidation(t *testing.T) {
	f := NewFeedback()

	result := f.Process("q1", "What is the meaning of life?", "The answer is 42", nil)

	assert.NotNil(t, result)
	assert.NotNil(t, result.ValidateResult)
}

func TestProcessWithClassification(t *testing.T) {
	f := NewFeedback()

	result := f.Process("q1", "Question", "Answer", nil)

	assert.NotNil(t, result)
	assert.NotNil(t, result.ClassifyResult)
}

func TestProcessTimestamp(t *testing.T) {
	f := NewFeedback()

	result := f.Process("q1", "Question", "Answer", nil)

	assert.NotEmpty(t, result.Timestamp)
}

func TestProcessFinalConfidence(t *testing.T) {
	f := NewFeedback()

	result := f.Process("q1", "What is the meaning of life?", "The definitive answer is 42", nil)

	assert.GreaterOrEqual(t, result.FinalConfidence, 0.0)
	assert.LessOrEqual(t, result.FinalConfidence, 1.0)
}

func TestProcessWithFailedStep(t *testing.T) {
	f := NewFeedback()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: false, Error: "SyntaxError", Confidence: 0.0},
	}

	result := f.Process("q1", "Question", "Answer", stepResults)

	assert.NotNil(t, result)
	assert.NotNil(t, result.ClassifyResult)
}

func TestProcessWithSuccessfulSteps(t *testing.T) {
	f := NewFeedback()

	stepResults := []*models.StepResult{
		{StepID: 1, Success: true, Confidence: 1.0},
		{StepID: 2, Success: true, Confidence: 1.0},
	}

	result := f.Process("q1", "Question", "Answer", stepResults)

	assert.NotNil(t, result)
}

func TestProcessEmptyAnswer(t *testing.T) {
	f := NewFeedback()

	result := f.Process("q1", "Question", "", nil)

	assert.NotNil(t, result)
	assert.False(t, result.ValidateResult.IsValid)
}

func TestProcessMultiple(t *testing.T) {
	f := NewFeedback()

	// Process multiple feedbacks
	f.Process("q1", "Question 1", "Answer 1", nil)
	f.Process("q2", "Question 2", "Answer 2", nil)

	stats := f.GetStatistics()

	assert.Equal(t, 2, stats.TotalCount)
}

func TestGetStatistics(t *testing.T) {
	f := NewFeedback()

	stats := f.GetStatistics()

	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.TotalCount)
}

func TestProcessFinalConfidenceRange(t *testing.T) {
	f := NewFeedback()

	// Test various confidence levels
	result1 := f.Process("q1", "Question", "I am very confident about this answer.", nil)
	result2 := f.Process("q2", "Question", "Maybe the answer is 42.", nil)

	assert.GreaterOrEqual(t, result1.FinalConfidence, result2.FinalConfidence)
}

func TestProcessWithLongAnswer(t *testing.T) {
	f := NewFeedback()

	longAnswer := "The answer is 42. This is because... [long explanation]"
	result := f.Process("q1", "Question", longAnswer, nil)

	assert.NotNil(t, result)
}

func TestProcessWithQuestionID(t *testing.T) {
	f := NewFeedback()

	result := f.Process("test-question-123", "Question", "Answer", nil)

	assert.Equal(t, "test-question-123", result.QuestionID)
}

func TestGetHistory(t *testing.T) {
	f := NewFeedback()

	f.Process("q1", "Question 1", "Answer 1", nil)
	f.Process("q2", "Question 2", "Answer 2", nil)

	history := f.GetHistory()

	assert.Len(t, history, 2)
}

func TestGetValidator(t *testing.T) {
	f := NewFeedback()

	validator := f.GetValidator()

	assert.NotNil(t, validator)
}

func TestGetClassifier(t *testing.T) {
	f := NewFeedback()

	classifier := f.GetClassifier()

	assert.NotNil(t, classifier)
}

func TestProcessIsAcceptable(t *testing.T) {
	f := NewFeedback()

	result := f.Process("q1", "Question", "The answer is 42", nil)

	assert.NotNil(t, result)
}

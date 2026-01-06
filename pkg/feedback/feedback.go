package feedback

import (
	"sync"
	"time"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// Feedback combines validation and classification for comprehensive feedback
type Feedback struct {
	validator  *Validator
	classifier *Classifier
	logger     *zap.Logger
	mu         sync.RWMutex
	history    []*FeedbackRecord
}

// FeedbackRecord represents a feedback record
type FeedbackRecord struct {
	QuestionID    string           `json:"question_id"`
	Timestamp     string           `json:"timestamp"`
	ValidateResult *ValidateResult   `json:"validate_result"`
	ClassifyResult *ClassificationResult `json:"classify_result"`
	FinalConfidence float64         `json:"final_confidence"`
	IsAcceptable   bool            `json:"is_acceptable"`
}

// NewFeedback creates a new Feedback instance
func NewFeedback() *Feedback {
	return &Feedback{
		validator:  NewValidator(nil),
		classifier: NewClassifier(),
		logger:     logging.WithComponent("Feedback"),
		history:    make([]*FeedbackRecord, 0),
	}
}

// Process processes step results and returns comprehensive feedback
func (f *Feedback) Process(questionID string, question string, answer string, stepResults []*models.StepResult) *FeedbackRecord {
	record := &FeedbackRecord{
		QuestionID: questionID,
		Timestamp:  getCurrentTimestamp(),
	}

	// Validate the answer
	validateResult := f.validator.Validate(question, answer, stepResults)
	record.ValidateResult = validateResult

	// Classify any errors
	classifyResult := f.classifier.Classify(stepResults)
	record.ClassifyResult = classifyResult

	// Calculate final confidence
	record.FinalConfidence = f.calculateFinalConfidence(validateResult, classifyResult)

	// Determine if answer is acceptable
	record.IsAcceptable = f.isAcceptable(record, stepResults)

	// Store in history
	f.mu.Lock()
	f.history = append(f.history, record)
	if len(f.history) > 100 {
		f.history = f.history[len(f.history)-100:]
	}
	f.mu.Unlock()

	f.logger.Debug("反馈处理完成",
		zap.String("question_id", questionID),
		zap.Bool("is_acceptable", record.IsAcceptable),
		zap.Float64("final_confidence", record.FinalConfidence),
		zap.Int("issues_count", len(validateResult.Issues)))

	return record
}

// getCurrentTimestamp returns the current timestamp in RFC3339 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

// calculateFinalConfidence calculates the final confidence score
func (f *Feedback) calculateFinalConfidence(validateResult *ValidateResult, classifyResult *ClassificationResult) float64 {
	// Base confidence from validation
	confidence := float64(validateResult.Confidence)

	// Adjust based on error classification
	if len(classifyResult.RelatedSteps) > 0 {
		errorRate := float64(len(classifyResult.RelatedSteps)) / 10.0 // Assuming max 10 steps
		confidence -= errorRate * 0.2
	}

	// Adjust based on issues
	for _, issue := range validateResult.Issues {
		switch issue.Severity {
		case SeverityCritical, SeverityError:
			confidence -= 0.15
		case SeverityWarning:
			confidence -= 0.05
		}
	}

	// Ensure confidence is between 0 and 1
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

// isAcceptable determines if the answer is acceptable
func (f *Feedback) isAcceptable(record *FeedbackRecord, stepResults []*models.StepResult) bool {
	// Must have acceptable confidence
	if record.FinalConfidence < 0.5 {
		return false
	}

	// Must have valid answer
	if !record.ValidateResult.IsValid {
		return false
	}

	// Check step success rate
	successCount := 0
	for _, sr := range stepResults {
		if sr.Success {
			successCount++
		}
	}
	successRate := float64(successCount) / float64(len(stepResults))
	if successRate < 0.5 {
		return false
	}

	return true
}

// GetHistory returns the feedback history
func (f *Feedback) GetHistory() []*FeedbackRecord {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make([]*FeedbackRecord, len(f.history))
	copy(result, f.history)
	return result
}

// GetStatistics returns feedback statistics
func (f *Feedback) GetStatistics() FeedbackStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := FeedbackStats{
		TotalCount:       len(f.history),
		AcceptableCount:  0,
		RejectCount:      0,
		AvgConfidence:    0,
		IssueCounts:      make(map[string]int),
		ErrorTypeCounts:  make(map[string]int),
	}

	var totalConf float64
	for _, record := range f.history {
		if record.IsAcceptable {
			stats.AcceptableCount++
		} else {
			stats.RejectCount++
		}

		totalConf += record.FinalConfidence

		// Count issues
		for _, issue := range record.ValidateResult.Issues {
			stats.IssueCounts[string(issue.Type)]++
		}

		// Count error types
		stats.ErrorTypeCounts[string(record.ClassifyResult.ErrorType)]++
	}

	if stats.TotalCount > 0 {
		stats.AvgConfidence = totalConf / float64(stats.TotalCount)
	}

	return stats
}

// FeedbackStats represents feedback statistics
type FeedbackStats struct {
	TotalCount      int              `json:"total_count"`
	AcceptableCount int              `json:"acceptable_count"`
	RejectCount     int              `json:"reject_count"`
	AvgConfidence   float64          `json:"avg_confidence"`
	IssueCounts     map[string]int   `json:"issue_counts"`
	ErrorTypeCounts map[string]int   `json:"error_type_counts"`
}

// GetValidator returns the validator
func (f *Feedback) GetValidator() *Validator {
	return f.validator
}

// GetClassifier returns the classifier
func (f *Feedback) GetClassifier() *Classifier {
	return f.classifier
}

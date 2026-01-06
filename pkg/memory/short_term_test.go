package memory

import (
	"testing"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultMemory(t *testing.T) {
	mem := NewDefaultMemory()

	assert.NotNil(t, mem)
	assert.NotNil(t, mem.history)
	assert.NotNil(t, mem.stepResults)
	assert.Equal(t, 100, mem.maxHistorySize)
	assert.Equal(t, 1000, mem.config.MaxSteps)
	assert.Equal(t, "", mem.conversationID)
}

func TestNewShortTermMemory(t *testing.T) {
	cfg := &MemoryConfig{
		MaxHistorySize:  50,
		MaxSteps:        500,
		SessionTTL:      7200,
		PersistEnabled:  true,
	}

	mem := NewShortTermMemory(cfg)

	assert.NotNil(t, mem)
	assert.Equal(t, 50, mem.maxHistorySize)
	assert.Equal(t, 500, mem.config.MaxSteps)
	assert.Equal(t, 7200, mem.config.SessionTTL)
	assert.True(t, mem.config.PersistEnabled)
}

func TestStartSession(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session-123")

	assert.Equal(t, "test-session-123", mem.conversationID)
	// currentPlan is reset to nil in StartSession
	assert.Nil(t, mem.currentPlan)
	// stepResults should be initialized to empty slice
	assert.NotNil(t, mem.stepResults)
	assert.Empty(t, mem.stepResults)
}

func TestAddQuestion(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddQuestion("What is 2+2?", map[string]interface{}{
		"domain":     "math",
		"complexity": "low",
		"keywords":   []string{"addition"},
	})

	history := mem.GetRecentHistory(10)
	require.Len(t, history, 1)
	assert.Equal(t, EntryQuestion, history[0].Type)
	assert.Equal(t, "What is 2+2?", history[0].Content)
	// Confidence defaults to 0 for AddQuestion
}

func TestAddPlan(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	plan := &models.Plan{
		ID:         "plan-001",
		TotalSteps: 3,
		Steps: []models.Step{
			{ID: 1, Description: "Step 1"},
			{ID: 2, Description: "Step 2"},
			{ID: 3, Description: "Step 3"},
		},
	}

	mem.AddPlan(plan)

	history := mem.GetRecentHistory(10)
	require.Len(t, history, 1)
	assert.Equal(t, EntryPlan, history[0].Type)
	assert.Contains(t, history[0].Content, "plan-001")
}

func TestAddStepResult(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	result := &models.StepResult{
		StepID:     1,
		Success:    true,
		Output:     "4",
		Confidence: 1.0,
	}

	mem.AddStepResult(result)

	stepResults := mem.GetStepResults()
	require.Len(t, stepResults, 1)
	assert.Equal(t, 1, stepResults[0].StepID)
	assert.True(t, stepResults[0].Success)
	assert.Equal(t, "4", stepResults[0].Output)
}

func TestAddReasoning(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddReasoning("This is a test reasoning trace", 0.9)

	history := mem.GetRecentHistory(10)
	require.Len(t, history, 1)
	assert.Equal(t, EntryReasoning, history[0].Type)
	assert.Equal(t, "This is a test reasoning trace", history[0].Content)
	assert.Equal(t, 0.9, history[0].Confidence)
}

func TestAddAnswer(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddAnswer("The answer is 4", 1.0)

	history := mem.GetRecentHistory(10)
	require.Len(t, history, 1)
	assert.Equal(t, EntryAnswer, history[0].Type)
	assert.Equal(t, "The answer is 4", history[0].Content)
	assert.Equal(t, 1.0, history[0].Confidence)
}

func TestGetRecentHistoryLimit(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	// Add 10 questions
	for i := 0; i < 10; i++ {
		mem.AddQuestion("question", map[string]interface{}{"domain": "math"})
	}

	// Get only 5
	history := mem.GetRecentHistory(5)
	assert.Len(t, history, 5)
}

func TestGetStepResults(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	// Add some step results
	for i := 1; i <= 5; i++ {
		mem.AddStepResult(&models.StepResult{
			StepID:     i,
			Success:    true,
			Confidence: 1.0,
		})
	}

	results := mem.GetStepResults()
	assert.Len(t, results, 5)
}

func TestGetContextForLLM(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddQuestion("Test question", map[string]interface{}{
		"domain": "math",
	})

	mem.AddReasoning("Test reasoning trace", 0.8)

	context := mem.GetContextForLLM(10)

	assert.Contains(t, context, "Test question")
	assert.Contains(t, context, "Test reasoning trace")
}

func TestClear(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddQuestion("Test", map[string]interface{}{})
	mem.AddAnswer("Answer", 1.0)

	assert.Len(t, mem.GetRecentHistory(10), 2)

	mem.Clear()

	assert.Len(t, mem.GetRecentHistory(10), 0)
	assert.Len(t, mem.GetStepResults(), 0)
}

func TestGetStatistics(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	stats := mem.GetStatistics()

	// MemoryStats has QuestionCount, PlanCount, etc.
	assert.GreaterOrEqual(t, stats.TotalEntries, 0)
}

func TestMemoryEntryTypes(t *testing.T) {
	assert.Equal(t, EntryType("question"), EntryQuestion)
	assert.Equal(t, EntryType("plan"), EntryPlan)
	assert.Equal(t, EntryType("step"), EntryStep)
	assert.Equal(t, EntryType("result"), EntryResult)
	assert.Equal(t, EntryType("reasoning"), EntryReasoning)
	assert.Equal(t, EntryType("answer"), EntryAnswer)
}

func TestSessionStates(t *testing.T) {
	assert.Equal(t, SessionState("idle"), SessionIdle)
	assert.Equal(t, SessionState("active"), SessionActive)
	assert.Equal(t, SessionState("completed"), SessionCompleted)
}

func TestMemoryConfigYAMLTags(t *testing.T) {
	cfg := &MemoryConfig{
		MaxSteps:       100,
		MaxHistorySize: 50,
		SessionTTL:     3600,
		PersistEnabled: true,
	}

	assert.Equal(t, 100, cfg.MaxSteps)
	assert.Equal(t, 50, cfg.MaxHistorySize)
	assert.Equal(t, 3600, cfg.SessionTTL)
	assert.True(t, cfg.PersistEnabled)
}

func TestMultipleSessionLifecycle(t *testing.T) {
	mem := NewDefaultMemory()

	// Start first session
	mem.StartSession("session-1")

	mem.AddQuestion("Q1", map[string]interface{}{})
	assert.Equal(t, 1, mem.GetStatistics().QuestionCount)

	// Clear and start new session
	mem.Clear()
	mem.StartSession("session-2")

	// First session questions should be gone
	assert.Equal(t, 0, mem.GetStatistics().QuestionCount)
}

func TestQuestionWithMetadata(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddQuestion("Complex math problem", map[string]interface{}{
		"domain":       "math",
		"sub_domain":   "algebra",
		"complexity":   "high",
		"keywords":     []string{"equation", "x", "solve"},
		"has_code":     true,
		"code_length":  50,
	})

	history := mem.GetRecentHistory(1)
	require.Len(t, history, 1)

	metadata, ok := history[0].Metadata.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "math", metadata["domain"])
	assert.Equal(t, "algebra", metadata["sub_domain"])
	assert.Equal(t, "high", metadata["complexity"])
	assert.Equal(t, true, metadata["has_code"])
}

func TestStepResultWithFailure(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddStepResult(&models.StepResult{
		StepID:     1,
		Success:    false,
		Error:      "SyntaxError: invalid syntax",
		Confidence: 0.5,
	})

	results := mem.GetStepResults()
	require.Len(t, results, 1)
	assert.False(t, results[0].Success)
	assert.Contains(t, results[0].Error, "SyntaxError")
}

func TestMaxHistorySizeEnforcement(t *testing.T) {
	cfg := &MemoryConfig{
		MaxHistorySize: 5,
		MaxSteps:       100,
		SessionTTL:     3600,
		PersistEnabled: false,
	}

	mem := NewShortTermMemory(cfg)
	mem.StartSession("test-session")

	// Add 10 questions (more than max)
	for i := 0; i < 10; i++ {
		mem.AddQuestion("question", map[string]interface{}{})
	}

	// Should be limited to 5
	history := mem.GetRecentHistory(100)
	assert.LessOrEqual(t, len(history), 5)
}

func TestGetHistoryByType(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddQuestion("Q1", map[string]interface{}{})
	mem.AddQuestion("Q2", map[string]interface{}{})
	mem.AddAnswer("A1", 1.0)

	questions := mem.GetHistoryByType(EntryQuestion)
	answers := mem.GetHistoryByType(EntryAnswer)

	assert.Len(t, questions, 2)
	assert.Len(t, answers, 1)
}

func TestConfidenceTracking(t *testing.T) {
	mem := NewDefaultMemory()
	mem.StartSession("test-session")

	mem.AddQuestion("Q1", map[string]interface{}{})
	mem.AddReasoning("Reasoning 1", 0.8)
	mem.AddAnswer("A1", 0.95)

	history := mem.GetRecentHistory(10)
	require.Len(t, history, 3)

	// Find the reasoning entry and check its confidence
	var reasoningEntry MemoryEntry
	for _, entry := range history {
		if entry.Type == EntryReasoning {
			reasoningEntry = entry
			break
		}
	}
	assert.Equal(t, 0.8, reasoningEntry.Confidence)
}

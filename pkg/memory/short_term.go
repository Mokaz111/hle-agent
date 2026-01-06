package memory

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hle-agent/hle-agent/internal/models"
	"github.com/hle-agent/hle-agent/pkg/logging"
	"go.uber.org/zap"
)

// ShortTermMemory manages conversation context and recent history
type ShortTermMemory struct {
	logger           *zap.Logger
	config           *MemoryConfig
	conversationID   string
	history          []MemoryEntry
	currentPlan      *models.Plan
	stepResults      []*models.StepResult
	maxHistorySize   int
	mu               sync.RWMutex
}

// MemoryConfig represents memory configuration
type MemoryConfig struct {
	MaxSteps        int    `yaml:"max_steps"`
	MaxHistorySize  int    `yaml:"max_history_size"`
	SessionTTL      int    `yaml:"session_ttl"` // in seconds
	PersistEnabled  bool   `yaml:"persist_enabled"`
}

// MemoryEntry represents a single memory entry
type MemoryEntry struct {
	Type        EntryType   `json:"type"`
	Content     string      `json:"content"`
	Metadata    interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
	Confidence  float64     `json:"confidence"`
}

// EntryType represents the type of memory entry
type EntryType string

const (
	EntryQuestion   EntryType = "question"
	EntryPlan       EntryType = "plan"
	EntryStep       EntryType = "step"
	EntryResult     EntryType = "result"
	EntryReasoning  EntryType = "reasoning"
	EntryAnswer     EntryType = "answer"
)

// SessionInfo represents current session information
type SessionInfo struct {
	SessionID      string        `json:"session_id"`
	StartTime      time.Time     `json:"start_time"`
	QuestionCount  int           `json:"question_count"`
	TotalSteps     int           `json:"total_steps"`
	AvgConfidence  float64       `json:"avg_confidence"`
	CurrentState   SessionState  `json:"current_state"`
}

// SessionState represents the current state of the session
type SessionState string

const (
	SessionIdle      SessionState = "idle"
	SessionActive    SessionState = "active"
	SessionCompleted SessionState = "completed"
)

// NewShortTermMemory creates a new short-term memory instance
func NewShortTermMemory(cfg *MemoryConfig) *ShortTermMemory {
	if cfg.MaxHistorySize <= 0 {
		cfg.MaxHistorySize = 100
	}
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 1000
	}

	return &ShortTermMemory{
		logger:         logging.WithComponent("ShortTermMemory"),
		config:         cfg,
		history:        make([]MemoryEntry, 0),
		stepResults:    make([]*models.StepResult, 0),
		maxHistorySize: cfg.MaxHistorySize,
		currentPlan:    nil,
	}
}

// NewDefaultMemory creates a memory instance with default config
func NewDefaultMemory() *ShortTermMemory {
	return NewShortTermMemory(&MemoryConfig{
		MaxHistorySize:  100,
		MaxSteps:        1000,
		SessionTTL:      3600, // 1 hour
		PersistEnabled:  false,
	})
}

// StartSession starts a new conversation session
func (m *ShortTermMemory) StartSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.conversationID = sessionID
	m.history = make([]MemoryEntry, 0)
	m.stepResults = make([]*models.StepResult, 0)
	m.currentPlan = nil

	m.logger.Info("开始新会话",
		zap.String("session_id", sessionID))
}

// GetSessionInfo returns current session information
func (m *ShortTermMemory) GetSessionInfo() SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgConf := m.calculateAverageConfidence()

	return SessionInfo{
		SessionID:     m.conversationID,
		StartTime:     m.getStartTime(),
		QuestionCount: m.countEntriesByType(EntryQuestion),
		TotalSteps:    len(m.stepResults),
		AvgConfidence: avgConf,
		CurrentState:  m.getSessionState(),
	}
}

// AddQuestion adds a question to memory
func (m *ShortTermMemory) AddQuestion(question string, metadata interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := MemoryEntry{
		Type:      EntryQuestion,
		Content:   truncateString(question, 500),
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	m.history = append(m.history, entry)
	m.pruneHistory()

	m.logger.Debug("添加问题到记忆",
		zap.String("session_id", m.conversationID),
		zap.Int("history_size", len(m.history)))
}

// AddPlan stores the current plan
func (m *ShortTermMemory) AddPlan(plan *models.Plan) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentPlan = plan

	entry := MemoryEntry{
		Type:        EntryPlan,
		Content:     plan.ID,
		Metadata:    plan,
		Timestamp:   time.Now(),
		Confidence:  1.0,
	}

	m.history = append(m.history, entry)

	m.logger.Debug("存储计划",
		zap.String("plan_id", plan.ID),
		zap.Int("steps", plan.TotalSteps))
}

// AddStepResult stores a step execution result
func (m *ShortTermMemory) AddStepResult(result *models.StepResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stepResults = append(m.stepResults, result)

	entry := MemoryEntry{
		Type:       EntryStep,
		Content:    fmt.Sprintf("步骤 %d", result.StepID),
		Metadata:   result,
		Timestamp:  time.Now(),
		Confidence: result.Confidence,
	}

	m.history = append(m.history, entry)

	m.logger.Debug("存储步骤结果",
		zap.Int("step_id", result.StepID),
		zap.Bool("success", result.Success),
		zap.Float64("confidence", result.Confidence))
}

// AddReasoning stores reasoning trace
func (m *ShortTermMemory) AddReasoning(reasoning string, confidence float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := MemoryEntry{
		Type:       EntryReasoning,
		Content:    reasoning,
		Timestamp:  time.Now(),
		Confidence: confidence,
	}

	m.history = append(m.history, entry)
	m.pruneHistory()
}

// AddAnswer stores the final answer
func (m *ShortTermMemory) AddAnswer(answer string, confidence float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := MemoryEntry{
		Type:       EntryAnswer,
		Content:    answer,
		Timestamp:  time.Now(),
		Confidence: confidence,
	}

	m.history = append(m.history, entry)

	m.logger.Info("存储最终答案",
		zap.Float64("confidence", confidence))
}

// GetRecentHistory returns the recent history entries
func (m *ShortTermMemory) GetRecentHistory(count int) []MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if count <= 0 || count > len(m.history) {
		count = len(m.history)
	}

	result := make([]MemoryEntry, count)
	copy(result, m.history[len(m.history)-count:])
	return result
}

// GetHistoryByType returns entries filtered by type
func (m *ShortTermMemory) GetHistoryByType(entryType EntryType) []MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]MemoryEntry, 0)
	for _, entry := range m.history {
		if entry.Type == entryType {
			result = append(result, entry)
		}
	}
	return result
}

// GetCurrentPlan returns the current plan if any
func (m *ShortTermMemory) GetCurrentPlan() *models.Plan {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.currentPlan
}

// GetStepResults returns all step results
func (m *ShortTermMemory) GetStepResults() []*models.StepResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.StepResult, len(m.stepResults))
	copy(result, m.stepResults)
	return result
}

// GetContextForLLM returns formatted context for LLM prompts
func (m *ShortTermMemory) GetContextForLLM(maxEntries int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history := m.GetRecentHistory(maxEntries)
	if len(history) == 0 {
		return ""
	}

	var context strings.Builder
	context.WriteString("=== 对话上下文 ===\n\n")

	for _, entry := range history {
		switch entry.Type {
		case EntryQuestion:
			context.WriteString("问题:\n")
			context.WriteString(entry.Content)
			context.WriteString("\n\n")
		case EntryStep:
			if metadata, ok := entry.Metadata.(*models.StepResult); ok {
				context.WriteString("步骤结果:\n")
				context.WriteString(fmt.Sprintf("步骤 %d - ", metadata.StepID))
				if metadata.Success {
					context.WriteString("成功")
				} else {
					context.WriteString("失败")
				}
				context.WriteString(fmt.Sprintf(" (置信度: %.0f%%)", metadata.Confidence*100))
				context.WriteString("\n\n")
			}
		case EntryReasoning:
			context.WriteString("推理:\n")
			context.WriteString(truncateString(entry.Content, 200))
			context.WriteString("\n\n")
		}

		// Limit context size
		if context.Len() > 3000 {
			break
		}
	}

	return context.String()
}

// Clear clears all memory
func (m *ShortTermMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.history = make([]MemoryEntry, 0)
	m.stepResults = make([]*models.StepResult, 0)
	m.currentPlan = nil

	m.logger.Info("清除短期记忆")
}

// GetStatistics returns memory statistics
func (m *ShortTermMemory) GetStatistics() MemoryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MemoryStats{
		TotalEntries:    len(m.history),
		QuestionCount:   m.countEntriesByType(EntryQuestion),
		PlanCount:       m.countEntriesByType(EntryPlan),
		StepCount:       m.countEntriesByType(EntryStep),
		ReasoningCount:  m.countEntriesByType(EntryReasoning),
		AnswerCount:     m.countEntriesByType(EntryAnswer),
		TotalSteps:      len(m.stepResults),
		AvgConfidence:   m.calculateAverageConfidence(),
		SessionDuration: time.Since(m.getStartTime()),
	}
}

// MemoryStats represents memory statistics
type MemoryStats struct {
	TotalEntries   int           `json:"total_entries"`
	QuestionCount  int           `json:"question_count"`
	PlanCount      int           `json:"plan_count"`
	StepCount      int           `json:"step_count"`
	ReasoningCount int           `json:"reasoning_count"`
	AnswerCount    int           `json:"answer_count"`
	TotalSteps     int           `json:"total_steps"`
	AvgConfidence  float64       `json:"avg_confidence"`
	SessionDuration time.Duration `json:"session_duration"`
}

// Helper functions

func (m *ShortTermMemory) pruneHistory() {
	if len(m.history) > m.maxHistorySize {
		m.history = m.history[len(m.history)-m.maxHistorySize:]
	}
}

func (m *ShortTermMemory) countEntriesByType(entryType EntryType) int {
	count := 0
	for _, entry := range m.history {
		if entry.Type == entryType {
			count++
		}
	}
	return count
}

func (m *ShortTermMemory) calculateAverageConfidence() float64 {
	if len(m.history) == 0 {
		return 0
	}

	var total float64
	count := 0
	for _, entry := range m.history {
		if entry.Confidence > 0 {
			total += entry.Confidence
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func (m *ShortTermMemory) getStartTime() time.Time {
	if len(m.history) == 0 {
		return time.Now()
	}
	return m.history[0].Timestamp
}

func (m *ShortTermMemory) getSessionState() SessionState {
	if m.currentPlan == nil {
		return SessionIdle
	}
	if len(m.stepResults) >= m.currentPlan.TotalSteps {
		return SessionCompleted
	}
	return SessionActive
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

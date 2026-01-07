package agent

import "context"

// contextKey 是一个类型安全的 context key
type contextKey string

const (
	// ContextKeyQuestion 用于在 context 中存储题目
	ContextKeyQuestion contextKey = "hle_agent_question"
	// ContextKeyUserMessage 用于在 context 中存储用户消息
	ContextKeyUserMessage contextKey = "hle_agent_user_message"
	// ContextKeyQuestionID 用于在 context 中存储题目ID
	ContextKeyQuestionID contextKey = "hle_agent_question_id"
	// ContextKeySessionID 用于在 context 中存储会话ID
	ContextKeySessionID contextKey = "hle_agent_session_id"
)

// WithQuestion 将题目添加到 context 中
func WithQuestion(ctx context.Context, question string) context.Context {
	return context.WithValue(ctx, ContextKeyQuestion, question)
}

// WithUserMessage 将用户消息添加到 context 中
func WithUserMessage(ctx context.Context, message string) context.Context {
	return context.WithValue(ctx, ContextKeyUserMessage, message)
}

// WithQuestionID 将题目ID添加到 context 中
func WithQuestionID(ctx context.Context, questionID string) context.Context {
	return context.WithValue(ctx, ContextKeyQuestionID, questionID)
}

// WithSessionID 将会话ID添加到 context 中
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ContextKeySessionID, sessionID)
}

// GetQuestion 从 context 中获取题目
func GetQuestion(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if question, ok := ctx.Value(ContextKeyQuestion).(string); ok {
		return question
	}
	return ""
}

// GetUserMessage 从 context 中获取用户消息
func GetUserMessage(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if message, ok := ctx.Value(ContextKeyUserMessage).(string); ok {
		return message
	}
	return ""
}

// GetQuestionID 从 context 中获取题目ID
func GetQuestionID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if questionID, ok := ctx.Value(ContextKeyQuestionID).(string); ok {
		return questionID
	}
	return ""
}

// GetSessionID 从 context 中获取会话ID
func GetSessionID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if sessionID, ok := ctx.Value(ContextKeySessionID).(string); ok {
		return sessionID
	}
	return ""
}

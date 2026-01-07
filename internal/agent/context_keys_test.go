package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithQuestion(t *testing.T) {
	ctx := context.Background()
	question := "What is 2+2?"
	
	ctxWithQuestion := WithQuestion(ctx, question)
	
	retrieved := GetQuestion(ctxWithQuestion)
	assert.Equal(t, question, retrieved)
}

func TestWithUserMessage(t *testing.T) {
	ctx := context.Background()
	message := "Hello, world!"
	
	ctxWithMessage := WithUserMessage(ctx, message)
	
	retrieved := GetUserMessage(ctxWithMessage)
	assert.Equal(t, message, retrieved)
}

func TestWithQuestionID(t *testing.T) {
	ctx := context.Background()
	questionID := "q_123"
	
	ctxWithID := WithQuestionID(ctx, questionID)
	
	retrieved := GetQuestionID(ctxWithID)
	assert.Equal(t, questionID, retrieved)
}

func TestWithSessionID(t *testing.T) {
	ctx := context.Background()
	sessionID := "session_456"
	
	ctxWithSession := WithSessionID(ctx, sessionID)
	
	retrieved := GetSessionID(ctxWithSession)
	assert.Equal(t, sessionID, retrieved)
}

func TestGetQuestionWithNilContext(t *testing.T) {
	result := GetQuestion(nil)
	assert.Empty(t, result)
}

func TestGetUserMessageWithNilContext(t *testing.T) {
	result := GetUserMessage(nil)
	assert.Empty(t, result)
}

func TestGetQuestionIDWithNilContext(t *testing.T) {
	result := GetQuestionID(nil)
	assert.Empty(t, result)
}

func TestGetSessionIDWithNilContext(t *testing.T) {
	result := GetSessionID(nil)
	assert.Empty(t, result)
}

func TestGetQuestionWithWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ContextKeyQuestion, 123)
	result := GetQuestion(ctx)
	assert.Empty(t, result)
}

func TestContextKeysChaining(t *testing.T) {
	ctx := context.Background()
	ctx = WithQuestion(ctx, "What is 2+2?")
	ctx = WithQuestionID(ctx, "q_123")
	ctx = WithSessionID(ctx, "session_456")
	
	assert.Equal(t, "What is 2+2?", GetQuestion(ctx))
	assert.Equal(t, "q_123", GetQuestionID(ctx))
	assert.Equal(t, "session_456", GetSessionID(ctx))
}

func TestExtractQuestionFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
	}{
		{
			name: "from question key",
			setupCtx: func() context.Context {
				return WithQuestion(context.Background(), "test question")
			},
			expected: "test question",
		},
		{
			name: "from user message key",
			setupCtx: func() context.Context {
				return WithUserMessage(context.Background(), "test message")
			},
			expected: "test message",
		},
		{
			name: "from legacy question key",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), "question", "legacy question")
			},
			expected: "legacy question",
		},
		{
			name: "from legacy user_message key",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), "user_message", "legacy message")
			},
			expected: "legacy message",
		},
		{
			name: "empty context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			result := extractQuestionFromContext(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}


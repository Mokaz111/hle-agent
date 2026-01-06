package llm

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/hle-agent/hle-agent/internal/config"
)

// Message represents a chat message (for backward compatibility)
type Message = schema.Message

// Client represents an LLM client using Eino OpenAI integration
type Client struct {
	chatModel *openai.ChatModel
	cfg       *config.ModelConfig
}

// NewClient creates a new LLM client using Eino OpenAI
func NewClient(cfg *config.ModelConfig) (*Client, error) {
	// Convert config values to pointers for Eino
	temp := float32(cfg.Temperature)
	maxTokens := cfg.MaxTokens

	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:       cfg.Model,
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		chatModel: chatModel,
		cfg:       cfg,
	}, nil
}

// Generate generates a chat completion
func (c *Client) Generate(ctx context.Context, messages []Message) (string, error) {
	// Convert to Eino schema messages
	einoMessages := make([]*schema.Message, len(messages))
	for i, msg := range messages {
		einoMessages[i] = &schema.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Generate response
	resp, err := c.chatModel.Generate(ctx, einoMessages)
	if err != nil {
		return "", err
	}

	if resp == nil {
		return "", nil
	}

	// Extract content from response
	// Eino returns *schema.Message directly
	return resp.Content, nil
}

// GenerateWithSystemPrompt generates a chat completion with system prompt
func (c *Client) GenerateWithSystemPrompt(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []*schema.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	if resp == nil {
		return "", nil
	}

	return resp.Content, nil
}

// Stream generates a chat completion with streaming
func (c *Client) Stream(ctx context.Context, messages []Message, callback func(string)) error {
	// Convert to Eino schema messages
	einoMessages := make([]*schema.Message, len(messages))
	for i, msg := range messages {
		einoMessages[i] = &schema.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Create stream reader
	stream, err := c.chatModel.Stream(ctx, einoMessages)
	if err != nil {
		return err
	}
	defer stream.Close()

	// Process stream
	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}
		if msg != nil {
			callback(msg.Content)
		}
	}

	return nil
}

package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider:    "openai",
		Model:       "gpt-4",
		BaseURL:     "https://api.openai.com/v1",
		APIKey:      "test-key",
		Timeout:     60,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	client := NewClient(cfg)

	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, cfg, client.cfg)
}

func TestMessageStruct(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello, world!",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello, world!", msg.Content)
}

func TestChatRequestStruct(t *testing.T) {
	req := ChatRequest{
		Model:       "gpt-4",
		Messages:    []Message{{Role: "user", Content: "Hi"}},
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	assert.Equal(t, "gpt-4", req.Model)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, 0.7, req.Temperature)
	assert.Equal(t, 1000, req.MaxTokens)
}

func TestChatResponseStruct(t *testing.T) {
	resp := ChatResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Choices: []Choice{
			{
				Index:        0,
				Message:      Message{Role: "assistant", Content: "Hello!"},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	assert.Equal(t, "chatcmpl-123", resp.ID)
	assert.Equal(t, "chat.completion", resp.Object)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, "Hello!", resp.Choices[0].Message.Content)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestChoiceStruct(t *testing.T) {
	choice := Choice{
		Index:        0,
		Message:      Message{Role: "assistant", Content: "Test"},
		FinishReason: "stop",
	}

	assert.Equal(t, 0, choice.Index)
	assert.Equal(t, "assistant", choice.Message.Role)
	assert.Equal(t, "stop", choice.FinishReason)
}

func TestUsageStruct(t *testing.T) {
	usage := Usage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	assert.Equal(t, 100, usage.PromptTokens)
	assert.Equal(t, 50, usage.CompletionTokens)
	assert.Equal(t, 150, usage.TotalTokens)
}

func TestClientWithMockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return a mock response
		resp := ChatResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Choices: []Choice{
				{
					Index:        0,
					Message:      Message{Role: "assistant", Content: "Test response"},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with mock server URL
	cfg := &config.ModelConfig{
		Provider:    "openai",
		Model:       "gpt-4",
		BaseURL:     server.URL,
		APIKey:      "test-key",
		Timeout:     60,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	client := NewClient(cfg)

	// Test Generate
	messages := []Message{
		{Role: "user", Content: "Say hello"},
	}

	response, err := client.Generate(context.Background(), messages)

	require.NoError(t, err)
	assert.Equal(t, "Test response", response)
}

func TestClientGenerateWithDifferentModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			ID:      "test-id",
			Choices: []Choice{{Message: Message{Content: "Response"}}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	testCases := []struct {
		name  string
		model string
	}{
		{"GPT-4", "gpt-4"},
		{"GPT-3.5", "gpt-3.5-turbo"},
		{"Claude", "claude-3-opus"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				Model:   tc.model,
				BaseURL: server.URL,
				APIKey:  "test-key",
				Timeout: 60,
			}

			client := NewClient(cfg)
			assert.Equal(t, tc.model, client.cfg.Model)
		})
	}
}

func TestClientTimeoutConfiguration(t *testing.T) {
	cfg := &config.ModelConfig{
		Model:   "gpt-4",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
		Timeout: 120,
	}

	client := NewClient(cfg)

	assert.NotNil(t, client.httpClient)
}

func TestClientRequestBody(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		resp := ChatResponse{
			ID:      "test-id",
			Choices: []Choice{{Message: Message{Content: "Response"}}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.ModelConfig{
		Model:       "gpt-4",
		BaseURL:     server.URL,
		APIKey:      "test-key",
		Timeout:     60,
		Temperature: 0.5,
		MaxTokens:   1000,
	}

	client := NewClient(cfg)
	client.Generate(context.Background(), []Message{{Role: "user", Content: "Test"}})

	require.NotNil(t, receivedBody)

	var req ChatRequest
	err := json.Unmarshal(receivedBody, &req)
	require.NoError(t, err)

	assert.Equal(t, "gpt-4", req.Model)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, 0.5, req.Temperature)
	assert.Equal(t, 1000, req.MaxTokens)
}

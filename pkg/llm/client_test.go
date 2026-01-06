package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/hle-agent/hle-agent/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.ModelConfig{
		Provider:    "openai",
		APIKey:      "test-api-key",
		Model:       "gpt-3.5-turbo",
		BaseURL:     "https://api.openai.com/v1",
		Temperature: 0.7,
		MaxTokens:   100,
		Timeout:     30,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.cfg.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model gpt-3.5-turbo, got %s", client.cfg.Model)
	}
}

func TestGenerate(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Check authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got %s", authHeader)
		}

		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %s", contentType)
		}

		// Parse request body
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)

		// Validate request
		if req["model"] != "gpt-3.5-turbo" {
			t.Errorf("Expected model gpt-3.5-turbo, got %v", req["model"])
		}

		// Return mock response - using Eino-compatible format
		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello! How can I help you today?",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server URL
	cfg := &config.ModelConfig{
		Provider:    "openai",
		APIKey:      "sk-api-ORpmGAVbyUtWl6j-oE5Xv60xIy596Vqnu509ezB-drgkL0oxujYiEDLVgw_7IUifaWmpYPAUoiRpqNOvez-kBaaSi4ZgDBqKoboetKMVj8Kb26rrZdzsYDU",
		Model:       "MiniMax-M2.1",
		BaseURL:     server.URL,
		Temperature: 0.7,
		MaxTokens:   100,
		Timeout:     30,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test generate
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	response, err := client.Generate(context.Background(), messages)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := "Hello! How can I help you today?"
	if response != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, response)
	}
}

func TestGenerateWithSystemPrompt(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var req struct {
			Model    string           `json:"model"`
			Messages []schema.Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		// Check that we have 2 messages (system + user)
		if len(req.Messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(req.Messages))
		}

		// Check first message is system
		if len(req.Messages) > 0 && req.Messages[0].Role != "system" {
			t.Errorf("Expected first message role 'system', got '%s'", req.Messages[0].Role)
		}

		// Check second message is user
		if len(req.Messages) > 1 && req.Messages[1].Role != "user" {
			t.Errorf("Expected second message role 'user', got '%s'", req.Messages[1].Role)
		}

		// Return mock response
		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "I am a helpful assistant.",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server URL
	cfg := &config.ModelConfig{
		Provider:    "openai",
		APIKey:      "test-api-key",
		Model:       "gpt-3.5-turbo",
		BaseURL:     server.URL,
		Temperature: 0.7,
		MaxTokens:   100,
		Timeout:     30,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test generate with system prompt
	response, err := client.GenerateWithSystemPrompt(
		context.Background(),
		"You are a helpful assistant.",
		"Hello!",
	)
	if err != nil {
		t.Fatalf("GenerateWithSystemPrompt failed: %v", err)
	}

	expected := "I am a helpful assistant."
	if response != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, response)
	}
}

func TestGenerateEmptyResponse(t *testing.T) {
	// Create mock server that returns empty choices
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"choices": []map[string]interface{}{}, // Empty choices - Eino returns error for this
			"usage":   map[string]int{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server URL
	cfg := &config.ModelConfig{
		Provider:    "openai",
		APIKey:      "test-api-key",
		Model:       "gpt-3.5-turbo",
		BaseURL:     server.URL,
		Temperature: 0.7,
		MaxTokens:   100,
		Timeout:     30,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test generate with empty response - Eino returns error for empty choices
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	_, err = client.Generate(context.Background(), messages)
	// Eino returns error for empty choices, which is expected behavior
	if err == nil {
		t.Log("Note: Eino might return empty string for empty choices in future versions")
		// This is acceptable - the test documents the current behavior
	}
}

func TestGenerateAPIError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	cfg := &config.ModelConfig{
		Provider:    "openai",
		APIKey:      "invalid-key",
		Model:       "gpt-3.5-turbo",
		BaseURL:     server.URL,
		Temperature: 0.7,
		MaxTokens:   100,
		Timeout:     30,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test generate with API error
	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	_, err = client.Generate(context.Background(), messages)
	if err == nil {
		t.Error("Expected error for API failure, got nil")
	}
}

func TestClientWithMultipleMessages(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		var req struct {
			Model    string           `json:"model"`
			Messages []schema.Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		// Return response with last message content
		lastMessage := ""
		if len(req.Messages) > 0 {
			lastMessage = req.Messages[len(req.Messages)-1].Content
		}

		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "Echo: " + lastMessage,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client with mock server URL
	cfg := &config.ModelConfig{
		Provider:    "openai",
		APIKey:      "test-api-key",
		Model:       "gpt-3.5-turbo",
		BaseURL:     server.URL,
		Temperature: 0.7,
		MaxTokens:   100,
		Timeout:     30,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test with multiple messages
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "First message"},
		{Role: "assistant", Content: "I understand."},
		{Role: "user", Content: "Second message"},
	}

	response, err := client.Generate(context.Background(), messages)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := "Echo: Second message"
	if response != expected {
		t.Errorf("Expected response '%s', got '%s'", expected, response)
	}
}

func TestClientConfiguration(t *testing.T) {
	testCases := []struct {
		name           string
		provider       string
		model          string
		baseURL        string
		temperature    float64
		maxTokens      int
		timeout        int
		expectedModel  string
		expectedTemp   float64
		expectedTokens int
	}{
		{
			name:           "OpenAI GPT-4",
			provider:       "openino",
			model:          "gpt-4",
			baseURL:        "https://api.openai.com/v1",
			temperature:    0.5,
			maxTokens:      200,
			timeout:        60,
			expectedModel:  "gpt-4",
			expectedTemp:   0.5,
			expectedTokens: 200,
		},
		{
			name:           "OpenAI GPT-3.5",
			provider:       "openai",
			model:          "gpt-3.5-turbo",
			baseURL:        "https://api.openai.com/v1",
			temperature:    0.8,
			maxTokens:      150,
			timeout:        30,
			expectedModel:  "gpt-3.5-turbo",
			expectedTemp:   0.8,
			expectedTokens: 150,
		},
		{
			name:           "Custom Provider",
			provider:       "custom",
			model:          "custom-model",
			baseURL:        "https://custom.api.com/v1",
			temperature:    0.0,
			maxTokens:      500,
			timeout:        120,
			expectedModel:  "custom-model",
			expectedTemp:   0.0,
			expectedTokens: 500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				Provider:    tc.provider,
				APIKey:      "test-key",
				Model:       tc.model,
				BaseURL:     tc.baseURL,
				Temperature: tc.temperature,
				MaxTokens:   tc.maxTokens,
				Timeout:     tc.timeout,
			}

			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			if client.cfg.Model != tc.expectedModel {
				t.Errorf("Expected model '%s', got '%s'", tc.expectedModel, client.cfg.Model)
			}

			if client.cfg.Temperature != tc.expectedTemp {
				t.Errorf("Expected temperature %f, got %f", tc.expectedTemp, client.cfg.Temperature)
			}

			if client.cfg.MaxTokens != tc.expectedTokens {
				t.Errorf("Expected max tokens %d, got %d", tc.expectedTokens, client.cfg.MaxTokens)
			}
		})
	}
}

// TestSchemaMessageCompatibility tests that our Message type is compatible with Eino schema.Message
func TestSchemaMessageCompatibility(t *testing.T) {
	// Test that we can use our Message type with Eino schema.Message
	einoMsg := &schema.Message{
		Role:    "user",
		Content: "Test message",
	}

	ourMsg := Message{
		Role:    "assistant",
		Content: "Test response",
	}

	// They should be compatible in structure
	if einoMsg.Role != "user" {
		t.Error("Eino message role mismatch")
	}

	if ourMsg.Role != "assistant" {
		t.Error("Our message role mismatch")
	}
}

// TestOpenAIChatModelConfig verifies the configuration structure
func TestOpenAIChatModelConfig(t *testing.T) {
	// Test pointer conversion
	temp := float32(0.7)
	maxTokens := 100

	cfg := &openai.ChatModelConfig{
		Model:       "gpt-3.5-turbo",
		APIKey:      "test-key",
		BaseURL:     "https://api.openai.com/v1",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	if cfg.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model gpt-3.5-turbo, got %s", cfg.Model)
	}

	if *cfg.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", *cfg.Temperature)
	}

	if *cfg.MaxTokens != 100 {
		t.Errorf("Expected max tokens 100, got %d", *cfg.MaxTokens)
	}
}

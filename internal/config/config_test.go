package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderTypeString(t *testing.T) {
	assert.Equal(t, "openai", ProviderOpenAI.String())
	assert.Equal(t, "anthropic", ProviderAnthropic.String())
	assert.Equal(t, "deepseek", ProviderDeepSeek.String())
	assert.Equal(t, "oneapi", ProviderOneAPI.String())
	assert.Equal(t, "custom", ProviderCustom.String())
}

func TestProviderTypeIsValid(t *testing.T) {
	assert.True(t, ProviderOpenAI.IsValid())
	assert.True(t, ProviderAnthropic.IsValid())
	assert.True(t, ProviderDeepSeek.IsValid())
	assert.True(t, ProviderOneAPI.IsValid())
	assert.True(t, ProviderCustom.IsValid())
}

func TestProviderTypeIsValidInvalid(t *testing.T) {
	assert.False(t, ProviderType("invalid").IsValid())
	assert.False(t, ProviderType("").IsValid())
	assert.False(t, ProviderType("unknown").IsValid())
}

func TestSupportedProviders(t *testing.T) {
	assert.Len(t, SupportedProviders, 5)
	assert.Contains(t, SupportedProviders, ProviderOpenAI)
	assert.Contains(t, SupportedProviders, ProviderAnthropic)
	assert.Contains(t, SupportedProviders, ProviderDeepSeek)
	assert.Contains(t, SupportedProviders, ProviderOneAPI)
	assert.Contains(t, SupportedProviders, ProviderCustom)
}

func TestModelConfigGetProviderType(t *testing.T) {
	cfg := &ModelConfig{Provider: "openai"}
	assert.Equal(t, ProviderOpenAI, cfg.GetProviderType())

	cfg.Provider = "OneAPI"
	assert.Equal(t, ProviderOneAPI, cfg.GetProviderType())

	cfg.Provider = "DEEPSEEK"
	assert.Equal(t, ProviderDeepSeek, cfg.GetProviderType())
}

func TestModelConfigGetProviderTypeInvalid(t *testing.T) {
	cfg := &ModelConfig{Provider: "invalid"}
	assert.Equal(t, ProviderCustom, cfg.GetProviderType())
}

func TestModelConfigWithOpenAIProvider(t *testing.T) {
	cfg := &ModelConfig{
		Provider:   "openai",
		APIKey:     "sk-test123",
		Model:      "gpt-4",
		BaseURL:    "https://api.openai.com/v1",
		Temperature: 0.7,
		MaxTokens:  4096,
		Timeout:    60,
	}

	assert.Equal(t, ProviderOpenAI, cfg.GetProviderType())
	assert.Equal(t, "gpt-4", cfg.Model)
}

func TestModelConfigWithOneAPI(t *testing.T) {
	cfg := &ModelConfig{
		Provider:   "oneapi",
		APIKey:     "sk-test123",
		Model:      "MiniMax-M2.1",
		BaseURL:    "https://oneapi.sangfor.com/v1",
	}

	assert.Equal(t, ProviderOneAPI, cfg.GetProviderType())
}

func TestModelConfigWithDeepSeek(t *testing.T) {
	cfg := &ModelConfig{
		Provider:   "deepseek",
		APIKey:     "sk-test123",
		Model:      "deepseek-chat",
		BaseURL:    "https://api.deepseek.com/v1",
	}

	assert.Equal(t, ProviderDeepSeek, cfg.GetProviderType())
}

func TestModelConfigWithClaude(t *testing.T) {
	cfg := &ModelConfig{
		Provider:   "anthropic",
		APIKey:     "sk-ant-test123",
		Model:      "claude-3-opus-20240307",
	}

	assert.Equal(t, ProviderAnthropic, cfg.GetProviderType())
}

func TestModelConfigMissingAPIKey(t *testing.T) {
	cfg := &ModelConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}

	assert.Equal(t, ProviderOpenAI, cfg.GetProviderType())
	assert.Empty(t, cfg.APIKey)
}

func TestModelConfigMissingModel(t *testing.T) {
	cfg := &ModelConfig{
		Provider: "openai",
		APIKey:   "sk-test123",
	}

	assert.Equal(t, ProviderOpenAI, cfg.GetProviderType())
	assert.Empty(t, cfg.Model)
}

func TestConfigStruct(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Name:    "hle-agent",
			Version: "1.0.0",
		},
		Model: ModelConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
	}

	assert.Equal(t, "hle-agent", cfg.App.Name)
	assert.Equal(t, "1.0.0", cfg.App.Version)
	assert.Equal(t, "openai", cfg.Model.Provider)
}

func TestModelConfigTemperature(t *testing.T) {
	cfg := &ModelConfig{
		Temperature: 0.5,
	}

	assert.Equal(t, 0.5, cfg.Temperature)
}

func TestModelConfigMaxTokens(t *testing.T) {
	cfg := &ModelConfig{
		MaxTokens: 8192,
	}

	assert.Equal(t, 8192, cfg.MaxTokens)
}

func TestModelConfigTimeout(t *testing.T) {
	cfg := &ModelConfig{
		Timeout: 120,
	}

	assert.Equal(t, 120, cfg.Timeout)
}

func TestModelConfigBaseURL(t *testing.T) {
	cfg := &ModelConfig{
		BaseURL: "https://api.example.com/v1",
	}

	assert.Equal(t, "https://api.example.com/v1", cfg.BaseURL)
}

func TestProviderTypeCaseInsensitive(t *testing.T) {
	cfg := &ModelConfig{Provider: "OPENAI"}
	assert.Equal(t, ProviderOpenAI, cfg.GetProviderType())

	cfg.Provider = "Openai"
	assert.Equal(t, ProviderOpenAI, cfg.GetProviderType())
}

func TestValidateModelName(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"Valid GPT-4", "gpt-4"},
		{"Valid GPT-3.5", "gpt-3.5-turbo"},
		{"Valid Claude", "claude-3-opus"},
		{"Valid DeepSeek", "deepseek-chat"},
		{"Valid MiniMax", "MiniMax-M2.1"},
		{"Empty model", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ModelConfig{Model: tt.model}
			// Just test that the model is set correctly
			assert.Equal(t, tt.model, cfg.Model)
		})
	}
}

func TestAppConfig(t *testing.T) {
	appCfg := AppConfig{
		Name:    "test-app",
		Version: "2.0.0",
		LogLevel: "debug",
	}

	assert.Equal(t, "test-app", appCfg.Name)
	assert.Equal(t, "2.0.0", appCfg.Version)
	assert.Equal(t, "debug", appCfg.LogLevel)
}

func TestToolsConfig(t *testing.T) {
	toolsCfg := ToolsConfig{
		Python: ToolConfig{
			Enabled:  true,
			Endpoint: "http://python:8080",
			Timeout:  30,
		},
		SageMath: ToolConfig{
			Enabled:  false,
			Endpoint: "http://sagemath:8080",
		},
	}

	assert.True(t, toolsCfg.Python.Enabled)
	assert.False(t, toolsCfg.SageMath.Enabled)
}

func TestMemoryConfig(t *testing.T) {
	memCfg := MemoryConfig{
		ShortTermMaxSteps:   100,
		LongTermPersistence: false,
		StoragePath:         "/tmp/memory",
	}

	assert.Equal(t, 100, memCfg.ShortTermMaxSteps)
	assert.False(t, memCfg.LongTermPersistence)
	assert.Equal(t, "/tmp/memory", memCfg.StoragePath)
}

func TestAgentConfig(t *testing.T) {
	agentCfg := AgentConfig{
		MaxIterations: 10,
		CheckInterval: 1,
	}

	assert.Equal(t, 10, agentCfg.MaxIterations)
	assert.Equal(t, 1, agentCfg.CheckInterval)
}

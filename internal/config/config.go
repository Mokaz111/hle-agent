package config

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	App     AppConfig     `yaml:"app"`
	Model   ModelConfig   `yaml:"model"`
	Agent   AgentConfig   `yaml:"agent"`
	Tools   ToolsConfig   `yaml:"tools"`
	Memory  MemoryConfig  `yaml:"memory"`
	Output  OutputConfig  `yaml:"output"`
	Audit   AuditConfig   `yaml:"audit"`
}

// AppConfig represents application settings
type AppConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	LogLevel string `yaml:"log_level"`
}

// ProviderType represents the LLM provider type
type ProviderType string

const (
	ProviderOpenAI  ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderDeepSeek ProviderType = "deepseek"
	ProviderOneAPI  ProviderType = "oneapi"
	ProviderCustom  ProviderType = "custom"
)

// SupportedProviders contains all supported provider types
var SupportedProviders = []ProviderType{
	ProviderOpenAI,
	ProviderAnthropic,
	ProviderDeepSeek,
	ProviderOneAPI,
	ProviderCustom,
}

// IsValid checks if the provider type is valid
func (p ProviderType) IsValid() bool {
	for _, supported := range SupportedProviders {
		if p == supported {
			return true
		}
	}
	return false
}

// String returns the string representation of the provider type
func (p ProviderType) String() string {
	return string(p)
}

// ModelConfig represents LLM model settings
type ModelConfig struct {
	Provider   string  `yaml:"provider"`
	APIKey     string  `yaml:"api_key"`
	Model      string  `yaml:"model"`
	BaseURL    string  `yaml:"base_url"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens  int     `yaml:"max_tokens"`
	Timeout    int     `yaml:"timeout"`
}

// GetProviderType returns the ProviderType from the string
func (c *ModelConfig) GetProviderType() ProviderType {
	provider := ProviderType(strings.ToLower(c.Provider))
	if provider.IsValid() {
		return provider
	}
	return ProviderCustom
}

// AgentConfig represents agent settings
type AgentConfig struct {
	Type          string `yaml:"type"`
	MaxIterations int    `yaml:"max_iterations"`
	CheckInterval int    `yaml:"check_interval"`
}

// ToolsConfig represents tool settings
type ToolsConfig struct {
	Python   ToolConfig `yaml:"python"`
	SageMath ToolConfig `yaml:"sagemath"`
	Retriever ToolConfig `yaml:"retriever"`
}

// ToolConfig represents a single tool configuration
type ToolConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Endpoint      string `yaml:"endpoint"`
	Timeout       int    `yaml:"timeout"`
	SandboxEnabled bool  `yaml:"sandbox_enabled,omitempty"`
	DockerImage   string `yaml:"docker_image,omitempty"`
	Type          string `yaml:"type,omitempty"`
}

// MemoryConfig represents memory settings
type MemoryConfig struct {
	ShortTermMaxSteps   int    `yaml:"short_term_max_steps"`
	LongTermPersistence bool   `yaml:"long_term_persistence"`
	StoragePath         string `yaml:"storage_path"`
}

// OutputConfig represents output settings
type OutputConfig struct {
	Format      string `yaml:"format"`
	DetailLevel string `yaml:"detail_level"`
	ResultDir   string `yaml:"result_dir"`
}

// AuditConfig represents audit settings
type AuditConfig struct {
	Enabled       bool   `yaml:"enabled"`
	LogFile       string `yaml:"log_file"`
	RequireApproval bool `yaml:"require_approval"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Expand environment variables
	expanded := os.ExpandEnv(string(data))
	
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Post-process configuration
	cfg.postProcess()
	
	return &cfg, nil
}

// postProcess performs post-processing on the configuration
func (c *Config) postProcess() {
	// Expand environment variables in API key
	c.Model.APIKey = os.ExpandEnv(c.Model.APIKey)
	
	// Set defaults if not specified
	if c.Agent.MaxIterations == 0 {
		c.Agent.MaxIterations = 10
	}
	
	if c.Agent.CheckInterval == 0 {
		c.Agent.CheckInterval = 1
	}
	
	if c.Memory.ShortTermMaxSteps == 0 {
		c.Memory.ShortTermMaxSteps = 100
	}
	
	if c.App.LogLevel == "" {
		c.App.LogLevel = "info"
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate model configuration
	if c.Model.Model == "" {
		return fmt.Errorf("model name is required")
	}

	// Validate provider type
	providerType := c.Model.GetProviderType()
	if providerType == ProviderCustom {
		// For custom providers, just warn but don't fail
		c.logger().Warn("使用自定义 LLM 提供商，请确保 API 兼容 OpenAI 格式")
	}

	if c.Model.BaseURL == "" {
		return fmt.Errorf("model base URL is required")
	}

	// Validate API key (skip for localhost/testing)
	if c.Model.APIKey == "" && !strings.Contains(c.Model.BaseURL, "localhost") &&
	   !strings.Contains(c.Model.BaseURL, "127.0.0.1") {
		return fmt.Errorf("API key is required for non-localhost endpoints")
	}

	// Validate common model names
	if err := c.validateModelName(providerType); err != nil {
		return err
	}

	// Validate agent configuration
	if c.Agent.MaxIterations <= 0 {
		return fmt.Errorf("max_iterations must be positive")
	}

	// Validate output configuration
	if c.Output.Format == "" {
		return fmt.Errorf("output format is required")
	}

	return nil
}

// validateModelName validates the model name for the given provider
func (c *Config) validateModelName(provider ProviderType) error {
	model := c.Model.Model

	// Common model name patterns
	commonModels := map[ProviderType][]string{
		ProviderOpenAI:   {"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "gpt-3.5"},
		ProviderAnthropic: {"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
		ProviderDeepSeek: {"deepseek-chat", "deepseek-coder"},
		ProviderOneAPI:   {"MiniMax-M2.1", "MiniMax-M2", "qwen-turbo", "qwen-plus", "glm-4"},
	}

	// Check if model matches known patterns
	if patterns, ok := commonModels[provider]; ok {
		for _, pattern := range patterns {
			if strings.Contains(model, pattern) || model == pattern {
				return nil // Known model, validation passed
			}
		}
	}

	// For unknown models, just warn
	c.logger().Warn("未识别的模型名称，请确保模型名称正确",
		zap.String("provider", provider.String()),
		zap.String("model", model))

	return nil
}

// logger returns a zap logger for config package
func (c *Config) logger() *zap.Logger {
	// Simple logger for config validation
	return zap.NewNop()
}

// GetModelConfig returns the model configuration
func (c *Config) GetModelConfig() *ModelConfig {
	return &c.Model
}

// GetAgentConfig returns the agent configuration
func (c *Config) GetAgentConfig() *AgentConfig {
	return &c.Agent
}

// GetToolsConfig returns the tools configuration
func (c *Config) GetToolsConfig() *ToolsConfig {
	return &c.Tools
}

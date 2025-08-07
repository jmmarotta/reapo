package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// ConfigFile represents the JSON configuration file structure
type ConfigFile struct {
	Model string `json:"model,omitempty"`
}

// Config holds the application configuration
type Config struct {
	Model         anthropic.Model
	ModelName     string
	MaxTokens     int
	ContextTokens int
}

// Global config instance
var AppConfig *Config

// ModelMapping maps user-friendly names to Anthropic model constants
var ModelMapping = map[string]anthropic.Model{
	"claude-sonnet-4":  anthropic.ModelClaude4Sonnet20250514,
	"claude-opus-4":    anthropic.ModelClaude4Opus20250514,
	"claude-opus-4-1":  anthropic.ModelClaudeOpus4_1_20250805,
	"claude-3-5-haiku": anthropic.ModelClaude3_5Haiku20241022,
}

// ModelContextLimits defines the context window size for each model (in tokens)
var ModelContextLimits = map[string]int{
	"claude-sonnet-4":  200000,
	"claude-opus-4":    200000,
	"claude-opus-4-1":  200000,
	"claude-3-5-haiku": 200000,
}

// ModelMaxTokens defines the maximum output tokens for each model
var ModelMaxTokens = map[string]int{
	"claude-sonnet-4":  64000,
	"claude-opus-4":    32000,
	"claude-opus-4-1":  32000,
	"claude-3-5-haiku": 8192,
}

// Init initializes the configuration
func Init() {
	// Default configuration
	AppConfig = &Config{
		Model:         anthropic.ModelClaude4Sonnet20250514,
		ModelName:     "claude-sonnet-4",
		MaxTokens:     64000,
		ContextTokens: 200000,
	}

	// Try to load from config file
	modelName := loadConfigFile()

	// Check environment variable if no config file setting
	if modelName == "" {
		modelName = os.Getenv("REAPO_MODEL")
	}

	// If a model is specified, update the config
	if modelName != "" {
		modelName = strings.ToLower(strings.TrimSpace(modelName))
		if model, ok := ModelMapping[modelName]; ok {
			AppConfig.Model = model
			AppConfig.ModelName = modelName
			if contextLimit, ok := ModelContextLimits[modelName]; ok {
				AppConfig.ContextTokens = contextLimit
			}
			if maxTokens, ok := ModelMaxTokens[modelName]; ok {
				AppConfig.MaxTokens = maxTokens
			}
		}
		// If model name not found in mapping, keep default
	}
}

// loadConfigFile loads the configuration from ~/.config/reapo/config.json
func loadConfigFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	configPath := filepath.Join(homeDir, ".config", "reapo", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var configFile ConfigFile
	if err := json.Unmarshal(data, &configFile); err != nil {
		return ""
	}

	return configFile.Model
}

// GetModel returns the configured model
func GetModel() anthropic.Model {
	if AppConfig == nil {
		Init()
	}
	return AppConfig.Model
}

// GetModelName returns the user-friendly model name
func GetModelName() string {
	if AppConfig == nil {
		Init()
	}
	return AppConfig.ModelName
}

// GetContextTokens returns the context window size for the current model
func GetContextTokens() int {
	if AppConfig == nil {
		Init()
	}
	return AppConfig.ContextTokens
}

// GetMaxTokens returns the max tokens for responses
func GetMaxTokens() int {
	if AppConfig == nil {
		Init()
	}
	return AppConfig.MaxTokens
}

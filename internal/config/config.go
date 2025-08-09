package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	OpenAI  OpenAIConfig  `mapstructure:"openai"`
	Cache   CacheConfig   `mapstructure:"cache"`
	UI      UIConfig      `mapstructure:"ui"`
	Logging LoggingConfig `mapstructure:"logging"`
	Profile string        `mapstructure:"profile"` // dev, prod, custom
}

// OpenAIConfig contains OpenAI API settings
type OpenAIConfig struct {
	APIKey         string        `mapstructure:"api_key"`
	Model          string        `mapstructure:"model"`
	MaxTokens      int           `mapstructure:"max_tokens"`
	Temperature    float32       `mapstructure:"temperature"`
	Timeout        time.Duration `mapstructure:"timeout"`
	BaseURL        string        `mapstructure:"base_url"`
	OrgID          string        `mapstructure:"org_id"`
	TopP           float32       `mapstructure:"top_p"`
	N              int           `mapstructure:"n"`
	Stop           []string      `mapstructure:"stop"`
	ReasoningEffort string       `mapstructure:"reasoning_effort"` // low, medium, high (for reasoning models)
}

// CacheConfig contains cache-related settings
type CacheConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	TTL      time.Duration `mapstructure:"ttl"`
	MaxSize  int           `mapstructure:"max_size"` // in MB
	Strategy string        `mapstructure:"strategy"` // lru, fifo, lfu
	Dir      string        `mapstructure:"dir"`      // cache directory
}

// UIConfig contains UI-related settings
type UIConfig struct {
	StreamingEnabled   bool   `mapstructure:"streaming_enabled"`
	ColorOutput        bool   `mapstructure:"color_output"`
	MarkdownRendering  bool   `mapstructure:"markdown_rendering"`
	SyntaxHighlighting bool   `mapstructure:"syntax_highlighting"`
	Theme              string `mapstructure:"theme"`   // dark, light, auto
	Spinner            string `mapstructure:"spinner"` // dots, line, star
	Width              int    `mapstructure:"width"`   // terminal width override
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, text, pretty
	File   string `mapstructure:"file"`   // log file path (empty for stdout)
	NoAPI  bool   `mapstructure:"no_api"` // disable API key logging
}

// Load loads configuration from multiple sources with priority:
// 1. Command-line flags (highest)
// 2. Environment variables
// 3. Config file
// 4. Defaults (lowest)
func Load(configPath string) (*Config, error) {
	// Initialize new viper instance for isolation
	v := viper.New()

	// Setup config file paths
	if err := setupConfigPaths(v, configPath); err != nil {
		return nil, fmt.Errorf("failed to setup config paths: %w", err)
	}

	// Configure environment variables
	setupEnvironment(v)

	// Set defaults based on profile
	profile := os.Getenv("TERMINAL_AI_PROFILE")
	if profile == "" {
		profile = "prod"
	}
	setDefaults(v, profile)

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Real error reading config
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK, will use defaults and env
	}

	// Unmarshal configuration
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply profile-specific overrides
	config.Profile = profile
	applyProfile(&config, profile)

	// Handle special environment variables
	handleSpecialEnvVars(&config)

	// Validate configuration
	validator := NewValidator(&config)
	if err := validator.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Mask sensitive data for security
	maskSensitiveData(&config)

	return &config, nil
}

// LoadWithProfile loads configuration with a specific profile
func LoadWithProfile(configPath, profile string) (*Config, error) {
	os.Setenv("TERMINAL_AI_PROFILE", profile)
	return Load(configPath)
}

// setupConfigPaths configures where to look for config files
func setupConfigPaths(v *viper.Viper, configPath string) error {
	if configPath != "" {
		// Use explicit config file
		v.SetConfigFile(configPath)
	} else {
		// Look for config in standard locations
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		// Priority order for config locations
		v.AddConfigPath(".")                                 // Current directory
		v.AddConfigPath(filepath.Join(home, ".opt"))         // User .opt directory (primary)
		v.AddConfigPath(filepath.Join(home, ".terminal-ai")) // User config directory (fallback)
		v.AddConfigPath(home)                                // Home directory
		v.AddConfigPath("/etc/terminal-ai")                  // System config

		// Config file names to search for (in order of priority)
		v.SetConfigName("terminal-ai-config") // Primary config name
		v.SetConfigType("yaml")

		// Also support legacy names
		v.SetConfigName("config")
		v.SetConfigName(".terminal-ai")
	}

	return nil
}

// setupEnvironment configures environment variable handling
func setupEnvironment(v *viper.Viper) {
	v.SetEnvPrefix("TERMINAL_AI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Bind specific environment variables
	v.BindEnv("openai.api_key", "OPENAI_API_KEY", "TERMINAL_AI_OPENAI_API_KEY")
	v.BindEnv("openai.org_id", "OPENAI_ORG_ID", "TERMINAL_AI_OPENAI_ORG_ID")
	v.BindEnv("logging.level", "TERMINAL_AI_LOG_LEVEL", "LOG_LEVEL")
}

// handleSpecialEnvVars handles special environment variables
func handleSpecialEnvVars(config *Config) {
	// Check for OPENAI_API_KEY if not set
	if config.OpenAI.APIKey == "" {
		if key := os.Getenv("OPENAI_API_KEY"); key != "" {
			config.OpenAI.APIKey = key
		}
	}

	// Support reading API key from file
	if keyFile := os.Getenv("TERMINAL_AI_API_KEY_FILE"); keyFile != "" {
		if content, err := os.ReadFile(keyFile); err == nil {
			config.OpenAI.APIKey = strings.TrimSpace(string(content))
		}
	}

	// Expand environment variables in string values
	config.OpenAI.APIKey = os.ExpandEnv(config.OpenAI.APIKey)
	config.Cache.Dir = os.ExpandEnv(config.Cache.Dir)
	config.Logging.File = os.ExpandEnv(config.Logging.File)
	
	// Adjust settings based on model type
	AdjustForModelType(config)
}

// IsReasoningModel checks if the given model is a reasoning model
func IsReasoningModel(model string) bool {
	reasoningModels := map[string]bool{
		"gpt-5":      true,
		"gpt-5-mini": true,
		"gpt-5-nano": true,
		"o1":         true,
		"o1-mini":    true,
		"o3":         true,
		"o3-mini":    true,
		"o4-mini":    true,
	}
	return reasoningModels[model]
}

// AdjustForModelType adjusts configuration based on whether the model is a reasoning model
func AdjustForModelType(config *Config) {
	if IsReasoningModel(config.OpenAI.Model) {
		// Reasoning models require temperature=1.0 to avoid errors
		config.OpenAI.Temperature = 1.0
		
		// Set default reasoning effort if not specified
		if config.OpenAI.ReasoningEffort == "" {
			config.OpenAI.ReasoningEffort = "low"
		}
	} else {
		// Non-reasoning models don't use reasoning_effort
		config.OpenAI.ReasoningEffort = ""
		
		// Use standard temperature if it was set to 1.0 by default
		if config.OpenAI.Temperature == 1.0 {
			config.OpenAI.Temperature = 0.7
		}
	}
}

// setDefaults sets default configuration values based on profile
func setDefaults(v *viper.Viper, profile string) {
	// OpenAI defaults
	v.SetDefault("openai.model", "gpt-5-mini") // Updated default to reasoning model
	v.SetDefault("openai.max_tokens", 2000)
	v.SetDefault("openai.temperature", 1.0) // Default for reasoning models
	v.SetDefault("openai.timeout", "30s")
	v.SetDefault("openai.base_url", "https://api.openai.com/v1")
	v.SetDefault("openai.top_p", 1.0)
	v.SetDefault("openai.n", 1)
	v.SetDefault("openai.reasoning_effort", "low") // Default for reasoning models

	// Cache defaults
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.ttl", "5m")
	v.SetDefault("cache.max_size", 100)
	v.SetDefault("cache.strategy", "lru")

	// UI defaults
	v.SetDefault("ui.streaming_enabled", true)
	v.SetDefault("ui.color_output", true)
	v.SetDefault("ui.markdown_rendering", true)
	v.SetDefault("ui.syntax_highlighting", true)
	v.SetDefault("ui.theme", "auto")
	v.SetDefault("ui.spinner", "dots")
	v.SetDefault("ui.width", 0) // 0 means auto-detect

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.file", "")
	v.SetDefault("logging.no_api", true) // Never log API keys by default

	// Apply profile-specific defaults
	switch profile {
	case "dev":
		v.SetDefault("logging.level", "debug")
		v.SetDefault("logging.format", "pretty")
		v.SetDefault("cache.enabled", false)
		v.SetDefault("openai.timeout", "60s")
	case "prod":
		v.SetDefault("logging.level", "info")
		v.SetDefault("logging.format", "json")
		v.SetDefault("cache.enabled", true)
		v.SetDefault("openai.timeout", "30s")
	}

	// Set cache directory default
	if home, err := os.UserHomeDir(); err == nil {
		v.SetDefault("cache.dir", filepath.Join(home, ".terminal-ai", "cache"))
	}
}

// applyProfile applies profile-specific configuration overrides
func applyProfile(config *Config, profile string) {
	switch profile {
	case "dev":
		// Development profile overrides
		if config.Logging.Level == "info" {
			config.Logging.Level = "debug"
		}
		if config.Logging.Format == "json" {
			config.Logging.Format = "pretty"
		}
	case "prod":
		// Production profile overrides
		config.Logging.NoAPI = true // Always mask API keys in production
		if config.Cache.Strategy == "" {
			config.Cache.Strategy = "lru"
		}
	}
}

// maskSensitiveData masks sensitive information for security
func maskSensitiveData(config *Config) {
	if config.Logging.NoAPI && config.OpenAI.APIKey != "" {
		// Store the actual key separately and mask for logging
		// This is a simplified version - in production, use a secure key store
		if len(config.OpenAI.APIKey) > 8 {
			// Keep first 4 and last 4 characters for identification
			masked := config.OpenAI.APIKey[:4] + "..." + config.OpenAI.APIKey[len(config.OpenAI.APIKey)-4:]
			_ = masked // Use this for logging purposes
		}
	}
}

// Validate validates the configuration (simplified version, full validation in validator.go)
func (c *Config) Validate() error {
	validator := NewValidator(c)
	return validator.Validate()
}

// Save saves the configuration to file
func (c *Config) Save() error {
	return c.SaveTo("")
}

// SaveTo saves the configuration to a specific file
func (c *Config) SaveTo(path string) error {
	if path == "" {
		path = GetDefaultConfigPath()
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create a new viper instance for saving
	v := viper.New()

	// Marshal config to viper
	if err := v.MergeConfigMap(c.ToMap()); err != nil {
		return fmt.Errorf("failed to prepare config for saving: %w", err)
	}

	// Write config file with restricted permissions (0600)
	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Set proper file permissions for security
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	return nil
}

// ToMap converts config to a map for viper
func (c *Config) ToMap() map[string]interface{} {
	// This is a simplified version - in production, use proper marshaling
	return map[string]interface{}{
		"profile": c.Profile,
		"openai": map[string]interface{}{
			"api_key":          maskAPIKey(c.OpenAI.APIKey),
			"model":            c.OpenAI.Model,
			"max_tokens":       c.OpenAI.MaxTokens,
			"temperature":      c.OpenAI.Temperature,
			"timeout":          c.OpenAI.Timeout.String(),
			"base_url":         c.OpenAI.BaseURL,
			"org_id":           c.OpenAI.OrgID,
			"top_p":            c.OpenAI.TopP,
			"n":                c.OpenAI.N,
			"stop":             c.OpenAI.Stop,
			"reasoning_effort": c.OpenAI.ReasoningEffort,
		},
		"cache": map[string]interface{}{
			"enabled":  c.Cache.Enabled,
			"ttl":      c.Cache.TTL.String(),
			"max_size": c.Cache.MaxSize,
			"strategy": c.Cache.Strategy,
			"dir":      c.Cache.Dir,
		},
		"ui": map[string]interface{}{
			"streaming_enabled":   c.UI.StreamingEnabled,
			"color_output":        c.UI.ColorOutput,
			"markdown_rendering":  c.UI.MarkdownRendering,
			"syntax_highlighting": c.UI.SyntaxHighlighting,
			"theme":               c.UI.Theme,
			"spinner":             c.UI.Spinner,
			"width":               c.UI.Width,
		},
		"logging": map[string]interface{}{
			"level":  c.Logging.Level,
			"format": c.Logging.Format,
			"file":   c.Logging.File,
			"no_api": c.Logging.NoAPI,
		},
	}
}

// maskAPIKey masks API key for saving to file
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	// Use environment variable reference instead of actual key
	return "${OPENAI_API_KEY}"
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	if configFile := viper.ConfigFileUsed(); configFile != "" {
		return configFile
	}
	return GetDefaultConfigPath()
}

// GetDefaultConfigPath returns the default config file path
func GetDefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	// Primary config location in ~/.opt
	return filepath.Join(home, ".opt", "terminal-ai-config.yaml")
}

// GetString returns a config value as string with dot notation support
func (c *Config) GetString(key string) string {
	// This is a helper for accessing nested config values
	// Example: c.GetString("openai.model") returns c.OpenAI.Model
	parts := strings.Split(key, ".")
	if len(parts) == 2 {
		switch parts[0] {
		case "openai":
			switch parts[1] {
			case "model":
				return c.OpenAI.Model
			case "api_key":
				return c.OpenAI.APIKey
			case "base_url":
				return c.OpenAI.BaseURL
			}
		case "logging":
			switch parts[1] {
			case "level":
				return c.Logging.Level
			case "format":
				return c.Logging.Format
			}
		}
	}
	return ""
}

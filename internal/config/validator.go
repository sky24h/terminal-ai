package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Validator provides configuration validation
type Validator struct {
	config *Config
	errors []string
}

// NewValidator creates a new configuration validator
func NewValidator(config *Config) *Validator {
	return &Validator{
		config: config,
		errors: []string{},
	}
}

// Validate performs full configuration validation
func (v *Validator) Validate() error {
	v.errors = []string{}

	// Validate all sections
	v.validateOpenAI()
	v.validateCache()
	v.validateUI()
	v.validateLogging()

	if len(v.errors) > 0 {
		return errors.New(strings.Join(v.errors, "; "))
	}

	return nil
}

// validateOpenAI validates OpenAI configuration
func (v *Validator) validateOpenAI() {
	// API Key validation
	if v.config.OpenAI.APIKey == "" {
		v.errors = append(v.errors, "OpenAI API key is required (set OPENAI_API_KEY or configure in file)")
		return // Skip other validations if no API key
	}

	// Validate API key format (basic check)
	if !v.isValidAPIKey(v.config.OpenAI.APIKey) {
		v.errors = append(v.errors, "invalid OpenAI API key format")
	}

	// Base URL validation
	if v.config.OpenAI.BaseURL != "" {
		if _, err := url.Parse(v.config.OpenAI.BaseURL); err != nil {
			v.errors = append(v.errors, fmt.Sprintf("invalid OpenAI base URL: %v", err))
		}
	}

	// Model validation
	if !v.isValidModel(v.config.OpenAI.Model) {
		v.errors = append(v.errors, fmt.Sprintf("unsupported model: %s", v.config.OpenAI.Model))
	}

	// Temperature validation - reasoning models require 1.0
	if IsReasoningModel(v.config.OpenAI.Model) {
		if v.config.OpenAI.Temperature != 1.0 {
			v.errors = append(v.errors, "reasoning models require temperature=1.0")
		}
	} else {
		// Non-reasoning models: 0.0 to 2.0
		if v.config.OpenAI.Temperature < 0 || v.config.OpenAI.Temperature > 2 {
			v.errors = append(v.errors, "temperature must be between 0 and 2")
		}
	}
	
	// Reasoning effort validation (only for reasoning models)
	if IsReasoningModel(v.config.OpenAI.Model) {
		if v.config.OpenAI.ReasoningEffort != "" && !v.isValidReasoningEffort(v.config.OpenAI.ReasoningEffort) {
			v.errors = append(v.errors, fmt.Sprintf("invalid reasoning_effort: %s (must be low, medium, or high)", v.config.OpenAI.ReasoningEffort))
		}
	}

	// TopP validation (0.0 to 1.0)
	if v.config.OpenAI.TopP < 0 || v.config.OpenAI.TopP > 1 {
		v.errors = append(v.errors, "top_p must be between 0 and 1")
	}

	// MaxTokens validation
	if v.config.OpenAI.MaxTokens < 1 {
		v.errors = append(v.errors, "max_tokens must be at least 1")
	}

	// Model-specific token limits
	maxTokensLimit := v.getMaxTokensForModel(v.config.OpenAI.Model)
	if v.config.OpenAI.MaxTokens > maxTokensLimit {
		v.errors = append(v.errors, fmt.Sprintf("max_tokens exceeds model limit of %d", maxTokensLimit))
	}

	// N validation (number of completions)
	if v.config.OpenAI.N < 1 || v.config.OpenAI.N > 10 {
		v.errors = append(v.errors, "n (number of completions) must be between 1 and 10")
	}

	// Timeout validation
	if v.config.OpenAI.Timeout <= 0 {
		v.errors = append(v.errors, "timeout must be positive")
	} else if v.config.OpenAI.Timeout < 5*time.Second {
		v.errors = append(v.errors, "timeout should be at least 5 seconds")
	} else if v.config.OpenAI.Timeout > 5*time.Minute {
		v.errors = append(v.errors, "timeout should not exceed 5 minutes")
	}

	// Organization ID validation (optional but check format if provided)
	if v.config.OpenAI.OrgID != "" && !v.isValidOrgID(v.config.OpenAI.OrgID) {
		v.errors = append(v.errors, "invalid organization ID format")
	}
}

// validateCache validates cache configuration
func (v *Validator) validateCache() {
	// TTL validation
	if v.config.Cache.TTL < 0 {
		v.errors = append(v.errors, "cache TTL must be non-negative")
	}

	// Max size validation
	if v.config.Cache.MaxSize < 0 {
		v.errors = append(v.errors, "cache max size must be non-negative")
	} else if v.config.Cache.MaxSize > 10240 { // 10GB limit
		v.errors = append(v.errors, "cache max size should not exceed 10GB")
	}

	// Strategy validation
	validStrategies := []string{"lru", "lfu", "fifo", ""}
	if !v.contains(validStrategies, v.config.Cache.Strategy) {
		v.errors = append(v.errors, fmt.Sprintf("invalid cache strategy: %s (must be lru, lfu, or fifo)", v.config.Cache.Strategy))
	}

	// Cache directory validation
	if v.config.Cache.Dir != "" {
		// Expand environment variables
		cacheDir := os.ExpandEnv(v.config.Cache.Dir)

		// Check write permissions if directory exists
		if info, err := os.Stat(cacheDir); err == nil {
			if !info.IsDir() {
				v.errors = append(v.errors, fmt.Sprintf("cache path exists but is not a directory: %s", cacheDir))
			}
			// Try to create a test file to check write permissions
			testFile := filepath.Join(cacheDir, ".write_test")
			if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
				v.errors = append(v.errors, fmt.Sprintf("cache directory is not writable: %s", cacheDir))
			} else {
				os.Remove(testFile)
			}
		} else if !os.IsNotExist(err) {
			// Some other error accessing the directory
			v.errors = append(v.errors, fmt.Sprintf("error accessing cache directory: %v", err))
		}
		// If directory doesn't exist, that's OK - it will be created on first use
	}
}

// validateUI validates UI configuration
func (v *Validator) validateUI() {
	// Theme validation
	validThemes := []string{"dark", "light", "auto"}
	if !v.contains(validThemes, v.config.UI.Theme) {
		v.errors = append(v.errors, fmt.Sprintf("invalid UI theme: %s (must be dark, light, or auto)", v.config.UI.Theme))
	}

	// Spinner validation
	validSpinners := []string{"dots", "line", "star", "arrow", ""}
	if !v.contains(validSpinners, v.config.UI.Spinner) {
		v.errors = append(v.errors, fmt.Sprintf("invalid spinner type: %s", v.config.UI.Spinner))
	}

	// Terminal width validation
	if v.config.UI.Width < 0 {
		v.errors = append(v.errors, "terminal width must be non-negative (0 for auto-detect)")
	} else if v.config.UI.Width > 0 && v.config.UI.Width < 40 {
		v.errors = append(v.errors, "terminal width should be at least 40 characters")
	} else if v.config.UI.Width > 500 {
		v.errors = append(v.errors, "terminal width should not exceed 500 characters")
	}
}

// validateLogging validates logging configuration
func (v *Validator) validateLogging() {
	// Log level validation
	validLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
	if !v.contains(validLevels, v.config.Logging.Level) {
		v.errors = append(v.errors, fmt.Sprintf("invalid log level: %s", v.config.Logging.Level))
	}

	// Log format validation
	validFormats := []string{"json", "text", "pretty"}
	if !v.contains(validFormats, v.config.Logging.Format) {
		v.errors = append(v.errors, fmt.Sprintf("invalid log format: %s", v.config.Logging.Format))
	}

	// Log file validation
	if v.config.Logging.File != "" {
		logFile := os.ExpandEnv(v.config.Logging.File)
		logDir := filepath.Dir(logFile)

		// Check if directory exists or can be created
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(logDir, 0755); err != nil {
				v.errors = append(v.errors, fmt.Sprintf("cannot create log directory: %s", logDir))
			} else {
				// Clean up test directory
				os.Remove(logDir)
			}
		}

		// Check if file is writable (if it exists)
		if _, err := os.Stat(logFile); err == nil {
			// File exists, check if writable
			if file, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644); err != nil {
				v.errors = append(v.errors, fmt.Sprintf("log file is not writable: %s", logFile))
			} else {
				file.Close()
			}
		}
	}
}

// isValidAPIKey performs basic validation of API key format
func (v *Validator) isValidAPIKey(key string) bool {
	// Skip validation for environment variable references
	if strings.HasPrefix(key, "${") && strings.HasSuffix(key, "}") {
		return true
	}

	// OpenAI API keys typically start with "sk-" and are 51 characters long
	// But we'll be more flexible to support different providers
	if len(key) < 20 {
		return false
	}

	// Check for common patterns
	patterns := []string{
		`^sk-[a-zA-Z0-9]{48}$`,      // OpenAI format
		`^sk-proj-[a-zA-Z0-9]{48}$`, // OpenAI project keys
		`^[a-zA-Z0-9]{32,}$`,        // Generic API key
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, key); matched {
			return true
		}
	}

	// Allow any reasonably long string for custom endpoints
	return len(key) >= 20 && len(key) <= 200
}

// isValidModel checks if a model name is valid
func (v *Validator) isValidModel(model string) bool {
	// Allow fine-tuned models
	if strings.HasPrefix(model, "ft:") || strings.HasPrefix(model, "ft-") {
		return true
	}

	// List of valid OpenAI models (as of 2025)
	validModels := []string{
		// GPT-5 reasoning models
		"gpt-5", "gpt-5-mini", "gpt-5-nano",
		
		// O-series reasoning models
		"o1", "o1-mini", "o3", "o3-mini", "o4-mini",
		
		// GPT-4.1 non-reasoning models
		"gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano",
		
		// GPT-4 models
		"gpt-4", "gpt-4-0314", "gpt-4-0613", "gpt-4-32k", "gpt-4-32k-0314", "gpt-4-32k-0613",
		"gpt-4-turbo", "gpt-4-turbo-preview", "gpt-4-1106-preview", "gpt-4-vision-preview",
		"gpt-4o", "gpt-4o-mini",

		// GPT-3.5 models
		"gpt-3.5-turbo", "gpt-3.5-turbo-0301", "gpt-3.5-turbo-0613", "gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-16k", "gpt-3.5-turbo-16k-0613", "gpt-3.5-turbo-instruct",

		// Legacy models (might still be in use)
		"text-davinci-003", "text-davinci-002", "code-davinci-002",

		// Embedding models (in case they're used)
		"text-embedding-ada-002", "text-embedding-3-small", "text-embedding-3-large",
	}

	return v.contains(validModels, model)
}

// getMaxTokensForModel returns the maximum tokens for a given model
func (v *Validator) getMaxTokensForModel(model string) int {
	// Model-specific limits
	modelLimits := map[string]int{
		// GPT-5 reasoning models
		"gpt-5":               200000,
		"gpt-5-mini":          100000,
		"gpt-5-nano":          50000,
		
		// O-series reasoning models
		"o1":                  200000,
		"o1-mini":             100000,
		"o3":                  200000,
		"o3-mini":             100000,
		"o4-mini":             100000,
		
		// GPT-4.1 non-reasoning models
		"gpt-4.1":             128000,
		"gpt-4.1-mini":        64000,
		"gpt-4.1-nano":        32000,
		
		// GPT-4 models
		"gpt-4":               8192,
		"gpt-4-32k":           32768,
		"gpt-4-turbo":         128000,
		"gpt-4-turbo-preview": 128000,
		"gpt-4o":              128000,
		"gpt-4o-mini":         16384,
		"gpt-3.5-turbo":       4096,
		"gpt-3.5-turbo-16k":   16384,
	}

	// Check for exact match
	if limit, ok := modelLimits[model]; ok {
		return limit
	}

	// Check for prefix match (for versioned models)
	for prefix, limit := range modelLimits {
		if strings.HasPrefix(model, prefix) {
			return limit
		}
	}

	// Default limit for unknown models
	return 4096
}

// isValidOrgID validates OpenAI organization ID format
func (v *Validator) isValidOrgID(orgID string) bool {
	// OpenAI org IDs typically start with "org-" and have 24 characters after
	pattern := `^org-[a-zA-Z0-9]{24}$`
	matched, _ := regexp.MatchString(pattern, orgID)
	return matched
}

// isValidReasoningEffort validates the reasoning_effort parameter
func (v *Validator) isValidReasoningEffort(effort string) bool {
	validEfforts := []string{"low", "medium", "high"}
	return v.contains(validEfforts, effort)
}

// contains checks if a slice contains a value
func (v *Validator) contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// GetErrors returns validation errors
func (v *Validator) GetErrors() []string {
	return v.errors
}

// ValidateAPIKeyFile validates an API key file
func ValidateAPIKeyFile(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("API key file not found: %w", err)
	}

	// Check file permissions (should be readable only by owner)
	mode := info.Mode()
	if mode.Perm()&0077 != 0 {
		return fmt.Errorf("API key file has insecure permissions %v (should be 0600)", mode.Perm())
	}

	// Check file size (API keys shouldn't be huge)
	if info.Size() > 1024 {
		return fmt.Errorf("API key file is too large (>1KB)")
	}

	// Read and validate content
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read API key file: %w", err)
	}

	key := strings.TrimSpace(string(content))
	if key == "" {
		return fmt.Errorf("API key file is empty")
	}

	validator := &Validator{}
	if !validator.isValidAPIKey(key) {
		return fmt.Errorf("invalid API key format in file")
	}

	return nil
}

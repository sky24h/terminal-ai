package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Test with environment variables
	t.Run("LoadFromEnvironment", func(t *testing.T) {
		// Set test environment variables
		os.Setenv("OPENAI_API_KEY", "sk-test1234567890abcdefghijklmnopqrstuvwxyz12345678")
		os.Setenv("TERMINAL_AI_OPENAI_MODEL", "gpt-4")
		os.Setenv("TERMINAL_AI_OPENAI_MAX_TOKENS", "4000")
		os.Setenv("TERMINAL_AI_CACHE_ENABLED", "false")
		os.Setenv("TERMINAL_AI_UI_THEME", "dark")
		os.Setenv("TERMINAL_AI_LOGGING_LEVEL", "debug")
		defer func() {
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("TERMINAL_AI_OPENAI_MODEL")
			os.Unsetenv("TERMINAL_AI_OPENAI_MAX_TOKENS")
			os.Unsetenv("TERMINAL_AI_CACHE_ENABLED")
			os.Unsetenv("TERMINAL_AI_UI_THEME")
			os.Unsetenv("TERMINAL_AI_LOGGING_LEVEL")
		}()

		config, err := Load("")
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Verify values
		if config.OpenAI.APIKey == "" {
			t.Error("API key should be loaded from environment")
		}
		if config.OpenAI.Model != "gpt-4" {
			t.Errorf("Expected model gpt-4, got %s", config.OpenAI.Model)
		}
		if config.OpenAI.MaxTokens != 4000 {
			t.Errorf("Expected max_tokens 4000, got %d", config.OpenAI.MaxTokens)
		}
		if config.Cache.Enabled != false {
			t.Error("Cache should be disabled")
		}
		if config.UI.Theme != "dark" {
			t.Errorf("Expected theme dark, got %s", config.UI.Theme)
		}
		if config.Logging.Level != "debug" {
			t.Errorf("Expected log level debug, got %s", config.Logging.Level)
		}
	})

	// Test profile loading
	t.Run("LoadWithProfile", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "sk-test1234567890abcdefghijklmnopqrstuvwxyz12345678")
		defer os.Unsetenv("OPENAI_API_KEY")

		// Test dev profile
		config, err := LoadWithProfile("", "dev")
		if err != nil {
			t.Fatalf("Failed to load config with dev profile: %v", err)
		}
		if config.Profile != "dev" {
			t.Errorf("Expected profile dev, got %s", config.Profile)
		}
		if config.Logging.Level != "debug" {
			t.Errorf("Dev profile should set debug logging, got %s", config.Logging.Level)
		}

		// Test prod profile
		config, err = LoadWithProfile("", "prod")
		if err != nil {
			t.Fatalf("Failed to load config with prod profile: %v", err)
		}
		if config.Profile != "prod" {
			t.Errorf("Expected profile prod, got %s", config.Profile)
		}
		if config.Logging.NoAPI != true {
			t.Error("Prod profile should mask API keys")
		}
	})

	// Test default values
	t.Run("DefaultValues", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "sk-test1234567890abcdefghijklmnopqrstuvwxyz12345678")
		defer os.Unsetenv("OPENAI_API_KEY")

		config, err := Load("")
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Check defaults
		if config.OpenAI.Temperature != 0.7 {
			t.Errorf("Expected default temperature 0.7, got %f", config.OpenAI.Temperature)
		}
		if config.OpenAI.TopP != 1.0 {
			t.Errorf("Expected default top_p 1.0, got %f", config.OpenAI.TopP)
		}
		if config.OpenAI.N != 1 {
			t.Errorf("Expected default n 1, got %d", config.OpenAI.N)
		}
		if config.Cache.Strategy != "lru" {
			t.Errorf("Expected default cache strategy lru, got %s", config.Cache.Strategy)
		}
		if config.UI.StreamingEnabled != true {
			t.Error("Streaming should be enabled by default")
		}
		if config.UI.ColorOutput != true {
			t.Error("Color output should be enabled by default")
		}
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("MissingAPIKey", func(t *testing.T) {
		config := &Config{}
		validator := NewValidator(config)
		err := validator.Validate()
		if err == nil {
			t.Error("Should fail validation without API key")
		}
	})

	t.Run("InvalidTemperature", func(t *testing.T) {
		config := &Config{
			OpenAI: OpenAIConfig{
				APIKey:      "sk-test1234567890abcdefghijklmnopqrstuvwxyz12345678",
				Model:       "gpt-3.5-turbo",
				Temperature: 3.0, // Invalid: > 2.0
				MaxTokens:   100,
				Timeout:     30 * time.Second,
				TopP:        1.0,
				N:           1,
			},
		}
		validator := NewValidator(config)
		err := validator.Validate()
		if err == nil {
			t.Error("Should fail validation with temperature > 2.0")
		}
	})

	t.Run("InvalidModel", func(t *testing.T) {
		config := &Config{
			OpenAI: OpenAIConfig{
				APIKey:      "sk-test1234567890abcdefghijklmnopqrstuvwxyz12345678",
				Model:       "invalid-model",
				Temperature: 0.7,
				MaxTokens:   100,
				Timeout:     30 * time.Second,
				TopP:        1.0,
				N:           1,
			},
		}
		validator := NewValidator(config)
		err := validator.Validate()
		if err == nil {
			t.Error("Should fail validation with invalid model")
		}
	})

	t.Run("ValidConfiguration", func(t *testing.T) {
		config := &Config{
			OpenAI: OpenAIConfig{
				APIKey:      "sk-test1234567890abcdefghijklmnopqrstuvwxyz12345678",
				Model:       "gpt-3.5-turbo",
				Temperature: 0.7,
				MaxTokens:   2000,
				Timeout:     30 * time.Second,
				BaseURL:     "https://api.openai.com/v1",
				TopP:        1.0,
				N:           1,
			},
			Cache: CacheConfig{
				Enabled:  true,
				TTL:      5 * time.Minute,
				MaxSize:  100,
				Strategy: "lru",
			},
			UI: UIConfig{
				StreamingEnabled:   true,
				ColorOutput:        true,
				MarkdownRendering:  true,
				SyntaxHighlighting: true,
				Theme:              "auto",
				Spinner:            "dots",
			},
			Logging: LoggingConfig{
				Level:  "info",
				Format: "json",
				NoAPI:  true,
			},
		}
		validator := NewValidator(config)
		err := validator.Validate()
		if err != nil {
			t.Errorf("Valid configuration should pass validation: %v", err)
		}
	})

	t.Run("TokenLimits", func(t *testing.T) {
		config := &Config{
			OpenAI: OpenAIConfig{
				APIKey:      "sk-test1234567890abcdefghijklmnopqrstuvwxyz12345678",
				Model:       "gpt-3.5-turbo",
				MaxTokens:   10000, // Exceeds model limit
				Temperature: 0.7,
				Timeout:     30 * time.Second,
				TopP:        1.0,
				N:           1,
			},
		}
		validator := NewValidator(config)
		err := validator.Validate()
		if err == nil {
			t.Error("Should fail validation when max_tokens exceeds model limit")
		}
	})
}

func TestConfigSave(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	config := &Config{
		Profile: "test",
		OpenAI: OpenAIConfig{
			APIKey:      "sk-test-key",
			Model:       "gpt-3.5-turbo",
			MaxTokens:   2000,
			Temperature: 0.7,
			Timeout:     30 * time.Second,
			BaseURL:     "https://api.openai.com/v1",
			TopP:        1.0,
			N:           1,
		},
		Cache: CacheConfig{
			Enabled:  true,
			TTL:      5 * time.Minute,
			MaxSize:  100,
			Strategy: "lru",
			Dir:      filepath.Join(tempDir, "cache"),
		},
		UI: UIConfig{
			StreamingEnabled:  true,
			ColorOutput:       true,
			MarkdownRendering: true,
			Theme:             "dark",
			Spinner:           "dots",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			NoAPI:  true,
		},
	}

	// Save config
	err := config.SaveTo(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	mode := info.Mode()
	if mode.Perm() != 0600 {
		t.Errorf("Config file should have 0600 permissions, got %v", mode.Perm())
	}

	// Verify API key is masked in saved file
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	if string(content) == "" {
		t.Error("Config file should not be empty")
	}
	// API key should be masked as ${OPENAI_API_KEY}
	if !contains(string(content), "${OPENAI_API_KEY}") {
		t.Error("API key should be masked in saved config")
	}
}

func TestAPIKeyFileSecurity(t *testing.T) {
	// Create a temporary API key file
	tempDir := t.TempDir()
	keyFile := filepath.Join(tempDir, "api.key")

	// Write API key with secure permissions
	testKey := "sk-test1234567890abcdefghijklmnopqrstuvwxyz12345678"
	err := os.WriteFile(keyFile, []byte(testKey), 0600)
	if err != nil {
		t.Fatalf("Failed to write API key file: %v", err)
	}

	// Test validation
	err = ValidateAPIKeyFile(keyFile)
	if err != nil {
		t.Errorf("Valid API key file should pass validation: %v", err)
	}

	// Test with insecure permissions
	os.Chmod(keyFile, 0644)
	err = ValidateAPIKeyFile(keyFile)
	if err == nil {
		t.Error("Should fail validation with insecure permissions")
	}

	// Test with empty file
	emptyFile := filepath.Join(tempDir, "empty.key")
	os.WriteFile(emptyFile, []byte(""), 0600)
	err = ValidateAPIKeyFile(emptyFile)
	if err == nil {
		t.Error("Should fail validation with empty file")
	}

	// Test with non-existent file
	err = ValidateAPIKeyFile(filepath.Join(tempDir, "nonexistent.key"))
	if err == nil {
		t.Error("Should fail validation with non-existent file")
	}
}

func TestGetString(t *testing.T) {
	config := &Config{
		OpenAI: OpenAIConfig{
			Model:   "gpt-4",
			APIKey:  "test-key",
			BaseURL: "https://api.openai.com/v1",
		},
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"openai.model", "gpt-4"},
		{"openai.api_key", "test-key"},
		{"openai.base_url", "https://api.openai.com/v1"},
		{"logging.level", "debug"},
		{"logging.format", "json"},
		{"invalid.key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := config.GetString(tt.key)
			if result != tt.expected {
				t.Errorf("GetString(%s) = %s, want %s", tt.key, result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

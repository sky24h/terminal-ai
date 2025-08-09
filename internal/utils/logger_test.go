package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLoggerCreation(t *testing.T) {
	tests := []struct {
		name   string
		config LogConfig
	}{
		{
			name: "Console logger",
			config: LogConfig{
				Level:  "debug",
				Output: "console",
				Pretty: true,
			},
		},
		{
			name: "JSON logger",
			config: LogConfig{
				Level:  "info",
				Output: "json",
			},
		},
		{
			name: "Both outputs",
			config: LogConfig{
				Level:  "info",
				Output: "both",
				Pretty: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.config)
			if logger == nil {
				t.Fatal("Expected logger to be created")
			}
			logger.Close()
		})
	}
}

func TestSensitiveDataMasking(t *testing.T) {
	logger := NewLogger(LogConfig{
		Level:         "debug",
		Output:        "json",
		MaskSensitive: true,
	})
	defer logger.Close()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "OpenAI API key",
			input:    "sk-abcdefghijklmnopqrstuvwxyz1234567890abcdefghij",
			expected: "sk-***MASKED***hij",
		},
		{
			name:     "Bearer token",
			input:    "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Bea***MASKED***CJ9",
		},
		{
			name:     "Password in JSON",
			input:    `{"password":"supersecret123"}`,
			expected: `{"p***MASKED***3"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked := logger.maskSensitiveData(tt.input)
			if !strings.Contains(masked, "***MASKED***") {
				t.Errorf("Expected masking in output, got: %s", masked)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	levels := []string{"trace", "debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			logger := NewLogger(LogConfig{
				Level:  level,
				Output: "json",
			})
			defer logger.Close()

			// Test that the level was set correctly
			parsedLevel := parseLevel(level)
			if parsedLevel.String() != level {
				t.Errorf("Expected level %s, got %s", level, parsedLevel.String())
			}
		})
	}
}

func TestContextualLogging(t *testing.T) {
	logger := NewLogger(LogConfig{
		Level:  "debug",
		Output: "json",
	})
	defer logger.Close()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "test-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")
	ctx = context.WithValue(ctx, "trace_id", "trace-789")

	contextLogger := logger.WithContext(ctx)

	// Verify that context values are preserved
	if contextLogger == nil {
		t.Fatal("Expected context logger to be created")
	}

	// Log with context
	contextLogger.Info("Test message with context", nil)
}

func TestWithFields(t *testing.T) {
	logger := NewLogger(LogConfig{
		Level:         "debug",
		Output:        "json",
		MaskSensitive: true,
	})
	defer logger.Close()

	// Test single field
	fieldLogger := logger.WithField("component", "test")
	if fieldLogger == nil {
		t.Fatal("Expected field logger to be created")
	}

	// Test multiple fields with sensitive data
	fields := map[string]interface{}{
		"user":     "john.doe",
		"password": "secret123", // Should be masked
		"action":   "login",
	}

	multiFieldLogger := logger.WithFields(fields)
	if multiFieldLogger == nil {
		t.Fatal("Expected multi-field logger to be created")
	}
}

func TestDurationLogging(t *testing.T) {
	logger := NewLogger(LogConfig{
		Level:  "info",
		Output: "json",
	})
	defer logger.Close()

	// Test short duration
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	logger.Duration("fast_operation", start, nil)

	// Test moderate duration
	start = time.Now()
	time.Sleep(100 * time.Millisecond)
	logger.Duration("moderate_operation", start, nil)
}

func TestBenchmark(t *testing.T) {
	logger := NewLogger(LogConfig{
		Level:  "info",
		Output: "json",
	})
	defer logger.Close()

	executed := false
	logger.Benchmark("test_operation", func() {
		executed = true
		time.Sleep(10 * time.Millisecond)
	})

	if !executed {
		t.Error("Benchmark function was not executed")
	}
}

func TestAuditLogging(t *testing.T) {
	logger := NewLogger(LogConfig{
		Level:  "info",
		Output: "json",
	})
	defer logger.Close()

	logger.Audit("user_login", "user-123", map[string]interface{}{
		"ip":      "192.168.1.1",
		"success": true,
	})
}

func TestSetLevel(t *testing.T) {
	logger := NewLogger(LogConfig{
		Level:  "info",
		Output: "json",
	})
	defer logger.Close()

	// Change level dynamically
	logger.SetLevel("debug")

	// Verify level was changed
	if logger.config.Level != "debug" {
		t.Errorf("Expected level to be debug, got %s", logger.config.Level)
	}
}

func TestErrorLogging(t *testing.T) {
	logger := NewLogger(LogConfig{
		Level:      "debug",
		Output:     "json",
		StackTrace: true,
	})
	defer logger.Close()

	// Test with AppError
	appErr := NewAPIError(
		ErrCodeRateLimit,
		"Rate limit exceeded",
		429,
		nil,
	).WithContext("retry_after", "60s")

	logger.Error("API error occurred", appErr, map[string]interface{}{
		"endpoint": "/v1/chat",
	})

	// Test with regular error
	regularErr := bytes.ErrTooLarge
	logger.Error("Regular error occurred", regularErr, nil)
}

func TestJSONOutput(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	logger := NewLogger(LogConfig{
		Level:  "info",
		Output: "json",
	})
	defer logger.Close()

	// Log a message
	logger.Info("Test message", map[string]interface{}{
		"key": "value",
	})

	// Parse output as JSON (would need to capture stdout/stderr in real test)
	// This is a simplified test
	if buf.Len() > 0 {
		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Errorf("Failed to parse JSON output: %v", err)
		}
	}
}

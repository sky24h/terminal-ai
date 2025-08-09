// Package main demonstrates logging and error handling features
// To run: go run examples/logging_example.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/user/terminal-ai/internal/utils"
)

func main() {
	// Initialize logger with comprehensive configuration
	logger := utils.NewLogger(utils.LogConfig{
		Level:         "debug",
		Output:        "both", // console and json
		Pretty:        true,
		FilePath:      "logs/app.log",
		MaskSensitive: true,
		StackTrace:    true,
	})
	defer logger.Close()

	// Initialize metrics collector
	metrics := utils.InitMetrics()

	// Start background metrics reporter (every 30 seconds)
	metrics.StartMetricsReporter(logger, 30*time.Second)

	// Example 1: Basic logging with different levels
	logger.Trace("Application starting", nil)
	logger.Debug("Debug information", map[string]interface{}{
		"config": "loaded",
		"mode":   "development",
	})
	logger.Info("Application initialized successfully", nil)

	// Example 2: Context-aware logging with request ID
	ctx := context.WithValue(context.Background(), "request_id", "req-123456")
	ctx = context.WithValue(ctx, "user_id", "user-789")

	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("Processing user request", map[string]interface{}{
		"action": "generate",
		"model":  "gpt-4",
	})

	// Example 3: Sensitive data masking
	logger.Info("API call with masked data", map[string]interface{}{
		"api_key": "sk-abcdefghijklmnopqrstuvwxyz1234567890abcdefghij",
		"token":   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"user":    "john.doe@example.com",
	})

	// Example 4: Performance tracking
	start := time.Now()
	simulateAPICall(logger, metrics)
	logger.Duration("api_operation", start, map[string]interface{}{
		"endpoint": "/v1/chat/completions",
		"method":   "POST",
	})

	// Example 5: Error handling with different error types
	demonstrateErrorHandling(logger)

	// Example 6: Metrics collection
	demonstrateMetrics(logger, metrics)

	// Example 7: Audit logging
	logger.Audit("user_login", "user-789", map[string]interface{}{
		"ip_address": "192.168.1.1",
		"user_agent": "Mozilla/5.0",
		"success":    true,
	})

	// Log final metrics summary
	metrics.LogMetricsSummary(logger)
}

func simulateAPICall(logger *utils.Logger, metrics *utils.MetricsCollector) {
	start := time.Now()

	// Simulate API call
	time.Sleep(100 * time.Millisecond)

	// Record metrics
	duration := time.Since(start)
	metrics.RecordAPICall("/v1/chat/completions", duration, http.StatusOK, nil)
	metrics.RecordTokenUsage(utils.TokenUsage{
		Model:            "gpt-4",
		PromptTokens:     150,
		CompletionTokens: 250,
		TotalTokens:      400,
	})

	logger.Debug("API call completed", map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
		"status":      200,
	})
}

func demonstrateErrorHandling(logger *utils.Logger) {
	// Example 1: Configuration error
	configErr := utils.NewConfigError(
		"Invalid configuration file",
		fmt.Errorf("file not found: config.yaml"),
	)
	logger.Error("Configuration error occurred", configErr, nil)

	// Example 2: API error with retry logic
	apiErr := utils.NewAPIError(
		utils.ErrCodeRateLimit,
		"Rate limit exceeded",
		http.StatusTooManyRequests,
		nil,
	).WithContext("retry_after", "60s")

	if utils.IsRetryable(apiErr) {
		logger.Warn("Retryable error encountered", map[string]interface{}{
			"error_code":  apiErr.Code,
			"will_retry":  true,
			"retry_after": apiErr.Context["retry_after"],
		})
	}

	// Example 3: Validation error
	validationErr := utils.NewValidationError(
		"Model name must be one of: gpt-4, gpt-3.5-turbo",
		"model",
	)
	logger.Error("Validation failed", validationErr, map[string]interface{}{
		"input": "invalid-model",
	})

	// Example 4: Network error
	networkErr := utils.NewNetworkError(
		"Failed to connect to OpenAI API",
		fmt.Errorf("connection timeout after 30s"),
	)
	logger.Error("Network error", networkErr, nil)

	// Example 5: Authentication error
	authErr := utils.NewAuthError("Invalid API key provided")
	logger.Error("Authentication failed", authErr, nil)
	fmt.Printf("User-friendly message: %s\n", authErr.GetUserMessage())

	// Example 6: Error group for multiple errors
	errorGroup := utils.NewErrorGroup()
	errorGroup.Add(fmt.Errorf("first error"))
	errorGroup.Add(fmt.Errorf("second error"))

	if errorGroup.HasErrors() {
		logger.Error("Multiple errors occurred", errorGroup, nil)
	}
}

func demonstrateMetrics(logger *utils.Logger, metrics *utils.MetricsCollector) {
	// Simulate cache operations
	for i := 0; i < 10; i++ {
		if i%3 == 0 {
			metrics.RecordCacheHit()
		} else {
			metrics.RecordCacheMiss()
		}
	}
	metrics.RecordCacheWrite()
	metrics.RecordCacheEviction()

	// Record various operation durations
	operations := []string{"parse_input", "generate_response", "format_output"}
	for i, op := range operations {
		duration := time.Duration(100+i*50) * time.Millisecond
		metrics.RecordDuration(op, duration)
	}

	// Get and log statistics
	cacheStats := metrics.GetCacheStats()
	logger.Info("Cache statistics", map[string]interface{}{
		"hit_rate": fmt.Sprintf("%.2f%%", cacheStats["hit_rate"].(float64)),
		"hits":     cacheStats["hits"],
		"misses":   cacheStats["misses"],
	})

	apiStats := metrics.GetAPIStats()
	logger.Info("API statistics", map[string]interface{}{
		"total_calls":  apiStats["total_calls"],
		"total_errors": apiStats["total_errors"],
	})

	memStats := metrics.GetMemoryStats()
	logger.Info("Memory usage", map[string]interface{}{
		"allocated_mb": memStats["alloc_mb"],
		"goroutines":   memStats["goroutines"],
	})
}

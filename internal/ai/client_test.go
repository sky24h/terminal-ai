package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/user/terminal-ai/internal/config"
)

func TestNewOpenAIClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				OpenAI: config.OpenAIConfig{
					APIKey:  "test-key",
					Model:   "gpt-3.5-turbo",
					Timeout: 30 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: &config.Config{
				OpenAI: config.OpenAIConfig{
					Model:   "gpt-3.5-turbo",
					Timeout: 30 * time.Second,
				},
			},
			wantErr: true,
		},
		{
			name: "custom base URL",
			config: &config.Config{
				OpenAI: config.OpenAIConfig{
					APIKey:  "test-key",
					BaseURL: "https://custom.openai.com/v1",
					Model:   "gpt-3.5-turbo",
					Timeout: 30 * time.Second,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewOpenAIClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewOpenAIClient() returned nil client")
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestRetryConfig(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}

	client := &OpenAIClient{
		retryConfig: config,
	}

	// Test backoff calculation
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second},  // Should cap at MaxDelay
		{10, 30 * time.Second}, // Should still cap at MaxDelay
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			delay := client.calculateBackoff(tt.attempt)
			if delay != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := &RateLimiter{
		minInterval:    100 * time.Millisecond,
		requestsPerMin: 10,
		windowStart:    time.Now(),
	}

	ctx := context.Background()

	// Test minimum interval enforcement
	start := time.Now()
	err := limiter.Wait(ctx)
	if err != nil {
		t.Fatalf("First Wait() failed: %v", err)
	}

	err = limiter.Wait(ctx)
	if err != nil {
		t.Fatalf("Second Wait() failed: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < limiter.minInterval {
		t.Errorf("Rate limiter didn't enforce minimum interval: elapsed %v, expected >= %v",
			elapsed, limiter.minInterval)
	}
}

func TestRateLimiterContextCancellation(t *testing.T) {
	limiter := &RateLimiter{
		minInterval:    1 * time.Second,
		requestsPerMin: 1,
		windowStart:    time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Make first request
	err := limiter.Wait(ctx)
	if err != nil {
		t.Fatalf("First Wait() failed: %v", err)
	}

	// Cancel context before second request
	cancel()

	// Second request should fail due to cancelled context
	err = limiter.Wait(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestConvertMessages(t *testing.T) {
	client := &OpenAIClient{}

	messages := []Message{
		{Role: "system", Content: "You are helpful"},
		{Role: "user", Content: "Hello", Name: "Alice"},
		{Role: "assistant", Content: "Hi there!"},
	}

	converted := client.convertMessages(messages)

	if len(converted) != len(messages) {
		t.Errorf("convertMessages() returned %d messages, expected %d", len(converted), len(messages))
	}

	// With the new API, we can't directly inspect the union fields
	// The test just verifies the conversion doesn't panic and returns the right number of messages
	// More detailed testing would require mocking the API calls
}

func TestClientClosed(t *testing.T) {
	cfg := &config.Config{
		OpenAI: config.OpenAIConfig{
			APIKey:  "test-key",
			Model:   "gpt-3.5-turbo",
			Timeout: 30 * time.Second,
		},
	}

	client, err := NewOpenAIClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Close the client
	err = client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}

	// Try to use closed client
	ctx := context.Background()
	_, err = client.Query(ctx, "test")
	if err == nil || err.Error() != "client is closed" {
		t.Errorf("Expected 'client is closed' error, got: %v", err)
	}

	// Close again should not error
	err = client.Close()
	if err != nil {
		t.Errorf("Closing already closed client returned error: %v", err)
	}
}

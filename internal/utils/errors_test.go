package utils

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestNewAppError(t *testing.T) {
	err := NewAppError(ErrCodeAPIFailure, "API call failed", fmt.Errorf("connection timeout"))

	if err.Code != ErrCodeAPIFailure {
		t.Errorf("Expected code %s, got %s", ErrCodeAPIFailure, err.Code)
	}

	if err.Message != "API call failed" {
		t.Errorf("Expected message 'API call failed', got %s", err.Message)
	}

	if err.Err == nil {
		t.Error("Expected wrapped error to be present")
	}

	if len(err.Stack) == 0 {
		t.Error("Expected stack trace to be captured")
	}
}

func TestErrorTypeConstructors(t *testing.T) {
	tests := []struct {
		name       string
		createErr  func() *AppError
		checkCode  string
		retryable  bool
		statusCode int
	}{
		{
			name: "ConfigError",
			createErr: func() *AppError {
				return NewConfigError("Invalid config", nil)
			},
			checkCode:  ErrCodeConfiguration,
			retryable:  false,
			statusCode: http.StatusInternalServerError,
		},
		{
			name: "APIError",
			createErr: func() *AppError {
				return NewAPIError(ErrCodeRateLimit, "Rate limited", 429, nil)
			},
			checkCode:  ErrCodeRateLimit,
			retryable:  true,
			statusCode: 429,
		},
		{
			name: "NetworkError",
			createErr: func() *AppError {
				return NewNetworkError("Connection failed", nil)
			},
			checkCode:  ErrCodeNetwork,
			retryable:  true,
			statusCode: http.StatusServiceUnavailable,
		},
		{
			name: "ValidationError",
			createErr: func() *AppError {
				return NewValidationError("Invalid input", "field_name")
			},
			checkCode:  ErrCodeValidation,
			retryable:  false,
			statusCode: http.StatusBadRequest,
		},
		{
			name: "CacheError",
			createErr: func() *AppError {
				return NewCacheError("Cache unavailable", nil)
			},
			checkCode: ErrCodeCache,
			retryable: true,
		},
		{
			name: "AuthError",
			createErr: func() *AppError {
				return NewAuthError("Invalid credentials")
			},
			checkCode:  ErrCodeAuthentication,
			retryable:  false,
			statusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createErr()

			if err.Code != tt.checkCode {
				t.Errorf("Expected code %s, got %s", tt.checkCode, err.Code)
			}

			if err.Retryable != tt.retryable {
				t.Errorf("Expected retryable %v, got %v", tt.retryable, err.Retryable)
			}

			if tt.statusCode != 0 && err.StatusCode != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, err.StatusCode)
			}
		})
	}
}

func TestErrorContext(t *testing.T) {
	err := NewAppError(ErrCodeAPIFailure, "Test error", nil)

	// Test WithContext
	err.WithContext("key1", "value1")
	if err.Context["key1"] != "value1" {
		t.Error("Expected context key1 to be value1")
	}

	// Test WithContextMap
	contextMap := map[string]interface{}{
		"key2": "value2",
		"key3": 123,
	}
	err.WithContextMap(contextMap)

	if err.Context["key2"] != "value2" {
		t.Error("Expected context key2 to be value2")
	}
	if err.Context["key3"] != 123 {
		t.Error("Expected context key3 to be 123")
	}

	// Test WithRequestID
	err.WithRequestID("req-123")
	if err.RequestID != "req-123" {
		t.Error("Expected request ID to be req-123")
	}
}

func TestErrorMethods(t *testing.T) {
	err := NewAPIError(ErrCodeRateLimit, "Rate limited", 429, fmt.Errorf("underlying error"))

	// Test Error() method
	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected error string to be non-empty")
	}

	// Test Unwrap()
	unwrapped := err.Unwrap()
	if unwrapped == nil {
		t.Error("Expected unwrapped error to be present")
	}

	// Test Is()
	rateLimitErr := &AppError{Code: ErrCodeRateLimit}
	if !err.Is(rateLimitErr) {
		t.Error("Expected Is() to return true for matching code")
	}
	timeoutErr := &AppError{Code: ErrCodeTimeout}
	if err.Is(timeoutErr) {
		t.Error("Expected Is() to return false for non-matching code")
	}

	// Test GetStack()
	stack := err.GetStack()
	if len(stack) == 0 {
		t.Error("Expected stack trace to be present")
	}
}

func TestGetUserMessage(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{
			code:     ErrCodeRateLimit,
			expected: "You've made too many requests. Please wait a moment and try again.",
		},
		{
			code:     ErrCodeAuthentication,
			expected: "Authentication failed. Please check your API key or credentials.",
		},
		{
			code:     ErrCodeNetwork,
			expected: "Network connection issue. Please check your internet connection.",
		},
		{
			code:     "UNKNOWN_CODE",
			expected: "Test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := NewAppError(tt.code, "Test message", nil)
			msg := err.GetUserMessage()

			if tt.code == "UNKNOWN_CODE" {
				if msg != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, msg)
				}
			} else {
				if msg != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, msg)
				}
			}
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		code           string
		expectedStatus int
	}{
		{ErrCodeValidation, http.StatusBadRequest},
		{ErrCodeAuthentication, http.StatusUnauthorized},
		{ErrCodePermissionDenied, http.StatusForbidden},
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeRateLimit, http.StatusTooManyRequests},
		{ErrCodeTimeout, http.StatusRequestTimeout},
		{ErrCodeServiceDown, http.StatusServiceUnavailable},
		{"UNKNOWN", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := NewAppError(tt.code, "Test", nil)
			status := err.HTTPStatus()

			if status != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, status)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "Retryable API error",
			err:       NewAPIError(ErrCodeRateLimit, "Rate limited", 429, nil),
			retryable: true,
		},
		{
			name:      "Non-retryable validation error",
			err:       NewValidationError("Invalid input", "field"),
			retryable: false,
		},
		{
			name:      "Network error is retryable",
			err:       NewNetworkError("Connection failed", nil),
			retryable: true,
		},
		{
			name:      "Standard error with retryable pattern",
			err:       errors.New("connection refused"),
			retryable: true,
		},
		{
			name:      "Standard error without retryable pattern",
			err:       errors.New("invalid argument"),
			retryable: false,
		},
		{
			name:      "Nil error",
			err:       nil,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("Expected IsRetryable to return %v, got %v", tt.retryable, result)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	// Test wrapping nil error
	wrapped := WrapError(nil, ErrCodeInternal, "Test")
	if wrapped != nil {
		t.Error("Expected nil when wrapping nil error")
	}

	// Test wrapping standard error
	stdErr := errors.New("standard error")
	wrapped = WrapError(stdErr, ErrCodeInternal, "Wrapped message")

	if wrapped.Code != ErrCodeInternal {
		t.Errorf("Expected code %s, got %s", ErrCodeInternal, wrapped.Code)
	}
	if wrapped.Message != "Wrapped message" {
		t.Errorf("Expected message 'Wrapped message', got %s", wrapped.Message)
	}

	// Test wrapping AppError (should preserve original)
	appErr := NewAPIError(ErrCodeRateLimit, "Original message", 429, nil)
	wrapped = WrapError(appErr, ErrCodeInternal, "Additional context")

	if wrapped.Code != ErrCodeRateLimit {
		t.Error("Expected original code to be preserved")
	}
	if !strings.Contains(wrapped.Message, "Additional context") {
		t.Error("Expected additional context in message")
	}
}

func TestErrorGroup(t *testing.T) {
	eg := NewErrorGroup()

	// Test empty group
	if eg.HasErrors() {
		t.Error("Expected no errors in new group")
	}
	if eg.Error() != "" {
		t.Error("Expected empty string for empty error group")
	}

	// Add errors
	eg.Add(errors.New("error 1"))
	eg.Add(errors.New("error 2"))
	eg.Add(nil) // Should be ignored

	if !eg.HasErrors() {
		t.Error("Expected errors to be present")
	}

	errs := eg.GetErrors()
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errs))
	}

	// Test Error() string
	errStr := eg.Error()
	if !strings.Contains(errStr, "Multiple errors occurred") {
		t.Error("Expected error string to contain 'Multiple errors occurred'")
	}
	if !strings.Contains(errStr, "error 1") {
		t.Error("Expected error string to contain 'error 1'")
	}
	if !strings.Contains(errStr, "error 2") {
		t.Error("Expected error string to contain 'error 2'")
	}
}

func TestIsAppError(t *testing.T) {
	// Test with AppError
	appErr := NewAppError(ErrCodeInternal, "Test", nil)
	if !IsAppError(appErr) {
		t.Error("Expected IsAppError to return true for AppError")
	}

	// Test with standard error
	stdErr := errors.New("standard error")
	if IsAppError(stdErr) {
		t.Error("Expected IsAppError to return false for standard error")
	}
}

func TestGetAppError(t *testing.T) {
	// Test with AppError
	appErr := NewAppError(ErrCodeInternal, "Test", nil)
	extracted := GetAppError(appErr)
	if extracted == nil {
		t.Error("Expected to extract AppError")
	}
	if extracted != appErr {
		t.Error("Expected extracted error to be the same instance")
	}

	// Test with standard error
	stdErr := errors.New("standard error")
	extracted = GetAppError(stdErr)
	if extracted != nil {
		t.Error("Expected nil when extracting from standard error")
	}
}

func TestSetRetryable(t *testing.T) {
	err := NewAppError(ErrCodeInternal, "Test", nil)

	// Initially should be based on code
	initialRetryable := err.Retryable

	// Set to opposite
	err.SetRetryable(!initialRetryable)

	if err.Retryable == initialRetryable {
		t.Error("Expected SetRetryable to change the retryable flag")
	}
}

func TestWithStatusCode(t *testing.T) {
	err := NewAppError(ErrCodeInternal, "Test", nil)

	// Set status code
	err.WithStatusCode(http.StatusBadGateway)

	if err.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected status code %d, got %d", http.StatusBadGateway, err.StatusCode)
	}

	// Should be retryable for 502
	if !err.Retryable {
		t.Error("Expected error to be retryable for status 502")
	}
}

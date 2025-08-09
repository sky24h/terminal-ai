package utils

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// AppError represents an application error with context
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Err        error                  `json:"-"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Stack      []string               `json:"-"`
	Retryable  bool                   `json:"retryable"`
	StatusCode int                    `json:"status_code,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
}

// Error codes
const (
	// Client errors (4xx)
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeValidation       = "VALIDATION_ERROR"
	ErrCodeAuthentication   = "AUTH_ERROR"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeRateLimit        = "RATE_LIMIT"
	ErrCodeQuotaExceeded    = "QUOTA_EXCEEDED"

	// Server errors (5xx)
	ErrCodeAPIFailure  = "API_FAILURE"
	ErrCodeInternal    = "INTERNAL_ERROR"
	ErrCodeTimeout     = "TIMEOUT"
	ErrCodeServiceDown = "SERVICE_UNAVAILABLE"

	// Network errors
	ErrCodeNetwork    = "NETWORK_ERROR"
	ErrCodeConnection = "CONNECTION_ERROR"
	ErrCodeDNS        = "DNS_ERROR"

	// Configuration errors
	ErrCodeConfiguration = "CONFIG_ERROR"
	ErrCodeMissingConfig = "MISSING_CONFIG"
	ErrCodeInvalidConfig = "INVALID_CONFIG"

	// Cache errors
	ErrCodeCache        = "CACHE_ERROR"
	ErrCodeCacheMiss    = "CACHE_MISS"
	ErrCodeCacheExpired = "CACHE_EXPIRED"

	// Operation errors
	ErrCodeCanceled = "CANCELED"
	ErrCodeAborted  = "ABORTED"
	ErrCodeDeadline = "DEADLINE_EXCEEDED"
)

// NewAppError creates a new application error
func NewAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:      code,
		Message:   message,
		Err:       err,
		Context:   make(map[string]interface{}),
		Stack:     captureStack(),
		Retryable: isRetryableCode(code),
		Timestamp: time.Now(),
	}
}

// Error type constructors for common scenarios

// NewConfigError creates a configuration error
func NewConfigError(message string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeConfiguration,
		Message:    message,
		Err:        err,
		Context:    make(map[string]interface{}),
		Stack:      captureStack(),
		Retryable:  false,
		StatusCode: http.StatusInternalServerError,
		Timestamp:  time.Now(),
	}
}

// NewAPIError creates an API error with status code
func NewAPIError(code, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		Context:    make(map[string]interface{}),
		Stack:      captureStack(),
		Retryable:  isRetryableStatusCode(statusCode),
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
}

// NewNetworkError creates a network error
func NewNetworkError(message string, err error) *AppError {
	return &AppError{
		Code:       ErrCodeNetwork,
		Message:    message,
		Err:        err,
		Context:    make(map[string]interface{}),
		Stack:      captureStack(),
		Retryable:  true,
		StatusCode: http.StatusServiceUnavailable,
		Timestamp:  time.Now(),
	}
}

// NewValidationError creates a validation error
func NewValidationError(message string, field string) *AppError {
	err := &AppError{
		Code:       ErrCodeValidation,
		Message:    message,
		Context:    make(map[string]interface{}),
		Stack:      captureStack(),
		Retryable:  false,
		StatusCode: http.StatusBadRequest,
		Timestamp:  time.Now(),
	}
	if field != "" {
		err.Context["field"] = field
	}
	return err
}

// NewCacheError creates a cache error
func NewCacheError(message string, err error) *AppError {
	return &AppError{
		Code:      ErrCodeCache,
		Message:   message,
		Err:       err,
		Context:   make(map[string]interface{}),
		Stack:     captureStack(),
		Retryable: true,
		Timestamp: time.Now(),
	}
}

// NewAuthError creates an authentication error
func NewAuthError(message string) *AppError {
	return &AppError{
		Code:       ErrCodeAuthentication,
		Message:    message,
		Context:    make(map[string]interface{}),
		Stack:      captureStack(),
		Retryable:  false,
		StatusCode: http.StatusUnauthorized,
		Timestamp:  time.Now(),
	}
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	e.Context[key] = value
	return e
}

// WithContextMap adds multiple context values
func (e *AppError) WithContextMap(context map[string]interface{}) *AppError {
	for k, v := range context {
		e.Context[k] = v
	}
	return e
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// WithStatusCode sets the HTTP status code
func (e *AppError) WithStatusCode(code int) *AppError {
	e.StatusCode = code
	e.Retryable = isRetryableStatusCode(code)
	return e
}

// SetRetryable sets whether the error is retryable
func (e *AppError) SetRetryable(retryable bool) *AppError {
	e.Retryable = retryable
	return e
}

// Is implements the errors.Is interface for error comparison
func (e *AppError) Is(target error) bool {
	if targetErr, ok := target.(*AppError); ok {
		return e.Code == targetErr.Code
	}
	return false
}

// GetStack returns the stack trace
func (e *AppError) GetStack() []string {
	return e.Stack
}

// GetUserMessage returns a user-friendly error message
func (e *AppError) GetUserMessage() string {
	switch e.Code {
	case ErrCodeRateLimit:
		return "You've made too many requests. Please wait a moment and try again."
	case ErrCodeAuthentication:
		return "Authentication failed. Please check your API key or credentials."
	case ErrCodeNetwork:
		return "Network connection issue. Please check your internet connection."
	case ErrCodeTimeout:
		return "The request took too long. Please try again."
	case ErrCodeValidation:
		return fmt.Sprintf("Invalid input: %s", e.Message)
	case ErrCodePermissionDenied:
		return "You don't have permission to perform this action."
	case ErrCodeNotFound:
		return "The requested resource was not found."
	case ErrCodeServiceDown:
		return "The service is temporarily unavailable. Please try again later."
	case ErrCodeQuotaExceeded:
		return "You've exceeded your usage quota. Please upgrade your plan or wait for the quota to reset."
	default:
		if e.Message != "" {
			return e.Message
		}
		return "An unexpected error occurred. Please try again."
	}
}

// HTTPStatus returns the appropriate HTTP status code
func (e *AppError) HTTPStatus() int {
	if e.StatusCode != 0 {
		return e.StatusCode
	}

	switch e.Code {
	case ErrCodeInvalidInput, ErrCodeValidation:
		return http.StatusBadRequest
	case ErrCodeAuthentication:
		return http.StatusUnauthorized
	case ErrCodePermissionDenied:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeRateLimit:
		return http.StatusTooManyRequests
	case ErrCodeTimeout, ErrCodeDeadline:
		return http.StatusRequestTimeout
	case ErrCodeServiceDown, ErrCodeNetwork:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// captureStack captures the current stack trace
func captureStack() []string {
	var stack []string
	for i := 2; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		// Skip runtime functions
		fnName := fn.Name()
		if strings.Contains(fnName, "runtime.") {
			continue
		}

		stack = append(stack, fmt.Sprintf("%s:%d %s", file, line, fnName))
	}
	return stack
}

// Common errors
var (
	ErrAPIKeyMissing = NewConfigError(
		"API key is missing",
		errors.New("please set your API key in the configuration or environment variable"),
	)

	ErrInvalidModel = NewValidationError(
		"Invalid model specified",
		"model",
	)

	ErrRateLimited = NewAPIError(
		ErrCodeRateLimit,
		"Rate limit exceeded",
		http.StatusTooManyRequests,
		errors.New("please wait before making another request"),
	)

	ErrRequestTimeout = NewAPIError(
		ErrCodeTimeout,
		"Request timed out",
		http.StatusRequestTimeout,
		nil,
	)

	ErrRequestCanceled = NewAppError(
		ErrCodeCanceled,
		"Request was canceled",
		nil,
	).SetRetryable(false)

	ErrServiceUnavailable = NewAPIError(
		ErrCodeServiceDown,
		"Service is temporarily unavailable",
		http.StatusServiceUnavailable,
		nil,
	)

	ErrQuotaExceeded = NewAPIError(
		ErrCodeQuotaExceeded,
		"API quota exceeded",
		http.StatusPaymentRequired,
		nil,
	)
)

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError extracts AppError from an error
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// WrapError wraps a standard error into an AppError
func WrapError(err error, code, message string) *AppError {
	if err == nil {
		return nil
	}

	// If it's already an AppError, preserve it but add context
	if appErr := GetAppError(err); appErr != nil {
		if message != "" && message != appErr.Message {
			appErr.Message = fmt.Sprintf("%s: %s", message, appErr.Message)
		}
		return appErr
	}

	return NewAppError(code, message, err)
}

// Helper functions

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if appErr := GetAppError(err); appErr != nil {
		return appErr.Retryable
	}

	// Check for common retryable error patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"too many requests",
		"service unavailable",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}

// isRetryableCode checks if an error code is retryable
func isRetryableCode(code string) bool {
	switch code {
	case ErrCodeNetwork, ErrCodeConnection, ErrCodeDNS,
		ErrCodeTimeout, ErrCodeServiceDown, ErrCodeCache:
		return true
	case ErrCodeRateLimit:
		return true // Retry with backoff
	default:
		return false
	}
}

// isRetryableStatusCode checks if an HTTP status code is retryable
func isRetryableStatusCode(code int) bool {
	switch code {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// ErrorGroup represents a collection of errors
type ErrorGroup struct {
	errors []error
}

// NewErrorGroup creates a new error group
func NewErrorGroup() *ErrorGroup {
	return &ErrorGroup{
		errors: make([]error, 0),
	}
}

// Add adds an error to the group
func (eg *ErrorGroup) Add(err error) {
	if err != nil {
		eg.errors = append(eg.errors, err)
	}
}

// HasErrors checks if there are any errors
func (eg *ErrorGroup) HasErrors() bool {
	return len(eg.errors) > 0
}

// Error implements the error interface
func (eg *ErrorGroup) Error() string {
	if len(eg.errors) == 0 {
		return ""
	}

	var messages []string
	for _, err := range eg.errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("Multiple errors occurred: %s", strings.Join(messages, "; "))
}

// GetErrors returns all errors
func (eg *ErrorGroup) GetErrors() []error {
	return eg.errors
}

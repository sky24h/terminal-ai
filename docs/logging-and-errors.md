# Logging and Error Handling Guide

## Overview

The terminal-ai project includes comprehensive logging, error handling, and metrics collection capabilities designed for production use.

## Logging System

### Configuration

The logger supports multiple output formats and configurations:

```go
logger := utils.NewLogger(utils.LogConfig{
    Level:         "debug",      // trace, debug, info, warn, error, fatal
    Output:        "both",        // console, json, or both
    Pretty:        true,          // Pretty print for console output
    FilePath:      "logs/app.log", // Optional file output
    MaskSensitive: true,          // Mask sensitive data
    StackTrace:    true,          // Include stack traces for errors
})
```

### Log Levels

- **Trace**: Most detailed debugging information
- **Debug**: Debugging information
- **Info**: General informational messages
- **Warn**: Warning messages
- **Error**: Error messages (includes stack traces if enabled)
- **Fatal**: Fatal errors that cause application exit

### Features

#### Sensitive Data Masking

Automatically masks sensitive information in logs:
- OpenAI API keys (sk-xxx...)
- Bearer tokens
- Passwords
- API keys in JSON

```go
logger.Info("API call", map[string]interface{}{
    "api_key": "sk-abcdef...", // Will be masked as "sk-***MASKED***"
})
```

#### Context-Aware Logging

Add context from Go contexts:

```go
ctx := context.WithValue(ctx, "request_id", "req-123")
contextLogger := logger.WithContext(ctx)
contextLogger.Info("Processing request", nil)
```

#### Performance Tracking

Track operation duration:

```go
start := time.Now()
// ... perform operation ...
logger.Duration("operation_name", start, nil)
```

#### Audit Logging

For security-critical events:

```go
logger.Audit("user_login", userID, map[string]interface{}{
    "ip_address": "192.168.1.1",
    "success": true,
})
```

## Error Handling

### Error Types

The system provides specialized error types for different scenarios:

#### Configuration Errors
```go
err := utils.NewConfigError("Invalid configuration", originalErr)
```

#### API Errors
```go
err := utils.NewAPIError(
    utils.ErrCodeRateLimit,
    "Rate limit exceeded",
    429,
    originalErr,
)
```

#### Network Errors
```go
err := utils.NewNetworkError("Connection failed", originalErr)
```

#### Validation Errors
```go
err := utils.NewValidationError("Invalid input", "fieldName")
```

#### Cache Errors
```go
err := utils.NewCacheError("Cache unavailable", originalErr)
```

#### Authentication Errors
```go
err := utils.NewAuthError("Invalid credentials")
```

### Error Features

#### Retryable Errors

Check if an error is retryable:

```go
if utils.IsRetryable(err) {
    // Implement retry logic with backoff
}
```

#### Error Context

Add context to errors:

```go
err.WithContext("retry_after", "60s")
   .WithRequestID("req-123")
   .WithStatusCode(429)
```

#### User-Friendly Messages

Get user-friendly error messages:

```go
userMessage := err.GetUserMessage()
// Returns: "You've made too many requests. Please wait a moment and try again."
```

#### Stack Traces

Errors automatically capture stack traces for debugging:

```go
stack := err.GetStack()
```

### Error Codes

Standard error codes for programmatic handling:

- **Client Errors (4xx)**
  - `INVALID_INPUT`: Invalid input data
  - `VALIDATION_ERROR`: Validation failed
  - `AUTH_ERROR`: Authentication failed
  - `PERMISSION_DENIED`: Insufficient permissions
  - `NOT_FOUND`: Resource not found
  - `RATE_LIMIT`: Rate limit exceeded
  - `QUOTA_EXCEEDED`: Quota exceeded

- **Server Errors (5xx)**
  - `API_FAILURE`: API call failed
  - `INTERNAL_ERROR`: Internal server error
  - `TIMEOUT`: Request timeout
  - `SERVICE_UNAVAILABLE`: Service unavailable

- **Network Errors**
  - `NETWORK_ERROR`: General network error
  - `CONNECTION_ERROR`: Connection failed
  - `DNS_ERROR`: DNS resolution failed

## Metrics Collection

### Available Metrics

#### API Metrics
- Total calls and errors
- Per-endpoint statistics
- Response time tracking
- Status code distribution

```go
metrics.RecordAPICall(endpoint, duration, statusCode, err)
```

#### Cache Metrics
- Hit/miss rates
- Write and eviction counts

```go
metrics.RecordCacheHit()
metrics.RecordCacheMiss()
```

#### Token Usage
- Total tokens consumed
- Per-model breakdown
- Cost estimates

```go
metrics.RecordTokenUsage(utils.TokenUsage{
    Model:            "gpt-5-mini",
    PromptTokens:     150,
    CompletionTokens: 250,
    TotalTokens:      400,
})
```

#### Performance Metrics
- Operation durations
- Min/max/average times

```go
metrics.RecordDuration("operation_name", duration)
```

### Retrieving Metrics

Get comprehensive statistics:

```go
// All stats
allStats := metrics.GetAllStats()

// Specific categories
apiStats := metrics.GetAPIStats()
cacheStats := metrics.GetCacheStats()
tokenStats := metrics.GetTokenStats()
memoryStats := metrics.GetMemoryStats()
perfStats := metrics.GetPerformanceStats()
```

### Background Reporting

Start automatic metrics reporting:

```go
metrics.StartMetricsReporter(logger, 30*time.Second)
```

## Best Practices

### 1. Use Structured Logging

```go
logger.Info("User action", map[string]interface{}{
    "user_id": userID,
    "action":  "generate",
    "model":   model,
    "tokens":  tokenCount,
})
```

### 2. Handle Errors Appropriately

```go
err := someOperation()
if err != nil {
    appErr := utils.WrapError(err, utils.ErrCodeAPIFailure, "Operation failed")
    
    if utils.IsRetryable(appErr) {
        // Retry with exponential backoff
        return retryWithBackoff(someOperation)
    }
    
    logger.Error("Operation failed", appErr, map[string]interface{}{
        "operation": "someOperation",
    })
    
    return appErr
}
```

### 3. Track Performance

```go
func processRequest(ctx context.Context) error {
    start := time.Now()
    defer func() {
        logger.Duration("request_processing", start, nil)
        metrics.RecordDuration("request_processing", time.Since(start))
    }()
    
    // ... process request ...
}
```

### 4. Use Request IDs

```go
requestID := generateRequestID()
logger := logger.WithRequestID(requestID)

// Use this logger throughout the request lifecycle
logger.Info("Starting request processing", nil)
```

### 5. Audit Critical Operations

```go
logger.Audit("data_access", userID, map[string]interface{}{
    "resource":    resourceID,
    "action":      "read",
    "ip_address":  clientIP,
    "timestamp":   time.Now(),
})
```

## Example Usage

See `/examples/logging_example.go` for a complete working example demonstrating all features.

## Testing

Run the comprehensive test suite:

```bash
go test ./internal/utils -v
```

Test specific components:

```bash
# Logger tests
go test ./internal/utils -v -run TestLogger

# Error handling tests
go test ./internal/utils -v -run TestError

# Metrics tests
go test ./internal/utils -v -run TestMetrics
```
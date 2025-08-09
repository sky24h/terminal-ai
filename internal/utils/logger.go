package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var (
	// Global logger instance
	globalLogger *Logger
	loggerOnce   sync.Once

	// Sensitive data patterns for masking
	sensitivePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(sk-[a-zA-Z0-9]{32,})`),                  // OpenAI API key
		regexp.MustCompile(`(Bearer\s+[a-zA-Z0-9\-\._~\+\/]+)`),      // Bearer tokens
		regexp.MustCompile(`("api[_-]?key"\s*:\s*"[^"]+")`),          // API keys in JSON
		regexp.MustCompile(`(password["']?\s*[:=]\s*["']?[^\s"']+)`), // Passwords
		regexp.MustCompile(`(token["']?\s*[:=]\s*["']?[^\s"']+)`),    // Tokens
	}
)

// Logger wraps zerolog for application logging
type Logger struct {
	logger    zerolog.Logger
	config    LogConfig
	logFile   io.WriteCloser
	maskingOn bool
	mu        sync.RWMutex
}

// LogConfig represents logger configuration
type LogConfig struct {
	Level         string
	Output        string // "console", "json", "both"
	TimeFormat    string
	Pretty        bool
	FilePath      string
	MaskSensitive bool // mask sensitive data
	StackTrace    bool // include stack traces for errors
}

// NewLogger creates a new logger instance
func NewLogger(config LogConfig) *Logger {
	// Set defaults
	if config.TimeFormat == "" {
		config.TimeFormat = time.RFC3339
	}
	if config.Output == "" {
		config.Output = "console"
	}

	// Enable stack traces for error level if configured
	if config.StackTrace {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	}

	// Set log level
	level := parseLevel(config.Level)
	zerolog.SetGlobalLevel(level)

	// Configure writers
	var writers []io.Writer

	// Console output
	if config.Output == "console" || config.Output == "both" {
		if config.Pretty {
			writers = append(writers, zerolog.ConsoleWriter{
				Out:           os.Stderr,
				TimeFormat:    config.TimeFormat,
				FieldsExclude: []string{"hostname", "pid"},
			})
		} else {
			writers = append(writers, os.Stderr)
		}
	}

	// JSON output
	if config.Output == "json" || config.Output == "both" {
		if config.Output != "both" {
			writers = append(writers, os.Stderr)
		}
	}

	// File output
	var fileWriter io.WriteCloser
	if config.FilePath != "" {
		// Ensure directory exists
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err == nil {
			file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				fileWriter = file
				writers = append(writers, fileWriter)
			}
		}
	}

	// Create multi-writer
	var writer io.Writer
	if len(writers) > 1 {
		writer = zerolog.MultiLevelWriter(writers...)
	} else if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = os.Stderr
	}

	// Create logger with context
	logger := zerolog.New(writer).With().
		Timestamp().
		Str("app", "terminal-ai").
		Str("version", "1.0.0").
		Int("pid", os.Getpid()).
		Logger()

	// Set as global logger
	log.Logger = logger

	l := &Logger{
		logger:    logger,
		config:    config,
		logFile:   fileWriter,
		maskingOn: config.MaskSensitive,
	}

	// Set global instance
	loggerOnce.Do(func() {
		globalLogger = l
	})

	return l
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		// Create default logger if not initialized
		globalLogger = NewLogger(LogConfig{
			Level:  "info",
			Output: "console",
			Pretty: true,
		})
	}
	return globalLogger
}

// SetLogger sets the global logger instance
func SetLogger(logger *Logger) {
	globalLogger = logger
}

// parseLevel parses log level from string
func parseLevel(level string) zerolog.Level {
	switch level {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// maskSensitiveData masks sensitive information in a string
func (l *Logger) maskSensitiveData(data string) string {
	if !l.maskingOn {
		return data
	}

	masked := data
	for _, pattern := range sensitivePatterns {
		masked = pattern.ReplaceAllStringFunc(masked, func(match string) string {
			if len(match) <= 8 {
				return "***MASKED***"
			}
			// Show first 3 and last 3 characters
			return match[:3] + "***MASKED***" + match[len(match)-3:]
		})
	}
	return masked
}

// Trace logs a trace message
func (l *Logger) Trace(msg string, fields ...map[string]interface{}) {
	msg = l.maskSensitiveData(msg)
	event := l.logger.Trace()
	if len(fields) > 0 {
		event = event.Fields(l.maskFields(fields[0]))
	}
	event.Msg(msg)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	msg = l.maskSensitiveData(msg)
	event := l.logger.Debug()
	if len(fields) > 0 {
		event = event.Fields(l.maskFields(fields[0]))
	}
	event.Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	msg = l.maskSensitiveData(msg)
	event := l.logger.Info()
	if len(fields) > 0 {
		event = event.Fields(l.maskFields(fields[0]))
	}
	event.Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	msg = l.maskSensitiveData(msg)
	event := l.logger.Warn()
	if len(fields) > 0 {
		event = event.Fields(l.maskFields(fields[0]))
	}
	event.Msg(msg)
}

// Error logs an error message with optional stack trace
func (l *Logger) Error(msg string, err error, fields ...map[string]interface{}) {
	msg = l.maskSensitiveData(msg)
	event := l.logger.Error()

	if err != nil {
		// Check if it's an AppError to include more context
		if appErr := GetAppError(err); appErr != nil {
			event = event.
				Str("error_code", appErr.Code).
				Interface("error_context", appErr.Context)
			if l.config.StackTrace && len(appErr.Stack) > 0 {
				event = event.Strs("stack_trace", appErr.Stack)
			}
		}
		event = event.Err(err)
	}

	if len(fields) > 0 {
		event = event.Fields(l.maskFields(fields[0]))
	}
	event.Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error, fields ...map[string]interface{}) {
	msg = l.maskSensitiveData(msg)
	event := l.logger.Fatal()

	if err != nil {
		if appErr := GetAppError(err); appErr != nil {
			event = event.
				Str("error_code", appErr.Code).
				Interface("error_context", appErr.Context)
			if l.config.StackTrace && len(appErr.Stack) > 0 {
				event = event.Strs("stack_trace", appErr.Stack)
			}
		}
		event = event.Err(err)
	}

	if len(fields) > 0 {
		event = event.Fields(l.maskFields(fields[0]))
	}
	event.Msg(msg)
}

// maskFields masks sensitive data in fields
func (l *Logger) maskFields(fields map[string]interface{}) map[string]interface{} {
	if !l.maskingOn {
		return fields
	}

	masked := make(map[string]interface{})
	for k, v := range fields {
		// Mask known sensitive field names
		if isSensitiveField(k) {
			masked[k] = "***MASKED***"
		} else if str, ok := v.(string); ok {
			masked[k] = l.maskSensitiveData(str)
		} else {
			masked[k] = v
		}
	}
	return masked
}

// isSensitiveField checks if a field name indicates sensitive data
func isSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	sensitiveNames := []string{"password", "token", "key", "secret", "credential", "auth"}
	for _, sensitive := range sensitiveNames {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}
	return false
}

// WithField creates a new logger with a field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	if isSensitiveField(key) && l.maskingOn {
		value = "***MASKED***"
	} else if str, ok := value.(string); ok && l.maskingOn {
		value = l.maskSensitiveData(str)
	}

	return &Logger{
		logger:    l.logger.With().Interface(key, value).Logger(),
		config:    l.config,
		logFile:   l.logFile,
		maskingOn: l.maskingOn,
	}
}

// WithFields creates a new logger with multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{
		logger:    l.logger.With().Fields(l.maskFields(fields)).Logger(),
		config:    l.config,
		logFile:   l.logFile,
		maskingOn: l.maskingOn,
	}
}

// WithContext creates a new logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	newLogger := l.logger.With().Logger()

	// Extract common context values
	if reqID := ctx.Value("request_id"); reqID != nil {
		newLogger = newLogger.With().Str("request_id", fmt.Sprint(reqID)).Logger()
	}
	if userID := ctx.Value("user_id"); userID != nil {
		newLogger = newLogger.With().Str("user_id", fmt.Sprint(userID)).Logger()
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		newLogger = newLogger.With().Str("trace_id", fmt.Sprint(traceID)).Logger()
	}

	return &Logger{
		logger:    newLogger,
		config:    l.config,
		logFile:   l.logFile,
		maskingOn: l.maskingOn,
	}
}

// WithRequestID creates a new logger with a request ID
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		logger:    l.logger.With().Str("request_id", requestID).Logger(),
		config:    l.config,
		logFile:   l.logFile,
		maskingOn: l.maskingOn,
	}
}

// Benchmark logs execution time
func (l *Logger) Benchmark(name string, fn func()) {
	start := time.Now()
	fn()
	duration := time.Since(start)

	l.logger.Info().
		Str("benchmark", name).
		Dur("duration", duration).
		Msg("Benchmark completed")
}

// Duration logs a duration for performance tracking
func (l *Logger) Duration(operation string, start time.Time, fields ...map[string]interface{}) {
	duration := time.Since(start)
	event := l.logger.Info().
		Str("operation", operation).
		Dur("duration", duration).
		Int64("duration_ms", duration.Milliseconds())

	if len(fields) > 0 {
		event = event.Fields(l.maskFields(fields[0]))
	}

	// Add performance warning for slow operations
	if duration > 5*time.Second {
		event.Str("performance", "slow")
	} else if duration > 1*time.Second {
		event.Str("performance", "moderate")
	} else {
		event.Str("performance", "fast")
	}

	event.Msg("Operation completed")
}

// Audit logs an audit event
func (l *Logger) Audit(action string, userID string, fields map[string]interface{}) {
	event := l.logger.Info().
		Str("audit_action", action).
		Str("user_id", userID).
		Time("audit_timestamp", time.Now())

	if fields != nil {
		event = event.Fields(l.maskFields(fields))
	}

	event.Msg("Audit event")
}

// GetZerolog returns the underlying zerolog logger
func (l *Logger) GetZerolog() zerolog.Logger {
	return l.logger
}

// SetLevel dynamically changes the log level
func (l *Logger) SetLevel(level string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLevel := parseLevel(level)
	zerolog.SetGlobalLevel(newLevel)
	l.config.Level = level
}

// Close closes any open file handles
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Flush ensures all buffered logs are written
func (l *Logger) Flush() {
	// Zerolog doesn't buffer, but if we add buffering later
	// this method will be useful
	if syncer, ok := l.logFile.(interface{ Sync() error }); ok {
		_ = syncer.Sync()
	}
}

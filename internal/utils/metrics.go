package utils

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects performance and usage metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// API metrics
	apiCalls       map[string]*APIMetrics
	totalAPICalls  int64
	totalAPIErrors int64

	// Cache metrics
	cacheHits      int64
	cacheMisses    int64
	cacheWrites    int64
	cacheEvictions int64

	// Token usage
	totalTokensUsed int64
	tokensByModel   map[string]int64

	// Performance metrics
	durations map[string]*DurationMetrics

	// Memory metrics
	memStats      runtime.MemStats
	lastMemUpdate time.Time

	// Start time for uptime calculation
	startTime time.Time
}

// APIMetrics tracks metrics for API calls
type APIMetrics struct {
	Count         int64
	Errors        int64
	TotalDuration time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	LastCallTime  time.Time
	StatusCodes   map[int]int64
	mu            sync.RWMutex
}

// DurationMetrics tracks duration statistics
type DurationMetrics struct {
	Count      int64
	Total      time.Duration
	Min        time.Duration
	Max        time.Duration
	Average    time.Duration
	LastUpdate time.Time
	mu         sync.RWMutex
}

// TokenUsage represents token usage for a request
type TokenUsage struct {
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

var (
	// Global metrics instance
	globalMetrics *MetricsCollector
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics collector
func InitMetrics() *MetricsCollector {
	metricsOnce.Do(func() {
		globalMetrics = &MetricsCollector{
			apiCalls:      make(map[string]*APIMetrics),
			tokensByModel: make(map[string]int64),
			durations:     make(map[string]*DurationMetrics),
			startTime:     time.Now(),
		}
	})
	return globalMetrics
}

// GetMetrics returns the global metrics instance
func GetMetrics() *MetricsCollector {
	if globalMetrics == nil {
		return InitMetrics()
	}
	return globalMetrics
}

// RecordAPICall records an API call
func (m *MetricsCollector) RecordAPICall(endpoint string, duration time.Duration, statusCode int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize endpoint metrics if needed
	if _, exists := m.apiCalls[endpoint]; !exists {
		m.apiCalls[endpoint] = &APIMetrics{
			StatusCodes: make(map[int]int64),
			MinDuration: duration,
			MaxDuration: duration,
		}
	}

	metrics := m.apiCalls[endpoint]
	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	// Update counts
	metrics.Count++
	atomic.AddInt64(&m.totalAPICalls, 1)

	if err != nil {
		metrics.Errors++
		atomic.AddInt64(&m.totalAPIErrors, 1)
	}

	// Update duration stats
	metrics.TotalDuration += duration
	if duration < metrics.MinDuration || metrics.MinDuration == 0 {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}

	// Record status code
	if statusCode > 0 {
		metrics.StatusCodes[statusCode]++
	}

	metrics.LastCallTime = time.Now()
}

// RecordCacheHit records a cache hit
func (m *MetricsCollector) RecordCacheHit() {
	atomic.AddInt64(&m.cacheHits, 1)
}

// RecordCacheMiss records a cache miss
func (m *MetricsCollector) RecordCacheMiss() {
	atomic.AddInt64(&m.cacheMisses, 1)
}

// RecordCacheWrite records a cache write
func (m *MetricsCollector) RecordCacheWrite() {
	atomic.AddInt64(&m.cacheWrites, 1)
}

// RecordCacheEviction records a cache eviction
func (m *MetricsCollector) RecordCacheEviction() {
	atomic.AddInt64(&m.cacheEvictions, 1)
}

// RecordTokenUsage records token usage
func (m *MetricsCollector) RecordTokenUsage(usage TokenUsage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.AddInt64(&m.totalTokensUsed, int64(usage.TotalTokens))

	if _, exists := m.tokensByModel[usage.Model]; !exists {
		m.tokensByModel[usage.Model] = 0
	}
	m.tokensByModel[usage.Model] += int64(usage.TotalTokens)
}

// RecordDuration records a duration for an operation
func (m *MetricsCollector) RecordDuration(operation string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.durations[operation]; !exists {
		m.durations[operation] = &DurationMetrics{
			Min: duration,
			Max: duration,
		}
	}

	metrics := m.durations[operation]
	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.Count++
	metrics.Total += duration

	if duration < metrics.Min || metrics.Min == 0 {
		metrics.Min = duration
	}
	if duration > metrics.Max {
		metrics.Max = duration
	}

	// Calculate average
	if metrics.Count > 0 {
		metrics.Average = time.Duration(int64(metrics.Total) / metrics.Count)
	}

	metrics.LastUpdate = time.Now()
}

// GetCacheHitRate returns the cache hit rate
func (m *MetricsCollector) GetCacheHitRate() float64 {
	hits := atomic.LoadInt64(&m.cacheHits)
	misses := atomic.LoadInt64(&m.cacheMisses)

	total := hits + misses
	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total) * 100
}

// GetAPIStats returns API call statistics
func (m *MetricsCollector) GetAPIStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_calls"] = atomic.LoadInt64(&m.totalAPICalls)
	stats["total_errors"] = atomic.LoadInt64(&m.totalAPIErrors)

	if m.totalAPICalls > 0 {
		stats["error_rate"] = float64(m.totalAPIErrors) / float64(m.totalAPICalls) * 100
	}

	// Endpoint-specific stats
	endpoints := make(map[string]map[string]interface{})
	for endpoint, metrics := range m.apiCalls {
		metrics.mu.RLock()
		endpointStats := map[string]interface{}{
			"count":           metrics.Count,
			"errors":          metrics.Errors,
			"avg_duration_ms": float64(metrics.TotalDuration.Milliseconds()) / float64(metrics.Count),
			"min_duration_ms": metrics.MinDuration.Milliseconds(),
			"max_duration_ms": metrics.MaxDuration.Milliseconds(),
			"last_call":       metrics.LastCallTime,
			"status_codes":    metrics.StatusCodes,
		}
		metrics.mu.RUnlock()
		endpoints[endpoint] = endpointStats
	}
	stats["endpoints"] = endpoints

	return stats
}

// GetCacheStats returns cache statistics
func (m *MetricsCollector) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"hits":      atomic.LoadInt64(&m.cacheHits),
		"misses":    atomic.LoadInt64(&m.cacheMisses),
		"writes":    atomic.LoadInt64(&m.cacheWrites),
		"evictions": atomic.LoadInt64(&m.cacheEvictions),
		"hit_rate":  m.GetCacheHitRate(),
	}
}

// GetTokenStats returns token usage statistics
func (m *MetricsCollector) GetTokenStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"total_tokens": atomic.LoadInt64(&m.totalTokensUsed),
		"by_model":     m.tokensByModel,
	}

	// Calculate cost estimates (example rates)
	costEstimates := make(map[string]float64)
	for model, tokens := range m.tokensByModel {
		// Example pricing per 1K tokens
		var rate float64
		switch model {
		case "gpt-4":
			rate = 0.03
		case "gpt-3.5-turbo":
			rate = 0.002
		default:
			rate = 0.01
		}
		costEstimates[model] = float64(tokens) / 1000.0 * rate
	}
	stats["cost_estimates"] = costEstimates

	return stats
}

// GetMemoryStats returns memory usage statistics
func (m *MetricsCollector) GetMemoryStats() map[string]interface{} {
	// Update memory stats if stale
	if time.Since(m.lastMemUpdate) > 5*time.Second {
		runtime.ReadMemStats(&m.memStats)
		m.lastMemUpdate = time.Now()
	}

	return map[string]interface{}{
		"alloc_mb":       m.memStats.Alloc / 1024 / 1024,
		"total_alloc_mb": m.memStats.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.memStats.Sys / 1024 / 1024,
		"num_gc":         m.memStats.NumGC,
		"goroutines":     runtime.NumGoroutine(),
	}
}

// GetPerformanceStats returns performance statistics
func (m *MetricsCollector) GetPerformanceStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	operations := make(map[string]map[string]interface{})
	for op, metrics := range m.durations {
		metrics.mu.RLock()
		opStats := map[string]interface{}{
			"count":       metrics.Count,
			"avg_ms":      metrics.Average.Milliseconds(),
			"min_ms":      metrics.Min.Milliseconds(),
			"max_ms":      metrics.Max.Milliseconds(),
			"total_ms":    metrics.Total.Milliseconds(),
			"last_update": metrics.LastUpdate,
		}
		metrics.mu.RUnlock()
		operations[op] = opStats
	}
	stats["operations"] = operations
	stats["uptime_seconds"] = time.Since(m.startTime).Seconds()

	return stats
}

// GetAllStats returns all statistics
func (m *MetricsCollector) GetAllStats() map[string]interface{} {
	return map[string]interface{}{
		"api":         m.GetAPIStats(),
		"cache":       m.GetCacheStats(),
		"tokens":      m.GetTokenStats(),
		"memory":      m.GetMemoryStats(),
		"performance": m.GetPerformanceStats(),
		"timestamp":   time.Now(),
	}
}

// Reset resets all metrics
func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset API metrics
	m.apiCalls = make(map[string]*APIMetrics)
	atomic.StoreInt64(&m.totalAPICalls, 0)
	atomic.StoreInt64(&m.totalAPIErrors, 0)

	// Reset cache metrics
	atomic.StoreInt64(&m.cacheHits, 0)
	atomic.StoreInt64(&m.cacheMisses, 0)
	atomic.StoreInt64(&m.cacheWrites, 0)
	atomic.StoreInt64(&m.cacheEvictions, 0)

	// Reset token metrics
	atomic.StoreInt64(&m.totalTokensUsed, 0)
	m.tokensByModel = make(map[string]int64)

	// Reset duration metrics
	m.durations = make(map[string]*DurationMetrics)

	// Update start time
	m.startTime = time.Now()
}

// LogMetricsSummary logs a summary of metrics
func (m *MetricsCollector) LogMetricsSummary(logger *Logger) {
	stats := m.GetAllStats()

	logger.Info("Metrics Summary", map[string]interface{}{
		"api_calls":      m.totalAPICalls,
		"api_errors":     m.totalAPIErrors,
		"cache_hit_rate": fmt.Sprintf("%.2f%%", m.GetCacheHitRate()),
		"total_tokens":   m.totalTokensUsed,
		"uptime":         fmt.Sprintf("%.2f hours", time.Since(m.startTime).Hours()),
		"memory_mb":      m.memStats.Alloc / 1024 / 1024,
		"goroutines":     runtime.NumGoroutine(),
	})

	// Log detailed stats at debug level
	logger.Debug("Detailed Metrics", map[string]interface{}{
		"full_stats": stats,
	})
}

// StartMetricsReporter starts a background metrics reporter
func (m *MetricsCollector) StartMetricsReporter(logger *Logger, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			m.LogMetricsSummary(logger)
		}
	}()
}

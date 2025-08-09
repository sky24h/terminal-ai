package utils

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestMetricsInitialization(t *testing.T) {
	metrics := InitMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be initialized")
	}

	// Test singleton pattern
	metrics2 := GetMetrics()
	if metrics != metrics2 {
		t.Error("Expected GetMetrics to return the same instance")
	}
}

func TestRecordAPICall(t *testing.T) {
	metrics := &MetricsCollector{
		apiCalls:      make(map[string]*APIMetrics),
		tokensByModel: make(map[string]int64),
		durations:     make(map[string]*DurationMetrics),
		startTime:     time.Now(),
	}

	// Record successful call
	metrics.RecordAPICall("/v1/chat", 100*time.Millisecond, http.StatusOK, nil)

	// Record failed call
	metrics.RecordAPICall("/v1/chat", 200*time.Millisecond, http.StatusInternalServerError, fmt.Errorf("error"))

	// Check metrics
	if metrics.totalAPICalls != 2 {
		t.Errorf("Expected 2 total API calls, got %d", metrics.totalAPICalls)
	}

	if metrics.totalAPIErrors != 1 {
		t.Errorf("Expected 1 total API error, got %d", metrics.totalAPIErrors)
	}

	// Check endpoint-specific metrics
	endpointMetrics := metrics.apiCalls["/v1/chat"]
	if endpointMetrics.Count != 2 {
		t.Errorf("Expected 2 calls for endpoint, got %d", endpointMetrics.Count)
	}

	if endpointMetrics.MinDuration != 100*time.Millisecond {
		t.Errorf("Expected min duration 100ms, got %v", endpointMetrics.MinDuration)
	}

	if endpointMetrics.MaxDuration != 200*time.Millisecond {
		t.Errorf("Expected max duration 200ms, got %v", endpointMetrics.MaxDuration)
	}
}

func TestCacheMetrics(t *testing.T) {
	metrics := &MetricsCollector{
		apiCalls:      make(map[string]*APIMetrics),
		tokensByModel: make(map[string]int64),
		durations:     make(map[string]*DurationMetrics),
		startTime:     time.Now(),
	}

	// Record cache operations
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheMiss()
	metrics.RecordCacheMiss()
	metrics.RecordCacheWrite()
	metrics.RecordCacheEviction()

	// Test hit rate calculation
	hitRate := metrics.GetCacheHitRate()
	expectedRate := 60.0 // 3 hits out of 5 total (3 hits + 2 misses)
	if hitRate != expectedRate {
		t.Errorf("Expected hit rate %.2f%%, got %.2f%%", expectedRate, hitRate)
	}

	// Test cache stats
	stats := metrics.GetCacheStats()
	if stats["hits"] != int64(3) {
		t.Errorf("Expected 3 hits, got %v", stats["hits"])
	}
	if stats["misses"] != int64(2) {
		t.Errorf("Expected 2 misses, got %v", stats["misses"])
	}
	if stats["writes"] != int64(1) {
		t.Errorf("Expected 1 write, got %v", stats["writes"])
	}
	if stats["evictions"] != int64(1) {
		t.Errorf("Expected 1 eviction, got %v", stats["evictions"])
	}
}

func TestTokenUsage(t *testing.T) {
	metrics := &MetricsCollector{
		apiCalls:      make(map[string]*APIMetrics),
		tokensByModel: make(map[string]int64),
		durations:     make(map[string]*DurationMetrics),
		startTime:     time.Now(),
	}

	// Record token usage
	metrics.RecordTokenUsage(TokenUsage{
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 200,
		TotalTokens:      300,
	})

	metrics.RecordTokenUsage(TokenUsage{
		Model:            "gpt-3.5-turbo",
		PromptTokens:     50,
		CompletionTokens: 100,
		TotalTokens:      150,
	})

	metrics.RecordTokenUsage(TokenUsage{
		Model:            "gpt-4",
		PromptTokens:     200,
		CompletionTokens: 300,
		TotalTokens:      500,
	})

	// Check total tokens
	if metrics.totalTokensUsed != 950 {
		t.Errorf("Expected 950 total tokens, got %d", metrics.totalTokensUsed)
	}

	// Check per-model tokens
	if metrics.tokensByModel["gpt-4"] != 800 {
		t.Errorf("Expected 800 tokens for gpt-4, got %d", metrics.tokensByModel["gpt-4"])
	}
	if metrics.tokensByModel["gpt-3.5-turbo"] != 150 {
		t.Errorf("Expected 150 tokens for gpt-3.5-turbo, got %d", metrics.tokensByModel["gpt-3.5-turbo"])
	}

	// Test token stats
	stats := metrics.GetTokenStats()
	if stats["total_tokens"] != int64(950) {
		t.Errorf("Expected 950 total tokens in stats, got %v", stats["total_tokens"])
	}
}

func TestDurationMetrics(t *testing.T) {
	metrics := &MetricsCollector{
		apiCalls:      make(map[string]*APIMetrics),
		tokensByModel: make(map[string]int64),
		durations:     make(map[string]*DurationMetrics),
		startTime:     time.Now(),
	}

	// Record durations for different operations
	operations := []struct {
		name     string
		duration time.Duration
	}{
		{"parse", 10 * time.Millisecond},
		{"parse", 20 * time.Millisecond},
		{"parse", 30 * time.Millisecond},
		{"generate", 100 * time.Millisecond},
		{"generate", 200 * time.Millisecond},
	}

	for _, op := range operations {
		metrics.RecordDuration(op.name, op.duration)
	}

	// Check parse operation metrics
	parseMetrics := metrics.durations["parse"]
	if parseMetrics.Count != 3 {
		t.Errorf("Expected 3 parse operations, got %d", parseMetrics.Count)
	}
	if parseMetrics.Min != 10*time.Millisecond {
		t.Errorf("Expected min 10ms, got %v", parseMetrics.Min)
	}
	if parseMetrics.Max != 30*time.Millisecond {
		t.Errorf("Expected max 30ms, got %v", parseMetrics.Max)
	}
	if parseMetrics.Average != 20*time.Millisecond {
		t.Errorf("Expected average 20ms, got %v", parseMetrics.Average)
	}

	// Check generate operation metrics
	genMetrics := metrics.durations["generate"]
	if genMetrics.Count != 2 {
		t.Errorf("Expected 2 generate operations, got %d", genMetrics.Count)
	}
	if genMetrics.Average != 150*time.Millisecond {
		t.Errorf("Expected average 150ms, got %v", genMetrics.Average)
	}
}

func TestGetAllStats(t *testing.T) {
	metrics := &MetricsCollector{
		apiCalls:      make(map[string]*APIMetrics),
		tokensByModel: make(map[string]int64),
		durations:     make(map[string]*DurationMetrics),
		startTime:     time.Now(),
	}

	// Add some data
	metrics.RecordAPICall("/v1/chat", 100*time.Millisecond, http.StatusOK, nil)
	metrics.RecordCacheHit()
	metrics.RecordTokenUsage(TokenUsage{
		Model:       "gpt-4",
		TotalTokens: 100,
	})

	// Get all stats
	allStats := metrics.GetAllStats()

	// Verify structure
	if _, ok := allStats["api"]; !ok {
		t.Error("Expected 'api' key in all stats")
	}
	if _, ok := allStats["cache"]; !ok {
		t.Error("Expected 'cache' key in all stats")
	}
	if _, ok := allStats["tokens"]; !ok {
		t.Error("Expected 'tokens' key in all stats")
	}
	if _, ok := allStats["memory"]; !ok {
		t.Error("Expected 'memory' key in all stats")
	}
	if _, ok := allStats["performance"]; !ok {
		t.Error("Expected 'performance' key in all stats")
	}
	if _, ok := allStats["timestamp"]; !ok {
		t.Error("Expected 'timestamp' key in all stats")
	}
}

func TestReset(t *testing.T) {
	metrics := &MetricsCollector{
		apiCalls:      make(map[string]*APIMetrics),
		tokensByModel: make(map[string]int64),
		durations:     make(map[string]*DurationMetrics),
		startTime:     time.Now(),
	}

	// Add some data
	metrics.RecordAPICall("/v1/chat", 100*time.Millisecond, http.StatusOK, nil)
	metrics.RecordCacheHit()
	metrics.RecordTokenUsage(TokenUsage{
		Model:       "gpt-4",
		TotalTokens: 100,
	})

	// Reset
	metrics.Reset()

	// Verify everything is reset
	if metrics.totalAPICalls != 0 {
		t.Error("Expected API calls to be reset")
	}
	if metrics.cacheHits != 0 {
		t.Error("Expected cache hits to be reset")
	}
	if metrics.totalTokensUsed != 0 {
		t.Error("Expected token usage to be reset")
	}
	if len(metrics.apiCalls) != 0 {
		t.Error("Expected API calls map to be reset")
	}
	if len(metrics.tokensByModel) != 0 {
		t.Error("Expected tokens by model map to be reset")
	}
}

func TestMemoryStats(t *testing.T) {
	metrics := &MetricsCollector{
		apiCalls:      make(map[string]*APIMetrics),
		tokensByModel: make(map[string]int64),
		durations:     make(map[string]*DurationMetrics),
		startTime:     time.Now(),
	}

	stats := metrics.GetMemoryStats()

	// Verify memory stats structure
	if _, ok := stats["alloc_mb"]; !ok {
		t.Error("Expected 'alloc_mb' in memory stats")
	}
	if _, ok := stats["total_alloc_mb"]; !ok {
		t.Error("Expected 'total_alloc_mb' in memory stats")
	}
	if _, ok := stats["sys_mb"]; !ok {
		t.Error("Expected 'sys_mb' in memory stats")
	}
	if _, ok := stats["num_gc"]; !ok {
		t.Error("Expected 'num_gc' in memory stats")
	}
	if _, ok := stats["goroutines"]; !ok {
		t.Error("Expected 'goroutines' in memory stats")
	}
}

func TestPerformanceStats(t *testing.T) {
	metrics := &MetricsCollector{
		apiCalls:      make(map[string]*APIMetrics),
		tokensByModel: make(map[string]int64),
		durations:     make(map[string]*DurationMetrics),
		startTime:     time.Now().Add(-1 * time.Hour), // Started 1 hour ago
	}

	// Add some duration data
	metrics.RecordDuration("operation1", 100*time.Millisecond)

	stats := metrics.GetPerformanceStats()

	// Verify structure
	if _, ok := stats["operations"]; !ok {
		t.Error("Expected 'operations' in performance stats")
	}
	if _, ok := stats["uptime_seconds"]; !ok {
		t.Error("Expected 'uptime_seconds' in performance stats")
	}

	// Check uptime is reasonable (should be around 3600 seconds)
	uptime := stats["uptime_seconds"].(float64)
	if uptime < 3599 || uptime > 3601 {
		t.Errorf("Expected uptime around 3600 seconds, got %f", uptime)
	}
}

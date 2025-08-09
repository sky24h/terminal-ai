# Caching System Documentation

## Overview

The terminal-ai caching system provides intelligent response caching to improve performance and reduce API costs. It implements an LRU (Least Recently Used) cache with TTL (Time To Live) support, thread-safe operations, and optional persistence.

## Features

### Core Features
- **LRU Eviction**: Automatically removes least recently used entries when cache is full
- **TTL Support**: Entries expire after a configurable time period
- **Size-based Limits**: Configure maximum cache size in MB
- **Thread-safe Operations**: Safe for concurrent access with read/write locks
- **Persistence**: Optional disk persistence for cache survival across restarts
- **Cache Statistics**: Track hits, misses, evictions, and hit rate
- **Pattern Invalidation**: Remove cache entries matching specific patterns
- **Cache Warming**: Preload cache with frequently used responses

### Performance Benefits
- Instant response for cached queries (sub-millisecond)
- Reduced API costs by avoiding redundant calls
- Lower latency for repeated queries
- Configurable cache strategies (LRU, FIFO, LFU)

## Configuration

Configure caching in your `config.yaml`:

```yaml
cache:
  enabled: true           # Enable/disable caching
  ttl: 5m                # Time to live for cache entries
  max_size: 100          # Maximum cache size in MB
  strategy: lru          # Eviction strategy: lru, fifo, lfu
  dir: ~/.terminal-ai/cache  # Directory for persistent cache
```

### Environment Variables

You can also configure caching via environment variables:

```bash
export TERMINAL_AI_CACHE_ENABLED=true
export TERMINAL_AI_CACHE_TTL=10m
export TERMINAL_AI_CACHE_MAX_SIZE=200
export TERMINAL_AI_CACHE_STRATEGY=lru
export TERMINAL_AI_CACHE_DIR=/path/to/cache
```

## Usage

### Command Line

```bash
# Show cache statistics
terminal-ai cache --stats

# Clear all cached responses
terminal-ai cache --clear

# Invalidate entries matching a pattern
terminal-ai cache --invalidate "chat_"
```

### Programmatic Usage

```go
// Create cache with configuration
cache := ai.NewInMemoryCache(config)
defer cache.Close()

// Generate cache key
key := cache.GenerateKey("What is AI?")

// Store response in cache
entry := &ai.CacheEntry{
    Response: response,
    TokenUsage: usage,
    CreatedAt: time.Now(),
    LastAccessedAt: time.Now(),
}
cache.Set(key, entry, 5*time.Minute)

// Retrieve from cache
if cached, found := cache.Get(key); found {
    return cached.Response
}
```

## Cache Entry Structure

Each cache entry contains:

```go
type CacheEntry struct {
    Response         *Response     // The AI response
    PromptHash       string       // Hash of the prompt
    TokenUsage       Usage        // Token usage metrics
    CreatedAt        time.Time    // When entry was created
    LastAccessedAt   time.Time    // Last access time
    ExpiresAt        time.Time    // Expiration time
    AccessCount      int64        // Number of accesses
    SizeBytes        int64        // Size in bytes
}
```

## Cache Statistics

The cache provides detailed statistics:

```go
type CacheStats struct {
    Hits          int64    // Number of cache hits
    Misses        int64    // Number of cache misses
    Evictions     int64    // Number of evictions
    Entries       int      // Current number of entries
    SizeBytes     int64    // Current size in bytes
    MaxSizeBytes  int64    // Maximum size in bytes
    HitRate       float64  // Hit rate (hits / total requests)
    LastCleanup   time.Time // Last cleanup time
}
```

## Implementation Details

### Cache Key Generation

Cache keys are generated using SHA-256 hashing:
- For simple queries: Hash of the prompt text
- For chat: Hash of messages array and chat options

### LRU Algorithm

The cache uses a doubly-linked list for O(1) LRU operations:
1. New entries are added to the front
2. Accessed entries are moved to the front
3. Eviction removes from the back (least recently used)

### Thread Safety

All cache operations are protected by read/write mutex:
- Multiple readers can access simultaneously
- Writers have exclusive access
- Cleanup runs in a separate goroutine

### Persistence

When configured with a cache directory:
1. Cache is saved to disk periodically
2. Cache is loaded on startup
3. Uses GOB encoding for efficient serialization
4. Atomic file operations prevent corruption

## Best Practices

### When to Use Caching

✅ **Good for:**
- Repeated queries with identical parameters
- Reference lookups (definitions, facts)
- Development and testing environments
- Rate-limited scenarios

❌ **Not suitable for:**
- Streaming responses
- Real-time or time-sensitive queries
- Queries requiring latest information
- Creative content generation

### Cache Sizing

Calculate appropriate cache size:
```
Cache Size (MB) = (Avg Response Size × Expected Entries) / 1024 / 1024
```

Example for 1000 entries averaging 5KB each:
```
(5KB × 1000) / 1024 / 1024 ≈ 5MB
```

### TTL Configuration

Choose TTL based on use case:
- **Short (1-5 minutes)**: Dynamic content, development
- **Medium (5-30 minutes)**: General queries, Q&A
- **Long (1-24 hours)**: Reference data, definitions

## Monitoring

Monitor cache performance with metrics:

```bash
# View cache statistics
terminal-ai cache --stats

# Output:
Cache Statistics
================

Performance:
  Hit Rate:     75.3%
  Total Hits:   1523
  Total Misses: 501

Storage:
  Entries:      245
  Size:         12.5 MB / 100.0 MB
  Usage:        12.5%

Evictions:
  Total:        23
```

## Troubleshooting

### Cache Not Working

1. Verify cache is enabled in configuration
2. Check cache directory permissions (if using persistence)
3. Ensure sufficient memory/disk space
4. Review logs for cache-related errors

### Low Hit Rate

1. Increase cache size if evictions are high
2. Adjust TTL for longer retention
3. Consider query normalization
4. Review access patterns

### Memory Issues

1. Reduce max_size configuration
2. Decrease TTL for shorter retention
3. Clear cache periodically
4. Monitor system memory usage

## Examples

### Basic Query Caching

```go
// Query with automatic caching
response, err := client.Query(ctx, "What is Go?")
// First call: hits API
// Subsequent calls: retrieved from cache
```

### Chat with Caching

```go
messages := []ai.Message{
    {Role: "system", Content: "You are helpful"},
    {Role: "user", Content: "Hello"},
}

options := ai.ChatOptions{
    Model: "gpt-5-mini",
    Temperature: 1.0,  // Required for reasoning models
}

// Chat with automatic caching
response, err := client.Chat(ctx, messages, options)
```

### Cache Warming

```go
// Preload common queries
commonQueries := map[string]string{
    "help": "Show available commands...",
    "version": "Current version is...",
}

for prompt, response := range commonQueries {
    entry := &ai.CacheEntry{
        Response: &ai.Response{Content: response},
        CreatedAt: time.Now(),
    }
    cache.Set(cache.GenerateKey(prompt), entry, 1*time.Hour)
}
```

## Performance Benchmarks

Typical performance improvements with caching:

| Operation | Without Cache | With Cache | Improvement |
|-----------|--------------|------------|-------------|
| Simple Query | 1.5s | 0.001s | 1500x |
| Chat Request | 2.0s | 0.002s | 1000x |
| Batch Queries (100) | 150s | 0.1s | 1500x |

## Security Considerations

1. **Sensitive Data**: Be cautious caching sensitive responses
2. **Cache Poisoning**: Validate cached data integrity
3. **Disk Persistence**: Secure cache directory permissions
4. **API Keys**: Never cache API keys or credentials

## Future Enhancements

Planned improvements:
- Distributed caching support (Redis)
- Compression for larger entries
- Smart invalidation based on content
- Cache analytics and insights
- Query normalization for better hit rates
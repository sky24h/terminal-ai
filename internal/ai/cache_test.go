package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/terminal-ai/internal/config"
)

func TestInMemoryCache_BasicOperations(t *testing.T) {
	cfg := &config.CacheConfig{
		Enabled:  true,
		TTL:      5 * time.Minute,
		MaxSize:  10, // 10 MB
		Strategy: "lru",
	}

	cache := NewInMemoryCache(cfg)
	defer cache.Close()

	// Test Set and Get
	entry := &CacheEntry{
		Response: &Response{
			Content: "Test response",
			Model:   "gpt-3.5-turbo",
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		},
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		AccessCount:    1,
	}

	key := cache.GenerateKey("test prompt")
	err := cache.Set(key, entry, 5*time.Minute)
	require.NoError(t, err)

	// Test Get
	cached, found := cache.Get(key)
	assert.True(t, found)
	assert.NotNil(t, cached)
	assert.Equal(t, "Test response", cached.Response.Content)
	assert.Equal(t, int64(2), cached.AccessCount) // Should be incremented

	// Test Stats
	stats := cache.Stats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, 1, stats.Entries)

	// Test Delete
	err = cache.Delete(key)
	assert.NoError(t, err)

	// Verify deletion
	_, found = cache.Get(key)
	assert.False(t, found)
}

func TestInMemoryCache_TTL(t *testing.T) {
	cfg := &config.CacheConfig{
		Enabled:  true,
		TTL:      100 * time.Millisecond,
		MaxSize:  10,
		Strategy: "lru",
	}

	cache := NewInMemoryCache(cfg)
	defer cache.Close()

	entry := &CacheEntry{
		Response: &Response{
			Content: "Temporary response",
		},
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	key := cache.GenerateKey("temp prompt")
	err := cache.Set(key, entry, 100*time.Millisecond)
	require.NoError(t, err)

	// Should exist immediately
	cached, found := cache.Get(key)
	assert.True(t, found)
	assert.NotNil(t, cached)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	cached, found = cache.Get(key)
	assert.False(t, found)
	assert.Nil(t, cached)
}

func TestInMemoryCache_LRUEviction(t *testing.T) {
	cfg := &config.CacheConfig{
		Enabled:  true,
		TTL:      5 * time.Minute,
		MaxSize:  1, // 1 MB cache
		Strategy: "lru",
	}

	cache := NewInMemoryCache(cfg)
	defer cache.Close()

	// Create entries that fit within 1MB but will trigger eviction
	entry1 := &CacheEntry{
		Response: &Response{
			Content: string(make([]byte, 400*1024)), // 400KB
			Model:   "gpt-3.5-turbo",
		},
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	entry2 := &CacheEntry{
		Response: &Response{
			Content: string(make([]byte, 400*1024)), // 400KB
			Model:   "gpt-3.5-turbo",
		},
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	entry3 := &CacheEntry{
		Response: &Response{
			Content: string(make([]byte, 400*1024)), // 400KB
			Model:   "gpt-3.5-turbo",
		},
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	// Add first entry
	key1 := cache.GenerateKey("prompt1")
	err := cache.Set(key1, entry1, 5*time.Minute)
	require.NoError(t, err)

	// Add second entry
	key2 := cache.GenerateKey("prompt2")
	err = cache.Set(key2, entry2, 5*time.Minute)
	require.NoError(t, err)

	// Add third entry - should evict the first (LRU)
	key3 := cache.GenerateKey("prompt3")
	err = cache.Set(key3, entry3, 5*time.Minute)
	require.NoError(t, err)

	// First entry should be evicted (it's the least recently used)
	_, found := cache.Get(key1)
	assert.False(t, found)

	// Second and third entries should exist
	_, found = cache.Get(key2)
	assert.True(t, found)
	_, found = cache.Get(key3)
	assert.True(t, found)

	// Check eviction stats
	stats := cache.Stats()
	assert.Greater(t, stats.Evictions, int64(0))
}

func TestInMemoryCache_GenerateChatKey(t *testing.T) {
	cfg := &config.CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 10,
	}

	cache := NewInMemoryCache(cfg)
	defer cache.Close()

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
	}

	options := ChatOptions{
		Model:       "gpt-3.5-turbo",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	key1 := cache.GenerateChatKey(messages, options)
	key2 := cache.GenerateChatKey(messages, options)

	// Same input should generate same key
	assert.Equal(t, key1, key2)

	// Different input should generate different key
	options.Temperature = 0.8
	key3 := cache.GenerateChatKey(messages, options)
	assert.NotEqual(t, key1, key3)
}

func TestInMemoryCache_Clear(t *testing.T) {
	cfg := &config.CacheConfig{
		Enabled:  true,
		TTL:      5 * time.Minute,
		MaxSize:  10,
		Strategy: "lru",
	}

	cache := NewInMemoryCache(cfg)
	defer cache.Close()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Response: &Response{
				Content: "Response " + string(rune(i)),
			},
			CreatedAt:      time.Now(),
			LastAccessedAt: time.Now(),
		}
		key := cache.GenerateKey("prompt" + string(rune(i)))
		err := cache.Set(key, entry, 5*time.Minute)
		require.NoError(t, err)
	}

	// Verify entries exist
	stats := cache.Stats()
	assert.Equal(t, 5, stats.Entries)

	// Clear cache
	err := cache.Clear()
	assert.NoError(t, err)

	// Verify cache is empty
	stats = cache.Stats()
	assert.Equal(t, 0, stats.Entries)
	assert.Equal(t, int64(0), stats.SizeBytes)
}

func TestInMemoryCache_InvalidatePattern(t *testing.T) {
	cfg := &config.CacheConfig{
		Enabled:  true,
		TTL:      5 * time.Minute,
		MaxSize:  10,
		Strategy: "lru",
	}

	cache := NewInMemoryCache(cfg)
	defer cache.Close()

	// Add entries with similar keys
	prefixes := []string{"chat_", "query_", "chat_"}
	for i, prefix := range prefixes {
		entry := &CacheEntry{
			Response: &Response{
				Content: "Response " + string(rune(i)),
			},
			CreatedAt:      time.Now(),
			LastAccessedAt: time.Now(),
		}
		// Use a simple key for pattern matching
		key := prefix + string(rune(i))
		cache.entries[key] = cache.lruList.PushFront(&lruNode{
			key:   key,
			entry: entry,
		})
	}

	// Invalidate all entries starting with "chat_"
	count, err := cache.InvalidatePattern("chat_")
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	// Verify only query_ entry remains
	assert.Equal(t, 1, len(cache.entries))
}

func TestInMemoryCache_ConcurrentAccess(t *testing.T) {
	cfg := &config.CacheConfig{
		Enabled:  true,
		TTL:      5 * time.Minute,
		MaxSize:  10,
		Strategy: "lru",
	}

	cache := NewInMemoryCache(cfg)
	defer cache.Close()

	// Run concurrent operations
	done := make(chan bool)

	// Writer
	go func() {
		for i := 0; i < 100; i++ {
			entry := &CacheEntry{
				Response: &Response{
					Content: "Response",
				},
				CreatedAt:      time.Now(),
				LastAccessedAt: time.Now(),
			}
			key := cache.GenerateKey("prompt" + string(rune(i%10)))
			cache.Set(key, entry, 5*time.Minute)
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 100; i++ {
			key := cache.GenerateKey("prompt" + string(rune(i%10)))
			cache.Get(key)
		}
		done <- true
	}()

	// Stats reader
	go func() {
		for i := 0; i < 50; i++ {
			cache.Stats()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify cache is still functional
	stats := cache.Stats()
	assert.Greater(t, stats.Entries, 0)
}

func TestInMemoryCache_Warm(t *testing.T) {
	cfg := &config.CacheConfig{
		Enabled:  true,
		TTL:      5 * time.Minute,
		MaxSize:  10,
		Strategy: "lru",
	}

	cache := NewInMemoryCache(cfg)
	defer cache.Close()

	// Prepare entries for warming
	warmEntries := make(map[string]*CacheEntry)
	for i := 0; i < 3; i++ {
		entry := &CacheEntry{
			Response: &Response{
				Content: "Warm response " + string(rune(i)),
			},
			CreatedAt:      time.Now(),
			LastAccessedAt: time.Now(),
		}
		key := cache.GenerateKey("warm" + string(rune(i)))
		warmEntries[key] = entry
	}

	// Warm the cache
	err := cache.Warm(warmEntries)
	assert.NoError(t, err)

	// Verify all entries were loaded
	stats := cache.Stats()
	assert.Equal(t, 3, stats.Entries)

	// Verify entries are accessible
	for key := range warmEntries {
		cached, found := cache.Get(key)
		assert.True(t, found)
		assert.NotNil(t, cached)
	}
}

package ai

import (
	"container/list"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/user/terminal-ai/internal/config"
)

// Cache defines the interface for caching implementations
type Cache interface {
	// Get retrieves a cached response by key
	Get(key string) (*CacheEntry, bool)
	// Set stores a response in cache with TTL
	Set(key string, entry *CacheEntry, ttl time.Duration) error
	// Delete removes a specific cache entry
	Delete(key string) error
	// Clear removes all entries from cache
	Clear() error
	// Stats returns cache statistics
	Stats() *CacheStats
	// GenerateKey generates a cache key from prompt
	GenerateKey(prompt string) string
	// GenerateChatKey generates a cache key from messages and options
	GenerateChatKey(messages []Message, options ChatOptions) string
	// Close gracefully shuts down the cache
	Close() error
}

// CacheEntry represents a cached response with metadata
type CacheEntry struct {
	Response       *Response `json:"response"`
	PromptHash     string    `json:"prompt_hash"`
	TokenUsage     Usage     `json:"token_usage"`
	CreatedAt      time.Time `json:"created_at"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	AccessCount    int64     `json:"access_count"`
	SizeBytes      int64     `json:"size_bytes"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits         int64     `json:"hits"`
	Misses       int64     `json:"misses"`
	Evictions    int64     `json:"evictions"`
	Entries      int       `json:"entries"`
	SizeBytes    int64     `json:"size_bytes"`
	MaxSizeBytes int64     `json:"max_size_bytes"`
	HitRate      float64   `json:"hit_rate"`
	LastCleanup  time.Time `json:"last_cleanup"`
}

// lruNode represents a node in the LRU list
type lruNode struct {
	key       string
	entry     *CacheEntry
	sizeBytes int64
}

// InMemoryCache implements an LRU cache with TTL support
type InMemoryCache struct {
	mu           sync.RWMutex
	entries      map[string]*list.Element // map of key to LRU list element
	lruList      *list.List               // LRU list
	maxSizeBytes int64                    // maximum cache size in bytes
	currentSize  int64                    // current cache size in bytes
	ttl          time.Duration            // default TTL
	stats        *CacheStats              // cache statistics
	config       *config.CacheConfig      // cache configuration
	persistPath  string                   // path for persistence
	stopCleanup  chan struct{}            // signal to stop cleanup goroutine
	wg           sync.WaitGroup           // wait group for goroutines
}

// NewInMemoryCache creates a new in-memory LRU cache
func NewInMemoryCache(cfg *config.CacheConfig) *InMemoryCache {
	maxSizeBytes := int64(cfg.MaxSize * 1024 * 1024) // Convert MB to bytes

	cache := &InMemoryCache{
		entries:      make(map[string]*list.Element),
		lruList:      list.New(),
		maxSizeBytes: maxSizeBytes,
		currentSize:  0,
		ttl:          cfg.TTL,
		config:       cfg,
		stopCleanup:  make(chan struct{}),
		stats: &CacheStats{
			MaxSizeBytes: maxSizeBytes,
		},
	}

	// Set persistence path if configured
	if cfg.Dir != "" {
		cache.persistPath = filepath.Join(cfg.Dir, "cache.gob")
		// Try to load existing cache
		if err := cache.Load(); err != nil {
			log.Debug().Err(err).Msg("Failed to load cache from disk")
		}
	}

	// Start cleanup goroutine
	cache.wg.Add(1)
	go cache.cleanupRoutine()

	return cache
}

// Get retrieves a cached entry and updates LRU
func (c *InMemoryCache) Get(key string) (*CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	node := elem.Value.(*lruNode)
	entry := node.entry

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		c.removeLocked(key)
		c.stats.Misses++
		return nil, false
	}

	// Update access metadata
	entry.LastAccessedAt = time.Now()
	entry.AccessCount++

	// Move to front of LRU list
	c.lruList.MoveToFront(elem)

	c.stats.Hits++
	return entry, true
}

// Set stores an entry in cache with TTL
func (c *InMemoryCache) Set(key string, entry *CacheEntry, ttl time.Duration) error {
	if entry == nil {
		return errors.New("cannot cache nil entry")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Calculate entry size
	entrySize := c.calculateEntrySize(entry)
	entry.SizeBytes = entrySize

	// Check if entry exceeds max cache size
	if entrySize > c.maxSizeBytes {
		return fmt.Errorf("entry size %d exceeds max cache size %d", entrySize, c.maxSizeBytes)
	}

	// If key already exists, remove old entry
	if elem, exists := c.entries[key]; exists {
		c.removeElementLocked(elem)
	}

	// Evict entries until we have enough space
	for c.currentSize+entrySize > c.maxSizeBytes && c.lruList.Len() > 0 {
		c.evictLRULocked()
	}

	// Set TTL
	if ttl == 0 {
		ttl = c.ttl
	}
	entry.ExpiresAt = time.Now().Add(ttl)
	entry.PromptHash = key

	// Add to cache
	node := &lruNode{
		key:       key,
		entry:     entry,
		sizeBytes: entrySize,
	}
	elem := c.lruList.PushFront(node)
	c.entries[key] = elem
	c.currentSize += entrySize

	// Update stats
	c.stats.Entries = len(c.entries)
	c.stats.SizeBytes = c.currentSize

	return nil
}

// Delete removes a specific cache entry
func (c *InMemoryCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.removeLocked(key) {
		return fmt.Errorf("key %s not found in cache", key)
	}

	return nil
}

// Clear removes all entries from cache
func (c *InMemoryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*list.Element)
	c.lruList.Init()
	c.currentSize = 0

	// Reset stats
	c.stats.Entries = 0
	c.stats.SizeBytes = 0
	c.stats.Evictions = 0

	// Clear persistent cache if configured
	if c.persistPath != "" {
		os.Remove(c.persistPath)
	}

	return nil
}

// Stats returns cache statistics
func (c *InMemoryCache) Stats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := *c.stats
	stats.Entries = len(c.entries)
	stats.SizeBytes = c.currentSize

	if totalRequests := stats.Hits + stats.Misses; totalRequests > 0 {
		stats.HitRate = float64(stats.Hits) / float64(totalRequests)
	}

	return &stats
}

// GenerateKey generates a cache key from prompt
func (c *InMemoryCache) GenerateKey(prompt string) string {
	hash := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(hash[:])
}

// GenerateChatKey generates a cache key from messages and options
func (c *InMemoryCache) GenerateChatKey(messages []Message, options ChatOptions) string {
	data := struct {
		Messages []Message
		Options  ChatOptions
	}{
		Messages: messages,
		Options:  options,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// Close gracefully shuts down the cache
func (c *InMemoryCache) Close() error {
	// Signal cleanup goroutine to stop
	close(c.stopCleanup)
	c.wg.Wait()

	// Save cache to disk if configured
	if c.persistPath != "" {
		return c.Save()
	}

	return nil
}

// Save persists the cache to disk
func (c *InMemoryCache) Save() error {
	if c.persistPath == "" {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(c.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create temporary file
	tempFile := c.persistPath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	// Encode cache data
	data := make(map[string]*CacheEntry)
	for key, elem := range c.entries {
		node := elem.Value.(*lruNode)
		data[key] = node.entry
	}

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to encode cache data: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, c.persistPath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to save cache file: %w", err)
	}

	return nil
}

// Load restores the cache from disk
func (c *InMemoryCache) Load() error {
	if c.persistPath == "" {
		return nil
	}

	file, err := os.Open(c.persistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file exists
		}
		return fmt.Errorf("failed to open cache file: %w", err)
	}
	defer file.Close()

	var data map[string]*CacheEntry
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode cache data: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Load entries that haven't expired
	now := time.Now()
	for key, entry := range data {
		if now.Before(entry.ExpiresAt) {
			node := &lruNode{
				key:       key,
				entry:     entry,
				sizeBytes: entry.SizeBytes,
			}
			elem := c.lruList.PushBack(node) // Add to back (oldest)
			c.entries[key] = elem
			c.currentSize += entry.SizeBytes
		}
	}

	c.stats.Entries = len(c.entries)
	c.stats.SizeBytes = c.currentSize

	log.Info().
		Int("entries_loaded", len(c.entries)).
		Int64("size_bytes", c.currentSize).
		Msg("Cache loaded from disk")

	return nil
}

// Warm preloads the cache with specified entries
func (c *InMemoryCache) Warm(entries map[string]*CacheEntry) error {
	for key, entry := range entries {
		if err := c.Set(key, entry, 0); err != nil {
			log.Warn().
				Err(err).
				Str("key", key).
				Msg("Failed to warm cache entry")
		}
	}
	return nil
}

// InvalidatePattern removes entries matching a pattern
func (c *InMemoryCache) InvalidatePattern(pattern string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key := range c.entries {
		// Simple prefix matching for now
		if len(pattern) > 0 && len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			if c.removeLocked(key) {
				count++
			}
		}
	}

	return count, nil
}

// Helper methods

// removeLocked removes an entry (must be called with lock held)
func (c *InMemoryCache) removeLocked(key string) bool {
	elem, exists := c.entries[key]
	if !exists {
		return false
	}

	c.removeElementLocked(elem)
	return true
}

// removeElementLocked removes an element from cache (must be called with lock held)
func (c *InMemoryCache) removeElementLocked(elem *list.Element) {
	node := elem.Value.(*lruNode)
	c.lruList.Remove(elem)
	delete(c.entries, node.key)
	c.currentSize -= node.sizeBytes
}

// evictLRULocked evicts the least recently used entry (must be called with lock held)
func (c *InMemoryCache) evictLRULocked() {
	elem := c.lruList.Back()
	if elem != nil {
		c.removeElementLocked(elem)
		c.stats.Evictions++
	}
}

// calculateEntrySize estimates the size of a cache entry in bytes
func (c *InMemoryCache) calculateEntrySize(entry *CacheEntry) int64 {
	// Estimate based on response content and metadata
	size := int64(0)

	if entry.Response != nil {
		size += int64(len(entry.Response.Content))
		size += int64(len(entry.Response.Model))
		size += int64(len(entry.Response.ID))
		size += 100 // Overhead for other fields
	}

	// Add metadata overhead
	size += int64(len(entry.PromptHash))
	size += 200 // Fixed overhead for timestamps and counters

	return size
}

// cleanupRoutine periodically removes expired entries
func (c *InMemoryCache) cleanupRoutine() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries
func (c *InMemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiredKeys := []string{}

	// Find expired entries
	for key, elem := range c.entries {
		node := elem.Value.(*lruNode)
		if now.After(node.entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Remove expired entries
	for _, key := range expiredKeys {
		c.removeLocked(key)
	}

	c.stats.LastCleanup = now

	if len(expiredKeys) > 0 {
		log.Debug().
			Int("expired_entries", len(expiredKeys)).
			Int("remaining_entries", len(c.entries)).
			Msg("Cache cleanup completed")
	}

	// Save to disk periodically if configured
	if c.persistPath != "" && len(c.entries) > 0 {
		if err := c.Save(); err != nil {
			log.Error().Err(err).Msg("Failed to save cache to disk")
		}
	}
}

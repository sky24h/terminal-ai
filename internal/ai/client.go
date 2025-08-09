package ai

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/shared"
	"github.com/rs/zerolog/log"
	"github.com/user/terminal-ai/internal/config"
)

// Client represents an AI client interface
type Client interface {
	// Query sends a simple text query and returns the response
	Query(ctx context.Context, prompt string) (string, error)
	// StreamQuery sends a query and streams the response token by token
	StreamQuery(ctx context.Context, prompt string, callback func(chunk string)) error
	// Chat sends a chat request with message history
	Chat(ctx context.Context, messages []Message, options ChatOptions) (*Response, error)
	// ChatStream sends a chat request and streams the response
	ChatStream(ctx context.Context, messages []Message, options ChatOptions) (<-chan StreamChunk, error)
	// ListModels lists available models
	ListModels(ctx context.Context) ([]string, error)
	// Close cleans up resources
	Close() error
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
	Name    string `json:"name,omitempty"` // Optional name for the message author
}

// ChatOptions contains options for chat requests
type ChatOptions struct {
	Model            string   `json:"model"`
	Temperature      float32  `json:"temperature"`
	MaxTokens        int      `json:"max_tokens"`
	TopP             float32  `json:"top_p"`
	N                int      `json:"n,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	PresencePenalty  float32  `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32  `json:"frequency_penalty,omitempty"`
	User             string   `json:"user,omitempty"`
	ReasoningEffort  string   `json:"reasoning_effort,omitempty"` // For reasoning models: low, medium, high
}

// Response represents an AI response
type Response struct {
	Content      string    `json:"content"`
	Model        string    `json:"model"`
	Usage        Usage     `json:"usage"`
	FinishReason string    `json:"finish_reason"`
	Created      time.Time `json:"created"`
	ID           string    `json:"id,omitempty"`
	Object       string    `json:"object,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a chunk of streamed response
type StreamChunk struct {
	Content string
	Error   error
	Done    bool
}

// OpenAIClient implements Client interface for OpenAI
type OpenAIClient struct {
	client        openai.Client
	config        *config.Config
	httpClient    *http.Client
	streamHandler *StreamHandler
	rateLimiter   *RateLimiter
	retryConfig   RetryConfig
	cache         Cache
	mu            sync.RWMutex
	closed        bool
}

// RetryConfig contains retry configuration
type RetryConfig struct {
	MaxRetries         int
	InitialDelay       time.Duration
	MaxDelay           time.Duration
	Multiplier         float64
	RetryableHTTPCodes []int
}

// RateLimiter implements rate limiting for API calls
type RateLimiter struct {
	mu              sync.Mutex
	lastRequestTime time.Time
	minInterval     time.Duration
	requestsPerMin  int
	requestCount    int
	windowStart     time.Time
}

// NewOpenAIClient creates a new OpenAI client with configuration
func NewOpenAIClient(cfg *config.Config) (*OpenAIClient, error) {
	if cfg.OpenAI.APIKey == "" {
		return nil, errors.New("OpenAI API key is required")
	}

	// Create custom HTTP client with connection pooling
	httpClient := &http.Client{
		Timeout: cfg.OpenAI.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}

	// Create OpenAI client with options
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.OpenAI.APIKey),
		option.WithHTTPClient(httpClient),
	}

	if cfg.OpenAI.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.OpenAI.BaseURL))
	}
	if cfg.OpenAI.OrgID != "" {
		opts = append(opts, option.WithHeader("OpenAI-Organization", cfg.OpenAI.OrgID))
	}

	openaiClient := openai.NewClient(opts...)

	// Create rate limiter (60 requests per minute by default)
	rateLimiter := &RateLimiter{
		minInterval:    time.Second, // Minimum 1 second between requests
		requestsPerMin: 60,
		windowStart:    time.Now(),
	}

	// Setup retry configuration
	retryConfig := RetryConfig{
		MaxRetries:         3,
		InitialDelay:       time.Second,
		MaxDelay:           30 * time.Second,
		Multiplier:         2.0,
		RetryableHTTPCodes: []int{429, 500, 502, 503, 504},
	}

	client := &OpenAIClient{
		client:        openaiClient,
		config:        cfg,
		httpClient:    httpClient,
		streamHandler: NewStreamHandler(openaiClient),
		rateLimiter:   rateLimiter,
		retryConfig:   retryConfig,
	}

	// Initialize cache if enabled
	if cfg.Cache.Enabled {
		client.cache = NewInMemoryCache(&cfg.Cache)
		log.Info().
			Bool("enabled", true).
			Str("strategy", cfg.Cache.Strategy).
			Int("max_size_mb", cfg.Cache.MaxSize).
			Dur("ttl", cfg.Cache.TTL).
			Msg("Cache initialized")
	}

	return client, nil
}

// Query sends a simple text query and returns the response
func (c *OpenAIClient) Query(ctx context.Context, prompt string) (string, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return "", errors.New("client is closed")
	}
	c.mu.RUnlock()

	// Check cache first if enabled
	if c.cache != nil {
		cacheKey := c.cache.GenerateKey(prompt)
		if cached, found := c.cache.Get(cacheKey); found {
			log.Debug().
				Str("key", cacheKey[:8]).
				Int64("access_count", cached.AccessCount).
				Msg("Cache hit for query")
			return cached.Response.Content, nil
		}
	}

	// Create messages for the query
	messages := []Message{
		{Role: "user", Content: prompt},
	}

	// Use default options from config
	options := ChatOptions{
		Model:           c.config.OpenAI.Model,
		Temperature:     c.config.OpenAI.Temperature,
		MaxTokens:       c.config.OpenAI.MaxTokens,
		TopP:            c.config.OpenAI.TopP,
		ReasoningEffort: c.config.OpenAI.ReasoningEffort,
	}

	resp, err := c.Chat(ctx, messages, options)
	if err != nil {
		return "", err
	}

	// Cache the response if caching is enabled
	if c.cache != nil && resp != nil {
		cacheKey := c.cache.GenerateKey(prompt)
		entry := &CacheEntry{
			Response:       resp,
			TokenUsage:     resp.Usage,
			CreatedAt:      time.Now(),
			LastAccessedAt: time.Now(),
			AccessCount:    1,
		}
		if err := c.cache.Set(cacheKey, entry, c.config.Cache.TTL); err != nil {
			log.Warn().Err(err).Msg("Failed to cache response")
		}
	}

	return resp.Content, nil
}

// StreamQuery sends a query and streams the response token by token
func (c *OpenAIClient) StreamQuery(ctx context.Context, prompt string, callback func(chunk string)) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return errors.New("client is closed")
	}
	c.mu.RUnlock()

	// Create messages for the query
	messages := []Message{
		{Role: "user", Content: prompt},
	}

	// Use default options from config
	options := ChatOptions{
		Model:           c.config.OpenAI.Model,
		Temperature:     c.config.OpenAI.Temperature,
		MaxTokens:       c.config.OpenAI.MaxTokens,
		TopP:            c.config.OpenAI.TopP,
		ReasoningEffort: c.config.OpenAI.ReasoningEffort,
	}

	// Convert to OpenAI messages
	openaiMessages := c.convertMessages(messages)

	// Process stream with callback
	return c.streamHandler.ProcessStreamWithCallback(ctx, openaiMessages, options, func(chunk string) error {
		callback(chunk)
		return nil
	})
}

// Chat sends a chat request and returns the response with retry logic
func (c *OpenAIClient) Chat(ctx context.Context, messages []Message, options ChatOptions) (*Response, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, errors.New("client is closed")
	}
	c.mu.RUnlock()

	// Check cache first if enabled
	if c.cache != nil {
		cacheKey := c.cache.GenerateChatKey(messages, options)
		if cached, found := c.cache.Get(cacheKey); found {
			log.Debug().
				Str("key", cacheKey[:8]).
				Int64("access_count", cached.AccessCount).
				Msg("Cache hit for chat")
			return cached.Response, nil
		}
	}

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiting error: %w", err)
	}

	// Convert messages to OpenAI format
	openaiMessages := c.convertMessages(messages)

	// Apply defaults if not specified
	if options.Model == "" {
		options.Model = c.config.OpenAI.Model
	}
	if options.N == 0 {
		options.N = c.config.OpenAI.N
	}

	// Create request parameters
	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(options.Model),
		Messages: openaiMessages,
	}

	// Add optional parameters
	if options.Temperature > 0 {
		params.Temperature = openai.Float(float64(options.Temperature))
	}
	if options.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(options.MaxTokens))
	}
	if options.TopP > 0 {
		params.TopP = openai.Float(float64(options.TopP))
	}
	if options.N > 0 {
		params.N = openai.Int(int64(options.N))
	}
	if len(options.Stop) > 0 {
		// Convert stop sequences to the union type
		if len(options.Stop) == 1 {
			params.Stop = openai.ChatCompletionNewParamsStopUnion{
				OfString: openai.String(options.Stop[0]),
			}
		} else {
			params.Stop = openai.ChatCompletionNewParamsStopUnion{
				OfStringArray: options.Stop,
			}
		}
	}
	if options.PresencePenalty != 0 {
		params.PresencePenalty = openai.Float(float64(options.PresencePenalty))
	}
	if options.FrequencyPenalty != 0 {
		params.FrequencyPenalty = openai.Float(float64(options.FrequencyPenalty))
	}
	if options.User != "" {
		params.User = openai.String(options.User)
	}

	// Handle ReasoningEffort for reasoning models
	if config.IsReasoningModel(options.Model) && options.ReasoningEffort != "" {
		switch options.ReasoningEffort {
		case "minimal":
			params.ReasoningEffort = shared.ReasoningEffortMinimal
		case "low":
			params.ReasoningEffort = shared.ReasoningEffortLow
		case "medium":
			params.ReasoningEffort = shared.ReasoningEffortMedium
		case "high":
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			params.ReasoningEffort = shared.ReasoningEffortMinimal
		}
		
		log.Debug().
			Str("model", options.Model).
			Str("reasoning_effort", options.ReasoningEffort).
			Msg("Using reasoning model with effort level")
	}

	var resp *openai.ChatCompletion
	var err error

	// Retry logic with exponential backoff
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		resp, err = c.client.Chat.Completions.New(ctx, params)

		if err == nil {
			break // Success
		}

		// Check if error is retryable
		if !c.isRetryableError(err) {
			log.Error().Err(err).Msg("Non-retryable error in chat completion")
			return nil, err
		}

		if attempt < c.retryConfig.MaxRetries {
			delay := c.calculateBackoff(attempt)
			log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Dur("delay", delay).
				Msg("Retrying chat completion")

			select {
			case <-time.After(delay):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to create chat completion after retries")
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, errors.New("no response choices returned")
	}

	// Convert response
	response := &Response{
		Content:      resp.Choices[0].Message.Content,
		Model:        resp.Model,
		FinishReason: string(resp.Choices[0].FinishReason),
		Created:      time.Unix(resp.Created, 0),
		ID:           resp.ID,
		Object:       string(resp.Object),
	}
	
	// Handle usage - it's a value, not a pointer
	response.Usage = Usage{
		PromptTokens:     int(resp.Usage.PromptTokens),
		CompletionTokens: int(resp.Usage.CompletionTokens),
		TotalTokens:      int(resp.Usage.TotalTokens),
	}

	// Cache the response if caching is enabled
	if c.cache != nil {
		cacheKey := c.cache.GenerateChatKey(messages, options)
		entry := &CacheEntry{
			Response:       response,
			TokenUsage:     response.Usage,
			CreatedAt:      time.Now(),
			LastAccessedAt: time.Now(),
			AccessCount:    1,
		}
		if err := c.cache.Set(cacheKey, entry, c.config.Cache.TTL); err != nil {
			log.Warn().Err(err).Msg("Failed to cache chat response")
		}
	}

	return response, nil
}

// ChatStream sends a chat request and returns a stream of responses
func (c *OpenAIClient) ChatStream(ctx context.Context, messages []Message, options ChatOptions) (<-chan StreamChunk, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, errors.New("client is closed")
	}
	c.mu.RUnlock()

	// Note: Streaming responses are not cached
	if c.cache != nil {
		log.Debug().Msg("Skipping cache for streaming response")
	}

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiting error: %w", err)
	}

	// Convert messages to OpenAI format
	openaiMessages := c.convertMessages(messages)

	// Apply defaults if not specified
	if options.Model == "" {
		options.Model = c.config.OpenAI.Model
	}

	// Delegate to stream handler
	return c.streamHandler.HandleStream(ctx, openaiMessages, options)
}

// ListModels lists available models
func (c *OpenAIClient) ListModels(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, errors.New("client is closed")
	}
	c.mu.RUnlock()

	// Create a page iterator for models
	iter := c.client.Models.ListAutoPaging(ctx)
	
	var modelNames []string
	for iter.Next() {
		model := iter.Current()
		modelNames = append(modelNames, model.ID)
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return modelNames, nil
}

// Close cleans up resources
func (c *OpenAIClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	// Close cache if enabled
	if c.cache != nil {
		if stats := c.cache.Stats(); stats != nil {
			log.Info().
				Int64("hits", stats.Hits).
				Int64("misses", stats.Misses).
				Float64("hit_rate", stats.HitRate).
				Int64("evictions", stats.Evictions).
				Msg("Cache statistics at shutdown")
		}
		if err := c.cache.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close cache")
		}
	}

	// Close HTTP client idle connections
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}

	return nil
}

// convertMessages converts internal messages to OpenAI format
func (c *OpenAIClient) convertMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case "system":
			// Note: The name field is not directly supported in the new API
			openaiMessages[i] = openai.SystemMessage(msg.Content)
		case "user":
			// Note: The name field is not directly supported in the new API
			openaiMessages[i] = openai.UserMessage(msg.Content)
		case "assistant":
			// Note: The name field is not directly supported in the new API
			openaiMessages[i] = openai.AssistantMessage(msg.Content)
		default:
			// Default to user message for unknown roles
			openaiMessages[i] = openai.UserMessage(msg.Content)
		}
	}
	return openaiMessages
}

// isRetryableError checks if an error is retryable
func (c *OpenAIClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific OpenAI API errors
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		// Rate limit errors are retryable
		if apiErr.StatusCode == 429 {
			return true
		}
		// Server errors are retryable
		for _, code := range c.retryConfig.RetryableHTTPCodes {
			if apiErr.StatusCode == code {
				return true
			}
		}
	}

	// Network errors are generally retryable
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	return false
}

// calculateBackoff calculates the backoff duration for a retry attempt
func (c *OpenAIClient) calculateBackoff(attempt int) time.Duration {
	delay := float64(c.retryConfig.InitialDelay) * math.Pow(c.retryConfig.Multiplier, float64(attempt))
	if delay > float64(c.retryConfig.MaxDelay) {
		delay = float64(c.retryConfig.MaxDelay)
	}
	return time.Duration(delay)
}

// GetCacheStats returns cache statistics if caching is enabled
func (c *OpenAIClient) GetCacheStats() *CacheStats {
	if c.cache != nil {
		return c.cache.Stats()
	}
	return nil
}

// ClearCache clears all cached responses
func (c *OpenAIClient) ClearCache() error {
	if c.cache != nil {
		return c.cache.Clear()
	}
	return nil
}

// InvalidateCachePattern invalidates cache entries matching a pattern
func (c *OpenAIClient) InvalidateCachePattern(pattern string) (int, error) {
	if c.cache != nil {
		if inMemCache, ok := c.cache.(*InMemoryCache); ok {
			return inMemCache.InvalidatePattern(pattern)
		}
	}
	return 0, nil
}

// Wait implements rate limiting
func (r *RateLimiter) Wait(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Reset window if needed
	if now.Sub(r.windowStart) >= time.Minute {
		r.windowStart = now
		r.requestCount = 0
	}

	// Check rate limit
	if r.requestCount >= r.requestsPerMin {
		// Calculate wait time until next window
		waitTime := time.Minute - now.Sub(r.windowStart)
		select {
		case <-time.After(waitTime):
			r.windowStart = time.Now()
			r.requestCount = 0
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Enforce minimum interval between requests
	if !r.lastRequestTime.IsZero() {
		elapsed := now.Sub(r.lastRequestTime)
		if elapsed < r.minInterval {
			waitTime := r.minInterval - elapsed
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	r.lastRequestTime = time.Now()
	r.requestCount++
	return nil
}

# OpenAI Client Implementation

This package provides a robust OpenAI client wrapper with advanced features for the terminal-ai project.

## Features

### Core Functionality
- **Simple Query Interface**: Easy-to-use `Query()` method for basic requests
- **Streaming Support**: Real-time token-by-token streaming with `StreamQuery()`
- **Chat Completions**: Full chat history support with `Chat()` and `ChatStream()`
- **Model Management**: List available models with `ListModels()`

### Advanced Features
- **Connection Pooling**: Efficient HTTP connection reuse for better performance
- **Exponential Backoff**: Automatic retry logic with configurable delays
- **Rate Limiting**: Built-in rate limiting to prevent API throttling
- **Context Support**: Full context cancellation support for all operations
- **Error Handling**: Comprehensive error handling with retryable error detection

## Usage Examples

### Simple Query
```go
client := ai.NewOpenAIClient(config)
response, err := client.Query(ctx, "What is Go?")
```

### Streaming Response
```go
err := client.StreamQuery(ctx, "Tell me a story", func(chunk string) {
    fmt.Print(chunk)
})
```

### Chat with History
```go
messages := []ai.Message{
    {Role: "system", Content: "You are a helpful assistant"},
    {Role: "user", Content: "What is the capital of France?"},
}

options := ai.ChatOptions{
    Model:           "gpt-5-mini",
    Temperature:     1.0,      // Auto-adjusted for reasoning models
    MaxTokens:       100,
    ReasoningEffort: "low",    // For reasoning models
    ServiceTier:     "default", // Service tier preference
}

response, err := client.Chat(ctx, messages, options)
```

### Streaming Chat
```go
chunks, err := client.ChatStream(ctx, messages, options)
for chunk := range chunks {
    if chunk.Error != nil {
        // Handle error
    }
    if chunk.Done {
        break
    }
    fmt.Print(chunk.Content)
}
```

## Configuration

The client integrates with the terminal-ai configuration system:

```yaml
openai:
  api_key: ${OPENAI_API_KEY}
  model: gpt-5-mini              # Default reasoning model
  max_tokens: 2000
  temperature: 1.0               # Must be 1.0 for reasoning models
  reasoning_effort: low          # low, medium, high (for reasoning models)
  service_tier: default          # auto, default, priority, flex, scale
  timeout: 30s
  base_url: https://api.openai.com/v1
  org_id: ""
  top_p: 1.0

# Supported Models:
# Reasoning: gpt-5, gpt-5-mini, gpt-5-nano, o1, o1-mini, o3, o3-mini, o4-mini
# Non-reasoning: gpt-4.1, gpt-4.1-mini, gpt-4.1-nano, gpt-4o, gpt-4o-mini
```

## Architecture

### Components

1. **Client Interface** (`client.go`)
   - Defines the main client interface
   - Implements OpenAIClient with all core methods
   - Manages connection pooling and HTTP client
   - Handles retry logic and rate limiting

2. **Stream Handler** (`stream.go`)
   - Processes Server-Sent Events (SSE)
   - Manages token-by-token streaming
   - Provides callback-based and channel-based interfaces
   - Includes advanced stream processing capabilities

3. **Models** (`pkg/models/models.go`)
   - Request/Response data structures
   - Message types (system, user, assistant, function)
   - Chat completion parameters
   - Token usage tracking
   - Error response models

### Key Design Decisions

1. **Connection Pooling**: Custom HTTP client with configurable connection limits
2. **Rate Limiting**: Token bucket algorithm with per-minute limits
3. **Retry Strategy**: Exponential backoff with jitter for transient failures
4. **Streaming**: Dual interface (callback and channel) for flexibility
5. **Context Support**: All operations support context cancellation

## Testing

Run tests with:
```bash
go test -v ./internal/ai/...
```

The test suite covers:
- Client initialization
- Retry logic and backoff calculations
- Rate limiting behavior
- Message conversion
- Context cancellation
- Client lifecycle management

## Performance Considerations

- **Connection Reuse**: Maintains persistent connections to reduce latency
- **Buffered Channels**: Uses buffered channels for smooth streaming
- **Concurrent Safety**: Thread-safe operations with proper synchronization
- **Resource Cleanup**: Proper cleanup with Close() method

## Error Handling

The client distinguishes between:
- **Retryable Errors**: Network issues, rate limits, server errors (5xx)
- **Non-Retryable Errors**: Authentication failures, invalid requests (4xx)
- **Context Errors**: Timeouts and cancellations

## Recent Updates

### GPT-5 and O-series Support
- Full support for reasoning models (gpt-5, gpt-5-mini, o1, o3, etc.)
- Automatic temperature adjustment to 1.0 for reasoning models
- reasoning_effort parameter support (low, medium, high)
- Service tier support for priority processing

### Service Tier Features
- Support for OpenAI's service tier parameter
- Options: auto, default, priority, flex, scale
- Configurable via config file or command-line flag
- All models default to "default" tier unless overridden

## Future Enhancements

- [ ] Function calling support
- [ ] Fine-tuning API integration
- [ ] Embeddings API support
- [ ] Token counting before requests
- [ ] Response caching layer
- [ ] Metrics and observability
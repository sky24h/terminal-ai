package ai

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/shared"
	"github.com/rs/zerolog/log"
	"github.com/user/terminal-ai/internal/config"
)

// StreamHandler interface defines methods for handling streaming responses
type StreamHandlerInterface interface {
	HandleStream(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, options ChatOptions) (<-chan StreamChunk, error)
	ProcessStreamWithCallback(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, options ChatOptions, callback func(chunk string) error) error
}

// StreamHandler handles streaming responses from OpenAI
type StreamHandler struct {
	client openai.Client
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(client openai.Client) *StreamHandler {
	return &StreamHandler{
		client: client,
	}
}

// HandleStream processes a streaming chat completion
func (h *StreamHandler) HandleStream(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, options ChatOptions) (<-chan StreamChunk, error) {
	chunks := make(chan StreamChunk, 100) // Larger buffer for smoother streaming

	// Create streaming request parameters
	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(options.Model),
		Messages: messages,
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
	}

	stream := h.client.Chat.Completions.NewStreaming(ctx, params)
	if err := stream.Err(); err != nil {
		close(chunks)
		return chunks, fmt.Errorf("failed to create stream: %w", err)
	}

	// Process stream in goroutine
	go func() {
		defer close(chunks)

		var totalContent strings.Builder
		hasContent := false

		for stream.Next() {
			select {
			case <-ctx.Done():
				// Context cancelled, send error and exit
				chunks <- StreamChunk{
					Error: ctx.Err(),
					Done:  true,
				}
				return
			default:
				chunk := stream.Current()
				
				// Process response chunks
				if len(chunk.Choices) > 0 {
					choice := chunk.Choices[0]
					if choice.Delta.Content != "" {
						hasContent = true
						totalContent.WriteString(choice.Delta.Content)
						chunks <- StreamChunk{
							Content: choice.Delta.Content,
						}
					}

					// Check for finish reason
					if choice.FinishReason != "" {
						log.Debug().
							Str("finish_reason", string(choice.FinishReason)).
							Msg("Stream finished with reason")
					}
				}
			}
		}

		// Check for stream errors
		if err := stream.Err(); err != nil {
			log.Error().Err(err).Msg("Stream error")
			chunks <- StreamChunk{
				Error: err,
				Done:  true,
			}
			return
		}

		// Stream completed successfully
		chunks <- StreamChunk{
			Done: true,
		}
		if hasContent {
			log.Debug().
				Str("total_content", totalContent.String()).
				Msg("Stream completed")
		}
	}()

	return chunks, nil
}

// ProcessStreamWithCallback processes stream with a callback function
func (h *StreamHandler) ProcessStreamWithCallback(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
	options ChatOptions,
	callback func(chunk string) error,
) error {
	chunks, err := h.HandleStream(ctx, messages, options)
	if err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}

	var totalContent strings.Builder
	for chunk := range chunks {
		// Handle errors in the stream
		if chunk.Error != nil {
			if errors.Is(chunk.Error, context.Canceled) {
				log.Debug().Msg("Stream cancelled by user")
				return chunk.Error
			}
			return fmt.Errorf("stream error: %w", chunk.Error)
		}

		// Check if stream is complete
		if chunk.Done {
			log.Debug().
				Str("total_content", totalContent.String()).
				Msg("Stream processing completed")
			break
		}

		// Process content chunk
		if chunk.Content != "" {
			totalContent.WriteString(chunk.Content)
			if err := callback(chunk.Content); err != nil {
				return fmt.Errorf("callback error: %w", err)
			}
		}
	}

	return nil
}

// SSEParser parses Server-Sent Events for streaming responses
type SSEParser struct {
	reader *bufio.Reader
}

// NewSSEParser creates a new SSE parser
func NewSSEParser(reader io.Reader) *SSEParser {
	return &SSEParser{
		reader: bufio.NewReader(reader),
	}
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// Parse reads and parses the next SSE event
func (p *SSEParser) Parse() (*SSEEvent, error) {
	event := &SSEEvent{}
	var dataBuilder strings.Builder

	for {
		line, err := p.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && dataBuilder.Len() > 0 {
				// Return the last event if we have data
				event.Data = strings.TrimSuffix(dataBuilder.String(), "\n")
				return event, nil
			}
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")

		// Empty line signals end of event
		if line == "" {
			if dataBuilder.Len() > 0 {
				event.Data = strings.TrimSuffix(dataBuilder.String(), "\n")
				return event, nil
			}
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		field := parts[0]
		value := strings.TrimPrefix(parts[1], " ")

		switch field {
		case "event":
			event.Event = value
		case "data":
			if dataBuilder.Len() > 0 {
				dataBuilder.WriteString("\n")
			}
			dataBuilder.WriteString(value)
		case "id":
			event.ID = value
		case "retry":
			// Parse retry value if needed
		}
	}
}

// StreamProcessor provides advanced stream processing capabilities
type StreamProcessor struct {
	onToken     func(token string) error
	onError     func(err error)
	onComplete  func()
	bufferSize  int
	enableStats bool
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		bufferSize: 100,
	}
}

// WithTokenHandler sets the token handler
func (sp *StreamProcessor) WithTokenHandler(handler func(token string) error) *StreamProcessor {
	sp.onToken = handler
	return sp
}

// WithErrorHandler sets the error handler
func (sp *StreamProcessor) WithErrorHandler(handler func(err error)) *StreamProcessor {
	sp.onError = handler
	return sp
}

// WithCompleteHandler sets the completion handler
func (sp *StreamProcessor) WithCompleteHandler(handler func()) *StreamProcessor {
	sp.onComplete = handler
	return sp
}

// WithBufferSize sets the buffer size for the stream
func (sp *StreamProcessor) WithBufferSize(size int) *StreamProcessor {
	sp.bufferSize = size
	return sp
}

// EnableStats enables statistics collection
func (sp *StreamProcessor) EnableStats() *StreamProcessor {
	sp.enableStats = true
	return sp
}

// Process processes a stream of chunks
func (sp *StreamProcessor) Process(ctx context.Context, chunks <-chan StreamChunk) error {
	var buffer bytes.Buffer
	var tokenCount int
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			if sp.onError != nil {
				sp.onError(ctx.Err())
			}
			return ctx.Err()

		case chunk, ok := <-chunks:
			if !ok {
				// Channel closed
				if sp.onComplete != nil {
					sp.onComplete()
				}
				if sp.enableStats {
					duration := time.Since(startTime)
					log.Info().
						Int("tokens", tokenCount).
						Dur("duration", duration).
						Float64("tokens_per_second", float64(tokenCount)/duration.Seconds()).
						Msg("Stream processing stats")
				}
				return nil
			}

			if chunk.Error != nil {
				if sp.onError != nil {
					sp.onError(chunk.Error)
				}
				return chunk.Error
			}

			if chunk.Done {
				if sp.onComplete != nil {
					sp.onComplete()
				}
				if sp.enableStats {
					duration := time.Since(startTime)
					log.Info().
						Int("tokens", tokenCount).
						Dur("duration", duration).
						Float64("tokens_per_second", float64(tokenCount)/duration.Seconds()).
						Msg("Stream processing stats")
				}
				return nil
			}

			if chunk.Content != "" {
				buffer.WriteString(chunk.Content)
				tokenCount++

				if sp.onToken != nil {
					if err := sp.onToken(chunk.Content); err != nil {
						return err
					}
				}
			}
		}
	}
}

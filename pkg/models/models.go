package models

import (
	"encoding/json"
	"time"
)

// Conversation represents a chat conversation
type Conversation struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Metadata  Metadata  `json:"metadata"`
}

// MessageRole represents the role of a message
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleFunction  MessageRole = "function"
)

// Message represents a single message in a conversation
type Message struct {
	ID           string      `json:"id"`
	Role         string      `json:"role"` // system, user, assistant, function
	Content      string      `json:"content"`
	Timestamp    time.Time   `json:"timestamp"`
	Metadata     Metadata    `json:"metadata"`
	Name         string      `json:"name,omitempty"`          // Optional name for the message author
	FunctionCall interface{} `json:"function_call,omitempty"` // For function calling
}

// Metadata represents additional data for messages or conversations
type Metadata map[string]interface{}

// Session represents a chat session
type Session struct {
	ID           string        `json:"id"`
	Conversation *Conversation `json:"conversation"`
	Settings     Settings      `json:"settings"`
	StartedAt    time.Time     `json:"started_at"`
	EndedAt      *time.Time    `json:"ended_at,omitempty"`
}

// Settings represents session settings
type Settings struct {
	Model            string             `json:"model"`
	Temperature      float32            `json:"temperature"`
	MaxTokens        int                `json:"max_tokens"`
	Stream           bool               `json:"stream"`
	TopP             float32            `json:"top_p,omitempty"`
	N                int                `json:"n,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	PresencePenalty  float32            `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32            `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float32 `json:"logit_bias,omitempty"`
	User             string             `json:"user,omitempty"`
}

// NewConversation creates a new conversation
func NewConversation(title string) *Conversation {
	return &Conversation{
		ID:        generateID(),
		Title:     title,
		Messages:  []Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(Metadata),
	}
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(role, content string) {
	message := Message{
		ID:        generateID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(Metadata),
	}

	c.Messages = append(c.Messages, message)
	c.UpdatedAt = time.Now()
}

// GetLastMessage returns the last message in the conversation
func (c *Conversation) GetLastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
	return &c.Messages[len(c.Messages)-1]
}

// GetMessageCount returns the number of messages
func (c *Conversation) GetMessageCount() int {
	return len(c.Messages)
}

// generateID generates a unique ID
func generateID() string {
	// Simple implementation using timestamp
	// In production, use a proper UUID library
	return time.Now().Format("20060102150405.999999999")
}

// ChatCompletionRequest represents a request to the chat completion API
type ChatCompletionRequest struct {
	Model            string             `json:"model"`
	Messages         []Message          `json:"messages"`
	Temperature      float32            `json:"temperature,omitempty"`
	TopP             float32            `json:"top_p,omitempty"`
	N                int                `json:"n,omitempty"`
	Stream           bool               `json:"stream,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	MaxTokens        int                `json:"max_tokens,omitempty"`
	PresencePenalty  float32            `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32            `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float32 `json:"logit_bias,omitempty"`
	User             string             `json:"user,omitempty"`
}

// ChatCompletionResponse represents a response from the chat completion API
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   TokenUsage             `json:"usage"`
}

// ChatCompletionChoice represents a single choice in a chat completion response
type ChatCompletionChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamResponse represents a streaming response chunk
type StreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice represents a choice in a streaming response
type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        DeltaContent `json:"delta"`
	FinishReason string       `json:"finish_reason,omitempty"`
}

// DeltaContent represents incremental content in a stream
type DeltaContent struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError represents an API error
type APIError struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Param   interface{} `json:"param,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// Error implements the error interface for APIError
func (e APIError) Error() string {
	return e.Message
}

// MarshalJSON implements custom JSON marshaling for APIError
func (e APIError) MarshalJSON() ([]byte, error) {
	type Alias APIError
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(&e),
	})
}

// Model represents an available model
type Model struct {
	ID         string       `json:"id"`
	Object     string       `json:"object"`
	Created    int64        `json:"created"`
	OwnedBy    string       `json:"owned_by"`
	Permission []Permission `json:"permission"`
	Root       string       `json:"root"`
	Parent     string       `json:"parent,omitempty"`
}

// Permission represents model permissions
type Permission struct {
	ID                 string  `json:"id"`
	Object             string  `json:"object"`
	Created            int64   `json:"created"`
	AllowCreateEngine  bool    `json:"allow_create_engine"`
	AllowSampling      bool    `json:"allow_sampling"`
	AllowLogprobs      bool    `json:"allow_logprobs"`
	AllowSearchIndices bool    `json:"allow_search_indices"`
	AllowView          bool    `json:"allow_view"`
	AllowFineTuning    bool    `json:"allow_fine_tuning"`
	Organization       string  `json:"organization"`
	Group              *string `json:"group"`
	IsBlocking         bool    `json:"is_blocking"`
}

// ModelList represents a list of available models
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

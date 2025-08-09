//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/user/terminal-ai/internal/ai"
	"github.com/user/terminal-ai/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ensure API key is set
	if cfg.OpenAI.APIKey == "" {
		cfg.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
		if cfg.OpenAI.APIKey == "" {
			log.Fatal("OPENAI_API_KEY environment variable is not set")
		}
	}

	// Create OpenAI client
	client, err := ai.NewOpenAIClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: Simple query
	fmt.Println("=== Simple Query Example ===")
	response, err := client.Query(ctx, "What is Go programming language in one sentence?")
	if err != nil {
		log.Printf("Query error: %v", err)
	} else {
		fmt.Printf("Response: %s\n\n", response)
	}

	// Example 2: Streaming query
	fmt.Println("=== Streaming Query Example ===")
	fmt.Print("Response: ")
	err = client.StreamQuery(ctx, "Tell me a very short story about a robot", func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		log.Printf("\nStreaming error: %v", err)
	}
	fmt.Println("\n")

	// Example 3: Chat with history
	fmt.Println("=== Chat with History Example ===")
	messages := []ai.Message{
		{Role: "system", Content: "You are a helpful assistant that speaks concisely."},
		{Role: "user", Content: "What is the capital of France?"},
		{Role: "assistant", Content: "The capital of France is Paris."},
		{Role: "user", Content: "What is its population?"},
	}

	options := ai.ChatOptions{
		Model:       "gpt-3.5-turbo",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	chatResponse, err := client.Chat(ctx, messages, options)
	if err != nil {
		log.Printf("Chat error: %v", err)
	} else {
		fmt.Printf("Response: %s\n", chatResponse.Content)
		fmt.Printf("Tokens used: %d\n\n", chatResponse.Usage.TotalTokens)
	}

	// Example 4: Streaming chat
	fmt.Println("=== Streaming Chat Example ===")
	messages2 := []ai.Message{
		{Role: "user", Content: "Write a haiku about programming"},
	}

	chunks, err := client.ChatStream(ctx, messages2, options)
	if err != nil {
		log.Printf("Stream chat error: %v", err)
	} else {
		fmt.Print("Response: ")
		for chunk := range chunks {
			if chunk.Error != nil {
				log.Printf("\nStream error: %v", chunk.Error)
				break
			}
			if chunk.Done {
				break
			}
			fmt.Print(chunk.Content)
		}
		fmt.Println("\n")
	}

	// Example 5: List available models
	fmt.Println("=== Available Models ===")
	models, err := client.ListModels(ctx)
	if err != nil {
		log.Printf("List models error: %v", err)
	} else {
		fmt.Println("First 5 available models:")
		for i, model := range models {
			if i >= 5 {
				break
			}
			fmt.Printf("  - %s\n", model)
		}
	}

	// Example 6: Context cancellation
	fmt.Println("\n=== Context Cancellation Example ===")
	ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.StreamQuery(ctx2, "Write a long essay about the history of computers", func(chunk string) {
		fmt.Print(chunk)
		time.Sleep(100 * time.Millisecond) // Simulate slow processing
	})
	if err != nil {
		fmt.Printf("\nExpected timeout error: %v\n", err)
	}
}

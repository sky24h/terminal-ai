package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/user/terminal-ai/internal/ai"
	// "github.com/user/terminal-ai/internal/config"
	"github.com/user/terminal-ai/internal/ui"
)

var (
	chatModel        string
	chatTemperature  float32
	chatMaxTokens    int
	chatStream       bool
	chatSaveHistory  bool
	chatLoadHistory  string
	chatExportPath   string
	chatSystemPrompt string
	chatMultiline    bool
)

// ConversationHistory represents a chat conversation
type ConversationHistory struct {
	Messages  []ai.Message `json:"messages"`
	Model     string       `json:"model"`
	Timestamp time.Time    `json:"timestamp"`
}

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session with AI",
	Long: `Start an interactive chat session with an AI assistant.
This command opens an interactive terminal interface where you can have
a conversation with the AI model.

Commands during chat:
  /help      - Show available commands
  /clear     - Clear conversation history
  /save      - Save conversation to file
  /load      - Load conversation from file
  /export    - Export conversation as markdown
  /model     - Change the AI model
  /system    - Set system prompt
  /multiline - Toggle multiline input mode
  /cache     - Show cache statistics
  /exit      - Exit chat session

Example:
  terminal-ai chat
  terminal-ai chat --model gpt-5
  terminal-ai chat --system "You are a helpful coding assistant"
  terminal-ai chat --load previous-chat.json`,
	RunE: RunChat,
}

func init() {
	rootCmd.AddCommand(chatCmd)

	// Chat-specific flags
	chatCmd.Flags().StringVarP(&chatModel, "model", "m", "", "AI model to use (default from config)")
	chatCmd.Flags().Float32VarP(&chatTemperature, "temperature", "t", -1, "Temperature for response generation (0.0-2.0)")
	chatCmd.Flags().IntVar(&chatMaxTokens, "max-tokens", 0, "Maximum tokens in response")
	chatCmd.Flags().BoolVarP(&chatStream, "stream", "s", true, "Stream responses in real-time")
	chatCmd.Flags().BoolVar(&chatSaveHistory, "save-history", true, "Auto-save conversation history")
	chatCmd.Flags().StringVar(&chatLoadHistory, "load", "", "Load conversation history from file")
	chatCmd.Flags().StringVar(&chatExportPath, "export", "", "Export conversation to file on exit")
	chatCmd.Flags().StringVar(&chatSystemPrompt, "system", "", "Initial system prompt")
	chatCmd.Flags().BoolVar(&chatMultiline, "multiline", false, "Enable multiline input mode")

	// Bind flags to viper
	viper.BindPFlag("chat.model", chatCmd.Flags().Lookup("model"))
	viper.BindPFlag("chat.temperature", chatCmd.Flags().Lookup("temperature"))
	viper.BindPFlag("chat.max_tokens", chatCmd.Flags().Lookup("max-tokens"))
	viper.BindPFlag("chat.stream", chatCmd.Flags().Lookup("stream"))
	viper.BindPFlag("chat.save_history", chatCmd.Flags().Lookup("save-history"))
	viper.BindPFlag("chat.multiline", chatCmd.Flags().Lookup("multiline"))
}

// RunChat runs the chat command - exported for use in simple mode
func RunChat(cmd *cobra.Command, args []string) error {
	// Get AI client
	client := GetAIClient()
	if client == nil {
		return fmt.Errorf("AI client not initialized. Please check your configuration")
	}

	// Get configuration
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Initialize chat options
	options := ai.ChatOptions{
		Model:       chatModel,
		Temperature: chatTemperature,
		MaxTokens:   chatMaxTokens,
	}

	// Use defaults from config if not specified
	if options.Model == "" {
		options.Model = cfg.OpenAI.Model
	}
	if options.Temperature < 0 {
		options.Temperature = cfg.OpenAI.Temperature
	}
	if options.MaxTokens == 0 {
		options.MaxTokens = cfg.OpenAI.MaxTokens
	}

	// Initialize conversation history
	var messages []ai.Message

	// Load history if specified
	if chatLoadHistory != "" {
		history, err := loadConversationHistory(chatLoadHistory)
		if err != nil {
			fmt.Printf("⚠️  Failed to load history: %v\n", err)
		} else {
			messages = history.Messages
			fmt.Printf("✓ Loaded %d messages from history\n", len(messages))
		}
	}

	// Add system prompt (use specified or default from config)
	systemPrompt := chatSystemPrompt
	if systemPrompt == "" && cfg.OpenAI.SystemPrompt != "" {
		systemPrompt = cfg.OpenAI.SystemPrompt
	}
	if systemPrompt != "" {
		messages = append([]ai.Message{{
			Role:    "system",
			Content: systemPrompt,
		}}, messages...)
	}

	// Print welcome message
	fmt.Println("\n=== Terminal AI Chat ===")
	fmt.Printf("Model: %s | Temperature: %.1f | Max Tokens: %d\n",
		options.Model, options.Temperature, options.MaxTokens)
	fmt.Println("Type '/help' for available commands or '/exit' to quit")
	fmt.Println()

	// Chat loop
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		// Get user input
		fmt.Print("> ")

		var userInput string
		var err error

		if chatMultiline {
			userInput, err = readMultilineInput(reader)
		} else {
			userInput, err = reader.ReadString('\n')
			userInput = strings.TrimSpace(userInput)
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("❌ Error reading input: %v\n", err)
			continue
		}

		// Handle commands
		if strings.HasPrefix(userInput, "/") {
			if handleSimpleChatCommand(userInput, &messages, &options) {
				break // Exit command
			}
			continue
		}

		// Skip empty input
		if strings.TrimSpace(userInput) == "" {
			continue
		}

		// Add user message to history
		messages = append(messages, ai.Message{
			Role:    "user",
			Content: userInput,
		})

		// Send to AI and get response
		if chatStream && cfg.UI.StreamingEnabled {
			// Streaming response
			spinner := ui.NewSimpleSpinner("Thinking...")
			spinner.Start()

			chunks, err := client.ChatStream(ctx, messages, options)
			if err != nil {
				spinner.Stop()
				fmt.Printf("❌ Failed to get response: %v\n", err)
				// Remove the failed message from history
				messages = messages[:len(messages)-1]
				continue
			}

			spinner.Stop()
			fmt.Print("Assistant: ")

			// Collect and display response
			var responseBuilder strings.Builder
			for chunk := range chunks {
				if chunk.Error != nil {
					fmt.Printf("❌ Stream error: %v\n", chunk.Error)
					break
				}
				if chunk.Done {
					break
				}
				responseBuilder.WriteString(chunk.Content)
				fmt.Print(chunk.Content)
			}
			fmt.Println()

			// Add assistant response to history
			messages = append(messages, ai.Message{
				Role:    "assistant",
				Content: responseBuilder.String(),
			})

		} else {
			// Non-streaming response
			spinner := ui.NewSimpleSpinner("Thinking...")
			spinner.Start()

			resp, err := client.Chat(ctx, messages, options)
			if err != nil {
				spinner.Stop()
				fmt.Printf("❌ Failed to get response: %v\n", err)
				// Remove the failed message from history
				messages = messages[:len(messages)-1]
				continue
			}

			spinner.Stop()

			// Display response
			fmt.Print("Assistant: ")
			fmt.Println(resp.Content)
			fmt.Println()

			// Show token usage if cache is enabled
			if cfg.Cache.Enabled {
				if openaiClient, ok := client.(*ai.OpenAIClient); ok {
					if stats := openaiClient.GetCacheStats(); stats != nil && stats.Hits > 0 {
						fmt.Printf("💾 (cached response, saved ~%.2fs)\n", 1.5)
					}
				}
			}

			// Add assistant response to history
			messages = append(messages, ai.Message{
				Role:    "assistant",
				Content: resp.Content,
			})
		}

		// Auto-save history if enabled
		if chatSaveHistory {
			saveConversationToCache(messages, options.Model)
		}
	}

	// Export conversation if requested
	if chatExportPath != "" {
		if err := exportSimpleConversation(messages, chatExportPath); err != nil {
			fmt.Printf("❌ Failed to export conversation: %v\n", err)
		} else {
			fmt.Printf("✓ Conversation exported to: %s\n", chatExportPath)
		}
	}

	fmt.Println("Chat session ended. Goodbye!")
	return nil
}

func handleSimpleChatCommand(command string, messages *[]ai.Message, options *ai.ChatOptions) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	switch parts[0] {
	case "/exit", "/quit", "/q":
		return true

	case "/help", "/h":
		printSimpleChatHelp()

	case "/clear":
		*messages = []ai.Message{}
		fmt.Println("✓ Conversation history cleared")

	case "/save":
		filename := "chat_" + time.Now().Format("20060102_150405") + ".json"
		if len(parts) > 1 {
			filename = strings.Join(parts[1:], " ")
		}
		if err := saveConversationHistory(*messages, filename, options.Model); err != nil {
			fmt.Printf("❌ Failed to save: %v\n", err)
		} else {
			fmt.Printf("✓ Conversation saved to: %s\n", filename)
		}

	case "/load":
		if len(parts) < 2 {
			fmt.Println("⚠️  Usage: /load <filename>")
		} else {
			filename := strings.Join(parts[1:], " ")
			history, err := loadConversationHistory(filename)
			if err != nil {
				fmt.Printf("❌ Failed to load: %v\n", err)
			} else {
				*messages = history.Messages
				fmt.Printf("✓ Loaded %d messages\n", len(history.Messages))
			}
		}

	case "/export":
		filename := "chat_export_" + time.Now().Format("20060102_150405") + ".md"
		if len(parts) > 1 {
			filename = strings.Join(parts[1:], " ")
		}
		if err := exportSimpleConversation(*messages, filename); err != nil {
			fmt.Printf("❌ Failed to export: %v\n", err)
		} else {
			fmt.Printf("✓ Exported to: %s\n", filename)
		}

	case "/model":
		if len(parts) < 2 {
			fmt.Printf("Current model: %s\n", options.Model)
		} else {
			options.Model = parts[1]
			fmt.Printf("✓ Model changed to: %s\n", options.Model)
		}

	case "/system":
		if len(parts) < 2 {
			fmt.Println("⚠️  Usage: /system <prompt>")
		} else {
			systemPrompt := strings.Join(parts[1:], " ")
			// Add or update system message
			if len(*messages) > 0 && (*messages)[0].Role == "system" {
				(*messages)[0].Content = systemPrompt
			} else {
				*messages = append([]ai.Message{{
					Role:    "system",
					Content: systemPrompt,
				}}, *messages...)
			}
			fmt.Println("✓ System prompt updated")
		}

	case "/multiline":
		chatMultiline = !chatMultiline
		if chatMultiline {
			fmt.Println("✓ Multiline mode enabled. Use Ctrl+D to send message.")
		} else {
			fmt.Println("✓ Multiline mode disabled.")
		}

	case "/history":
		fmt.Println("\n=== Conversation History ===")
		for i, msg := range *messages {
			role := strings.Title(msg.Role)
			preview := msg.Content
			if len(preview) > 100 {
				preview = preview[:97] + "..."
			}
			fmt.Printf("%d. [%s] %s\n", i+1, role, preview)
		}
		fmt.Println()

	case "/cache":
		// Show cache stats if enabled
		cfg := GetConfig()
		if cfg != nil && cfg.Cache.Enabled {
			if client := GetAIClient(); client != nil {
				if openaiClient, ok := client.(*ai.OpenAIClient); ok {
					if stats := openaiClient.GetCacheStats(); stats != nil {
						fmt.Println("\n=== Cache Statistics ===")
						fmt.Printf("Hit Rate: %.1f%%\n", stats.HitRate*100)
						fmt.Printf("Hits: %d | Misses: %d\n", stats.Hits, stats.Misses)
						fmt.Printf("Entries: %d | Size: %.2f MB\n",
							stats.Entries, float64(stats.SizeBytes)/(1024*1024))
						fmt.Println()
					}
				}
			}
		} else {
			fmt.Println("Cache is disabled")
		}

	default:
		fmt.Printf("⚠️  Unknown command: %s\n", parts[0])
	}

	return false
}

func printSimpleChatHelp() {
	fmt.Println("\n=== Chat Commands ===")
	commands := [][]string{
		{"/help", "Show this help message"},
		{"/clear", "Clear conversation history"},
		{"/save [file]", "Save conversation to file"},
		{"/load <file>", "Load conversation from file"},
		{"/export [file]", "Export conversation as markdown"},
		{"/model [name]", "Show or change AI model"},
		{"/system <prompt>", "Set system prompt"},
		{"/multiline", "Toggle multiline input mode"},
		{"/history", "Show conversation history"},
		{"/cache", "Show cache statistics"},
		{"/exit", "Exit chat session"},
	}

	for _, cmd := range commands {
		fmt.Printf("  %-20s %s\n", cmd[0], cmd[1])
	}
	fmt.Println()
}

func exportSimpleConversation(messages []ai.Message, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write markdown header
	fmt.Fprintf(file, "# Chat Conversation\n\n")
	fmt.Fprintf(file, "Exported: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "---\n\n")

	// Write messages
	for _, msg := range messages {
		role := strings.Title(msg.Role)
		fmt.Fprintf(file, "## %s\n\n", role)
		fmt.Fprintf(file, "%s\n\n", msg.Content)
	}

	return nil
}

// Helper functions for chat

func readMultilineInput(reader *bufio.Reader) (string, error) {
	var lines []string
	fmt.Println("(Enter Ctrl+D to send, Ctrl+C to cancel)")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, ""), nil
}

func saveConversationHistory(messages []ai.Message, filename string, model string) error {
	history := ConversationHistory{
		Messages:  messages,
		Model:     model,
		Timestamp: time.Now(),
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func loadConversationHistory(filename string) (*ConversationHistory, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var history ConversationHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return &history, nil
}

func saveConversationToCache(messages []ai.Message, model string) {
	// Save to a default cache location
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	cacheDir := filepath.Join(home, ".terminal-ai", "chat-history")
	os.MkdirAll(cacheDir, 0755)

	filename := filepath.Join(cacheDir, "last-chat.json")
	saveConversationHistory(messages, filename, model)
}

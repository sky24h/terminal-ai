package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/user/terminal-ai/internal/ai"
	"github.com/user/terminal-ai/internal/ui"
)

var (
	queryModel       string
	queryStream      bool
	queryContext     string
	queryOutput      string
	queryTemperature float32
	queryMaxTokens   int
	querySystem      string
	queryFormat      string
	queryShowTokens  bool
	queryTopP        float32
)

// queryCmd represents the query command
var queryCmd = &cobra.Command{
	Use:   "query [question]",
	Short: "Send a quick query to AI and get a response",
	Long: `Send a one-off query to the AI model and receive a response.
This is useful for quick questions without starting an interactive session.

Examples:
  terminal-ai query "What is the capital of France?"
  terminal-ai query "Explain quantum computing" --model gpt-4
  terminal-ai query "Write a Python function to sort a list" --output result.txt
  terminal-ai query "Translate to Spanish: Hello world" --format plain
  terminal-ai query "Code review this function" --system "You are a code reviewer" --context "def add(a,b): return a+b"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		question := strings.Join(args, " ")
		return runQuery(question)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	// Query-specific flags
	queryCmd.Flags().StringVarP(&queryModel, "model", "m", "", "AI model to use (default from config)")
	queryCmd.Flags().BoolVarP(&queryStream, "stream", "s", true, "Stream the response")
	queryCmd.Flags().StringVarP(&queryContext, "context", "c", "", "Additional context for the query")
	queryCmd.Flags().StringVarP(&queryOutput, "output", "o", "", "Save response to file")
	queryCmd.Flags().Float32VarP(&queryTemperature, "temperature", "t", -1, "Temperature for response generation (0.0-2.0)")
	queryCmd.Flags().IntVar(&queryMaxTokens, "max-tokens", 0, "Maximum tokens in response")
	queryCmd.Flags().StringVar(&querySystem, "system", "", "System prompt to set behavior")
	queryCmd.Flags().StringVarP(&queryFormat, "format", "f", "markdown", "Output format (plain, markdown, json)")
	queryCmd.Flags().BoolVar(&queryShowTokens, "tokens", false, "Show token usage information")
	queryCmd.Flags().Float32Var(&queryTopP, "top-p", -1, "Top-p sampling parameter")

	// Bind flags to viper
	viper.BindPFlag("query.model", queryCmd.Flags().Lookup("model"))
	viper.BindPFlag("query.stream", queryCmd.Flags().Lookup("stream"))
	viper.BindPFlag("query.context", queryCmd.Flags().Lookup("context"))
	viper.BindPFlag("query.output", queryCmd.Flags().Lookup("output"))
	viper.BindPFlag("query.temperature", queryCmd.Flags().Lookup("temperature"))
	viper.BindPFlag("query.max_tokens", queryCmd.Flags().Lookup("max-tokens"))
	viper.BindPFlag("query.format", queryCmd.Flags().Lookup("format"))
}

func runQuery(question string) error {
	// Get AI client
	client := GetAIClient()
	if client == nil {
		return fmt.Errorf("AI client not initialized. Please check your configuration")
	}

	// Get configuration
	config := GetConfig()
	if config == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Prepare messages
	var messages []ai.Message

	// Add system message (use specified or default from config)
	systemPrompt := querySystem
	if systemPrompt == "" && config.OpenAI.SystemPrompt != "" {
		systemPrompt = config.OpenAI.SystemPrompt
	}
	if systemPrompt != "" {
		messages = append(messages, ai.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Build the user message with context
	userContent := question
	if queryContext != "" {
		userContent = fmt.Sprintf("Context:\n%s\n\nQuestion: %s", queryContext, question)
	}

	messages = append(messages, ai.Message{
		Role:    "user",
		Content: userContent,
	})

	// Prepare chat options
	options := ai.ChatOptions{
		Model:       queryModel,
		Temperature: queryTemperature,
		MaxTokens:   queryMaxTokens,
		TopP:        queryTopP,
	}

	// Use defaults from config if not specified
	if options.Model == "" {
		options.Model = config.OpenAI.Model
	}
	if options.Temperature < 0 {
		options.Temperature = config.OpenAI.Temperature
	}
	if options.MaxTokens == 0 {
		options.MaxTokens = config.OpenAI.MaxTokens
	}
	if options.TopP < 0 {
		options.TopP = config.OpenAI.TopP
	}

	// Create UI components
	formatter := ui.NewFormatter(ui.FormatterOptions{
		ColorEnabled:       config.UI.ColorOutput,
		MarkdownEnabled:    config.UI.MarkdownRendering && queryFormat == "markdown",
		SyntaxHighlighting: config.UI.SyntaxHighlighting,
		Width:              ui.GetTerminalWidth(),
	})

	// Show query info if verbose
	logger := GetLogger()
	if logger != nil && verbose {
		logger.Debug("Sending query to AI", map[string]interface{}{
			"model":       options.Model,
			"temperature": options.Temperature,
			"max_tokens":  options.MaxTokens,
			"stream":      queryStream,
		})
	}

	ctx := context.Background()
	var response string
	var usage ai.Usage

	if queryStream && queryFormat != "json" {
		// Streaming response
		spinner := ui.NewSimpleSpinner("Thinking...")
		spinner.Start()

		chunks, err := client.ChatStream(ctx, messages, options)
		if err != nil {
			spinner.StopWithError(fmt.Sprintf("Failed to send query: %v", err))
			return err
		}

		spinner.Stop()
		fmt.Println() // New line after spinner

		// Collect response chunks
		var responseBuilder strings.Builder
		for chunk := range chunks {
			if chunk.Error != nil {
				return fmt.Errorf("stream error: %w", chunk.Error)
			}
			if chunk.Done {
				break
			}
			responseBuilder.WriteString(chunk.Content)
			// Print chunk immediately for streaming
			fmt.Print(chunk.Content)
		}
		response = responseBuilder.String()
		fmt.Println() // Final newline

	} else {
		// Non-streaming response
		spinner := ui.NewSimpleSpinner("Processing query...")
		spinner.Start()

		resp, err := client.Chat(ctx, messages, options)
		if err != nil {
			spinner.StopWithError(fmt.Sprintf("Failed to send query: %v", err))
			return err
		}

		spinner.StopWithSuccess("Response received")
		response = resp.Content
		usage = resp.Usage

		// Format and display response
		switch queryFormat {
		case "plain":
			fmt.Println(response)
		case "json":
			// For JSON, just print as-is
			fmt.Println(response)
		default: // markdown
			formatted := formatter.FormatMarkdown(response)
			fmt.Println(formatted)
		}
	}

	// Show token usage if requested
	if queryShowTokens && usage.TotalTokens > 0 {
		fmt.Println()
		formatter.PrintInfo(fmt.Sprintf("Token Usage: Prompt=%d, Completion=%d, Total=%d",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens))
	}

	// Save to file if requested
	if queryOutput != "" {
		if err := saveResponseToFile(response, queryOutput); err != nil {
			formatter.PrintError(fmt.Sprintf("Failed to save response: %v", err))
			return err
		}
		formatter.PrintSuccess(fmt.Sprintf("Response saved to: %s", queryOutput))
	}

	return nil
}

func saveResponseToFile(content, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write file
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

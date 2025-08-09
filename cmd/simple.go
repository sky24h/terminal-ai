package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/terminal-ai/internal/ai"
)

var (
	queryFlag  bool
	shellFlag  bool
	chatFlag   bool
	modelFlag  string
	streamFlag bool
)

const (
	helpfulAssistantPrompt = "You are a helpful assistant, answer as concisely as possible to the user."
	shellCommandPrompt     = "You are a shell command assistant. Respond only with the exact command to execute, nothing else. Include pipes, flags, and arguments as needed. Assume bash/Linux unless specified otherwise. No explanations, no markdown, just the command."
	// TODO: When OpenAI Go SDK supports structured outputs, enforce response format:
	// ResponseFormat: {type: "json_schema", json_schema: {name: "command", schema: {type: "object", properties: {command: {type: "string"}}, required: ["command"]}}}
)

func init() {
	// Add simple mode flags to root command
	rootCmd.Flags().BoolVarP(&queryFlag, "query", "q", false, "Quick query mode - ask a question and get a concise answer")
	rootCmd.Flags().BoolVarP(&shellFlag, "shell", "s", false, "Shell command mode (default) - generate and optionally execute shell commands")
	rootCmd.Flags().BoolVarP(&chatFlag, "chat", "c", false, "Interactive chat mode with the AI assistant")
	rootCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Override default model")
	rootCmd.Flags().BoolVar(&streamFlag, "stream", true, "Enable streaming responses")

	// Set the Run function for root command and allow unknown args
	rootCmd.Run = runSimpleMode
	rootCmd.FParseErrWhitelist.UnknownFlags = true
}

func runSimpleMode(cmd *cobra.Command, args []string) {
	// Check if any mode flag is set
	modeCount := 0
	if queryFlag {
		modeCount++
	}
	if shellFlag {
		modeCount++
	}
	if chatFlag {
		modeCount++
	}

	// If multiple modes selected, show error
	if modeCount > 1 {
		fmt.Println("Error: Please specify only one mode: -q (query), -s (shell), or -c (chat)")
		os.Exit(1)
	}

	// If no mode selected and no subcommand, handle as shell if args provided
	if modeCount == 0 {
		if len(args) > 0 {
			// Default to shell mode if text is provided
			shellFlag = true
		} else {
			// Show help if no mode and no args
			cmd.Help()
			return
		}
	}

	// Initialize app if not already done
	if appConfig == nil {
		if err := initializeApp(); err != nil {
			fmt.Printf("Error initializing: %v\n", err)
			os.Exit(1)
		}
	}

	// Get the question/prompt from args
	var prompt string
	if len(args) > 0 {
		prompt = strings.Join(args, " ")
	}

	// Handle different modes
	switch {
	case queryFlag:
		runQueryMode(prompt)
	case shellFlag:
		runShellMode(prompt)
	case chatFlag:
		runChatMode()
	}
}

func runQueryMode(prompt string) {
	if prompt == "" {
		fmt.Println("Error: Please provide a question")
		fmt.Println("Usage: terminal-ai -q \"Your question here\"")
		os.Exit(1)
	}

	ctx := context.Background()
	client := GetAIClient()
	config := GetConfig()

	// Prepare messages with helpful assistant prompt
	messages := []ai.Message{
		{
			Role:    "system",
			Content: helpfulAssistantPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Prepare options
	options := ai.ChatOptions{
		Model:       config.OpenAI.Model,
		Temperature: config.OpenAI.Temperature,
		MaxTokens:   config.OpenAI.MaxTokens,
		TopP:        config.OpenAI.TopP,
	}

	// Override model if specified
	if modelFlag != "" {
		options.Model = modelFlag
	}

	if streamFlag {
		// Stream response
		chunks, err := client.ChatStream(ctx, messages, options)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		for chunk := range chunks {
			if chunk.Error != nil {
				fmt.Printf("\nError: %v\n", chunk.Error)
				os.Exit(1)
			}
			if chunk.Content != "" {
				fmt.Print(chunk.Content)
			}
		}
		fmt.Println()
	} else {
		// Non-streaming response
		resp, err := client.Chat(ctx, messages, options)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(resp.Content)
	}
}

func runShellMode(prompt string) {
	if prompt == "" {
		// Interactive mode - get prompt from user
		fmt.Print("What do you want to do? > ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		prompt = strings.TrimSpace(input)
		if prompt == "" {
			fmt.Println("No input provided")
			return
		}
	}

	ctx := context.Background()
	client := GetAIClient()
	config := GetConfig()

	for {
		// Prepare messages with shell command prompt
		messages := []ai.Message{
			{
				Role:    "system",
				Content: shellCommandPrompt,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		}

		// Prepare options
		options := ai.ChatOptions{
			Model:       config.OpenAI.Model,
			Temperature: config.OpenAI.Temperature,
			MaxTokens:   config.OpenAI.MaxTokens,
			TopP:        config.OpenAI.TopP,
		}

		// Override model if specified
		if modelFlag != "" {
			options.Model = modelFlag
		}

		// Get the command suggestion
		resp, err := client.Chat(ctx, messages, options)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		command := strings.TrimSpace(resp.Content)
		fmt.Printf("\n📝 Command: %s\n", command)

		// Ask for confirmation
		fmt.Print("\n🔸 Execute? [Enter/E=Execute, N=No, Q=Quit]: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "", "e", "execute", "y", "yes":
			// Execute the command
			fmt.Println("\n🚀 Executing...")
			if err := executeCommand(command); err != nil {
				fmt.Printf("❌ Execution failed: %v\n", err)
			}
			return
		case "n", "no", "esc":
			// Ask for clarification
			fmt.Print("\n✏️  What should be different? > ")
			clarification, _ := reader.ReadString('\n')
			clarification = strings.TrimSpace(clarification)
			if clarification == "" {
				fmt.Println("No clarification provided, exiting.")
				return
			}
			// Update prompt with clarification
			prompt = fmt.Sprintf("%s\n\nUser feedback: %s", prompt, clarification)
			continue
		case "q", "quit", "exit":
			fmt.Println("Exiting.")
			return
		default:
			fmt.Println("Invalid input. Please try again.")
			// Re-ask for the same command
			fmt.Printf("\n📝 Command: %s\n", command)
			continue
		}
	}
}

func runChatMode() {
	// This will call the existing chat command implementation
	// but with the helpful assistant system prompt
	chatSystemPrompt = helpfulAssistantPrompt
	if err := RunChat(&cobra.Command{}, []string{}); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func executeCommand(command string) error {
	var cmd *exec.Cmd
	
	// Detect the operating system and use appropriate shell
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		// Unix-like systems (Linux, macOS)
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell, "-c", command)
	}

	// Set up the command to use the current terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	return cmd.Run()
}
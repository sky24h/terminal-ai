package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/terminal-ai/internal/ai"
	"github.com/user/terminal-ai/internal/config"
	"github.com/user/terminal-ai/internal/ui"
	"gopkg.in/yaml.v3"
)

var (
	configInit     bool
	configWizard   bool
	configValidate bool
	configTest     bool
	configShow     bool
	configEdit     bool
	configLocation bool
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage terminal-ai configuration",
	Long: `Manage terminal-ai configuration settings.
You can initialize, view, edit, and validate the configuration.

Examples:
  terminal-ai config --init       # Initialize configuration with wizard
  terminal-ai config --show       # Show current configuration
  terminal-ai config --validate   # Validate configuration
  terminal-ai config --test       # Test API connection
  terminal-ai config --edit       # Open config in editor
  terminal-ai config --location   # Show config file location
  terminal-ai config set openai.model gpt-5-mini
  terminal-ai config get openai.model`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle flags
		if configInit || configWizard {
			return runConfigWizard()
		}
		if configShow {
			return showConfig()
		}
		if configValidate {
			return validateConfig()
		}
		if configTest {
			return testAPIConnection()
		}
		if configEdit {
			return editConfig()
		}
		if configLocation {
			return showConfigLocation()
		}

		// If no flags, show help
		return cmd.Help()
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value using dot notation.

Examples:
  terminal-ai config set openai.api_key "sk-..."
  terminal-ai config set openai.model gpt-5-mini
  terminal-ai config set ui.color_output false
  terminal-ai config set cache.enabled true`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setConfigValue(args[0], args[1])
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long: `Get a configuration value using dot notation.

Examples:
  terminal-ai config get openai.model
  terminal-ai config get ui.color_output
  terminal-ai config get cache.ttl`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return getConfigValue(args[0])
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)

	// Config flags
	configCmd.Flags().BoolVarP(&configInit, "init", "i", false, "Initialize configuration with wizard")
	configCmd.Flags().BoolVarP(&configWizard, "wizard", "w", false, "Run configuration wizard")
	configCmd.Flags().BoolVarP(&configShow, "show", "s", false, "Show current configuration")
	configCmd.Flags().BoolVar(&configValidate, "validate", false, "Validate configuration")
	configCmd.Flags().BoolVarP(&configTest, "test", "t", false, "Test API connection")
	configCmd.Flags().BoolVarP(&configEdit, "edit", "e", false, "Edit configuration file in default editor")
	configCmd.Flags().BoolVarP(&configLocation, "location", "l", false, "Show configuration file location")
}

func runConfigWizard() error {
	formatter := ui.NewFormatter(ui.FormatterOptions{
		ColorEnabled: true,
		Width:        ui.GetTerminalWidth(),
	})

	formatter.PrintTitle("Terminal AI Configuration Wizard")
	fmt.Println("This wizard will help you set up terminal-ai configuration.")
	fmt.Println()

	// Create input helper
	input := ui.NewInput(ui.InputOptions{
		Prompt: "> ",
	})

	// Get API key
	formatter.PrintSection("OpenAI API Configuration")
	apiKey := ""

	// Check environment variable first
	if envKey := os.Getenv("OPENAI_API_KEY"); envKey != "" {
		formatter.PrintInfo("Found OPENAI_API_KEY in environment")
		fmt.Print("Use this API key? (Y/n): ")
		response, _ := input.ReadLine()
		if response == "" || strings.ToLower(response)[0] == 'y' {
			apiKey = envKey
		}
	}

	if apiKey == "" {
		fmt.Print("Enter your OpenAI API key: ")
		apiKey, _ = input.ReadPassword()
		fmt.Println() // New line after password input
	}

	if apiKey == "" {
		formatter.PrintError("API key is required")
		return fmt.Errorf("API key not provided")
	}

	// Select model
	formatter.PrintSection("Model Selection")
	models := []string{
		"gpt-3.5-turbo",
		"gpt-4",
		"gpt-4-turbo-preview",
		"gpt-4o",
		"gpt-4o-mini",
	}

	fmt.Println("Available models:")
	for i, model := range models {
		fmt.Printf("  %d. %s\n", i+1, model)
	}

	fmt.Print("Select model (1-5) [1]: ")
	modelChoice, _ := input.ReadLine()
	selectedModel := "gpt-3.5-turbo"

	if modelChoice != "" {
		if idx := parseInt(modelChoice); idx > 0 && idx <= len(models) {
			selectedModel = models[idx-1]
		}
	}

	// Temperature setting
	formatter.PrintSection("Response Settings")
	fmt.Print("Temperature (0.0-2.0) [0.7]: ")
	tempStr, _ := input.ReadLine()
	temperature := float32(0.7)
	if tempStr != "" {
		if temp := parseFloat32(tempStr); temp >= 0 && temp <= 2.0 {
			temperature = temp
		}
	}

	// Max tokens
	fmt.Print("Max tokens [2000]: ")
	maxTokensStr, _ := input.ReadLine()
	maxTokens := 2000
	if maxTokensStr != "" {
		if tokens := parseInt(maxTokensStr); tokens > 0 {
			maxTokens = tokens
		}
	}

	// UI preferences
	formatter.PrintSection("UI Preferences")
	fmt.Print("Enable colored output? (Y/n): ")
	colorResponse, _ := input.ReadLine()
	colorEnabled := colorResponse == "" || strings.ToLower(colorResponse)[0] == 'y'

	fmt.Print("Enable markdown rendering? (Y/n): ")
	markdownResponse, _ := input.ReadLine()
	markdownEnabled := markdownResponse == "" || strings.ToLower(markdownResponse)[0] == 'y'

	fmt.Print("Enable streaming responses? (Y/n): ")
	streamResponse, _ := input.ReadLine()
	streamEnabled := streamResponse == "" || strings.ToLower(streamResponse)[0] == 'y'

	// Cache settings
	formatter.PrintSection("Cache Settings")
	fmt.Print("Enable response caching? (Y/n): ")
	cacheResponse, _ := input.ReadLine()
	cacheEnabled := cacheResponse == "" || strings.ToLower(cacheResponse)[0] == 'y'

	// Create configuration
	cfg := &config.Config{
		OpenAI: config.OpenAIConfig{
			APIKey:      apiKey,
			Model:       selectedModel,
			Temperature: temperature,
			MaxTokens:   maxTokens,
			Timeout:     30 * time.Second,
			BaseURL:     "https://api.openai.com/v1",
			TopP:        1.0,
			N:           1,
		},
		UI: config.UIConfig{
			ColorOutput:        colorEnabled,
			MarkdownRendering:  markdownEnabled,
			StreamingEnabled:   streamEnabled,
			SyntaxHighlighting: true,
			Theme:              "auto",
			Spinner:            "dots",
		},
		Cache: config.CacheConfig{
			Enabled:  cacheEnabled,
			TTL:      5 * time.Minute,
			MaxSize:  100,
			Strategy: "lru",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			NoAPI:  true,
		},
		Profile: "prod",
	}

	// Set cache directory
	home, _ := os.UserHomeDir()
	cfg.Cache.Dir = filepath.Join(home, ".terminal-ai", "cache")

	// Save configuration
	configPath := config.GetDefaultConfigPath()
	if err := cfg.SaveTo(configPath); err != nil {
		formatter.PrintError(fmt.Sprintf("Failed to save configuration: %v", err))
		return err
	}

	formatter.PrintSuccess(fmt.Sprintf("Configuration saved to: %s", configPath))

	// Test connection
	fmt.Println()
	fmt.Print("Test API connection now? (Y/n): ")
	testResponse, _ := input.ReadLine()
	if testResponse == "" || strings.ToLower(testResponse)[0] == 'y' {
		return testAPIConnectionWithConfig(cfg)
	}

	return nil
}

func showConfig() error {
	cfg := GetConfig()
	if cfg == nil {
		// Try to load config directly
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	formatter := ui.NewFormatter(ui.FormatterOptions{
		ColorEnabled: !noColor,
		Width:        ui.GetTerminalWidth(),
	})

	formatter.PrintTitle("Current Configuration")

	// Convert to YAML for display (mask sensitive data)
	displayConfig := map[string]interface{}{
		"profile": cfg.Profile,
		"openai": map[string]interface{}{
			"api_key":     maskAPIKey(cfg.OpenAI.APIKey),
			"model":       cfg.OpenAI.Model,
			"max_tokens":  cfg.OpenAI.MaxTokens,
			"temperature": cfg.OpenAI.Temperature,
			"timeout":     cfg.OpenAI.Timeout.String(),
			"base_url":    cfg.OpenAI.BaseURL,
		},
		"ui": map[string]interface{}{
			"color_output":        cfg.UI.ColorOutput,
			"markdown_rendering":  cfg.UI.MarkdownRendering,
			"streaming_enabled":   cfg.UI.StreamingEnabled,
			"syntax_highlighting": cfg.UI.SyntaxHighlighting,
			"theme":               cfg.UI.Theme,
		},
		"cache": map[string]interface{}{
			"enabled":  cfg.Cache.Enabled,
			"ttl":      cfg.Cache.TTL.String(),
			"max_size": cfg.Cache.MaxSize,
			"strategy": cfg.Cache.Strategy,
			"dir":      cfg.Cache.Dir,
		},
		"logging": map[string]interface{}{
			"level":  cfg.Logging.Level,
			"format": cfg.Logging.Format,
			"file":   cfg.Logging.File,
		},
	}

	data, err := yaml.Marshal(displayConfig)
	if err != nil {
		return fmt.Errorf("failed to format configuration: %w", err)
	}

	fmt.Println(string(data))

	// Show config file location
	formatter.PrintInfo(fmt.Sprintf("Config file: %s", config.GetConfigPath()))

	return nil
}

func validateConfig() error {
	cfg := GetConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	formatter := ui.NewFormatter(ui.FormatterOptions{
		ColorEnabled: !noColor,
		Width:        ui.GetTerminalWidth(),
	})

	formatter.PrintTitle("Validating Configuration")

	validator := config.NewValidator(cfg)
	if err := validator.Validate(); err != nil {
		formatter.PrintError(fmt.Sprintf("Configuration validation failed: %v", err))
		return err
	}

	formatter.PrintSuccess("Configuration is valid")

	// Warnings are handled internally by the validator
	// Additional warnings could be added here if needed

	return nil
}

func testAPIConnection() error {
	cfg := GetConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	return testAPIConnectionWithConfig(cfg)
}

func testAPIConnectionWithConfig(cfg *config.Config) error {
	formatter := ui.NewFormatter(ui.FormatterOptions{
		ColorEnabled: cfg.UI.ColorOutput && !noColor,
		Width:        ui.GetTerminalWidth(),
	})

	formatter.PrintTitle("Testing API Connection")

	// Create AI client
	spinner := ui.NewSimpleSpinner("Connecting to OpenAI API...")
	spinner.Start()

	client, err := ai.NewOpenAIClient(cfg)
	if err != nil {
		spinner.StopWithError(fmt.Sprintf("Failed to create client: %v", err))
		return err
	}
	defer client.Close()

	// Test with a simple query
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := client.Query(ctx, "Say 'Hello, Terminal AI!' if you can hear me.")
	if err != nil {
		spinner.StopWithError(fmt.Sprintf("API test failed: %v", err))
		return err
	}

	spinner.StopWithSuccess("API connection successful")

	// Show response
	formatter.PrintSection("Test Response")
	fmt.Println(response)

	// List available models
	fmt.Println()
	formatter.PrintSection("Available Models")

	models, err := client.ListModels(ctx)
	if err != nil {
		formatter.PrintWarning(fmt.Sprintf("Failed to list models: %v", err))
	} else {
		// Filter and show relevant models
		relevantModels := []string{}
		for _, model := range models {
			if strings.Contains(model, "gpt") || strings.Contains(model, "dall-e") {
				relevantModels = append(relevantModels, model)
			}
		}

		if len(relevantModels) > 0 {
			for _, model := range relevantModels {
				fmt.Printf("  - %s\n", model)
			}
		}
	}

	return nil
}

func editConfig() error {
	configFile := config.GetConfigPath()

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		formatter := ui.NewFormatter(ui.FormatterOptions{
			ColorEnabled: !noColor,
		})
		formatter.PrintError("Configuration file not found")
		fmt.Println("Run 'terminal-ai config --init' to create one")
		return err
	}

	// Determine editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Default editors by platform
		switch runtime.GOOS {
		case "windows":
			editor = "notepad"
		case "darwin":
			editor = "nano"
		default:
			editor = "vi"
		}
	}

	// Open editor
	cmd := exec.Command(editor, configFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	// Validate after editing
	fmt.Println()
	fmt.Print("Validate configuration? (Y/n): ")

	var response string
	fmt.Scanln(&response)

	if response == "" || strings.ToLower(response)[0] == 'y' {
		return validateConfig()
	}

	return nil
}

func showConfigLocation() error {
	formatter := ui.NewFormatter(ui.FormatterOptions{
		ColorEnabled: !noColor,
	})

	configFile := config.GetConfigPath()
	defaultFile := config.GetDefaultConfigPath()

	formatter.PrintTitle("Configuration Locations")

	fmt.Printf("Current config file: %s\n", configFile)
	fmt.Printf("Default location:    %s\n", defaultFile)

	// Check if file exists
	if _, err := os.Stat(configFile); err == nil {
		info, _ := os.Stat(configFile)
		fmt.Printf("File size:           %d bytes\n", info.Size())
		fmt.Printf("Last modified:       %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	} else {
		formatter.PrintWarning("Configuration file does not exist")
		fmt.Println("Run 'terminal-ai config --init' to create one")
	}

	// Show environment variables
	fmt.Println()
	formatter.PrintSection("Environment Variables")

	envVars := []string{
		"TERMINAL_AI_CONFIG",
		"TERMINAL_AI_PROFILE",
		"OPENAI_API_KEY",
		"TERMINAL_AI_OPENAI_API_KEY",
		"TERMINAL_AI_LOG_LEVEL",
	}

	for _, env := range envVars {
		value := os.Getenv(env)
		if value != "" {
			if strings.Contains(env, "KEY") {
				value = maskAPIKey(value)
			}
			fmt.Printf("  %s = %s\n", env, value)
		}
	}

	return nil
}

func setConfigValue(key, value string) error {
	// Load current config
	cfg := GetConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			// Create new config if it doesn't exist
			cfg = &config.Config{}
		}
	}

	// Parse and set value based on key
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid key format. Use dot notation (e.g., openai.model)")
	}

	switch parts[0] {
	case "openai":
		switch parts[1] {
		case "api_key":
			cfg.OpenAI.APIKey = value
		case "model":
			cfg.OpenAI.Model = value
		case "max_tokens":
			cfg.OpenAI.MaxTokens = parseInt(value)
		case "temperature":
			cfg.OpenAI.Temperature = parseFloat32(value)
		case "base_url":
			cfg.OpenAI.BaseURL = value
		case "timeout":
			if d, err := time.ParseDuration(value); err == nil {
				cfg.OpenAI.Timeout = d
			}
		default:
			return fmt.Errorf("unknown OpenAI config key: %s", parts[1])
		}
	case "ui":
		switch parts[1] {
		case "color_output":
			cfg.UI.ColorOutput = parseBool(value)
		case "markdown_rendering":
			cfg.UI.MarkdownRendering = parseBool(value)
		case "streaming_enabled":
			cfg.UI.StreamingEnabled = parseBool(value)
		case "syntax_highlighting":
			cfg.UI.SyntaxHighlighting = parseBool(value)
		case "theme":
			cfg.UI.Theme = value
		default:
			return fmt.Errorf("unknown UI config key: %s", parts[1])
		}
	case "cache":
		switch parts[1] {
		case "enabled":
			cfg.Cache.Enabled = parseBool(value)
		case "ttl":
			if d, err := time.ParseDuration(value); err == nil {
				cfg.Cache.TTL = d
			}
		case "max_size":
			cfg.Cache.MaxSize = parseInt(value)
		case "strategy":
			cfg.Cache.Strategy = value
		case "dir":
			cfg.Cache.Dir = value
		default:
			return fmt.Errorf("unknown cache config key: %s", parts[1])
		}
	case "logging":
		switch parts[1] {
		case "level":
			cfg.Logging.Level = value
		case "format":
			cfg.Logging.Format = value
		case "file":
			cfg.Logging.File = value
		case "no_api":
			cfg.Logging.NoAPI = parseBool(value)
		default:
			return fmt.Errorf("unknown logging config key: %s", parts[1])
		}
	default:
		return fmt.Errorf("unknown config section: %s", parts[0])
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	formatter := ui.NewFormatter(ui.FormatterOptions{
		ColorEnabled: !noColor,
	})
	formatter.PrintSuccess(fmt.Sprintf("Set %s = %s", key, value))

	return nil
}

func getConfigValue(key string) error {
	cfg := GetConfig()
	if cfg == nil {
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	value := cfg.GetString(key)
	if value == "" {
		return fmt.Errorf("key '%s' not found or empty", key)
	}

	// Mask API keys
	if strings.Contains(key, "api_key") {
		value = maskAPIKey(value)
	}

	fmt.Printf("%s = %s\n", key, value)
	return nil
}

// Helper functions
func maskAPIKey(key string) string {
	if key == "" {
		return "<not set>"
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func parseFloat32(s string) float32 {
	var f float32
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "yes" || s == "1" || s == "y"
}

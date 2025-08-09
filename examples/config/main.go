// Package main demonstrates how to use the configuration management system
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/user/terminal-ai/internal/config"
)

func main() {
	// Example 1: Load configuration with defaults and environment variables
	fmt.Println("=== Example 1: Loading configuration ===")
	cfg, err := config.Load("")
	if err != nil {
		// If no API key is set, provide helpful instructions
		if err.Error() == "configuration validation failed: OpenAI API key is required (set OPENAI_API_KEY or configure in file)" {
			fmt.Println("Please set your OpenAI API key:")
			fmt.Println("  export OPENAI_API_KEY=your-api-key-here")
			fmt.Println("Or create a config file at ~/.terminal-ai/config.yaml")
			os.Exit(1)
		}
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Display loaded configuration (with API key masked)
	fmt.Printf("Profile: %s\n", cfg.Profile)
	fmt.Printf("Model: %s\n", cfg.OpenAI.Model)
	fmt.Printf("Max Tokens: %d\n", cfg.OpenAI.MaxTokens)
	fmt.Printf("Temperature: %.2f\n", cfg.OpenAI.Temperature)
	fmt.Printf("Cache Enabled: %v\n", cfg.Cache.Enabled)
	fmt.Printf("UI Theme: %s\n", cfg.UI.Theme)
	fmt.Printf("Log Level: %s\n", cfg.Logging.Level)
	fmt.Println()

	// Example 2: Load configuration with a specific profile
	fmt.Println("=== Example 2: Loading with dev profile ===")
	devCfg, err := config.LoadWithProfile("", "dev")
	if err != nil {
		log.Printf("Failed to load dev profile: %v", err)
	} else {
		fmt.Printf("Dev Profile Log Level: %s\n", devCfg.Logging.Level)
		fmt.Printf("Dev Profile Cache Enabled: %v\n", devCfg.Cache.Enabled)
	}
	fmt.Println()

	// Example 3: Validate configuration
	fmt.Println("=== Example 3: Configuration validation ===")
	validator := config.NewValidator(cfg)
	if err := validator.Validate(); err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		for _, e := range validator.GetErrors() {
			fmt.Printf("  - %s\n", e)
		}
	} else {
		fmt.Println("Configuration is valid!")
	}
	fmt.Println()

	// Example 4: Save configuration to file
	fmt.Println("=== Example 4: Saving configuration ===")
	tempFile := "/tmp/terminal-ai-example.yaml"
	if err := cfg.SaveTo(tempFile); err != nil {
		log.Printf("Failed to save configuration: %v", err)
	} else {
		fmt.Printf("Configuration saved to: %s\n", tempFile)
		fmt.Println("Note: API key is automatically masked in saved files")

		// Read and display a snippet of the saved file
		content, _ := os.ReadFile(tempFile)
		fmt.Printf("First few lines of saved config:\n%s\n", string(content[:min(200, len(content))])+"...")

		// Clean up
		os.Remove(tempFile)
	}
	fmt.Println()

	// Example 5: Access nested configuration values
	fmt.Println("=== Example 5: Accessing configuration values ===")
	fmt.Printf("OpenAI Base URL: %s\n", cfg.GetString("openai.base_url"))
	fmt.Printf("OpenAI Model: %s\n", cfg.GetString("openai.model"))
	fmt.Printf("Logging Format: %s\n", cfg.GetString("logging.format"))
	fmt.Println()

	// Example 6: Environment variable priority
	fmt.Println("=== Example 6: Environment variable priority ===")
	fmt.Println("Configuration priority (highest to lowest):")
	fmt.Println("1. Command-line flags (when implemented)")
	fmt.Println("2. Environment variables (TERMINAL_AI_* or OPENAI_API_KEY)")
	fmt.Println("3. Config file (~/.terminal-ai/config.yaml or .terminal-ai.yaml)")
	fmt.Println("4. Default values")
	fmt.Println()

	// Example 7: Secure API key handling
	fmt.Println("=== Example 7: Secure API key handling ===")
	fmt.Println("API keys are:")
	fmt.Println("- Never logged when logging.no_api is true")
	fmt.Println("- Masked as ${OPENAI_API_KEY} when saved to files")
	fmt.Println("- Validated for proper format")
	fmt.Println("- Can be loaded from files with restricted permissions (0600)")

	// Demonstrate API key validation
	if keyFile := os.Getenv("TERMINAL_AI_API_KEY_FILE"); keyFile != "" {
		if err := config.ValidateAPIKeyFile(keyFile); err != nil {
			fmt.Printf("API key file validation failed: %v\n", err)
		} else {
			fmt.Println("API key file is secure and valid")
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

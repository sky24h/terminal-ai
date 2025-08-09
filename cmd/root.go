package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/user/terminal-ai/internal/ai"
	"github.com/user/terminal-ai/internal/config"
	"github.com/user/terminal-ai/internal/utils"
)

var (
	cfgFile   string
	verbose   bool
	noColor   bool
	profile   string
	aiClient  ai.Client
	appConfig *config.Config
	logger    *utils.Logger
)

const version = "0.1.0"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "terminal-ai",
	Short: "A powerful terminal-based AI assistant",
	Long: `Terminal AI is a command-line interface for interacting with AI models.
It provides chat capabilities, quick queries, and streaming responses
directly in your terminal.

Quick Start:
  terminal-ai -q "What is Go?"
  terminal-ai -s "list docker containers"
  terminal-ai -c`,
	Version: version,
	DisableFlagParsing: false,
	TraverseChildren: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.opt/terminal-ai-config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "config profile to use (dev, prod)")

	// Bind flags to viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("ui.color_output", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of terminal-ai",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("terminal-ai version %s\n", version)
		},
	})
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
			os.Exit(1)
		}

		// Search config in multiple locations (priority order)
		viper.AddConfigPath(filepath.Join(home, ".opt"))         // Primary location
		viper.AddConfigPath(filepath.Join(home, ".terminal-ai")) // Legacy location
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("terminal-ai-config") // Primary config name

		// Support multiple config file names for backward compatibility
		if _, err := os.Stat(filepath.Join(home, ".opt", "terminal-ai-config.yaml")); err == nil {
			viper.SetConfigFile(filepath.Join(home, ".opt", "terminal-ai-config.yaml"))
		} else if _, err := os.Stat(filepath.Join(home, ".terminal-ai.yaml")); err == nil {
			viper.SetConfigFile(filepath.Join(home, ".terminal-ai.yaml"))
		}
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("TERMINAL_AI")
	viper.AutomaticEnv()

	// Support OPENAI_API_KEY environment variable
	viper.BindEnv("openai.api_key", "OPENAI_API_KEY")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

// initializeApp initializes the application configuration and AI client
func initializeApp() error {
	var err error

	// Load configuration
	if profile != "" {
		appConfig, err = config.LoadWithProfile(cfgFile, profile)
	} else {
		appConfig, err = config.Load(cfgFile)
	}

	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply command-line flag overrides
	if noColor {
		appConfig.UI.ColorOutput = false
	}

	// Initialize logger
	logLevel := appConfig.Logging.Level
	if verbose {
		logLevel = "debug"
	}

	logger = utils.NewLogger(utils.LogConfig{
		Level:         logLevel,
		Output:        appConfig.Logging.Format, // "console", "json", or "both"
		FilePath:      appConfig.Logging.File,
		Pretty:        appConfig.Logging.Format == "pretty" || appConfig.Logging.Format == "text",
		MaskSensitive: appConfig.Logging.NoAPI,
		StackTrace:    verbose || logLevel == "debug",
	})

	// Set global logger
	utils.SetLogger(logger)

	// Initialize AI client
	aiClient, err = ai.NewOpenAIClient(appConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize AI client: %w", err)
	}

	return nil
}

// Cleanup performs cleanup operations
func Cleanup() {
	if aiClient != nil {
		aiClient.Close()
	}
	if logger != nil {
		logger.Close()
	}
}

// GetAIClient returns the initialized AI client
func GetAIClient() ai.Client {
	return aiClient
}

// GetConfig returns the application configuration
func GetConfig() *config.Config {
	return appConfig
}

// GetLogger returns the application logger
func GetLogger() *utils.Logger {
	return logger
}

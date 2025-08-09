package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/user/terminal-ai/cmd"
	"github.com/user/terminal-ai/internal/utils"
)

func main() {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		// Cleanup on exit
		cmd.Cleanup()
		os.Exit(0)
	}()

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		logger := utils.GetLogger()
		if logger != nil {
			logger.Error("Failed to execute command", err)
		}
		os.Exit(1)
	}
}

package ui

import (
	"golang.org/x/term"
	"os"
)

// GetTerminalWidth returns the current terminal width
func GetTerminalWidth() int {
	width, _, err := getTerminalSize()
	if err != nil || width <= 0 {
		return 80 // Default width
	}
	return width
}

// GetTerminalHeight returns the current terminal height
func GetTerminalHeight() int {
	_, height, err := getTerminalSize()
	if err != nil || height <= 0 {
		return 24 // Default height
	}
	return height
}

// getTerminalSize returns the width and height of the terminal
func getTerminalSize() (int, int, error) {
	fd := int(os.Stdout.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		// Fallback to default sizes
		return 80, 24, nil
	}
	return width, height, nil
}

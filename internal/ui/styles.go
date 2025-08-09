package ui

import (
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

var (
	currentTheme *Theme
	themeMutex   sync.RWMutex
)

// Theme represents a color theme for the UI
type Theme struct {
	// Base colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Tertiary  lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Error   lipgloss.Color
	Warning lipgloss.Color
	Info    lipgloss.Color

	// Text colors
	Text       lipgloss.Color
	TextMuted  lipgloss.Color
	TextBright lipgloss.Color

	// AI/User distinction colors
	AIResponse lipgloss.Color // Rose red for AI responses
	UserInput  lipgloss.Color // Different color for user input

	// Background colors
	Background    lipgloss.Color
	BackgroundAlt lipgloss.Color

	// Border colors
	Border      lipgloss.Color
	BorderFocus lipgloss.Color

	// Code colors
	CodeBackground lipgloss.Color
	CodeText       lipgloss.Color
	CodeKeyword    lipgloss.Color
	CodeString     lipgloss.Color
	CodeComment    lipgloss.Color
	CodeFunction   lipgloss.Color
}

// DarkTheme returns a dark color theme
func DarkTheme() *Theme {
	return &Theme{
		// Base colors
		Primary:   lipgloss.Color("205"), // Pink
		Secondary: lipgloss.Color("86"),  // Cyan
		Tertiary:  lipgloss.Color("99"),  // Purple

		// Status colors
		Success: lipgloss.Color("42"),  // Green
		Error:   lipgloss.Color("196"), // Red
		Warning: lipgloss.Color("214"), // Orange
		Info:    lipgloss.Color("69"),  // Blue

		// Text colors
		Text:       lipgloss.Color("252"), // Light gray
		TextMuted:  lipgloss.Color("245"), // Medium gray
		TextBright: lipgloss.Color("255"), // White

		// AI/User distinction colors
		AIResponse: lipgloss.Color("211"), // Rose red for AI responses
		UserInput:  lipgloss.Color("86"),  // Cyan for user input

		// Background colors
		Background:    lipgloss.Color("235"), // Dark gray
		BackgroundAlt: lipgloss.Color("238"), // Slightly lighter gray

		// Border colors
		Border:      lipgloss.Color("240"), // Gray
		BorderFocus: lipgloss.Color("205"), // Pink

		// Code colors
		CodeBackground: lipgloss.Color("236"), // Very dark gray
		CodeText:       lipgloss.Color("252"), // Light gray
		CodeKeyword:    lipgloss.Color("205"), // Pink
		CodeString:     lipgloss.Color("42"),  // Green
		CodeComment:    lipgloss.Color("245"), // Gray
		CodeFunction:   lipgloss.Color("86"),  // Cyan
	}
}

// LightTheme returns a light color theme
func LightTheme() *Theme {
	return &Theme{
		// Base colors
		Primary:   lipgloss.Color("162"), // Magenta
		Secondary: lipgloss.Color("33"),  // Blue
		Tertiary:  lipgloss.Color("99"),  // Purple

		// Status colors
		Success: lipgloss.Color("34"),  // Green
		Error:   lipgloss.Color("160"), // Red
		Warning: lipgloss.Color("178"), // Yellow
		Info:    lipgloss.Color("33"),  // Blue

		// Text colors
		Text:       lipgloss.Color("235"), // Dark gray
		TextMuted:  lipgloss.Color("244"), // Medium gray
		TextBright: lipgloss.Color("232"), // Black

		// AI/User distinction colors
		AIResponse: lipgloss.Color("204"), // Rose red for AI responses
		UserInput:  lipgloss.Color("33"),  // Blue for user input

		// Background colors
		Background:    lipgloss.Color("255"), // White
		BackgroundAlt: lipgloss.Color("254"), // Off-white

		// Border colors
		Border:      lipgloss.Color("250"), // Light gray
		BorderFocus: lipgloss.Color("162"), // Magenta

		// Code colors
		CodeBackground: lipgloss.Color("254"), // Off-white
		CodeText:       lipgloss.Color("235"), // Dark gray
		CodeKeyword:    lipgloss.Color("162"), // Magenta
		CodeString:     lipgloss.Color("34"),  // Green
		CodeComment:    lipgloss.Color("244"), // Gray
		CodeFunction:   lipgloss.Color("33"),  // Blue
	}
}

// GetCurrentTheme returns the current theme
func GetCurrentTheme() *Theme {
	themeMutex.RLock()
	defer themeMutex.RUnlock()

	if currentTheme == nil {
		// Default to dark theme
		return DarkTheme()
	}
	return currentTheme
}

// SetTheme sets the current theme
func SetTheme(theme *Theme) {
	themeMutex.Lock()
	defer themeMutex.Unlock()
	currentTheme = theme
}

// SetThemeFromEnv sets the theme based on environment
func SetThemeFromEnv() {
	// Check for theme preference in environment
	if os.Getenv("TERMINAL_AI_THEME") == "light" {
		SetTheme(LightTheme())
	} else {
		SetTheme(DarkTheme())
	}
}

// MessageStyle represents styling for different message types
type MessageStyle struct {
	Icon      string
	IconStyle lipgloss.Style
	TextStyle lipgloss.Style
	BoxStyle  lipgloss.Style
}

// GetMessageStyles returns styles for different message types
func GetMessageStyles(theme *Theme) map[string]MessageStyle {
	return map[string]MessageStyle{
		"success": {
			Icon:      "✓",
			IconStyle: lipgloss.NewStyle().Foreground(theme.Success),
			TextStyle: lipgloss.NewStyle().Foreground(theme.Text),
			BoxStyle: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Success).
				Padding(0, 1),
		},
		"error": {
			Icon:      "✗",
			IconStyle: lipgloss.NewStyle().Foreground(theme.Error),
			TextStyle: lipgloss.NewStyle().Foreground(theme.Text),
			BoxStyle: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Error).
				Padding(0, 1),
		},
		"warning": {
			Icon:      "⚠",
			IconStyle: lipgloss.NewStyle().Foreground(theme.Warning),
			TextStyle: lipgloss.NewStyle().Foreground(theme.Text),
			BoxStyle: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Warning).
				Padding(0, 1),
		},
		"info": {
			Icon:      "ℹ",
			IconStyle: lipgloss.NewStyle().Foreground(theme.Info),
			TextStyle: lipgloss.NewStyle().Foreground(theme.Text),
			BoxStyle: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Info).
				Padding(0, 1),
		},
	}
}

// CommonStyles provides commonly used styles
type CommonStyles struct {
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Header   lipgloss.Style
	Body     lipgloss.Style
	Footer   lipgloss.Style

	Bold      lipgloss.Style
	Italic    lipgloss.Style
	Underline lipgloss.Style

	Highlight lipgloss.Style
	Selected  lipgloss.Style
	Focused   lipgloss.Style

	CodeBlock  lipgloss.Style
	CodeInline lipgloss.Style

	Table       lipgloss.Style
	TableHeader lipgloss.Style
	TableRow    lipgloss.Style
	TableCell   lipgloss.Style
}

// GetCommonStyles returns commonly used styles for the theme
func GetCommonStyles(theme *Theme) *CommonStyles {
	return &CommonStyles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			MarginBottom(1),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBright).
			Background(theme.BackgroundAlt).
			Padding(0, 1),

		Body: lipgloss.NewStyle().
			Foreground(theme.Text),

		Footer: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			MarginTop(1),

		Bold: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBright),

		Italic: lipgloss.NewStyle().
			Italic(true).
			Foreground(theme.Text),

		Underline: lipgloss.NewStyle().
			Underline(true).
			Foreground(theme.Text),

		Highlight: lipgloss.NewStyle().
			Background(theme.Warning).
			Foreground(theme.Background),

		Selected: lipgloss.NewStyle().
			Background(theme.Primary).
			Foreground(theme.Background),

		Focused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderFocus),

		CodeBlock: lipgloss.NewStyle().
			Background(theme.CodeBackground).
			Foreground(theme.CodeText).
			Padding(1).
			MarginTop(1).
			MarginBottom(1),

		CodeInline: lipgloss.NewStyle().
			Background(theme.CodeBackground).
			Foreground(theme.CodeText).
			Padding(0, 1),

		Table: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Border),

		TableHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBright).
			Background(theme.BackgroundAlt).
			Padding(0, 1),

		TableRow: lipgloss.NewStyle().
			Foreground(theme.Text),

		TableCell: lipgloss.NewStyle().
			Padding(0, 1),
	}
}

// IsTTY checks if the output is a TTY
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// AdaptiveStyle returns a style that adapts to TTY/non-TTY environments
type AdaptiveStyle struct {
	TTYStyle    lipgloss.Style
	NonTTYStyle lipgloss.Style
}

// Render renders content with the appropriate style
func (a *AdaptiveStyle) Render(content string) string {
	if IsTTY() {
		return a.TTYStyle.Render(content)
	}
	return a.NonTTYStyle.Render(content)
}

// NewAdaptiveStyle creates a new adaptive style
func NewAdaptiveStyle(ttyStyle, nonTTYStyle lipgloss.Style) *AdaptiveStyle {
	return &AdaptiveStyle{
		TTYStyle:    ttyStyle,
		NonTTYStyle: nonTTYStyle,
	}
}

func init() {
	// Set theme from environment on package initialization
	SetThemeFromEnv()
}

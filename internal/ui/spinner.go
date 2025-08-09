package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// SpinnerStyle represents different spinner animation styles
type SpinnerStyle int

const (
	SpinnerDots SpinnerStyle = iota
	SpinnerLine
	SpinnerGlobe
	SpinnerMoon
	SpinnerBouncingBar
	SpinnerPulse
	SpinnerArrows
	SpinnerCircle
)

// getSpinnerFrames returns frames for the given spinner style
func getSpinnerFrames(style SpinnerStyle) spinner.Spinner {
	switch style {
	case SpinnerLine:
		return spinner.Line
	case SpinnerGlobe:
		return spinner.Globe
	case SpinnerMoon:
		return spinner.Moon
	case SpinnerBouncingBar:
		return spinner.Jump
	case SpinnerPulse:
		return spinner.Pulse
	case SpinnerArrows:
		// Arrow spinner doesn't exist, use Points instead
		return spinner.Points
	case SpinnerCircle:
		// Circle spinner doesn't exist, use MiniDot instead
		return spinner.MiniDot
	default:
		return spinner.Dot
	}
}

// SpinnerModel represents a loading spinner with customizable messages
type SpinnerModel struct {
	spinner       spinner.Model
	message       string
	quitting      bool
	style         SpinnerStyle
	customMessage string
	theme         *Theme
}

// NewSpinner creates a new spinner model
func NewSpinner(message string) *SpinnerModel {
	return NewSpinnerWithStyle(message, SpinnerDots)
}

// NewSpinnerWithStyle creates a new spinner with a specific style
func NewSpinnerWithStyle(message string, style SpinnerStyle) *SpinnerModel {
	theme := GetCurrentTheme()
	return NewSpinnerWithTheme(message, style, theme)
}

// NewSpinnerWithTheme creates a new spinner with a specific style and theme
func NewSpinnerWithTheme(message string, style SpinnerStyle, theme *Theme) *SpinnerModel {
	s := spinner.New()
	s.Spinner = getSpinnerFrames(style)
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	return &SpinnerModel{
		spinner: s,
		message: message,
		style:   style,
		theme:   theme,
	}
}

// SetMessage updates the spinner message
func (m *SpinnerModel) SetMessage(message string) {
	m.message = message
}

// SetCustomMessage sets a custom loading message
func (m *SpinnerModel) SetCustomMessage(message string) {
	m.customMessage = message
}

// Init initializes the spinner
func (m *SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles spinner updates
func (m *SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}

// View renders the spinner
func (m *SpinnerModel) View() string {
	if m.quitting {
		return ""
	}

	msg := m.message
	if m.customMessage != "" {
		msg = m.customMessage
	}

	str := fmt.Sprintf("%s %s", m.spinner.View(), msg)
	return str
}

// Spinner provides a thread-safe spinner for concurrent operations
type Spinner struct {
	frames        []string
	current       int
	message       string
	customMessage string
	stop          chan bool
	stopped       bool
	mu            sync.Mutex
	isActive      bool
	isTTY         bool
	theme         *Theme
}

// NewSimpleSpinner creates a simple spinner
func NewSimpleSpinner(message string) *Spinner {
	return NewSimpleSpinnerWithStyle(message, SpinnerDots)
}

// NewSimpleSpinnerWithStyle creates a spinner with a specific style
func NewSimpleSpinnerWithStyle(message string, style SpinnerStyle) *Spinner {
	theme := GetCurrentTheme()
	return NewSimpleSpinnerWithTheme(message, style, theme)
}

// NewSimpleSpinnerWithTheme creates a spinner with a specific style and theme
func NewSimpleSpinnerWithTheme(message string, style SpinnerStyle, theme *Theme) *Spinner {
	var frames []string

	switch style {
	case SpinnerLine:
		frames = []string{"-", "\\", "|", "/"}
	case SpinnerGlobe:
		frames = []string{"🌍", "🌎", "🌏"}
	case SpinnerMoon:
		frames = []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}
	case SpinnerBouncingBar:
		frames = []string{"[    ]", "[=   ]", "[==  ]", "[=== ]", "[====]", "[ ===]", "[  ==]", "[   =]"}
	case SpinnerPulse:
		frames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	case SpinnerArrows:
		frames = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	case SpinnerCircle:
		frames = []string{"◐", "◓", "◑", "◒"}
	default:
		frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	}

	return &Spinner{
		frames:  frames,
		current: 0,
		message: message,
		stop:    make(chan bool, 1),
		stopped: false,
		isTTY:   isatty.IsTerminal(os.Stdout.Fd()),
		theme:   theme,
	}
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// SetCustomMessage sets a custom loading message
func (s *Spinner) SetCustomMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.customMessage = message
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.isActive {
		s.mu.Unlock()
		return
	}
	s.isActive = true
	s.stopped = false
	s.mu.Unlock()

	// Don't show spinner in non-TTY environments
	if !s.isTTY {
		fmt.Println(s.message)
		return
	}

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				if !s.stopped {
					msg := s.message
					if s.customMessage != "" {
						msg = s.customMessage
					}
					fmt.Printf("\r%s %s", s.frames[s.current], msg)
					s.current = (s.current + 1) % len(s.frames)
				}
				s.mu.Unlock()
			case <-s.stop:
				if s.isTTY {
					fmt.Print("\r\033[K") // Clear line
				}
				s.mu.Lock()
				s.isActive = false
				s.mu.Unlock()
				return
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.isActive {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	select {
	case s.stop <- true:
	default:
		// Channel already has a value, ignore
	}

	// Wait a bit for the goroutine to clean up
	time.Sleep(50 * time.Millisecond)
}

// StopWithMessage stops the spinner and displays a final message
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	if s.isTTY {
		fmt.Println(message)
	}
}

// StopWithSuccess stops the spinner with a success message
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	if s.isTTY {
		style := lipgloss.NewStyle().Foreground(s.theme.Success)
		fmt.Println(style.Render("✓ " + message))
	} else {
		fmt.Println(message)
	}
}

// StopWithError stops the spinner with an error message
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	if s.isTTY {
		style := lipgloss.NewStyle().Foreground(s.theme.Error)
		fmt.Println(style.Render("✗ " + message))
	} else {
		fmt.Printf("Error: %s\n", message)
	}
}

// StopWithWarning stops the spinner with a warning message
func (s *Spinner) StopWithWarning(message string) {
	s.Stop()
	if s.isTTY {
		style := lipgloss.NewStyle().Foreground(s.theme.Warning)
		fmt.Println(style.Render("⚠ " + message))
	} else {
		fmt.Printf("Warning: %s\n", message)
	}
}

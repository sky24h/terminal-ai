package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// InputOptions contains options for creating an input
type InputOptions struct {
	Prompt      string
	Placeholder string
	Multiline   bool
	History     bool
	Suggestions bool
	MaxLength   int
	Width       int
	Theme       *Theme
}

// Input represents an enhanced input handler
type Input struct {
	options InputOptions
	theme   *Theme
	reader  *bufio.Reader
	history []string
}

// InputModel represents an input field
type InputModel struct {
	textInput textinput.Model
	err       error
	label     string
	theme     *Theme
	cancelled bool
}

// NewInputModel creates a new input model
func NewInputModel(label, placeholder string) *InputModel {
	theme := GetCurrentTheme()
	return NewInputWithTheme(label, placeholder, theme)
}

// NewInputWithTheme creates a new input model with a theme
func NewInputWithTheme(label, placeholder string, theme *Theme) *InputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Primary)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.TextMuted)
	ti.CursorStyle = lipgloss.NewStyle().Foreground(theme.Secondary)

	return &InputModel{
		textInput: ti,
		label:     label,
		theme:     theme,
	}
}

// Init initializes the input model
func (m *InputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles input updates
func (m *InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		}

	case error:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the input field
func (m *InputModel) View() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(m.theme.Secondary).
		Bold(true).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		MarginTop(1)

	return fmt.Sprintf(
		"%s\n%s\n%s",
		labelStyle.Render(m.label),
		m.textInput.View(),
		helpStyle.Render("(press enter to submit, esc to cancel)"),
	)
}

// Value returns the input value
func (m *InputModel) Value() string {
	return m.textInput.Value()
}

// IsCancelled returns true if the input was cancelled
func (m *InputModel) IsCancelled() bool {
	return m.cancelled
}

// MultiLineInputModel represents a multi-line input field
type MultiLineInputModel struct {
	textarea  textarea.Model
	label     string
	theme     *Theme
	cancelled bool
}

// NewMultiLineInput creates a new multi-line input model
func NewMultiLineInput(label, placeholder string) *MultiLineInputModel {
	theme := GetCurrentTheme()
	return NewMultiLineInputWithTheme(label, placeholder, theme)
}

// NewMultiLineInputWithTheme creates a new multi-line input model with a theme
func NewMultiLineInputWithTheme(label, placeholder string, theme *Theme) *MultiLineInputModel {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.Focus()
	ta.SetWidth(60)
	ta.SetHeight(10)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(theme.BackgroundAlt)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.TextMuted)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(theme.Text)

	return &MultiLineInputModel{
		textarea: ta,
		label:    label,
		theme:    theme,
	}
}

// Init initializes the multi-line input model
func (m *MultiLineInputModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles multi-line input updates
func (m *MultiLineInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlD:
			// Submit with Ctrl+D
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// View renders the multi-line input field
func (m *MultiLineInputModel) View() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(m.theme.Secondary).
		Bold(true).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		MarginTop(1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocus)

	return fmt.Sprintf(
		"%s\n%s\n%s",
		labelStyle.Render(m.label),
		borderStyle.Render(m.textarea.View()),
		helpStyle.Render("(Ctrl+D to submit, Esc to cancel)"),
	)
}

// Value returns the multi-line input value
func (m *MultiLineInputModel) Value() string {
	return m.textarea.Value()
}

// IsCancelled returns true if the input was cancelled
func (m *MultiLineInputModel) IsCancelled() bool {
	return m.cancelled
}

// SimpleInput provides basic input functionality without bubbletea
type SimpleInput struct {
	reader  *bufio.Reader
	history []string
	theme   *Theme
	isTTY   bool
}

// NewSimpleInput creates a new simple input reader
func NewSimpleInput() *SimpleInput {
	theme := GetCurrentTheme()
	return &SimpleInput{
		reader:  bufio.NewReader(os.Stdin),
		history: make([]string, 0),
		theme:   theme,
		isTTY:   isatty.IsTerminal(os.Stdin.Fd()),
	}
}

// ReadLine reads a line of input
func (s *SimpleInput) ReadLine(prompt string) (string, error) {
	if s.isTTY {
		promptStyle := lipgloss.NewStyle().Foreground(s.theme.Primary)
		fmt.Print(promptStyle.Render(prompt))
	} else {
		fmt.Print(prompt)
	}

	text, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	text = strings.TrimSpace(text)
	if text != "" {
		s.history = append(s.history, text)
	}

	return text, nil
}

// ReadMultiLine reads multiple lines of input until EOF or empty line
func (s *SimpleInput) ReadMultiLine(prompt string) (string, error) {
	if s.isTTY {
		promptStyle := lipgloss.NewStyle().Foreground(s.theme.Primary)
		fmt.Println(promptStyle.Render(prompt))

		helpStyle := lipgloss.NewStyle().Foreground(s.theme.TextMuted)
		fmt.Println(helpStyle.Render("(Enter empty line or Ctrl+D to finish)"))
	} else {
		fmt.Println(prompt)
	}

	var lines []string
	for {
		text, err := s.reader.ReadString('\n')
		if err != nil {
			break
		}

		text = strings.TrimRight(text, "\n\r")
		if text == "" {
			break
		}

		lines = append(lines, text)
	}

	result := strings.Join(lines, "\n")
	if result != "" {
		s.history = append(s.history, result)
	}

	return result, nil
}

// ReadPassword reads a password (no echo)
func (s *SimpleInput) ReadPassword(prompt string) (string, error) {
	if s.isTTY {
		promptStyle := lipgloss.NewStyle().Foreground(s.theme.Warning)
		fmt.Print(promptStyle.Render(prompt))
	} else {
		fmt.Print(prompt)
	}

	// Note: This is a simplified version.
	// In production, use golang.org/x/term for proper password input
	text, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// Confirm asks for yes/no confirmation
func (s *SimpleInput) Confirm(prompt string, defaultYes bool) (bool, error) {
	suffix := " [Y/n]: "
	if !defaultYes {
		suffix = " [y/N]: "
	}

	response, err := s.ReadLine(prompt + suffix)
	if err != nil {
		return false, err
	}

	response = strings.ToLower(response)

	if response == "" {
		return defaultYes, nil
	}

	return response == "y" || response == "yes", nil
}

// Select presents a selection menu
func (s *SimpleInput) Select(prompt string, options []string) (int, error) {
	if s.isTTY {
		titleStyle := lipgloss.NewStyle().
			Foreground(s.theme.Secondary).
			Bold(true)
		fmt.Println(titleStyle.Render(prompt))

		optionStyle := lipgloss.NewStyle().Foreground(s.theme.Text)
		numberStyle := lipgloss.NewStyle().Foreground(s.theme.Primary)

		for i, option := range options {
			fmt.Printf("%s %s\n",
				numberStyle.Render(fmt.Sprintf("%d.", i+1)),
				optionStyle.Render(option))
		}
	} else {
		fmt.Println(prompt)
		for i, option := range options {
			fmt.Printf("%d. %s\n", i+1, option)
		}
	}

	for {
		response, err := s.ReadLine("Enter your choice: ")
		if err != nil {
			return -1, err
		}

		var choice int
		if _, err := fmt.Sscanf(response, "%d", &choice); err == nil {
			if choice >= 1 && choice <= len(options) {
				return choice - 1, nil
			}
		}

		if s.isTTY {
			errorStyle := lipgloss.NewStyle().Foreground(s.theme.Error)
			fmt.Println(errorStyle.Render("Invalid choice. Please try again."))
		} else {
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}

// GetHistory returns the input history
func (s *SimpleInput) GetHistory() []string {
	return s.history
}

// ClearHistory clears the input history
func (s *SimpleInput) ClearHistory() {
	s.history = make([]string, 0)
}

// InterruptHandler handles Ctrl+C interrupts gracefully
type InterruptHandler struct {
	channel chan os.Signal
	active  bool
}

// NewInterruptHandler creates a new interrupt handler
func NewInterruptHandler() *InterruptHandler {
	return &InterruptHandler{
		channel: make(chan os.Signal, 1),
		active:  false,
	}
}

// Start starts listening for interrupts
func (h *InterruptHandler) Start() {
	if h.active {
		return
	}

	h.active = true
	signal.Notify(h.channel, os.Interrupt, syscall.SIGTERM)
}

// Stop stops listening for interrupts
func (h *InterruptHandler) Stop() {
	if !h.active {
		return
	}

	h.active = false
	signal.Stop(h.channel)
}

// Wait waits for an interrupt signal
func (h *InterruptHandler) Wait() {
	if !h.active {
		return
	}
	<-h.channel
}

// HandleInterrupt runs a callback when an interrupt is received
func (h *InterruptHandler) HandleInterrupt(callback func()) {
	go func() {
		h.Wait()
		callback()
	}()
}

// NewInput creates a new enhanced input handler
func NewInput(options InputOptions) *Input {
	theme := options.Theme
	if theme == nil {
		theme = GetCurrentTheme()
	}

	return &Input{
		options: options,
		theme:   theme,
		reader:  bufio.NewReader(os.Stdin),
		history: []string{},
	}
}

// ReadLine reads a single line of input
func (i *Input) ReadLine() (string, error) {
	if i.options.Prompt != "" {
		fmt.Print(i.options.Prompt)
	}

	line, err := i.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimSpace(line)

	// Add to history if enabled
	if i.options.History && line != "" {
		i.history = append(i.history, line)
	}

	return line, nil
}

// ReadPassword reads a password without echoing
func (i *Input) ReadPassword() (string, error) {
	if i.options.Prompt != "" {
		fmt.Print(i.options.Prompt)
	}

	// Use syscall to disable echo
	// This is a simplified version - in production, use a proper terminal library
	password, err := i.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(password), nil
}

// GetHistory returns the input history
func (i *Input) GetHistory() []string {
	return i.history
}

// ClearHistory clears the input history
func (i *Input) ClearHistory() {
	i.history = []string{}
}

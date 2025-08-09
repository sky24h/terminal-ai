package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// ProgressModel represents a progress bar
type ProgressModel struct {
	progress  progress.Model
	current   float64
	total     float64
	label     string
	theme     *Theme
	completed bool
}

// NewProgressModel creates a new progress model
func NewProgressModel(label string, total float64) *ProgressModel {
	theme := GetCurrentTheme()
	return NewProgressModelWithTheme(label, total, theme)
}

// NewProgressModelWithTheme creates a new progress model with a theme
func NewProgressModelWithTheme(label string, total float64, theme *Theme) *ProgressModel {
	prog := progress.New(progress.WithDefaultGradient())
	prog.ShowPercentage = true
	prog.PercentageStyle = lipgloss.NewStyle().Foreground(theme.Primary)

	return &ProgressModel{
		progress: prog,
		current:  0,
		total:    total,
		label:    label,
		theme:    theme,
	}
}

// SetCurrent sets the current progress value
func (m *ProgressModel) SetCurrent(current float64) {
	m.current = current
	if m.current >= m.total {
		m.completed = true
	}
}

// IncrementBy increments the progress by a value
func (m *ProgressModel) IncrementBy(value float64) {
	m.SetCurrent(m.current + value)
}

// Init initializes the progress model
func (m *ProgressModel) Init() tea.Cmd {
	return nil
}

// Update handles progress updates
func (m *ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 4
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

// View renders the progress bar
func (m *ProgressModel) View() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(m.theme.Secondary).
		Bold(true)

	percentage := m.current / m.total
	if percentage > 1 {
		percentage = 1
	}

	statusIcon := "⏳"
	if m.completed {
		statusIcon = "✓"
	}

	return fmt.Sprintf("%s %s\n%s %.0f/%.0f",
		statusIcon,
		labelStyle.Render(m.label),
		m.progress.ViewAs(percentage),
		m.current,
		m.total,
	)
}

// SimpleProgress provides a simple progress bar without bubbletea
type SimpleProgress struct {
	current  float64
	total    float64
	label    string
	width    int
	mu       sync.Mutex
	lastDraw time.Time
	theme    *Theme
	isTTY    bool
}

// NewSimpleProgress creates a new simple progress bar
func NewSimpleProgress(label string, total float64) *SimpleProgress {
	theme := GetCurrentTheme()
	width := 40 // Default width

	if w, _, err := getTerminalSize(); err == nil && w > 0 {
		width = w / 2 // Use half the terminal width
	}

	return &SimpleProgress{
		current:  0,
		total:    total,
		label:    label,
		width:    width,
		theme:    theme,
		isTTY:    isatty.IsTerminal(os.Stdout.Fd()),
		lastDraw: time.Now(),
	}
}

// SetCurrent sets the current progress value
func (p *SimpleProgress) SetCurrent(current float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	p.draw()
}

// IncrementBy increments the progress by a value
func (p *SimpleProgress) IncrementBy(value float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current += value
	p.draw()
}

// draw renders the progress bar
func (p *SimpleProgress) draw() {
	// Rate limit drawing to avoid flickering
	if time.Since(p.lastDraw) < 100*time.Millisecond {
		return
	}
	p.lastDraw = time.Now()

	if !p.isTTY {
		// In non-TTY, just print percentage updates
		percentage := int(p.current / p.total * 100)
		fmt.Printf("%s: %d%%\n", p.label, percentage)
		return
	}

	percentage := p.current / p.total
	if percentage > 1 {
		percentage = 1
	}

	filled := int(float64(p.width) * percentage)
	empty := p.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	labelStyle := lipgloss.NewStyle().Foreground(p.theme.Secondary)
	barStyle := lipgloss.NewStyle().Foreground(p.theme.Primary)
	percentStyle := lipgloss.NewStyle().Foreground(p.theme.TextBright)

	fmt.Printf("\r%s [%s] %s %.0f/%.0f",
		labelStyle.Render(p.label),
		barStyle.Render(bar),
		percentStyle.Render(fmt.Sprintf("%3.0f%%", percentage*100)),
		p.current,
		p.total,
	)
}

// Finish completes the progress bar
func (p *SimpleProgress) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.draw()

	if p.isTTY {
		fmt.Println() // New line after completion
	}
}

// Clear clears the progress bar
func (p *SimpleProgress) Clear() {
	if !p.isTTY {
		return
	}

	fmt.Print("\r" + strings.Repeat(" ", p.width+len(p.label)+20) + "\r")
}

// StreamingDisplay handles streaming text display with token counting
type StreamingDisplay struct {
	buffer     strings.Builder
	tokenCount int
	lineCount  int
	theme      *Theme
	isTTY      bool
	mu         sync.Mutex
	onToken    func(string)
	formatter  *Formatter
}

// NewStreamingDisplay creates a new streaming display
func NewStreamingDisplay() *StreamingDisplay {
	theme := GetCurrentTheme()
	return &StreamingDisplay{
		theme:     theme,
		isTTY:     isatty.IsTerminal(os.Stdout.Fd()),
		formatter: NewFormatterWithTheme(theme),
	}
}

// SetOnToken sets a callback for each token
func (s *StreamingDisplay) SetOnToken(callback func(string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onToken = callback
}

// WriteToken writes a single token
func (s *StreamingDisplay) WriteToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer.WriteString(token)
	s.tokenCount++

	// Count newlines
	s.lineCount += strings.Count(token, "\n")

	// Display the token
	if s.isTTY {
		fmt.Print(token)
	} else {
		fmt.Print(token)
	}

	// Call callback if set
	if s.onToken != nil {
		s.onToken(token)
	}
}

// WriteLine writes a complete line
func (s *StreamingDisplay) WriteLine(line string) {
	s.WriteToken(line + "\n")
}

// GetContent returns the accumulated content
func (s *StreamingDisplay) GetContent() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buffer.String()
}

// GetTokenCount returns the number of tokens written
func (s *StreamingDisplay) GetTokenCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokenCount
}

// GetLineCount returns the number of lines written
func (s *StreamingDisplay) GetLineCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lineCount
}

// Clear clears the buffer
func (s *StreamingDisplay) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer.Reset()
	s.tokenCount = 0
	s.lineCount = 0
}

// ShowStats displays statistics about the streamed content
func (s *StreamingDisplay) ShowStats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	statsBox := s.formatter.Box(
		fmt.Sprintf("Tokens: %d\nLines: %d\nCharacters: %d",
			s.tokenCount,
			s.lineCount,
			s.buffer.Len()),
		"Stream Statistics",
	)

	fmt.Println(statsBox)
}

// TokenCounter provides token counting functionality
type TokenCounter struct {
	count     int
	startTime time.Time
	mu        sync.Mutex
	theme     *Theme
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		count:     0,
		startTime: time.Now(),
		theme:     GetCurrentTheme(),
	}
}

// Increment increments the token count
func (t *TokenCounter) Increment() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.count++
}

// IncrementBy increments the token count by a value
func (t *TokenCounter) IncrementBy(value int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.count += value
}

// GetCount returns the current token count
func (t *TokenCounter) GetCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.count
}

// GetRate returns tokens per second
func (t *TokenCounter) GetRate() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	elapsed := time.Since(t.startTime).Seconds()
	if elapsed == 0 {
		return 0
	}

	return float64(t.count) / elapsed
}

// Reset resets the counter
func (t *TokenCounter) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.count = 0
	t.startTime = time.Now()
}

// Display shows the token count and rate
func (t *TokenCounter) Display() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	rate := t.GetRate()
	elapsed := time.Since(t.startTime)

	style := lipgloss.NewStyle().Foreground(t.theme.TextMuted)
	return style.Render(fmt.Sprintf("Tokens: %d | Rate: %.1f/s | Time: %s",
		t.count,
		rate,
		elapsed.Round(time.Second),
	))
}

// LoadingIndicator provides various loading indicators
type LoadingIndicator struct {
	style   string
	message string
	active  bool
	mu      sync.Mutex
	stop    chan bool
	theme   *Theme
	isTTY   bool
}

// NewLoadingIndicator creates a new loading indicator
func NewLoadingIndicator(style string, message string) *LoadingIndicator {
	return &LoadingIndicator{
		style:   style,
		message: message,
		stop:    make(chan bool, 1),
		theme:   GetCurrentTheme(),
		isTTY:   isatty.IsTerminal(os.Stdout.Fd()),
	}
}

// Start starts the loading indicator
func (l *LoadingIndicator) Start() {
	l.mu.Lock()
	if l.active {
		l.mu.Unlock()
		return
	}
	l.active = true
	l.mu.Unlock()

	if !l.isTTY {
		fmt.Println(l.message)
		return
	}

	go func() {
		frames := l.getFrames()
		current := 0
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		msgStyle := lipgloss.NewStyle().Foreground(l.theme.Text)
		frameStyle := lipgloss.NewStyle().Foreground(l.theme.Primary)

		for {
			select {
			case <-ticker.C:
				l.mu.Lock()
				if l.active {
					fmt.Printf("\r%s %s",
						frameStyle.Render(frames[current]),
						msgStyle.Render(l.message))
					current = (current + 1) % len(frames)
				}
				l.mu.Unlock()

			case <-l.stop:
				fmt.Print("\r" + strings.Repeat(" ", len(l.message)+10) + "\r")
				return
			}
		}
	}()
}

// Stop stops the loading indicator
func (l *LoadingIndicator) Stop() {
	l.mu.Lock()
	if !l.active {
		l.mu.Unlock()
		return
	}
	l.active = false
	l.mu.Unlock()

	if l.isTTY {
		select {
		case l.stop <- true:
		default:
		}
	}
}

// SetMessage updates the loading message
func (l *LoadingIndicator) SetMessage(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.message = message
}

// getFrames returns animation frames based on style
func (l *LoadingIndicator) getFrames() []string {
	switch l.style {
	case "dots":
		return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	case "line":
		return []string{"-", "\\", "|", "/"}
	case "star":
		return []string{"✶", "✸", "✹", "✺", "✹", "✸"}
	case "square":
		return []string{"◰", "◳", "◲", "◱"}
	case "circle":
		return []string{"◐", "◓", "◑", "◒"}
	default:
		return []string{".", "..", "...", "....", ".....", "......"}
	}
}

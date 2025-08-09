# Terminal UI Components

This document describes the UI components available in the terminal-ai project.

## Overview

The UI package provides a comprehensive set of terminal UI components that work in both TTY and non-TTY environments. All components support theming and graceful degradation.

## Components

### 1. Spinner (`spinner.go`)

Loading spinners with multiple animation styles.

**Features:**
- Multiple styles: Dots, Line, Globe, Moon, BouncingBar, Pulse, Arrows, Circle
- Custom loading messages
- Thread-safe operation
- TTY/non-TTY support
- Success/Error/Warning stop states

**Usage:**
```go
// Simple spinner
spinner := ui.NewSimpleSpinner("Loading...")
spinner.Start()
// ... do work ...
spinner.StopWithSuccess("Done!")

// With custom style
spinner := ui.NewSimpleSpinnerWithStyle("Processing", ui.SpinnerPulse)

// Interactive with Bubbletea
model := ui.NewSpinner("Loading...")
p := tea.NewProgram(model)
p.Run()
```

### 2. Formatter (`formatter.go`)

Rich text formatting with markdown support.

**Features:**
- Markdown rendering
- Code syntax highlighting (Go, Python, JavaScript, TypeScript)
- Table formatting
- Color themes (Light/Dark)
- Text styling (Bold, Italic, Underline, Highlight)
- Box/Border layouts
- Non-TTY fallback

**Usage:**
```go
formatter := ui.NewFormatter()

// Basic formatting
fmt.Println(formatter.Title("Welcome"))
fmt.Println(formatter.Success("Operation complete"))
fmt.Println(formatter.Error("An error occurred"))

// Code blocks
code := `func main() { fmt.Println("Hello") }`
fmt.Println(formatter.Code(code, "go"))

// Tables
headers := []string{"Name", "Value"}
rows := [][]string{{"Key1", "Val1"}}
fmt.Println(formatter.Table(headers, rows))

// Markdown
fmt.Println(formatter.Markdown("# Header\n**Bold** text"))
```

### 3. Input (`input.go`)

Interactive input handling with history support.

**Features:**
- Single-line input
- Multi-line input (textarea)
- Password input
- Confirmation dialogs
- Selection menus
- Input history
- Ctrl+C handling

**Usage:**
```go
// Simple input
input := ui.NewSimpleInput()
name, _ := input.ReadLine("Enter name: ")
confirmed, _ := input.Confirm("Continue?", true)

// Selection
options := []string{"Option A", "Option B"}
choice, _ := input.Select("Choose:", options)

// Interactive with Bubbletea
model := ui.NewInput("Enter text:", "placeholder")
p := tea.NewProgram(model)
finalModel, _ := p.Run()
value := finalModel.(*ui.InputModel).Value()
```

### 4. Progress (`progress.go`)

Progress bars and streaming displays.

**Features:**
- Progress bars with percentage
- Token counters with rate calculation
- Streaming text display
- Loading indicators
- Statistics tracking

**Usage:**
```go
// Progress bar
progress := ui.NewSimpleProgress("Downloading", 100)
for i := 0; i <= 100; i += 10 {
    progress.SetCurrent(float64(i))
    time.Sleep(100 * time.Millisecond)
}
progress.Finish()

// Token counter
counter := ui.NewTokenCounter()
counter.IncrementBy(10)
fmt.Println(counter.Display()) // Shows count, rate, time

// Streaming display
streamer := ui.NewStreamingDisplay()
streamer.WriteToken("Hello ")
streamer.WriteToken("World")
streamer.ShowStats()

// Loading indicator
loader := ui.NewLoadingIndicator("dots", "Processing...")
loader.Start()
// ... do work ...
loader.Stop()
```

### 5. Styles (`styles.go`)

Consistent theming and styling system.

**Features:**
- Dark and Light themes
- Color schemes for different message types
- Adaptive styles for TTY/non-TTY
- Common style presets
- Environment-based theme selection

**Usage:**
```go
// Set theme from environment (TERMINAL_AI_THEME=light/dark)
ui.SetThemeFromEnv()

// Or set manually
ui.SetTheme(ui.LightTheme())

// Get current theme
theme := ui.GetCurrentTheme()

// Get message styles
styles := ui.GetMessageStyles(theme)
successStyle := styles["success"]

// Common styles
common := ui.GetCommonStyles(theme)
titleStyle := common.Title
```

## Environment Variables

- `TERMINAL_AI_THEME`: Set to "light" or "dark" (default: dark)

## TTY Detection

All components automatically detect TTY environments and adapt their output:
- **TTY mode**: Full styling, colors, and interactive features
- **Non-TTY mode**: Plain text fallback with preserved functionality

## Thread Safety

Components marked as thread-safe:
- `SimpleSpinner`
- `SimpleProgress`
- `TokenCounter`
- `StreamingDisplay`
- `LoadingIndicator`

## Testing

Run tests with:
```bash
go test ./internal/ui -v
```

## Examples

See `examples/ui_demo.go` for a comprehensive demonstration of all components.

```bash
go run examples/ui_demo.go
```

## Dependencies

- github.com/charmbracelet/bubbles
- github.com/charmbracelet/bubbletea
- github.com/charmbracelet/lipgloss
- github.com/mattn/go-isatty
- golang.org/x/term
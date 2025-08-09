package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/terminal-ai/internal/ui"
)

func main() {
	fmt.Println("Terminal AI - UI Components Demo")
	fmt.Println("=================================")

	// Initialize theme
	ui.SetThemeFromEnv()

	// Demo 1: Formatter
	demoFormatter()

	// Demo 2: Simple Spinner
	demoSpinner()

	// Demo 3: Progress Bar
	demoProgress()

	// Demo 4: Simple Input
	demoInput()

	// Demo 5: Streaming Display
	demoStreaming()

	fmt.Println("\nDemo completed!")
}

func demoFormatter() {
	fmt.Println("\n--- Formatter Demo ---")

	formatter := ui.NewFormatter(ui.FormatterOptions{})

	// Title and subtitle
	fmt.Println(formatter.Title("Welcome to Terminal AI"))
	fmt.Println(formatter.Subtitle("Advanced UI Components"))

	// Message types
	fmt.Println(formatter.Success("Operation completed successfully"))
	fmt.Println(formatter.Error("An error occurred"))
	fmt.Println(formatter.Warning("This is a warning"))
	fmt.Println(formatter.Info("Information message"))

	// Code block
	code := `func main() {
    fmt.Println("Hello, World!")
}`
	fmt.Println(formatter.Code(code, "go"))

	// List
	items := []string{"First item", "Second item", "Third item"}
	fmt.Println("\nUnordered list:")
	fmt.Println(formatter.List(items, false))
	fmt.Println("\nOrdered list:")
	fmt.Println(formatter.List(items, true))

	// Table
	headers := []string{"Name", "Age", "City"}
	rows := [][]string{
		{"Alice", "30", "New York"},
		{"Bob", "25", "San Francisco"},
		{"Charlie", "35", "Chicago"},
	}
	fmt.Println("\nTable:")
	fmt.Println(formatter.Table(headers, rows))

	// Box
	fmt.Println(formatter.Box("This is content inside a box", "Box Title"))

	// Text styling
	fmt.Println(formatter.Bold("Bold text"))
	fmt.Println(formatter.Italic("Italic text"))
	fmt.Println(formatter.Underline("Underlined text"))
	fmt.Println(formatter.Highlight("Highlighted text"))
	fmt.Println(formatter.Muted("Muted text"))

	// Markdown
	markdown := `# Header 1
## Header 2
### Header 3

This is **bold** and this is *italic*.

- List item 1
- List item 2

> This is a blockquote

Code inline: ` + "`fmt.Println()`"

	fmt.Println("\nMarkdown rendering:")
	fmt.Println(formatter.Markdown(markdown))
}

func demoSpinner() {
	fmt.Println("\n--- Spinner Demo ---")

	// Simple spinner
	spinner := ui.NewSimpleSpinner("Loading data...")
	spinner.Start()
	time.Sleep(2 * time.Second)
	spinner.SetMessage("Processing...")
	time.Sleep(1 * time.Second)
	spinner.StopWithSuccess("Data loaded successfully")

	// Different spinner styles
	styles := []ui.SpinnerStyle{
		ui.SpinnerDots,
		ui.SpinnerLine,
		ui.SpinnerPulse,
		ui.SpinnerArrows,
		ui.SpinnerCircle,
	}

	for _, style := range styles {
		spinner := ui.NewSimpleSpinnerWithStyle("Loading with style...", style)
		spinner.Start()
		time.Sleep(1 * time.Second)
		spinner.Stop()
	}
}

func demoProgress() {
	fmt.Println("\n--- Progress Bar Demo ---")

	// Simple progress bar
	progress := ui.NewSimpleProgress("Downloading", 100)

	for i := 0; i <= 100; i += 10 {
		progress.SetCurrent(float64(i))
		time.Sleep(200 * time.Millisecond)
	}
	progress.Finish()

	// Token counter
	fmt.Println("\n--- Token Counter Demo ---")
	counter := ui.NewTokenCounter()

	for i := 0; i < 50; i++ {
		counter.IncrementBy(5)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println(counter.Display())
}

func demoInput() {
	fmt.Println("\n--- Input Demo ---")

	input := ui.NewSimpleInput()

	// Read line
	name, err := input.ReadLine("Enter your name: ")
	if err == nil {
		fmt.Printf("Hello, %s!\n", name)
	}

	// Confirmation
	confirmed, err := input.Confirm("Do you want to continue?", true)
	if err == nil {
		if confirmed {
			fmt.Println("Continuing...")
		} else {
			fmt.Println("Cancelled")
		}
	}

	// Selection
	options := []string{"Option A", "Option B", "Option C"}
	choice, err := input.Select("Choose an option:", options)
	if err == nil {
		fmt.Printf("You selected: %s\n", options[choice])
	}
}

func demoStreaming() {
	fmt.Println("\n--- Streaming Display Demo ---")

	streamer := ui.NewStreamingDisplay()

	text := "This is a streaming text display. It simulates text being received token by token, like from an AI model. "

	for _, char := range text {
		streamer.WriteToken(string(char))
		time.Sleep(30 * time.Millisecond)
	}

	fmt.Println()
	streamer.ShowStats()
}

// Interactive spinner demo using bubbletea
func demoInteractiveSpinner() {
	fmt.Println("\n--- Interactive Spinner Demo (Press q to quit) ---")

	model := ui.NewSpinner("Processing your request...")
	p := tea.NewProgram(model)

	go func() {
		time.Sleep(3 * time.Second)
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// Interactive input demo using bubbletea
func demoInteractiveInput() {
	fmt.Println("\n--- Interactive Input Demo ---")

	model := ui.NewInputModel("Enter your message:", "Type here...")
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if inputModel, ok := finalModel.(*ui.InputModel); ok {
		if !inputModel.IsCancelled() {
			fmt.Printf("You entered: %s\n", inputModel.Value())
		} else {
			fmt.Println("Input cancelled")
		}
	}
}

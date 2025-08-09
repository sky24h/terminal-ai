package ui

import (
	"strings"
	"testing"
	"time"
)

func TestThemes(t *testing.T) {
	tests := []struct {
		name  string
		theme *Theme
	}{
		{"DarkTheme", DarkTheme()},
		{"LightTheme", LightTheme()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.theme == nil {
				t.Error("Theme should not be nil")
			}
			if tt.theme.Primary == "" {
				t.Error("Primary color should not be empty")
			}
			if tt.theme.Success == "" {
				t.Error("Success color should not be empty")
			}
			if tt.theme.Error == "" {
				t.Error("Error color should not be empty")
			}
		})
	}
}

func TestFormatter(t *testing.T) {
	formatter := NewFormatter(FormatterOptions{})

	t.Run("Title", func(t *testing.T) {
		result := formatter.Title("Test Title")
		if !strings.Contains(result, "Test Title") {
			t.Error("Title should contain the text")
		}
	})

	t.Run("Success", func(t *testing.T) {
		result := formatter.Success("Success message")
		if !strings.Contains(result, "Success message") {
			t.Error("Success should contain the message")
		}
	})

	t.Run("Error", func(t *testing.T) {
		result := formatter.Error("Error message")
		if !strings.Contains(result, "Error message") {
			t.Error("Error should contain the message")
		}
	})

	t.Run("List", func(t *testing.T) {
		items := []string{"Item 1", "Item 2"}

		// Unordered list
		result := formatter.List(items, false)
		if !strings.Contains(result, "Item 1") || !strings.Contains(result, "Item 2") {
			t.Error("List should contain all items")
		}

		// Ordered list
		result = formatter.List(items, true)
		if !strings.Contains(result, "Item 1") || !strings.Contains(result, "Item 2") {
			t.Error("Ordered list should contain all items")
		}
	})

	t.Run("Table", func(t *testing.T) {
		headers := []string{"Name", "Value"}
		rows := [][]string{
			{"Row1", "Val1"},
			{"Row2", "Val2"},
		}

		result := formatter.Table(headers, rows)
		if !strings.Contains(result, "Name") || !strings.Contains(result, "Value") {
			t.Error("Table should contain headers")
		}
		if !strings.Contains(result, "Row1") || !strings.Contains(result, "Val1") {
			t.Error("Table should contain row data")
		}
	})

	t.Run("Code", func(t *testing.T) {
		code := "fmt.Println(\"Hello\")"
		result := formatter.Code(code, "go")
		if !strings.Contains(result, code) {
			t.Error("Code block should contain the code")
		}
	})

	t.Run("InlineCode", func(t *testing.T) {
		code := "variable"
		result := formatter.InlineCode(code)
		if !strings.Contains(result, code) {
			t.Error("Inline code should contain the code")
		}
	})

	t.Run("Box", func(t *testing.T) {
		content := "Box content"
		title := "Box Title"
		result := formatter.Box(content, title)
		if !strings.Contains(result, content) {
			t.Error("Box should contain the content")
		}
	})

	t.Run("Wrap", func(t *testing.T) {
		// Test in non-TTY mode where wrapping doesn't occur
		formatter.isTTY = false
		longText := strings.Repeat("word ", 50)
		result := formatter.Wrap(longText)
		// In non-TTY mode, text is not wrapped
		if result != longText {
			t.Error("In non-TTY mode, text should not be wrapped")
		}

		// Test in TTY mode with narrow width
		formatter.isTTY = true
		formatter.width = 20 // Set narrow width for testing
		result = formatter.Wrap(longText)
		lines := strings.Split(result, "\n")
		if len(lines) < 2 {
			t.Error("Long text should be wrapped into multiple lines in TTY mode")
		}
	})

	t.Run("Center", func(t *testing.T) {
		text := "Centered"
		result := formatter.Center(text)
		if !strings.Contains(result, text) {
			t.Error("Centered text should contain the original text")
		}
	})
}

func TestSpinnerStyles(t *testing.T) {
	styles := []SpinnerStyle{
		SpinnerDots,
		SpinnerLine,
		SpinnerGlobe,
		SpinnerMoon,
		SpinnerBouncingBar,
		SpinnerPulse,
		SpinnerArrows,
		SpinnerCircle,
	}

	for _, style := range styles {
		t.Run("Style", func(t *testing.T) {
			spinner := NewSimpleSpinnerWithStyle("Test", style)
			if spinner == nil {
				t.Error("Spinner should not be nil")
			}
			if len(spinner.frames) == 0 {
				t.Error("Spinner should have frames")
			}
		})
	}
}

func TestSimpleSpinner(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		spinner := NewSimpleSpinner("Loading...")
		if spinner == nil {
			t.Fatal("Spinner should not be nil")
		}
		if spinner.message != "Loading..." {
			t.Error("Spinner message should be set")
		}
	})

	t.Run("SetMessage", func(t *testing.T) {
		spinner := NewSimpleSpinner("Initial")
		spinner.SetMessage("Updated")
		if spinner.message != "Updated" {
			t.Error("Message should be updated")
		}
	})

	t.Run("SetCustomMessage", func(t *testing.T) {
		spinner := NewSimpleSpinner("Initial")
		spinner.SetCustomMessage("Custom")
		if spinner.customMessage != "Custom" {
			t.Error("Custom message should be set")
		}
	})
}

func TestSimpleProgress(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		progress := NewSimpleProgress("Test", 100)
		if progress == nil {
			t.Fatal("Progress should not be nil")
		}
		if progress.total != 100 {
			t.Error("Total should be set")
		}
	})

	t.Run("SetCurrent", func(t *testing.T) {
		progress := NewSimpleProgress("Test", 100)
		progress.SetCurrent(50)
		if progress.current != 50 {
			t.Error("Current should be updated")
		}
	})

	t.Run("IncrementBy", func(t *testing.T) {
		progress := NewSimpleProgress("Test", 100)
		progress.IncrementBy(25)
		progress.IncrementBy(25)
		if progress.current != 50 {
			t.Error("Current should be incremented")
		}
	})
}

func TestTokenCounter(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		counter := NewTokenCounter()
		if counter == nil {
			t.Fatal("Counter should not be nil")
		}
		if counter.GetCount() != 0 {
			t.Error("Initial count should be 0")
		}
	})

	t.Run("Increment", func(t *testing.T) {
		counter := NewTokenCounter()
		counter.Increment()
		counter.Increment()
		if counter.GetCount() != 2 {
			t.Error("Count should be 2")
		}
	})

	t.Run("IncrementBy", func(t *testing.T) {
		counter := NewTokenCounter()
		counter.IncrementBy(5)
		counter.IncrementBy(3)
		if counter.GetCount() != 8 {
			t.Error("Count should be 8")
		}
	})

	t.Run("Reset", func(t *testing.T) {
		counter := NewTokenCounter()
		counter.IncrementBy(10)
		counter.Reset()
		if counter.GetCount() != 0 {
			t.Error("Count should be reset to 0")
		}
	})

	t.Run("GetRate", func(t *testing.T) {
		counter := NewTokenCounter()
		counter.IncrementBy(100)
		time.Sleep(100 * time.Millisecond)
		rate := counter.GetRate()
		if rate <= 0 {
			t.Error("Rate should be positive")
		}
	})
}

func TestStreamingDisplay(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		display := NewStreamingDisplay()
		if display == nil {
			t.Fatal("Display should not be nil")
		}
	})

	t.Run("WriteToken", func(t *testing.T) {
		display := NewStreamingDisplay()
		display.WriteToken("Hello ")
		display.WriteToken("World")

		content := display.GetContent()
		if content != "Hello World" {
			t.Errorf("Content should be 'Hello World', got '%s'", content)
		}

		if display.GetTokenCount() != 2 {
			t.Error("Token count should be 2")
		}
	})

	t.Run("WriteLine", func(t *testing.T) {
		display := NewStreamingDisplay()
		display.WriteLine("Line 1")
		display.WriteLine("Line 2")

		content := display.GetContent()
		if !strings.Contains(content, "Line 1\n") || !strings.Contains(content, "Line 2\n") {
			t.Error("Content should contain both lines with newlines")
		}

		if display.GetLineCount() != 2 {
			t.Error("Line count should be 2")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		display := NewStreamingDisplay()
		display.WriteToken("Test")
		display.Clear()

		if display.GetContent() != "" {
			t.Error("Content should be empty after clear")
		}
		if display.GetTokenCount() != 0 {
			t.Error("Token count should be 0 after clear")
		}
	})

	t.Run("Callback", func(t *testing.T) {
		display := NewStreamingDisplay()
		callbackCalled := false

		display.SetOnToken(func(token string) {
			callbackCalled = true
		})

		display.WriteToken("Test")

		if !callbackCalled {
			t.Error("Callback should be called")
		}
	})
}

func TestSimpleInput(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		input := NewSimpleInput()
		if input == nil {
			t.Fatal("Input should not be nil")
		}
		if input.reader == nil {
			t.Error("Reader should be initialized")
		}
	})

	t.Run("History", func(t *testing.T) {
		input := NewSimpleInput()

		// Initially empty
		history := input.GetHistory()
		if len(history) != 0 {
			t.Error("Initial history should be empty")
		}

		// Clear history
		input.history = []string{"item1", "item2"}
		input.ClearHistory()
		if len(input.GetHistory()) != 0 {
			t.Error("History should be cleared")
		}
	})
}

func TestInterruptHandler(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		handler := NewInterruptHandler()
		if handler == nil {
			t.Fatal("Handler should not be nil")
		}
		if handler.active {
			t.Error("Handler should not be active initially")
		}
	})

	t.Run("StartStop", func(t *testing.T) {
		handler := NewInterruptHandler()

		handler.Start()
		if !handler.active {
			t.Error("Handler should be active after Start")
		}

		handler.Stop()
		if handler.active {
			t.Error("Handler should not be active after Stop")
		}
	})

	t.Run("Callback", func(t *testing.T) {
		handler := NewInterruptHandler()
		handler.Start()

		callbackExecuted := make(chan bool, 1)
		handler.HandleInterrupt(func() {
			callbackExecuted <- true
		})

		// Clean up
		handler.Stop()
	})
}

func TestLoadingIndicator(t *testing.T) {
	styles := []string{"dots", "line", "star", "square", "circle", "default"}

	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			indicator := NewLoadingIndicator(style, "Loading...")
			if indicator == nil {
				t.Fatal("Indicator should not be nil")
			}

			frames := indicator.getFrames()
			if len(frames) == 0 {
				t.Error("Indicator should have frames")
			}
		})
	}

	t.Run("SetMessage", func(t *testing.T) {
		indicator := NewLoadingIndicator("dots", "Initial")
		indicator.SetMessage("Updated")
		if indicator.message != "Updated" {
			t.Error("Message should be updated")
		}
	})
}

func TestInputModel(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		input := NewInputModel("Label", "Placeholder")
		if input == nil {
			t.Fatal("Input should not be nil")
		}
		if input.label != "Label" {
			t.Error("Label should be set")
		}
	})

	t.Run("Value", func(t *testing.T) {
		input := NewInputModel("Label", "Placeholder")
		// The textinput model is initialized with empty value
		if input.Value() != "" {
			t.Error("Initial value should be empty")
		}
	})

	t.Run("IsCancelled", func(t *testing.T) {
		input := NewInputModel("Label", "Placeholder")
		if input.IsCancelled() {
			t.Error("Should not be cancelled initially")
		}

		input.cancelled = true
		if !input.IsCancelled() {
			t.Error("Should be cancelled after setting flag")
		}
	})
}

func TestMultiLineInputModel(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		input := NewMultiLineInput("Label", "Placeholder")
		if input == nil {
			t.Fatal("Input should not be nil")
		}
		if input.label != "Label" {
			t.Error("Label should be set")
		}
	})

	t.Run("Value", func(t *testing.T) {
		input := NewMultiLineInput("Label", "Placeholder")
		if input.Value() != "" {
			t.Error("Initial value should be empty")
		}
	})

	t.Run("IsCancelled", func(t *testing.T) {
		input := NewMultiLineInput("Label", "Placeholder")
		if input.IsCancelled() {
			t.Error("Should not be cancelled initially")
		}

		input.cancelled = true
		if !input.IsCancelled() {
			t.Error("Should be cancelled after setting flag")
		}
	})
}

func TestProgressModel(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		progress := NewProgressModel("Test", 100)
		if progress == nil {
			t.Fatal("Progress should not be nil")
		}
		if progress.total != 100 {
			t.Error("Total should be set")
		}
		if progress.label != "Test" {
			t.Error("Label should be set")
		}
	})

	t.Run("SetCurrent", func(t *testing.T) {
		progress := NewProgressModel("Test", 100)
		progress.SetCurrent(50)
		if progress.current != 50 {
			t.Error("Current should be updated")
		}
		if progress.completed {
			t.Error("Should not be completed at 50%")
		}

		progress.SetCurrent(100)
		if !progress.completed {
			t.Error("Should be completed at 100%")
		}
	})

	t.Run("IncrementBy", func(t *testing.T) {
		progress := NewProgressModel("Test", 100)
		progress.IncrementBy(25)
		if progress.current != 25 {
			t.Error("Current should be incremented")
		}
	})
}

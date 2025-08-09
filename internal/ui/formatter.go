package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// FormatterOptions contains options for creating a formatter
type FormatterOptions struct {
	ColorEnabled       bool
	MarkdownEnabled    bool
	SyntaxHighlighting bool
	Width              int
	Theme              *Theme
}

// Formatter handles text formatting and styling
type Formatter struct {
	theme              *Theme
	isTTY              bool
	width              int
	styles             *CommonStyles
	msgStyles          map[string]MessageStyle
	colorEnabled       bool
	markdownEnabled    bool
	syntaxHighlighting bool
}

// NewFormatter creates a new formatter with options
func NewFormatter(options FormatterOptions) *Formatter {
	theme := options.Theme
	if theme == nil {
		theme = GetCurrentTheme()
	}

	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	width := options.Width
	if width == 0 {
		width = 80 // Default width
		if isTTY {
			// Try to get terminal width
			if w, _, err := getTerminalSize(); err == nil && w > 0 {
				width = w
			}
		}
	}

	return &Formatter{
		theme:              theme,
		isTTY:              isTTY && options.ColorEnabled,
		width:              width,
		styles:             GetCommonStyles(theme),
		msgStyles:          GetMessageStyles(theme),
		colorEnabled:       options.ColorEnabled,
		markdownEnabled:    options.MarkdownEnabled,
		syntaxHighlighting: options.SyntaxHighlighting,
	}
}

// NewFormatterWithTheme creates a new formatter with the given theme (backward compatibility)
func NewFormatterWithTheme(theme *Theme) *Formatter {
	return NewFormatter(FormatterOptions{
		ColorEnabled:       true,
		MarkdownEnabled:    true,
		SyntaxHighlighting: true,
		Theme:              theme,
	})
}

// SetWidth sets the formatter width
func (f *Formatter) SetWidth(width int) {
	f.width = width
}

// Title formats a title
func (f *Formatter) Title(text string) string {
	if !f.isTTY {
		return fmt.Sprintf("# %s\n", text)
	}
	return f.styles.Title.Render(text)
}

// Subtitle formats a subtitle
func (f *Formatter) Subtitle(text string) string {
	if !f.isTTY {
		return fmt.Sprintf("## %s\n", text)
	}
	return f.styles.Subtitle.Render(text)
}

// Success formats a success message
func (f *Formatter) Success(text string) string {
	style := f.msgStyles["success"]
	if !f.isTTY {
		return fmt.Sprintf("SUCCESS: %s", text)
	}
	return fmt.Sprintf("%s %s",
		style.IconStyle.Render(style.Icon),
		style.TextStyle.Render(text))
}

// Error formats an error message
func (f *Formatter) Error(text string) string {
	style := f.msgStyles["error"]
	if !f.isTTY {
		return fmt.Sprintf("ERROR: %s", text)
	}
	return fmt.Sprintf("%s %s",
		style.IconStyle.Render(style.Icon),
		style.TextStyle.Render(text))
}

// Warning formats a warning message
func (f *Formatter) Warning(text string) string {
	style := f.msgStyles["warning"]
	if !f.isTTY {
		return fmt.Sprintf("WARNING: %s", text)
	}
	return fmt.Sprintf("%s %s",
		style.IconStyle.Render(style.Icon),
		style.TextStyle.Render(text))
}

// Info formats an info message
func (f *Formatter) Info(text string) string {
	style := f.msgStyles["info"]
	if !f.isTTY {
		return fmt.Sprintf("INFO: %s", text)
	}
	return fmt.Sprintf("%s %s",
		style.IconStyle.Render(style.Icon),
		style.TextStyle.Render(text))
}

// Code formats code blocks with syntax highlighting
func (f *Formatter) Code(code string, language string) string {
	if !f.isTTY {
		return fmt.Sprintf("```%s\n%s\n```", language, code)
	}

	// Apply basic syntax highlighting based on language
	highlighted := f.highlightCode(code, language)

	if language != "" {
		header := lipgloss.NewStyle().
			Background(f.theme.BackgroundAlt).
			Foreground(f.theme.Secondary).
			Padding(0, 1).
			Render(language)

		return header + "\n" + f.styles.CodeBlock.Render(highlighted)
	}

	return f.styles.CodeBlock.Render(highlighted)
}

// highlightCode applies basic syntax highlighting
func (f *Formatter) highlightCode(code string, language string) string {
	if !f.isTTY {
		return code
	}

	// Basic keyword highlighting for common languages
	keywords := map[string][]string{
		"go":         {"func", "var", "const", "type", "interface", "struct", "package", "import", "return", "if", "else", "for", "range", "switch", "case", "default"},
		"python":     {"def", "class", "import", "from", "return", "if", "else", "elif", "for", "while", "in", "with", "as", "try", "except", "finally"},
		"javascript": {"function", "var", "let", "const", "return", "if", "else", "for", "while", "switch", "case", "default", "import", "export", "class"},
		"typescript": {"function", "var", "let", "const", "return", "if", "else", "for", "while", "switch", "case", "default", "import", "export", "class", "interface", "type"},
	}

	keywordList, hasLanguage := keywords[strings.ToLower(language)]
	if !hasLanguage {
		return code
	}

	// Apply keyword highlighting
	keywordStyle := lipgloss.NewStyle().Foreground(f.theme.CodeKeyword)
	stringStyle := lipgloss.NewStyle().Foreground(f.theme.CodeString)
	commentStyle := lipgloss.NewStyle().Foreground(f.theme.CodeComment)

	lines := strings.Split(code, "\n")
	for i, line := range lines {
		// Highlight comments
		if strings.Contains(line, "//") {
			idx := strings.Index(line, "//")
			lines[i] = line[:idx] + commentStyle.Render(line[idx:])
			continue
		}

		// Highlight strings (basic)
		line = regexp.MustCompile(`"[^"]*"`).ReplaceAllStringFunc(line, func(s string) string {
			return stringStyle.Render(s)
		})
		line = regexp.MustCompile(`'[^']*'`).ReplaceAllStringFunc(line, func(s string) string {
			return stringStyle.Render(s)
		})

		// Highlight keywords
		for _, keyword := range keywordList {
			pattern := fmt.Sprintf(`\b%s\b`, keyword)
			line = regexp.MustCompile(pattern).ReplaceAllStringFunc(line, func(s string) string {
				return keywordStyle.Render(s)
			})
		}

		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

// InlineCode formats inline code
func (f *Formatter) InlineCode(code string) string {
	if !f.isTTY {
		return fmt.Sprintf("`%s`", code)
	}
	return f.styles.CodeInline.Render(code)
}

// Box creates a bordered box around content
func (f *Formatter) Box(content string, title string) string {
	if !f.isTTY {
		if title != "" {
			return fmt.Sprintf("=== %s ===\n%s\n==========", title, content)
		}
		return fmt.Sprintf("==========\n%s\n==========", content)
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(f.theme.Primary).
		Padding(1, 2).
		Width(f.width - 4)

	if title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(f.theme.Primary).
			Bold(true)

		titleLine := titleStyle.Render(title)
		separator := strings.Repeat("─", len(title)+4)

		content = titleLine + "\n" + separator + "\n" + content
	}

	return boxStyle.Render(content)
}

// List formats a list of items
func (f *Formatter) List(items []string, ordered bool) string {
	var result strings.Builder

	for i, item := range items {
		if ordered {
			if f.isTTY {
				bullet := fmt.Sprintf("%d.", i+1)
				result.WriteString(f.formatListItem(bullet, item))
			} else {
				result.WriteString(fmt.Sprintf("%d. %s", i+1, item))
			}
		} else {
			if f.isTTY {
				result.WriteString(f.formatListItem("•", item))
			} else {
				result.WriteString(fmt.Sprintf("- %s", item))
			}
		}

		if i < len(items)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// formatListItem formats a single list item
func (f *Formatter) formatListItem(bullet, text string) string {
	bulletStyle := lipgloss.NewStyle().
		Foreground(f.theme.Secondary).
		MarginRight(1)

	return bulletStyle.Render(bullet) + text
}

// Table formats data as a table
func (f *Formatter) Table(headers []string, rows [][]string) string {
	if !f.isTTY {
		return f.renderPlainTable(headers, rows)
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Build the table
	var result strings.Builder

	// Header
	headerStyle := f.styles.TableHeader
	for i, header := range headers {
		padded := f.padString(header, widths[i])
		result.WriteString(headerStyle.Render(padded))
		if i < len(headers)-1 {
			result.WriteString(" │ ")
		}
	}
	result.WriteString("\n")

	// Separator
	for i, width := range widths {
		result.WriteString(strings.Repeat("─", width+2))
		if i < len(widths)-1 {
			result.WriteString("┼")
		}
	}
	result.WriteString("\n")

	// Rows
	rowStyle := f.styles.TableRow
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				padded := f.padString(cell, widths[i])
				result.WriteString(rowStyle.Render(padded))
				if i < len(row)-1 {
					result.WriteString(" │ ")
				}
			}
		}
		result.WriteString("\n")
	}

	return f.styles.Table.Render(result.String())
}

// renderPlainTable renders a table without styling
func (f *Formatter) renderPlainTable(headers []string, rows [][]string) string {
	var result strings.Builder

	// Headers
	result.WriteString(strings.Join(headers, " | "))
	result.WriteString("\n")
	result.WriteString(strings.Repeat("-", len(strings.Join(headers, " | "))))
	result.WriteString("\n")

	// Rows
	for _, row := range rows {
		result.WriteString(strings.Join(row, " | "))
		result.WriteString("\n")
	}

	return result.String()
}

// padString pads a string to the specified width
func (f *Formatter) padString(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// Markdown renders markdown-like content
func (f *Formatter) Markdown(content string) string {
	if !f.isTTY {
		return content
	}

	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Headers
		if strings.HasPrefix(line, "# ") {
			result = append(result, f.Title(strings.TrimPrefix(line, "# ")))
		} else if strings.HasPrefix(line, "## ") {
			result = append(result, f.Subtitle(strings.TrimPrefix(line, "## ")))
		} else if strings.HasPrefix(line, "### ") {
			headerStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(f.theme.Tertiary)
			result = append(result, headerStyle.Render(strings.TrimPrefix(line, "### ")))
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			// Unordered list
			item := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			result = append(result, f.formatListItem("•", item))
		} else if matched, _ := regexp.MatchString(`^\d+\. `, line); matched {
			// Ordered list
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 {
				result = append(result, f.formatListItem(parts[0]+".", parts[1]))
			}
		} else if strings.HasPrefix(line, "> ") {
			// Blockquote
			quoteStyle := lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(f.theme.Secondary).
				PaddingLeft(1).
				Foreground(f.theme.TextMuted)
			result = append(result, quoteStyle.Render(strings.TrimPrefix(line, "> ")))
		} else if strings.HasPrefix(line, "```") {
			// Code block (simplified)
			result = append(result, line)
		} else {
			// Regular text with inline formatting
			line = f.formatInlineMarkdown(line)
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// formatInlineMarkdown formats inline markdown elements
func (f *Formatter) formatInlineMarkdown(text string) string {
	if !f.isTTY {
		return text
	}

	// Bold
	boldStyle := f.styles.Bold
	text = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllStringFunc(text, func(s string) string {
		content := strings.Trim(s, "*")
		return boldStyle.Render(content)
	})

	// Italic
	italicStyle := f.styles.Italic
	text = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllStringFunc(text, func(s string) string {
		content := strings.Trim(s, "*")
		return italicStyle.Render(content)
	})

	// Inline code
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllStringFunc(text, func(s string) string {
		content := strings.Trim(s, "`")
		return f.InlineCode(content)
	})

	return text
}

// Highlight highlights text
func (f *Formatter) Highlight(text string) string {
	if !f.isTTY {
		return fmt.Sprintf("[HIGHLIGHT] %s [/HIGHLIGHT]", text)
	}
	return f.styles.Highlight.Render(text)
}

// Muted formats muted text
func (f *Formatter) Muted(text string) string {
	if !f.isTTY {
		return text
	}
	style := lipgloss.NewStyle().Foreground(f.theme.TextMuted)
	return style.Render(text)
}

// Bold formats bold text
func (f *Formatter) Bold(text string) string {
	if !f.isTTY {
		return text
	}
	return f.styles.Bold.Render(text)
}

// Italic formats italic text
func (f *Formatter) Italic(text string) string {
	if !f.isTTY {
		return text
	}
	return f.styles.Italic.Render(text)
}

// Underline formats underlined text
func (f *Formatter) Underline(text string) string {
	if !f.isTTY {
		return text
	}
	return f.styles.Underline.Render(text)
}

// Center centers text
func (f *Formatter) Center(text string) string {
	if !f.isTTY {
		return text
	}

	lines := strings.Split(text, "\n")
	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	padding := (f.width - maxWidth) / 2
	if padding < 0 {
		padding = 0
	}

	var result []string
	for _, line := range lines {
		result = append(result, strings.Repeat(" ", padding)+line)
	}

	return strings.Join(result, "\n")
}

// Wrap wraps text to the formatter width
func (f *Formatter) Wrap(text string) string {
	if !f.isTTY {
		return text
	}

	words := strings.Fields(text)
	var lines []string
	var currentLine []string
	currentLength := 0

	for _, word := range words {
		wordLength := len(word)
		if currentLength+wordLength+1 > f.width && currentLength > 0 {
			lines = append(lines, strings.Join(currentLine, " "))
			currentLine = []string{word}
			currentLength = wordLength
		} else {
			currentLine = append(currentLine, word)
			if currentLength > 0 {
				currentLength += 1 // space
			}
			currentLength += wordLength
		}
	}

	if len(currentLine) > 0 {
		lines = append(lines, strings.Join(currentLine, " "))
	}

	return strings.Join(lines, "\n")
}

// PrintTitle prints a title message
func (f *Formatter) PrintTitle(title string) {
	if f.isTTY {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(f.theme.Primary).
			MarginBottom(1)
		fmt.Println(style.Render(title))
	} else {
		fmt.Printf("=== %s ===\n", title)
	}
}

// PrintSection prints a section header
func (f *Formatter) PrintSection(section string) {
	if f.isTTY {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(f.theme.Secondary).
			MarginTop(1)
		fmt.Println(style.Render(section))
	} else {
		fmt.Printf("\n--- %s ---\n", section)
	}
}

// PrintInfo prints an info message
func (f *Formatter) PrintInfo(message string) {
	if f.isTTY {
		style := lipgloss.NewStyle().Foreground(f.theme.Info)
		fmt.Println(style.Render("ℹ " + message))
	} else {
		fmt.Printf("INFO: %s\n", message)
	}
}

// PrintSuccess prints a success message
func (f *Formatter) PrintSuccess(message string) {
	if f.isTTY {
		style := lipgloss.NewStyle().Foreground(f.theme.Success)
		fmt.Println(style.Render("✓ " + message))
	} else {
		fmt.Printf("SUCCESS: %s\n", message)
	}
}

// PrintWarning prints a warning message
func (f *Formatter) PrintWarning(message string) {
	if f.isTTY {
		style := lipgloss.NewStyle().Foreground(f.theme.Warning)
		fmt.Println(style.Render("⚠ " + message))
	} else {
		fmt.Printf("WARNING: %s\n", message)
	}
}

// PrintError prints an error message
func (f *Formatter) PrintError(message string) {
	if f.isTTY {
		style := lipgloss.NewStyle().Foreground(f.theme.Error)
		fmt.Println(style.Render("✗ " + message))
	} else {
		fmt.Printf("ERROR: %s\n", message)
	}
}

// FormatPrompt formats a prompt for user input
func (f *Formatter) FormatPrompt(prompt string) string {
	if f.isTTY {
		style := lipgloss.NewStyle().
			Foreground(f.theme.Primary).
			Bold(true)
		return style.Render(prompt)
	}
	return prompt
}

// FormatAssistant formats assistant output
func (f *Formatter) FormatAssistant(label string) string {
	if f.isTTY {
		style := lipgloss.NewStyle().
			Foreground(f.theme.Secondary).
			Bold(true)
		return style.Render(label)
	}
	return label
}

// FormatMarkdown formats markdown content
func (f *Formatter) FormatMarkdown(content string) string {
	if !f.markdownEnabled {
		return content
	}

	// Simple markdown formatting
	// Bold
	content = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllStringFunc(content, func(s string) string {
		text := s[2 : len(s)-2]
		if f.isTTY {
			return lipgloss.NewStyle().Bold(true).Render(text)
		}
		return text
	})

	// Italic
	content = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllStringFunc(content, func(s string) string {
		text := s[1 : len(s)-1]
		if f.isTTY {
			return lipgloss.NewStyle().Italic(true).Render(text)
		}
		return text
	})

	// Inline code
	content = regexp.MustCompile("`([^`]+)`").ReplaceAllStringFunc(content, func(s string) string {
		code := s[1 : len(s)-1]
		return f.InlineCode(code)
	})

	// Headers
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			text := line[2:]
			if f.isTTY {
				lines[i] = lipgloss.NewStyle().Bold(true).Foreground(f.theme.Primary).Render(text)
			}
		} else if strings.HasPrefix(line, "## ") {
			text := line[3:]
			if f.isTTY {
				lines[i] = lipgloss.NewStyle().Bold(true).Foreground(f.theme.Secondary).Render(text)
			}
		} else if strings.HasPrefix(line, "### ") {
			text := line[4:]
			if f.isTTY {
				lines[i] = lipgloss.NewStyle().Foreground(f.theme.Tertiary).Render(text)
			}
		}
	}

	return strings.Join(lines, "\n")
}

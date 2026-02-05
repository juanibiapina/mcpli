package terminal

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// GetWidth returns the terminal width, defaulting to 80 if it cannot be determined
func GetWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

// WrapText wraps text to the specified width with the given indent for continuation lines.
// The first line has no indent, subsequent lines are indented.
func WrapText(text string, width int, indent string) string {
	if width <= 0 {
		return text
	}

	// Normalize whitespace - replace newlines and multiple spaces with single space
	text = strings.Join(strings.Fields(text), " ")

	if len(text) <= width {
		return text
	}

	var lines []string
	indentLen := len(indent)
	firstLineWidth := width
	nextLineWidth := width - indentLen

	if nextLineWidth <= 0 {
		nextLineWidth = width
	}

	// First line
	line, remaining := wrapLine(text, firstLineWidth)
	lines = append(lines, line)

	// Subsequent lines with indent
	for remaining != "" {
		line, remaining = wrapLine(remaining, nextLineWidth)
		lines = append(lines, indent+line)
	}

	return strings.Join(lines, "\n")
}

// wrapLine extracts one line of at most width characters, breaking at word boundaries
func wrapLine(text string, width int) (line, remaining string) {
	text = strings.TrimSpace(text)
	if len(text) <= width {
		return text, ""
	}

	// Find last space before width
	breakPoint := width
	for breakPoint > 0 && text[breakPoint] != ' ' {
		breakPoint--
	}

	// If no space found, force break at width
	if breakPoint == 0 {
		breakPoint = width
	}

	return strings.TrimSpace(text[:breakPoint]), strings.TrimSpace(text[breakPoint:])
}

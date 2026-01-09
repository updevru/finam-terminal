package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ApplyOrangeRedGradient applies a horizontal gradient from Orange to Red to the text.
// It returns the text with tview color tags.
func ApplyOrangeRedGradient(text string) string {
	return applyGradient(text, func(r, g, b int) string {
		return fmt.Sprintf("[#%02x%02x%02x]", r, g, b)
	})
}

// ApplyOrangeRedGradientANSI applies a horizontal gradient using ANSI escape codes.
func ApplyOrangeRedGradientANSI(text string) string {
	return applyGradient(text, func(r, g, b int) string {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	}) + "\x1b[0m" // Reset at the end
}

// applyGradient is a helper to apply gradient logic with a custom color formatter
func applyGradient(text string, colorFunc func(r, g, b int) string) string {
	lines := strings.Split(text, "\n")
	maxWidth := 0
	for _, line := range lines {
		w := utf8.RuneCountInString(line)
		if w > maxWidth {
			maxWidth = w
		}
	}

	if maxWidth == 0 {
		return text
	}

	var sb strings.Builder
	for i, line := range lines {
		runes := []rune(line)
		for j, r := range runes {
			// Calculate ratio 0.0 to 1.0
			ratio := float64(j) / float64(maxWidth)
			if maxWidth <= 1 {
				ratio = 0
			}

			// Orange (255, 165, 0) to Red (255, 0, 0)
			rVal := 255
			gVal := int(165 * (1 - ratio))
			bVal := 0

			sb.WriteString(colorFunc(rVal, gVal, bVal))
			sb.WriteRune(r)
		}
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

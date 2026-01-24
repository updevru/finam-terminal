package ui

import (
	"strings"
	"testing"
)

func TestApplyOrangeRedGradient(t *testing.T) {
	text := "HELLO"
	coloredText := ApplyOrangeRedGradient(text)

	if coloredText == text {
		t.Error("ApplyOrangeRedGradient should modify the text")
	}

	if !strings.Contains(coloredText, "[#") {
		t.Error("ApplyOrangeRedGradient should contain tview color tags")
	}
	// Check for start color (approx Orange)
	if !strings.Contains(coloredText, "ff") { // Red component
		t.Error("Should contain red component ff")
	}
}

func TestApplyOrangeRedGradientANSI(t *testing.T) {
	text := "HELLO"
	coloredText := ApplyOrangeRedGradientANSI(text)

	if coloredText == text {
		t.Error("ApplyOrangeRedGradientANSI should modify the text")
	}

	if !strings.Contains(coloredText, "\x1b[38;2;") {
		t.Error("ApplyOrangeRedGradientANSI should contain ANSI color codes")
	}

	if !strings.HasSuffix(coloredText, "\x1b[0m") {
		t.Error("ApplyOrangeRedGradientANSI should reset colors at the end")
	}
}

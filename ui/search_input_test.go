package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestSearchModal_RussianInput(t *testing.T) {
	app := tview.NewApplication()
	modal := NewSearchModal(app, nil, nil, nil, nil)

	// Simulate typing "Привет"
	runes := []rune{'П', 'р', 'и', 'в', 'е', 'т'}

	for _, r := range runes {
		event := tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
		modal.Input.InputHandler()(event, func(p tview.Primitive) {})
	}

	text := modal.Input.GetText()
	expected := "Привет"
	if text != expected {
		t.Errorf("Expected input text to be %q, got %q", expected, text)
	}
}

package ui

import (
	"testing"

	"github.com/rivo/tview"
)

func TestNewSearchModal(t *testing.T) {
	app := tview.NewApplication()
	// Mock callback
	onSelect := func(ticker string) {}
	onCancel := func() {}

	modal := NewSearchModal(app, nil, onSelect, onCancel)

	if modal == nil {
		t.Fatal("Expected NewSearchModal to return a modal, got nil")
	}

	if modal.Input == nil {
		t.Error("Expected SearchModal.Input to be initialized")
	}

	if modal.Table == nil {
		t.Error("Expected SearchModal.Table to be initialized")
	}

	if modal.Layout == nil {
		t.Error("Expected SearchModal.Layout to be initialized")
	}
}

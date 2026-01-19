package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestInputHandler_ModalOpen_EscClosesModal(t *testing.T) {
	app := NewApp(nil, nil)
	
	// Manually open modal
	app.pages.AddPage("modal", tview.NewBox(), true, true)
	
	if !app.IsModalOpen() {
		t.Fatal("Expected modal to be open")
	}

	// Simulate Esc key
	setupInputHandlers(app)
	capture := app.app.GetInputCapture()
	
	event := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	res := capture(event)
	
	if res != nil {
		t.Error("Expected Esc event to be consumed (return nil)")
	}
	
	if app.IsModalOpen() {
		t.Error("Expected modal to be closed after Esc")
	}
}

func TestInputHandler_ModalOpen_TabPassedThrough(t *testing.T) {
	app := NewApp(nil, nil)
	
	// Manually open modal
	app.pages.AddPage("modal", tview.NewBox(), true, true)
	
	if !app.IsModalOpen() {
		t.Fatal("Expected modal to be open")
	}

	// Simulate Tab key
	setupInputHandlers(app)
	capture := app.app.GetInputCapture()
	
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	res := capture(event)
	
	if res == nil {
		t.Error("Expected Tab event to be passed through (return event)")
	}
}

func TestInputHandler_ModalClosed_TabSwitchesFocus(t *testing.T) {
	app := NewApp(nil, nil)
	// Ensure modal is closed
	app.pages.HidePage("modal")
	
	setupInputHandlers(app)
	capture := app.app.GetInputCapture()
	
	// Initial focus on AccountTable (implicit default in setup)
	app.app.SetFocus(app.portfolioView.AccountTable)
	
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	res := capture(event)
	
	if res != nil {
		t.Error("Expected Tab event to be consumed")
	}
	
	if app.app.GetFocus() != app.portfolioView.PositionsTable {
		t.Error("Expected focus to switch to PositionsTable")
	}
}

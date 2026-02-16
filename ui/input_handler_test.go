package ui

import (
	"finam-terminal/models"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestInputHandler_ModalOpen_EscClosesModal(t *testing.T) {
	app := NewApp(&mockClient{}, nil)

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
	app := NewApp(&mockClient{}, nil)

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

func TestInputHandler_TabSwitchesFocus(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	setupInputHandlers(app)

	// Focus on AccountTable
	app.app.SetFocus(app.portfolioView.AccountTable)

	// Get capture for Application
	capture := app.app.GetInputCapture()

	// Simulate Tab key
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	capture(event)

	if app.app.GetFocus() == app.portfolioView.AccountTable {
		t.Error("Expected focus to switch away from AccountTable")
	}

	// Simulate Tab key again
	capture(event)
	if app.app.GetFocus() != app.portfolioView.AccountTable {
		t.Error("Expected focus to switch back to AccountTable")
	}
}

func TestInputHandler_ArrowsSwitchTabs(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	setupInputHandlers(app)

	// Focus on PositionsTable
	app.app.SetFocus(app.portfolioView.TabbedView.PositionsTable)

	// Get capture for Application
	capture := app.app.GetInputCapture()

	// Simulate Right arrow key
	event := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	capture(event)

	if app.portfolioView.TabbedView.ActiveTab != TabHistory {
		t.Errorf("Expected ActiveTab to be TabHistory, got %v", app.portfolioView.TabbedView.ActiveTab)
	}

	if app.app.GetFocus() != app.portfolioView.TabbedView.HistoryTable {
		t.Error("Expected focus to switch to HistoryTable")
	}
}

func TestInputHandler_F2RefreshesHistory(t *testing.T) {
	historyCalled := make(chan bool, 1)
	client := &mockClient{
		GetTradeHistoryFunc: func(accountID string) ([]models.Trade, error) {
			historyCalled <- true
			return nil, nil
		},
	}
	app := NewApp(client, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	setupInputHandlers(app)

	// Set active tab to History
	app.portfolioView.TabbedView.SetTab(TabHistory)

	// Get capture for Application
	capture := app.app.GetInputCapture()

	// Simulate F2 key
	event := tcell.NewEventKey(tcell.KeyF2, 0, tcell.ModNone)
	capture(event)

	select {
	case <-historyCalled:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("Timed out waiting for GetTradeHistory to be called on F2")
	}
}

func TestInputHandler_F2RefreshesOrders(t *testing.T) {
	ordersCalled := make(chan bool, 1)
	client := &mockClient{
		GetActiveOrdersFunc: func(accountID string) ([]models.Order, error) {
			ordersCalled <- true
			return nil, nil
		},
	}
	app := NewApp(client, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	setupInputHandlers(app)

	// Set active tab to Orders
	app.portfolioView.TabbedView.SetTab(TabOrders)

	// Get capture for Application
	capture := app.app.GetInputCapture()

	// Simulate F2 key
	event := tcell.NewEventKey(tcell.KeyF2, 0, tcell.ModNone)
	capture(event)

	select {
	case <-ordersCalled:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("Timed out waiting for GetActiveOrders to be called on F2")
	}
}

package ui

import (
	"finam-terminal/models"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestSearchToOrderTransition(t *testing.T) {
	client := &mockClient{
		SearchSecuritiesFunc: func(query string) ([]models.SecurityInfo, error) {
			return []models.SecurityInfo{{Ticker: "AAPL", Name: "Apple"}}, nil
		},
		GetSnapshotsFunc: func(symbols []string) (map[string]models.Quote, error) {
			return map[string]models.Quote{"AAPL": {Last: "150.00"}}, nil
		},
		GetAccountsFunc: func() ([]models.AccountInfo, error) {
			return []models.AccountInfo{{ID: "acc1"}}, nil
		},
	}

	app := NewApp(client, []models.AccountInfo{{ID: "acc1"}})

	// Manually add pages that are normally added in Run()
	app.pages.AddPage("main", tview.NewBox(), true, true)
	app.pages.AddPage("modal", app.orderModal.Layout, true, false)
	app.pages.AddPage("search_modal", app.searchModal.Layout, true, false)

	// Initially search modal should be closed
	if app.IsSearchModalOpen() {
		t.Error("Search modal should be closed initially")
	}

	// Open Search Modal
	app.OpenSearchModal()
	if !app.IsSearchModalOpen() {
		t.Error("Search modal should be open")
	}

	// Simulate selecting AAPL and pressing 'A'
	app.searchModal.results = []models.SecurityInfo{{Ticker: "AAPL", Name: "Apple"}}
	app.searchModal.updateTable(nil)
	app.searchModal.Table.Select(1, 0) // Select first row after header

	// Trigger 'A' key
	event := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	app.searchModal.Table.InputHandler()(event, func(p tview.Primitive) {})

	// Search modal should close
	if app.IsSearchModalOpen() {
		t.Error("Search modal should be closed after selection")
	}

	// Order modal should open with AAPL
	if !app.IsModalOpen() {
		t.Error("Order modal should be open")
	}

	if app.orderModal.GetInstrument() != "AAPL" {
		t.Errorf("Expected AAPL in order modal, got %s", app.orderModal.GetInstrument())
	}
}

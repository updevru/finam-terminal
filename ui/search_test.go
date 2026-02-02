package ui

import (
	"finam-terminal/models"
	"testing"

	"github.com/gdamore/tcell/v2"
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

func TestSearchModal_Search(t *testing.T) {
	app := tview.NewApplication()
	client := &mockClient{
		SearchSecuritiesFunc: func(query string) ([]models.SecurityInfo, error) {
			return []models.SecurityInfo{
				{Ticker: "SBER", Name: "Sberbank", Lot: 10, Currency: "RUB"},
			}, nil
		},
		GetSnapshotsFunc: func(symbols []string) (map[string]models.Quote, error) {
			return map[string]models.Quote{
				"SBER": {Last: "250.00"},
			}, nil
		},
	}

	modal := NewSearchModal(app, client, nil, nil)

	// We test PerformSearch directly to avoid QueueUpdateDraw/Application loop issues in tests
	// But we want to make sure updateTable works.
	modal.results = []models.SecurityInfo{
		{Ticker: "SBER", Name: "Sberbank", Lot: 10, Currency: "RUB"},
	}
	quotes := map[string]models.Quote{
		"SBER": {Last: "250.00"},
	}
	
	modal.updateTable(quotes)

	if modal.Table.GetRowCount() != 2 { // Header + 1 row
		t.Errorf("Expected 2 rows in table, got %d", modal.Table.GetRowCount())
	}

	cell := modal.Table.GetCell(1, 0)
	if cell.Text != "SBER" {
		t.Errorf("Expected cell text SBER, got %s", cell.Text)
	}
	
	priceCell := modal.Table.GetCell(1, 4)
	if priceCell.Text != "250.00" {
		t.Errorf("Expected price 250.00, got %s", priceCell.Text)
	}
}

func TestSearchModal_Navigation(t *testing.T) {
	app := tview.NewApplication()
	modal := NewSearchModal(app, nil, nil, nil)

	app.SetFocus(modal.Input)
	if app.GetFocus() != modal.Input {
		t.Fatal("Expected Input to have focus")
	}

	// Simulate Tab
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	modal.Input.InputHandler()(event, func(p tview.Primitive) {})

	if app.GetFocus() != modal.Table {
		t.Errorf("Expected Table to have focus after Tab, got %T", app.GetFocus())
	}

	// Simulate Tab back
	modal.Table.InputHandler()(event, func(p tview.Primitive) {})

	if app.GetFocus() != modal.Input {
		t.Errorf("Expected Input to have focus after second Tab, got %T", app.GetFocus())
	}
}

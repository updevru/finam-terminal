package ui

import (
	"finam-terminal/models"
	"testing"
)

func TestApp_OpenOrderModal_PrefillsInstrument(t *testing.T) {
	// Setup App
	accounts := []models.AccountInfo{{ID: "acc1"}}
	app := NewApp(nil, accounts)

	// Mock positions
	app.positions["acc1"] = []models.Position{
		{Ticker: "GAZP", MIC: "TQBR", Symbol: "GAZP@TQBR"},
		{Ticker: "SBER", MIC: "TQBR", Symbol: "SBER@TQBR"},
	}
	app.selectedIdx = 0

	// Populate table manually (since we can't run the full UI loop)
	updatePositionsTable(app)

	// Select the second row (SBER) - Row 0 is header, Row 1 is GAZP, Row 2 is SBER
	app.portfolioView.PositionsTable.Select(2, 0)

	// Action
	app.OpenOrderModal()

	// Assert
	if app.orderModal.GetInstrument() != "SBER" {
		t.Errorf("Expected instrument SBER, got %s", app.orderModal.GetInstrument())
	}

	// Verify "modal" page is shown?
	// tview.Pages doesn't easily expose state.
	// But we can verify the side effect on the modal component.
}

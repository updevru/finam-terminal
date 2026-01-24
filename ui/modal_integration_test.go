package ui

import (
	"finam-terminal/models"
	"testing"
)

func TestOpenCloseModal(t *testing.T) {
	accounts := []models.AccountInfo{{ID: "acc1"}}
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient, accounts)
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{
		{Ticker: "SBER", Quantity: "10", CurrentPrice: "250.50", UnrealizedPnL: "100"},
	}

	// Mock table selection (row 1 is first position)
	app.portfolioView.PositionsTable.Select(1, 0)

	// Setup pages for testing
	app.pages.AddPage("close_modal", app.closeModal.Layout, true, false)

	// Act
	app.OpenCloseModal()

	// Assert
	if !app.IsCloseModalOpen() {
		t.Error("Expected close modal to be open")
	}

	if app.closeModal.GetSymbol() != "SBER" {
		t.Errorf("Expected symbol SBER, got %s", app.closeModal.GetSymbol())
	}

	// Quantity field is intentionally cleared for user input
	if app.closeModal.GetQuantity() != 0 {
		t.Errorf("Expected quantity 0, got %f", app.closeModal.GetQuantity())
	}
}

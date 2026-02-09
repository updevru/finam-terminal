package ui

import (
	"finam-terminal/models"
	"fmt"
	"testing"
)

func TestSubmitOrder_Success(t *testing.T) {
	accounts := []models.AccountInfo{{ID: "acc1"}}
	mockClient := &mockClient{
		PlaceOrderFunc: func(id string, sym string, side string, qty float64) (string, error) {
			if id != "acc1" {
				return "", fmt.Errorf("wrong account")
			}
			if sym != "SBER" {
				return "", fmt.Errorf("wrong symbol")
			}
			if qty != 10 {
				return "", fmt.Errorf("wrong qty")
			}
			return "ord1", nil
		},
	}
	app := NewApp(mockClient, accounts)
	app.selectedIdx = 0

	// Act
	err := app.SubmitOrder("SBER", 10, "Buy")

	// Assert
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSubmitOrder_Error(t *testing.T) {
	accounts := []models.AccountInfo{{ID: "acc1"}}
	mockClient := &mockClient{
		PlaceOrderFunc: func(id string, sym string, side string, qty float64) (string, error) {
			return "", fmt.Errorf("api error")
		},
	}
	app := NewApp(mockClient, accounts)
	app.selectedIdx = 0

	// Act
	err := app.SubmitOrder("SBER", 10, "Buy")

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "api error" {
		t.Errorf("Expected 'api error', got '%v'", err)
	}
}

func TestSubmitClosePosition_Success(t *testing.T) {
	accounts := []models.AccountInfo{{ID: "acc1"}}
	mockClient := &mockClient{
		ClosePositionFunc: func(id string, sym string, curQty string, closeQty float64) (string, error) {
			// CRITICAL: Verify we receive the full symbol (Ticker@MIC)
			if sym != "SBER@TQBR" {
				return "", fmt.Errorf("expected symbol 'SBER@TQBR', got '%s'", sym)
			}
			if closeQty != 5 {
				return "", fmt.Errorf("wrong qty")
			}
			return "cls1", nil
		},
	}
	app := NewApp(mockClient, accounts)
	app.selectedIdx = 0
	// Setup position with Ticker and full Symbol
	app.positions["acc1"] = []models.Position{{Ticker: "SBER", Symbol: "SBER@TQBR", Quantity: "10"}}

	// Mock table selection
	app.portfolioView.TabbedView.PositionsTable.Select(1, 0)

	// Act
	err := app.SubmitClosePosition(5)

	// Assert
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSubmitClosePosition_Error(t *testing.T) {
	accounts := []models.AccountInfo{{ID: "acc1"}}
	mockClient := &mockClient{
		ClosePositionFunc: func(id string, sym string, curQty string, closeQty float64) (string, error) {
			return "", fmt.Errorf("api error")
		},
	}
	app := NewApp(mockClient, accounts)
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{{Ticker: "SBER", Quantity: "10"}}

	// Mock table selection
	app.portfolioView.TabbedView.PositionsTable.Select(1, 0)

	// Act
	err := app.SubmitClosePosition(5)

	// Assert
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "api error" {
		t.Errorf("Expected 'api error', got '%v'", err)
	}
}

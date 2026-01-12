package ui

import (
	"finam-terminal/models"
	"fmt"
	"testing"
)

type MockAPIClient struct {
	PlaceOrderFunc func(accountID string, symbol string, buySell string, quantity float64) (string, error)
}

func (m *MockAPIClient) GetAccounts() ([]models.AccountInfo, error) { return nil, nil }
func (m *MockAPIClient) GetAccountDetails(id string) (*models.AccountInfo, []models.Position, error) { return nil, nil, nil }
func (m *MockAPIClient) GetQuotes(syms []string) (map[string]*models.Quote, error) { return nil, nil }
func (m *MockAPIClient) PlaceOrder(accountID string, symbol string, buySell string, quantity float64) (string, error) {
	if m.PlaceOrderFunc != nil {
		return m.PlaceOrderFunc(accountID, symbol, buySell, quantity)
	}
	return "123", nil
}

func TestSubmitOrder_Success(t *testing.T) {
	accounts := []models.AccountInfo{{ID: "acc1"}}
	mockClient := &MockAPIClient{
		PlaceOrderFunc: func(id string, sym string, side string, qty float64) (string, error) {
			if id != "acc1" { return "", fmt.Errorf("wrong account") }
			if sym != "SBER" { return "", fmt.Errorf("wrong symbol") }
			if qty != 10 { return "", fmt.Errorf("wrong qty") }
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
	mockClient := &MockAPIClient{
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

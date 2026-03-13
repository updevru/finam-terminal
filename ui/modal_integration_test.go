package ui

import (
	"finam-terminal/models"
	"fmt"
	"testing"
)

func TestOpenCloseModal(t *testing.T) {
	accounts := []models.AccountInfo{{ID: "acc1"}}
	mockClient := &mockClient{}
	app := NewApp(mockClient, accounts)
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{
		{Ticker: "SBER", Quantity: "10", CurrentPrice: "250.50", UnrealizedPnL: "100"},
	}

	// Mock table selection (row 1 is first position)
	app.portfolioView.TabbedView.PositionsTable.Select(1, 0)

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

func TestGetSelectedOrder(t *testing.T) {
	mock := &mockClient{}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "SBER", Side: "Buy", Type: "Limit", Status: "Active"},
		{ID: "O2", Symbol: "GAZP", Side: "Sell", Type: "Stop", Status: "Active"},
	}

	updateOrdersTable(app)

	// Select first data row (row 1, since row 0 is header)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	order, err := app.getSelectedOrder()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if order.ID != "O1" {
		t.Errorf("Expected order ID O1, got %s", order.ID)
	}

	// Select second row
	app.portfolioView.TabbedView.OrdersTable.Select(2, 0)
	order, err = app.getSelectedOrder()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if order.ID != "O2" {
		t.Errorf("Expected order ID O2, got %s", order.ID)
	}
}

func TestCancelSelectedOrder_Success(t *testing.T) {
	cancelCalled := false
	mock := &mockClient{}
	mock.CancelOrderFunc = func(accountID, orderID string) error {
		cancelCalled = true
		if accountID != "acc1" {
			t.Errorf("Expected accountID acc1, got %s", accountID)
		}
		if orderID != "O1" {
			t.Errorf("Expected orderID O1, got %s", orderID)
		}
		return nil
	}
	mock.GetActiveOrdersFunc = func(accountID string) ([]models.Order, error) {
		return nil, nil
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "SBER", Side: "Buy", Type: "Limit", Status: "Active"},
	}

	err := app.cancelOrder("acc1", "O1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !cancelCalled {
		t.Error("Expected CancelOrder to be called")
	}
}

func TestCancelSelectedOrder_Error(t *testing.T) {
	mock := &mockClient{}
	mock.CancelOrderFunc = func(accountID, orderID string) error {
		return fmt.Errorf("order already filled")
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})

	err := app.cancelOrder("acc1", "O1")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

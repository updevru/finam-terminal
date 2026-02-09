package ui

import (
	"finam-terminal/models"
	"testing"
	"time"
)

func TestTabbedView_DataLoading(t *testing.T) {
	mock := &mockClient{}
	mock.GetTradeHistoryFunc = func(accountID string) ([]models.Trade, error) {
		return []models.Trade{{ID: "T1", Symbol: "SBER"}}, nil
	}
	mock.GetActiveOrdersFunc = func(accountID string) ([]models.Order, error) {
		return []models.Order{{ID: "O1", Symbol: "GAZP"}}, nil
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	setupInputHandlers(app)

	// Switch to History tab
	app.portfolioView.TabbedView.SetTab(TabHistory)
	app.loadHistoryAsync("acc1")

	// Wait for async load (rough wait for test)
	time.Sleep(100 * time.Millisecond)

	// Manually trigger update since QueueUpdateDraw doesn't run in tests
	updateHistoryTable(app)

	rowCount := app.portfolioView.TabbedView.HistoryTable.GetRowCount()
	if rowCount != 2 { // header + 1 row
		t.Errorf("Expected 2 rows in history table, got %d", rowCount)
	}

	// Switch to Orders tab
	app.portfolioView.TabbedView.SetTab(TabOrders)
	app.loadOrdersAsync("acc1")

	time.Sleep(100 * time.Millisecond)

	// Manually trigger update
	updateOrdersTable(app)

	rowCount = app.portfolioView.TabbedView.OrdersTable.GetRowCount()
	if rowCount != 2 { // header + 1 row
		t.Errorf("Expected 2 rows in orders table, got %d", rowCount)
	}
}

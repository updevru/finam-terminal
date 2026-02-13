package ui

import (
	"finam-terminal/models"
	"testing"
	"time"
)

func TestHistoryTable_LotBasedQuantity(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 {
		if ticker == "SBER" || ticker == "SBER@TQBR" {
			return 10
		}
		return 1
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.history["acc1"] = []models.Trade{
		{ID: "T1", Symbol: "SBER@TQBR", Side: "Buy", Quantity: "100", Price: "250.00", Total: "25000.00"},
	}

	updateHistoryTable(app)

	// Row 0 is header, row 1 is the trade
	// Header column 3 should be "Qty (Lots)"
	headerCell := app.portfolioView.TabbedView.HistoryTable.GetCell(0, 3)
	if headerCell.Text != "Qty (Lots)" {
		t.Errorf("Expected history header 'Qty (Lots)', got '%s'", headerCell.Text)
	}

	// Quantity should be displayed as lots: 100 shares / 10 lot size = 10 lots
	qtyCell := app.portfolioView.TabbedView.HistoryTable.GetCell(1, 3)
	if qtyCell.Text != "10" {
		t.Errorf("Expected history qty '10' (lots), got '%s'", qtyCell.Text)
	}
}

func TestOrdersTable_LotBasedQuantity(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 {
		if ticker == "GAZP" || ticker == "GAZP@TQBR" {
			return 10
		}
		return 1
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "GAZP@TQBR", Side: "Sell", Type: "Market", Status: "New", Quantity: "50"},
	}

	updateOrdersTable(app)

	// Header column 4 should be "Qty (Lots)"
	headerCell := app.portfolioView.TabbedView.OrdersTable.GetCell(0, 4)
	if headerCell.Text != "Qty (Lots)" {
		t.Errorf("Expected orders header 'Qty (Lots)', got '%s'", headerCell.Text)
	}

	// Quantity should be lots: 50 shares / 10 lot size = 5 lots
	qtyCell := app.portfolioView.TabbedView.OrdersTable.GetCell(1, 4)
	if qtyCell.Text != "5" {
		t.Errorf("Expected orders qty '5' (lots), got '%s'", qtyCell.Text)
	}
}

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

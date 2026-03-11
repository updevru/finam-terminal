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

	// Header column 4 should be "Qty"
	headerCell := app.portfolioView.TabbedView.OrdersTable.GetCell(0, 4)
	if headerCell.Text != "Qty" {
		t.Errorf("Expected orders header 'Qty', got '%s'", headerCell.Text)
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

func TestPositionsTable_InstrumentHeader(t *testing.T) {
	mock := &mockClient{}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.positions["acc1"] = []models.Position{
		{Symbol: "SBER@TQBR", Ticker: "SBER", Name: "Сбербанк", Quantity: "10", LotSize: 1},
	}

	updatePositionsTable(app)

	headerCell := app.portfolioView.TabbedView.PositionsTable.GetCell(0, 0)
	if headerCell.Text != "Instrument" {
		t.Errorf("Expected header 'Instrument', got '%s'", headerCell.Text)
	}

	// Should display Name when available
	nameCell := app.portfolioView.TabbedView.PositionsTable.GetCell(1, 0)
	if nameCell.Text != "Сбербанк" {
		t.Errorf("Expected instrument name 'Сбербанк', got '%s'", nameCell.Text)
	}
}

func TestPositionsTable_FallbackToTicker(t *testing.T) {
	mock := &mockClient{}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.positions["acc1"] = []models.Position{
		{Symbol: "UNKNOWN@TQBR", Ticker: "UNKNOWN", Name: "", Quantity: "10", LotSize: 1},
	}

	updatePositionsTable(app)

	nameCell := app.portfolioView.TabbedView.PositionsTable.GetCell(1, 0)
	if nameCell.Text != "UNKNOWN" {
		t.Errorf("Expected fallback ticker 'UNKNOWN', got '%s'", nameCell.Text)
	}
}

func TestHistoryTable_InstrumentHeader(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.history["acc1"] = []models.Trade{
		{ID: "T1", Symbol: "SBER", Name: "Сбербанк", Side: "Buy", Quantity: "10", Price: "250", Total: "2500"},
	}

	updateHistoryTable(app)

	headerCell := app.portfolioView.TabbedView.HistoryTable.GetCell(0, 0)
	if headerCell.Text != "Instrument" {
		t.Errorf("Expected header 'Instrument', got '%s'", headerCell.Text)
	}

	nameCell := app.portfolioView.TabbedView.HistoryTable.GetCell(1, 0)
	if nameCell.Text != "Сбербанк" {
		t.Errorf("Expected instrument name 'Сбербанк', got '%s'", nameCell.Text)
	}
}

func TestOrdersTable_InstrumentHeader(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "GAZP", Name: "Газпром", Side: "Buy", Type: "Market", Status: "New", Quantity: "10"},
	}

	updateOrdersTable(app)

	headerCell := app.portfolioView.TabbedView.OrdersTable.GetCell(0, 0)
	if headerCell.Text != "Instrument" {
		t.Errorf("Expected header 'Instrument', got '%s'", headerCell.Text)
	}

	nameCell := app.portfolioView.TabbedView.OrdersTable.GetCell(1, 0)
	if nameCell.Text != "Газпром" {
		t.Errorf("Expected instrument name 'Газпром', got '%s'", nameCell.Text)
	}
}

func TestStatusBar_OrdersTabShortcuts(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})

	// Add a cancellable order
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "SBER", Side: "Buy", Type: "Limit", Status: "New", Quantity: "10"},
	}

	// Switch to Orders tab and focus the table
	app.portfolioView.TabbedView.SetTab(TabOrders)
	app.app.SetFocus(app.portfolioView.TabbedView.OrdersTable)

	updateStatusBar(app)

	statusText := app.statusBar.GetText(false)
	if !contains(statusText, "Cancel") {
		t.Errorf("Expected status bar to contain 'Cancel' when Orders tab focused, got '%s'", statusText)
	}
	if !contains(statusText, "Modify") {
		t.Errorf("Expected status bar to contain 'Modify' when Orders tab focused, got '%s'", statusText)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestOrdersTable_EnhancedColumns(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 10 }
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})

	app.activeOrders["acc1"] = []models.Order{
		{
			ID: "STOP-1", Symbol: "SBER@TQBR", Name: "Сбербанк", Side: "Sell",
			Type: "Stop", Status: "New", Quantity: "100",
			StopPrice: "240.00", StopCondition: "Last Down", Validity: "GTC",
			ExecutedQty: "0", RemainingQty: "100",
		},
		{
			ID: "SLTP-1", Symbol: "GAZP@TQBR", Name: "Газпром", Side: "Sell",
			Type: "SL/TP", Status: "New", Quantity: "",
			SLPrice: "170.00", TPPrice: "200.00", SLQty: "10", TPQty: "10", Validity: "GTC",
		},
	}

	updateOrdersTable(app)

	// Verify header includes new columns (no Validity — it's inlined into Price/Condition)
	expectedHeaders := []string{"Instrument", "Side", "Type", "Status", "Qty", "Executed", "Price/Condition", "Time"}
	for i, expected := range expectedHeaders {
		cell := app.portfolioView.TabbedView.OrdersTable.GetCell(0, i)
		if cell.Text != expected {
			t.Errorf("Header[%d]: expected '%s', got '%s'", i, expected, cell.Text)
		}
	}

	// Row 1: Stop order — Price/Condition should show "SL: 240.00 ↓" (GTC omitted)
	priceCell := app.portfolioView.TabbedView.OrdersTable.GetCell(1, 6)
	if priceCell.Text != "SL: 240.00 ↓" {
		t.Errorf("Stop order Price/Condition: expected 'SL: 240.00 ↓', got '%s'", priceCell.Text)
	}

	// Executed column
	execCell := app.portfolioView.TabbedView.OrdersTable.GetCell(1, 5)
	if execCell.Text != "0" {
		t.Errorf("Stop order Executed: expected '0', got '%s'", execCell.Text)
	}

	// Row 2: SL/TP order — Price/Condition should show combined SL/TP
	sltpPriceCell := app.portfolioView.TabbedView.OrdersTable.GetCell(2, 6)
	if sltpPriceCell.Text != "SL:170.00 / TP:200.00" {
		t.Errorf("SL/TP order Price/Condition: expected 'SL:170.00 / TP:200.00', got '%s'", sltpPriceCell.Text)
	}
}

func TestOrdersTable_NonGTCValidityInlined(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})

	app.activeOrders["acc1"] = []models.Order{
		{
			ID: "LIM-1", Symbol: "SBER", Side: "Buy",
			Type: "Limit", Status: "Active", Quantity: "10",
			LimitPrice: "250.00", Validity: "Day",
		},
	}

	updateOrdersTable(app)

	// Price/Condition should include "(Day)"
	priceCell := app.portfolioView.TabbedView.OrdersTable.GetCell(1, 6)
	if priceCell.Text != "250.00 (Day)" {
		t.Errorf("Expected '250.00 (Day)', got '%s'", priceCell.Text)
	}
}

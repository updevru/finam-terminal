package ui

import (
	"finam-terminal/models"
	"fmt"
	"testing"

	"github.com/rivo/tview"
)

func TestMapOrderTypeToModal(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Market", models.OrderTypeMarket},
		{"Limit", models.OrderTypeLimit},
		{"Stop", models.OrderTypeStop},
		{"Take-Profit", models.OrderTypeTakeProfit},
		{"SL/TP", models.OrderTypeSLTP},
		{"Unknown", ""},        // unsupported type
		{"Stop-Limit", ""},     // unsupported type
	}

	for _, tt := range tests {
		result := mapOrderTypeToModal(tt.input)
		if result != tt.expected {
			t.Errorf("mapOrderTypeToModal(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func setupModalPage(app *App) {
	modalFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(app.orderModal.Layout, 20, 1, true).
			AddItem(nil, 0, 1, false), 50, 1, true).
		AddItem(nil, 0, 1, false)
	app.pages.AddPage("modal", modalFlex, true, false)
}

func TestShowModifyOrderModal_PreFillsLimitOrder(t *testing.T) {
	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 10 },
		GetInstrumentNameFunc: func(key string) string {
			if key == "SBER@MOEX" {
				return "Sberbank"
			}
			return ""
		},
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	setupModalPage(app)
	app.activeOrders["acc1"] = []models.Order{
		{
			ID:         "O1",
			Symbol:     "SBER@MOEX",
			Name:       "Sberbank",
			Side:       "Buy",
			Type:       "Limit",
			Status:     "Active",
			Quantity:   "100",
			LimitPrice: "250.50",
		},
	}

	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	app.ShowModifyOrderModal()

	// Verify modal is open
	if !app.IsModalOpen() {
		t.Fatal("Expected order modal to be open")
	}

	// Verify pre-filled values
	if got := app.orderModal.GetInstrument(); got != "SBER@MOEX" {
		t.Errorf("Instrument = %q, want %q", got, "SBER@MOEX")
	}
	if got := app.orderModal.GetDirection(); got != "Buy" {
		t.Errorf("Direction = %q, want %q", got, "Buy")
	}
	if got := app.orderModal.GetOrderType(); got != models.OrderTypeLimit {
		t.Errorf("OrderType = %q, want %q", got, models.OrderTypeLimit)
	}
	// Quantity: 100 shares / 10 lot size = 10 lots
	if got := app.orderModal.GetQuantity(); got != 10 {
		t.Errorf("Quantity = %v, want %v", got, 10.0)
	}
	// Limit price field should be set
	if app.orderModal.limitPriceField == nil {
		t.Fatal("Expected limitPriceField to be created")
	}
	if got := app.orderModal.getPriceFieldValue(app.orderModal.limitPriceField); got != 250.50 {
		t.Errorf("LimitPrice = %v, want %v", got, 250.50)
	}
}

func TestShowModifyOrderModal_PreFillsStopOrder(t *testing.T) {
	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 1 },
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{
			ID:        "O2",
			Symbol:    "GAZP@MOEX",
			Side:      "Sell",
			Type:      "Stop",
			Status:    "Active",
			Quantity:  "50",
			StopPrice: "180.00",
		},
	}

	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	app.ShowModifyOrderModal()

	if got := app.orderModal.GetDirection(); got != "Sell" {
		t.Errorf("Direction = %q, want %q", got, "Sell")
	}
	if got := app.orderModal.GetOrderType(); got != models.OrderTypeStop {
		t.Errorf("OrderType = %q, want %q", got, models.OrderTypeStop)
	}
	if app.orderModal.stopPriceField == nil {
		t.Fatal("Expected stopPriceField to be created")
	}
	if got := app.orderModal.getPriceFieldValue(app.orderModal.stopPriceField); got != 180.00 {
		t.Errorf("StopPrice = %v, want %v", got, 180.00)
	}
}

func TestShowModifyOrderModal_PreFillsSLTPOrder(t *testing.T) {
	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 1 },
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{
			ID:       "O3",
			Symbol:   "VTBR@MOEX",
			Side:     "Buy",
			Type:     "SL/TP",
			Status:   "Active",
			Quantity: "200",
			SLPrice:  "90.00",
			TPPrice:  "120.00",
		},
	}

	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	app.ShowModifyOrderModal()

	if got := app.orderModal.GetOrderType(); got != models.OrderTypeSLTP {
		t.Errorf("OrderType = %q, want %q", got, models.OrderTypeSLTP)
	}
	if app.orderModal.slPriceField == nil {
		t.Fatal("Expected slPriceField to be created")
	}
	if app.orderModal.tpPriceField == nil {
		t.Fatal("Expected tpPriceField to be created")
	}
	if got := app.orderModal.getPriceFieldValue(app.orderModal.slPriceField); got != 90.00 {
		t.Errorf("SLPrice = %v, want %v", got, 90.00)
	}
	if got := app.orderModal.getPriceFieldValue(app.orderModal.tpPriceField); got != 120.00 {
		t.Errorf("TPPrice = %v, want %v", got, 120.00)
	}
}

func TestShowModifyOrderModal_NonCancellableOrderRejected(t *testing.T) {
	mock := &mockClient{}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "SBER", Status: "Filled", Type: "Market"},
	}

	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	app.ShowModifyOrderModal()

	// Modal should NOT be open for non-cancellable orders
	if app.IsModalOpen() {
		t.Error("Expected modal to NOT be open for filled order")
	}
}

func TestShowModifyOrderModal_TitleShowsModify(t *testing.T) {
	mock := &mockClient{
		GetLotSizeFunc:        func(ticker string) float64 { return 1 },
		GetInstrumentNameFunc: func(key string) string { return "Sberbank" },
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "SBER", Name: "Sberbank", Side: "Buy", Type: "Market", Status: "Active", Quantity: "10"},
	}

	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	app.ShowModifyOrderModal()

	// Title should contain "Modify"
	title := app.orderModal.Layout.GetTitle()
	if title == "" {
		t.Fatal("Expected non-empty title")
	}
	if title != " Modify Order — Sberbank " {
		t.Errorf("Title = %q, want %q", title, " Modify Order — Sberbank ")
	}
}

func TestModifyOrderFlow_CancelThenPlace(t *testing.T) {
	cancelCalled := false
	placeCalled := false
	var placedParams *models.OrderParams
	done := make(chan struct{})

	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 1 },
		CancelOrderFunc: func(accountID, orderID string) error {
			cancelCalled = true
			if orderID != "O1" {
				t.Errorf("Expected cancel orderID O1, got %s", orderID)
			}
			return nil
		},
		PlaceOrderFunc: func(accountID, symbol, buySell string, quantity float64, params *models.OrderParams) (string, error) {
			placeCalled = true
			placedParams = params
			return "NEW-1", nil
		},
		GetActiveOrdersFunc: func(accountID string) ([]models.Order, error) {
			defer func() { close(done) }()
			return nil, nil
		},
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.activeOrders["acc1"] = []models.Order{
		{
			ID:         "O1",
			Symbol:     "SBER",
			Side:       "Buy",
			Type:       "Limit",
			Status:     "Active",
			Quantity:   "10",
			LimitPrice: "250",
		},
	}

	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	// Open modify modal
	app.ShowModifyOrderModal()

	// Simulate the callback being invoked (as if user pressed Create)
	sub := app.orderModal.buildSubmission()
	app.orderModal.GetCallback()(sub)

	// Wait for the goroutine to complete
	<-done

	if !cancelCalled {
		t.Error("Expected CancelOrder to be called")
	}
	if !placeCalled {
		t.Error("Expected PlaceOrder to be called")
	}
	if placedParams == nil {
		t.Fatal("Expected order params to be set for Limit order")
	}
}

func TestModifyOrderFlow_CancelFails_NoNewOrder(t *testing.T) {
	placeCalled := false
	done := make(chan struct{})

	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 1 },
		CancelOrderFunc: func(accountID, orderID string) error {
			defer func() { close(done) }()
			return fmt.Errorf("rpc error: code = NotFound desc = Order already executed")
		},
		PlaceOrderFunc: func(accountID, symbol, buySell string, quantity float64, params *models.OrderParams) (string, error) {
			placeCalled = true
			return "NEW-1", nil
		},
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	setupModalPage(app)
	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "SBER", Side: "Buy", Type: "Limit", Status: "Active", Quantity: "10", LimitPrice: "250"},
	}

	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	app.ShowModifyOrderModal()

	sub := app.orderModal.buildSubmission()
	app.orderModal.GetCallback()(sub)

	// Wait for the goroutine to complete
	<-done

	if placeCalled {
		t.Error("PlaceOrder should NOT be called when cancel fails")
	}
}

func TestModifyOrderFlow_CallbackRestoredAfterModify(t *testing.T) {
	originalCallbackCalled := false
	done := make(chan struct{})

	mock := &mockClient{
		GetLotSizeFunc: func(ticker string) float64 { return 1 },
		CancelOrderFunc: func(accountID, orderID string) error {
			return nil
		},
		PlaceOrderFunc: func(accountID, symbol, buySell string, quantity float64, params *models.OrderParams) (string, error) {
			return "NEW-1", nil
		},
		GetActiveOrdersFunc: func(accountID string) ([]models.Order, error) {
			defer func() {
				select {
				case <-done:
				default:
					close(done)
				}
			}()
			return nil, nil
		},
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	setupModalPage(app)

	// Set a callback that will be saved as original
	app.orderModal.callback = func(sub OrderSubmission) {
		originalCallbackCalled = true
	}

	app.activeOrders["acc1"] = []models.Order{
		{ID: "O1", Symbol: "SBER", Side: "Buy", Type: "Market", Status: "Active", Quantity: "10"},
	}

	updateOrdersTable(app)
	app.portfolioView.TabbedView.OrdersTable.Select(1, 0)

	app.ShowModifyOrderModal()

	// Invoke the modify callback
	sub := app.orderModal.buildSubmission()
	app.orderModal.GetCallback()(sub)

	// Wait for the goroutine to complete
	<-done

	// Now invoke the callback again — should be the original
	app.orderModal.GetCallback()(sub)

	if !originalCallbackCalled {
		t.Error("Original callback should be restored after modify")
	}
}

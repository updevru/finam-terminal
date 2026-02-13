package ui

import (
	"testing"

	"github.com/rivo/tview"
)

func TestOrderModal_Initialization(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, func(instrument string, quantity float64, buySell string) {}, nil)

	if modal == nil {
		t.Fatal("NewOrderModal returned nil")
	}

	if modal.GetInstrument() != "" {
		t.Errorf("Expected empty instrument, got %s", modal.GetInstrument())
	}
}

func TestOrderModal_SetInstrument(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil, nil)

	modal.SetInstrument("SBER")
	if modal.GetInstrument() != "SBER" {
		t.Errorf("Expected SBER, got %s", modal.GetInstrument())
	}
}

func TestOrderModal_Validation(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil, nil)

	if modal.Validate() {
		t.Error("Expected validation to fail with empty inputs")
	}

	modal.SetInstrument("GAZP")
	// Quantity is 0 by default
	if modal.Validate() {
		t.Error("Expected validation to fail with quantity 0")
	}

	modal.SetQuantity(10)
	if !modal.Validate() {
		t.Error("Expected validation to pass with valid inputs")
	}
}

func TestOrderModal_Direction(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil, nil)

	// Default direction
	if modal.GetDirection() != "Buy" {
		t.Errorf("Expected default direction Buy, got %s", modal.GetDirection())
	}

	// Change selection to Sell (index 1)
	modal.direction.SetCurrentOption(1)

	if modal.GetDirection() != "Sell" {
		t.Errorf("Expected direction Sell after change, got %s", modal.GetDirection())
	}
}

func TestOrderModal_SetLotSize(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil, nil)

	// Default lot size should be 0
	if modal.GetLotSize() != 0 {
		t.Errorf("Expected default lot size 0, got %v", modal.GetLotSize())
	}

	modal.SetLotSize(10)
	if modal.GetLotSize() != 10 {
		t.Errorf("Expected lot size 10, got %v", modal.GetLotSize())
	}

	// Quantity label should include lot size info
	label := modal.quantity.GetLabel()
	if label != "Lots (size - 10): " {
		t.Errorf("Expected quantity label 'Lots (size - 10): ', got '%s'", label)
	}
}

func TestOrderModal_LotBasedCalculation(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil, nil)

	modal.SetLotSize(10)
	modal.SetPrice(250.50)
	modal.SetQuantity(2) // 2 lots

	// Total shares = 2 * 10 = 20
	totalShares := modal.GetTotalShares()
	if totalShares != 20 {
		t.Errorf("Expected total shares 20, got %v", totalShares)
	}

	// Estimated cost = 20 * 250.50 = 5010
	estimatedCost := modal.GetEstimatedCost()
	if estimatedCost != 5010 {
		t.Errorf("Expected estimated cost 5010, got %v", estimatedCost)
	}
}

func TestOrderModal_LotInfoDisplay(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil, nil)

	modal.SetLotSize(10)
	modal.SetPrice(250.50)
	modal.SetQuantity(3)

	// Quantity label should show lot size
	label := modal.quantity.GetLabel()
	if label != "Lots (size - 10): " {
		t.Errorf("Expected quantity label with lot size, got '%s'", label)
	}

	// Info area should show estimated cost
	infoText := modal.infoArea.GetText(true)
	if infoText == "" {
		t.Error("Expected info area to contain estimated cost")
	}
}

func TestOrderModal_DisplayName(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil, nil)

	modal.SetInstrument("SBER")
	modal.SetDisplayName("Сбербанк")

	title := modal.Layout.GetTitle()
	if title != " New Order — Сбербанк " {
		t.Errorf("Expected title ' New Order — Сбербанк ', got '%s'", title)
	}

	// Empty name should keep default title
	modal2 := NewOrderModal(app, nil, nil)
	modal2.SetInstrument("UNKNOWN")
	modal2.SetDisplayName("")

	title2 := modal2.Layout.GetTitle()
	if title2 != " New Order " {
		t.Errorf("Expected default title ' New Order ', got '%s'", title2)
	}
}

func TestOrderModal_QuantityIsInLots(t *testing.T) {
	// Verify the callback receives lot quantity (not shares)
	var receivedQty float64
	app := tview.NewApplication()
	modal := NewOrderModal(app, func(instrument string, quantity float64, buySell string) {
		receivedQty = quantity
	}, nil)

	modal.SetInstrument("SBER")
	modal.SetLotSize(10)
	modal.SetQuantity(5) // 5 lots

	// GetQuantity should return the lot-based quantity (5), not shares (50)
	if modal.GetQuantity() != 5 {
		t.Errorf("Expected GetQuantity to return 5 (lots), got %v", modal.GetQuantity())
	}

	// Simulate clicking Create
	if modal.Validate() {
		modal.callback(modal.GetInstrument(), modal.GetQuantity(), modal.currentDir)
	}

	if receivedQty != 5 {
		t.Errorf("Expected callback to receive 5 (lots), got %v", receivedQty)
	}
}

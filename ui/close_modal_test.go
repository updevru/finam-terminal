package ui

import (
	"testing"

	"github.com/rivo/tview"
)

func TestClosePositionModal_Initialization(t *testing.T) {
	app := tview.NewApplication()
	modal := NewClosePositionModal(app, nil, nil)

	if modal == nil {
		t.Fatal("NewClosePositionModal returned nil")
	}

	if modal.Layout == nil {
		t.Error("Expected Layout to be initialized")
	}
	if modal.Form == nil {
		t.Error("Expected Form to be initialized")
	}
}

func TestClosePositionModal_SetPositionData(t *testing.T) {
	app := tview.NewApplication()
	modal := NewClosePositionModal(app, nil, nil)

	modal.SetPositionData("SBER", 100, 250.5, 500.0)

	if modal.GetSymbol() != "SBER" {
		t.Errorf("Expected symbol SBER, got %s", modal.GetSymbol())
	}
	if modal.GetQuantity() != 100 {
		t.Errorf("Expected quantity 100, got %f", modal.GetQuantity())
	}
}

func TestClosePositionModal_Validation(t *testing.T) {
	app := tview.NewApplication()
	modal := NewClosePositionModal(app, nil, nil)
	
	// Set data: 100 shares
	modal.SetPositionData("SBER", 100, 250.0, 1000.0)
	
	// Empty/Zero
	modal.quantityField.SetText("0")
	if modal.Validate() {
		t.Error("Expected validation to fail for 0 quantity")
	}

	// Negative
	modal.quantityField.SetText("-5")
	if modal.Validate() {
		t.Error("Expected validation to fail for negative quantity")
	}

	// Within limits
	modal.quantityField.SetText("50")
	if !modal.Validate() {
		t.Error("Expected validation to pass for partial close (50/100)")
	}

	// Exact limit
	modal.quantityField.SetText("100")
	if !modal.Validate() {
		t.Error("Expected validation to pass for full close (100/100)")
	}

	// Exceed limit
	modal.quantityField.SetText("101")
	if modal.Validate() {
		t.Error("Expected validation to fail for exceeding max quantity (101/100)")
	}
}

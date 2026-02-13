package ui

import (
	"testing"

	"github.com/rivo/tview"
)

func TestClosePositionModal_Initialization(t *testing.T) {
	app := tview.NewApplication()
	modal := NewClosePositionModal(app, nil, nil, nil)

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
	modal := NewClosePositionModal(app, nil, nil, nil)

	modal.SetPositionData("SBER", 100, 250.5, 500.0)

	if modal.GetSymbol() != "SBER" {
		t.Errorf("Expected symbol SBER, got %s", modal.GetSymbol())
	}
	if modal.GetQuantity() != 0 {
		t.Errorf("Expected quantity 0 (empty field), got %f", modal.GetQuantity())
	}
}

func TestClosePositionModal_Validation(t *testing.T) {
	app := tview.NewApplication()
	modal := NewClosePositionModal(app, nil, nil, nil)

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

func TestClosePositionModal_ValidateBehavior(t *testing.T) {
	// Setup
	modal := NewClosePositionModal(nil, nil, nil, nil)
	modal.SetPositionData("TEST", 100.0, 10.0, 50.0)
	modal.quantityField.SetText("100")

	// Case 1: Default Valid
	if !modal.Validate() {
		t.Error("Position data with 100 quantity should be valid")
	}

	// Case 2: Zero Quantity
	modal.quantityField.SetText("0")
	if modal.Validate() {
		t.Error("Zero quantity should be invalid")
	}

	// Case 3: Negative Quantity
	modal.quantityField.SetText("-10")
	if modal.Validate() {
		t.Error("Negative quantity should be invalid")
	}

	// Case 4: Exceeds Max
	modal.quantityField.SetText("101")
	if modal.Validate() {
		t.Error("Quantity > Max should be invalid")
	}

	// Case 5: Float Quantity (should be valid in logic, even if UI blocks it)
	// We want to verify that the validation logic ITSELF supports floats.
	modal.SetPositionData("TEST", 1.5, 10.0, 50.0)
	modal.quantityField.SetText("1.5")
	if !modal.Validate() {
		t.Error("Float quantity 1.5 should be valid (Max 1.5)")
	}
}

func TestClosePositionModal_LotBasedDisplay(t *testing.T) {
	modal := NewClosePositionModal(nil, nil, nil, nil)

	// SetPositionData with lot size: 100 shares, lot size 10 = 10 lots
	modal.SetPositionDataWithLots("SBER", 100.0, 250.50, 500.0, 10.0)

	// Symbol should be set
	if modal.GetSymbol() != "SBER" {
		t.Errorf("Expected symbol SBER, got %s", modal.GetSymbol())
	}

	// Max quantity should be in lots (100 shares / 10 lot size = 10 lots)
	if modal.maxQuantity != 10 {
		t.Errorf("Expected maxQuantity 10 (lots), got %v", modal.maxQuantity)
	}

	// Lot info should be displayed
	if modal.GetLotSize() != 10 {
		t.Errorf("Expected lot size 10, got %v", modal.GetLotSize())
	}
}

func TestClosePositionModal_LotValidation(t *testing.T) {
	modal := NewClosePositionModal(nil, nil, nil, nil)

	// 100 shares, lot size 10 = max 10 lots
	modal.SetPositionDataWithLots("SBER", 100.0, 250.50, 500.0, 10.0)

	// Valid: 5 lots (within 10 max)
	modal.quantityField.SetText("5")
	if !modal.Validate() {
		t.Error("Expected validation to pass for 5 lots (max 10)")
	}

	// Valid: exact max (10 lots)
	modal.quantityField.SetText("10")
	if !modal.Validate() {
		t.Error("Expected validation to pass for 10 lots (exact max)")
	}

	// Invalid: 11 lots (exceeds max)
	modal.quantityField.SetText("11")
	if modal.Validate() {
		t.Error("Expected validation to fail for 11 lots (max 10)")
	}
}

func TestClosePositionModal_LotInfoText(t *testing.T) {
	modal := NewClosePositionModal(nil, nil, nil, nil)

	modal.SetPositionDataWithLots("SBER", 100.0, 250.50, 500.0, 10.0)

	infoText := modal.infoArea.GetText(true)
	if infoText == "" {
		t.Error("Expected info area to display lot information")
	}
}

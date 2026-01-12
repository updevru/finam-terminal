package ui

import (
	"testing"

	"github.com/rivo/tview"
)

func TestOrderModal_Initialization(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, func(instrument string, quantity float64, buySell string) {})

	if modal == nil {
		t.Fatal("NewOrderModal returned nil")
	}

	if modal.GetInstrument() != "" {
		t.Errorf("Expected empty instrument, got %s", modal.GetInstrument())
	}
}

func TestOrderModal_SetInstrument(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil)

	modal.SetInstrument("SBER")
	if modal.GetInstrument() != "SBER" {
		t.Errorf("Expected SBER, got %s", modal.GetInstrument())
	}
}

func TestOrderModal_Validation(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil)

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
	modal := NewOrderModal(app, nil)

	// Default direction
	if modal.GetDirection() != "Buy" {
		t.Errorf("Expected default direction Buy, got %s", modal.GetDirection())
	}

	modal.ToggleDirection()
	if modal.GetDirection() != "Sell" {
		t.Errorf("Expected direction Sell after toggle, got %s", modal.GetDirection())
	}
}

func TestOrderModal_Validity(t *testing.T) {
	app := tview.NewApplication()
	modal := NewOrderModal(app, nil)

	// Default
	if modal.GetValidity() != "Day" {
		t.Errorf("Expected default validity Day, got %s", modal.GetValidity())
	}

	modal.ToggleValidity()
	if modal.GetValidity() != "GTC" {
		t.Errorf("Expected validity GTC after toggle, got %s", modal.GetValidity())
	}
}

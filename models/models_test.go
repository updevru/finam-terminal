package models

import (
	"testing"
	"time"
)

func TestAccountInfoConsistency(t *testing.T) {
	// This test ensures that AccountInfo has the expected fields
	// and serves as a check for the rename refactoring.
	acc := AccountInfo{
		ID:            "test-id",
		Type:          "test-type",
		Status:        "test-status",
		Equity:        "100.0",
		UnrealizedPnL: "10.0", // Verification of renamed field
		OpenDate:      time.Now(),
	}

	if acc.UnrealizedPnL != "10.0" {
		t.Errorf("Expected UnrealizedPnL to be 10.0, got %s", acc.UnrealizedPnL)
	}
}

func TestPosition_GetCloseDirection(t *testing.T) {
	tests := []struct {
		name     string
		quantity string
		want     string
	}{
		{"Long position", "100", "Sell"},
		{"Short position", "-50", "Buy"},
		{"Zero position", "0", ""},
		{"Invalid quantity", "abc", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Position{Quantity: tt.quantity}
			if got := p.GetCloseDirection(); got != tt.want {
				t.Errorf("Position.GetCloseDirection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecurityInfoStructure(t *testing.T) {
	// Verify SecurityInfo struct exists with correct fields
	sec := SecurityInfo{
		Ticker:   "AAPL",
		Name:     "Apple Inc.",
		Lot:      100,
		Currency: "USD",
	}

	if sec.Ticker != "AAPL" {
		t.Errorf("Expected Ticker AAPL, got %s", sec.Ticker)
	}
	if sec.Name != "Apple Inc." {
		t.Errorf("Expected Name Apple Inc., got %s", sec.Name)
	}
	if sec.Lot != 100 {
		t.Errorf("Expected Lot 100, got %d", sec.Lot)
	}
	if sec.Currency != "USD" {
		t.Errorf("Expected Currency USD, got %s", sec.Currency)
	}
}

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
		t.Errorf("Expected Lot 100, got %f", sec.Lot)
	}
	if sec.Currency != "USD" {
		t.Errorf("Expected Currency USD, got %s", sec.Currency)
	}
}

func TestTradeModel(t *testing.T) {
	tr := Trade{
		ID:       "T1",
		Symbol:   "SBER",
		Side:     "Buy",
		Price:    "250.00",
		Quantity: "10",
		Total:    "2500.00",
		Name:     "Сбербанк",
	}
	if tr.Total != "2500.00" {
		t.Errorf("Expected Total 2500.00, got %s", tr.Total)
	}
	if tr.Name != "Сбербанк" {
		t.Errorf("Expected Name Сбербанк, got %s", tr.Name)
	}
}

func TestOrderModel(t *testing.T) {
	o := Order{
		ID:     "O1",
		Symbol: "GAZP",
		Status: "Active",
		Name:   "Газпром",
	}
	if o.ID != "O1" {
		t.Errorf("Expected ID O1, got %s", o.ID)
	}
	if o.Name != "Газпром" {
		t.Errorf("Expected Name Газпром, got %s", o.Name)
	}
}

func TestOrderModel_ExtendedFields(t *testing.T) {
	o := Order{
		ID:            "O2",
		Symbol:        "SBER@TQBR",
		Name:          "Сбербанк",
		Side:          "Buy",
		Type:          "Stop",
		Status:        "New",
		Quantity:      "100",
		Executed:      "0",
		Price:         "250.00",
		StopCondition: "Last Down",
		LimitPrice:    "248.00",
		StopPrice:     "250.00",
		Validity:      "GTC",
		ExecutedQty:   "0",
		RemainingQty:  "100",
		SLQty:         "50",
		TPQty:         "50",
		SLPrice:       "240.00",
		TPPrice:       "260.00",
	}

	if o.StopCondition != "Last Down" {
		t.Errorf("Expected StopCondition 'Last Down', got '%s'", o.StopCondition)
	}
	if o.LimitPrice != "248.00" {
		t.Errorf("Expected LimitPrice '248.00', got '%s'", o.LimitPrice)
	}
	if o.StopPrice != "250.00" {
		t.Errorf("Expected StopPrice '250.00', got '%s'", o.StopPrice)
	}
	if o.Validity != "GTC" {
		t.Errorf("Expected Validity 'GTC', got '%s'", o.Validity)
	}
	if o.ExecutedQty != "0" {
		t.Errorf("Expected ExecutedQty '0', got '%s'", o.ExecutedQty)
	}
	if o.RemainingQty != "100" {
		t.Errorf("Expected RemainingQty '100', got '%s'", o.RemainingQty)
	}
	if o.SLQty != "50" {
		t.Errorf("Expected SLQty '50', got '%s'", o.SLQty)
	}
	if o.TPQty != "50" {
		t.Errorf("Expected TPQty '50', got '%s'", o.TPQty)
	}
	if o.SLPrice != "240.00" {
		t.Errorf("Expected SLPrice '240.00', got '%s'", o.SLPrice)
	}
	if o.TPPrice != "260.00" {
		t.Errorf("Expected TPPrice '260.00', got '%s'", o.TPPrice)
	}
}

func TestPositionLotSize(t *testing.T) {
	p := Position{
		Symbol:   "SBER",
		LotSize:  10,
		Quantity: "100",
	}

	if p.LotSize != 10 {
		t.Errorf("Expected LotSize 10, got %f", p.LotSize)
	}
}

func TestPositionNameField(t *testing.T) {
	p := Position{
		Symbol: "SBER@MISX",
		Ticker: "SBER",
		Name:   "Сбербанк",
	}
	if p.Name != "Сбербанк" {
		t.Errorf("Expected Name Сбербанк, got %s", p.Name)
	}
}

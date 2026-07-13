package ui

import (
	"finam-terminal/models"
	"testing"
)

// TestHistoryTable_AccruedInterestColumn verifies the combined НКД column:
// a bond trade shows "accrued currency" ("12.34 RUB"), an equity trade is blank,
// and Price/Total columns are untouched (7 columns total).
func TestHistoryTable_AccruedInterestColumn(t *testing.T) {
	mock := &mockClient{}
	mock.GetLotSizeFunc = func(ticker string) float64 { return 1 }

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.history["acc1"] = []models.Trade{
		// Equity trade — no accrued interest
		{ID: "T1", Symbol: "SBER@TQBR", Name: "Сбербанк", Side: "Buy", Price: "250.00", Quantity: "10", Total: "2500.00"},
		// Bond trade — accrued interest + currency
		{ID: "T2", Symbol: "SU26238@TQOB", Name: "ОФЗ 26238", Side: "Buy", Price: "650.10", Quantity: "3", Total: "1950.30", AccruedInterest: "12.34", Currency: "RUB"},
	}

	updateHistoryTable(app)

	table := app.portfolioView.TabbedView.HistoryTable

	// Header: 7 columns with НКД present, and its index resolved dynamically.
	expectedHeaders := []string{"Instrument", "Side", "Price", "Qty (Lots)", "Total", "НКД", "Time"}
	if got := table.GetColumnCount(); got != len(expectedHeaders) {
		t.Fatalf("expected %d columns, got %d", len(expectedHeaders), got)
	}
	nkdCol := -1
	for i, h := range expectedHeaders {
		cell := table.GetCell(0, i)
		if cell.Text != h {
			t.Errorf("Header[%d]: expected %q, got %q", i, h, cell.Text)
		}
		if h == "НКД" {
			nkdCol = i
		}
	}
	if nkdCol < 0 {
		t.Fatal("НКД column not found in header")
	}

	// Equity row → НКД blank
	if cell := table.GetCell(1, nkdCol); cell.Text != "" {
		t.Errorf("equity НКД: expected blank, got %q", cell.Text)
	}
	// Bond row → "12.34 RUB"
	if cell := table.GetCell(2, nkdCol); cell.Text != "12.34 RUB" {
		t.Errorf("bond НКД: expected %q, got %q", "12.34 RUB", cell.Text)
	}

	// Price/Total columns untouched
	if cell := table.GetCell(2, 2); cell.Text != "650.10" {
		t.Errorf("bond Price: expected 650.10, got %q", cell.Text)
	}
	if cell := table.GetCell(2, 4); cell.Text != "1950.30" {
		t.Errorf("bond Total: expected 1950.30, got %q", cell.Text)
	}
}

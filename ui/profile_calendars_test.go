package ui

import (
	"strings"
	"testing"

	"finam-terminal/models"

	"github.com/rivo/tview"
)

// equityProfile builds a minimal equity profile (no contract/strike/bond markers).
func equityProfile() *models.InstrumentProfile {
	return &models.InstrumentProfile{
		Symbol: "SBER@TQBR",
		Details: &models.AssetDetails{
			Name:          "Сбербанк",
			Type:          "Stock",
			Board:         "TQBR",
			QuoteCurrency: "RUB",
			LotSize:       "10",
		},
	}
}

func TestProfilePanel_EquityDividendsAndSplits(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	profile := equityProfile()
	profile.Dividends = []models.Dividend{
		{Date: "2025-12-15", Amount: "10.0", Currency: "RUB", IsFuture: false},
		{Date: "2026-03-15", Amount: "15.5", Currency: "RUB", IsFuture: true},
		{Date: "2026-06-15", Amount: "16.0", Currency: "RUB", IsFuture: true},
		{Date: "2026-09-15", Amount: "17.0", Currency: "RUB", IsFuture: true},
		{Date: "2026-12-15", Amount: "18.0", Currency: "RUB", IsFuture: true}, // 4th future → dropped
		{Date: "2027-03-15", Amount: "19.0", Currency: "RUB", IsFuture: true}, // 5th future → dropped
	}
	profile.Splits = []models.Split{
		{Date: "2025-06-01", OldRatio: "1", NewRatio: "10", NewLot: "1", ConvType: "ORDINARY", IsFuture: false},
		{Date: "2026-08-01", OldRatio: "2", NewRatio: "1", NewLot: "", ConvType: "TENDER_OFFER", IsFuture: true},
	}

	panel.Update(profile)
	text := panel.InfoPanel.GetText(false)

	// Dividends section present
	if !strings.Contains(text, "Dividends") {
		t.Error("expected Dividends section header")
	}
	// Nearest 3 future dividends shown
	for _, d := range []string{"2026-03-15", "2026-06-15", "2026-09-15"} {
		if !strings.Contains(text, d) {
			t.Errorf("expected dividend row for %s", d)
		}
	}
	// Past dividend shown
	if !strings.Contains(text, "2025-12-15") {
		t.Error("expected past dividend row for 2025-12-15")
	}
	// 4th/5th future dividends capped away
	if strings.Contains(text, "2027-03-15") {
		t.Error("far future dividend 2027-03-15 should be capped away")
	}
	// Overflow hint present
	if !strings.Contains(text, "…") {
		t.Error("expected '…' overflow hint for capped dividends")
	}

	// Splits section present with coefficient and future row
	if !strings.Contains(text, "Splits") {
		t.Error("expected Splits section header")
	}
	if !strings.Contains(text, "1→10") {
		t.Error("expected split coefficient 1→10")
	}
	if !strings.Contains(text, "2026-08-01") {
		t.Error("expected future split row for 2026-08-01")
	}
}

// A futures instrument must not render equity calendars even if slices are set.
func TestProfilePanel_FuturesNoEquityCalendars(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	profile := &models.InstrumentProfile{
		Symbol: "SiH6@RTSX",
		Details: &models.AssetDetails{
			Name:         "Si-3.26",
			Type:         "Futures",
			ContractSize: "1000",
		},
		Dividends: []models.Dividend{{Date: "2026-03-15", Amount: "15.5", Currency: "RUB", IsFuture: true}},
	}

	panel.Update(profile)
	text := panel.InfoPanel.GetText(false)

	if strings.Contains(text, "Dividends") {
		t.Error("futures profile must not render a Dividends section")
	}
}

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

func TestProfilePanel_BondEventsSections(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	profile := &models.InstrumentProfile{
		Symbol: "SU26238@TQOB",
		Details: &models.AssetDetails{
			Name:             "ОФЗ 26238",
			Type:             "Bond",
			BondFaceValue:    "1000",
			BondFaceCurrency: "RUB",
		},
		BondEvents: []models.BondEvent{
			{Date: "2026-01-20", Kind: models.BondEventCoupon, Value: "34.9", Currency: "RUB",
				RecordDate: "2026-01-18", Percent: "6.98", IsFuture: false},
			{Date: "2026-10-20", Kind: models.BondEventAmortization,
				NewFaceValue: "800", InitialFaceValue: "1000", Percent: "20", IsFuture: true},
			{Date: "2026-11-15", Kind: models.BondEventOffer,
				Type: "PUT", Price: "100", Start: "2026-11-10", End: "2026-11-14", IsFuture: true},
		},
	}

	panel.Update(profile)
	text := panel.InfoPanel.GetText(false)

	// Coupons: rate % and record date
	if !strings.Contains(text, "Coupons") {
		t.Error("expected Coupons section header")
	}
	if !strings.Contains(text, "6.98%") {
		t.Error("expected coupon rate 6.98%")
	}
	if !strings.Contains(text, "2026-01-18") {
		t.Error("expected coupon record date 2026-01-18")
	}

	// Amortization: percent and new face value
	if !strings.Contains(text, "Amortization") {
		t.Error("expected Amortization section header")
	}
	if !strings.Contains(text, "20%") {
		t.Error("expected amortization percent 20%")
	}
	if !strings.Contains(text, "800") {
		t.Error("expected amortization new face value 800")
	}

	// Offers: type, price, date window
	if !strings.Contains(text, "Offers") {
		t.Error("expected Offers section header")
	}
	if !strings.Contains(text, "PUT") {
		t.Error("expected offer type PUT")
	}
	if !strings.Contains(text, "100") {
		t.Error("expected offer price 100")
	}
	if !strings.Contains(text, "2026-11-10") || !strings.Contains(text, "2026-11-14") {
		t.Error("expected offer date window 2026-11-10..2026-11-14")
	}
}

// A bond must not render equity Dividends/Splits sections.
func TestProfilePanel_BondNoEquityCalendars(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	profile := &models.InstrumentProfile{
		Symbol: "SU26238@TQOB",
		Details: &models.AssetDetails{
			Name:          "ОФЗ 26238",
			BondFaceValue: "1000",
		},
		Dividends: []models.Dividend{{Date: "2026-03-15", Amount: "15.5", IsFuture: true}},
	}

	panel.Update(profile)
	text := panel.InfoPanel.GetText(false)

	if strings.Contains(text, "Dividends") {
		t.Error("bond profile must not render a Dividends section")
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

package ui

import (
	"strings"
	"testing"
	"time"

	"finam-terminal/models"

	"github.com/rivo/tview"
)

func TestNewProfilePanel(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	if panel == nil {
		t.Fatal("NewProfilePanel returned nil")
	}
	if panel.Layout == nil {
		t.Error("Layout is nil")
	}
	if panel.InfoPanel == nil {
		t.Error("InfoPanel is nil")
	}
	if panel.ChartView == nil {
		t.Error("ChartView is nil")
	}
	if panel.Footer == nil {
		t.Error("Footer is nil")
	}
	if panel.GetTimeframe() != 2 {
		t.Errorf("Expected default timeframe 2 (Daily), got %d", panel.GetTimeframe())
	}
}

func TestProfilePanel_Update(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	profile := &models.InstrumentProfile{
		Symbol: "SBER@TQBR",
		Details: &models.AssetDetails{
			Name:          "Сбербанк",
			Type:          "Stock",
			Board:         "TQBR",
			ISIN:          "RU0009029540",
			QuoteCurrency: "RUB",
			LotSize:       "10",
			Decimals:      2,
			MinStep:       1,
		},
		Quote: &models.Quote{
			Last:    "280.50",
			Bid:     "280.40",
			BidSize: "100",
			Ask:     "280.60",
			AskSize: "50",
			Volume:  "1000000",
			Open:    "278.00",
			High:    "282.00",
			Low:     "277.50",
			Close:   "279.00",
		},
		Params: &models.AssetParams{
			IsTradable: true,
			Longable:   "Available",
			Shortable:  "Not Available",
		},
		Schedule: []models.TradingSession{
			{
				Type:      "MAIN",
				StartTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local),
				EndTime:   time.Date(2024, 1, 15, 18, 45, 0, 0, time.Local),
			},
		},
	}

	panel.Update(profile)

	text := panel.InfoPanel.GetText(false)

	if !strings.Contains(text, "Сбербанк") {
		t.Error("InfoPanel should contain instrument name")
	}
	if !strings.Contains(text, "SBER@TQBR") {
		t.Error("InfoPanel should contain symbol")
	}
	if !strings.Contains(text, "Stock") {
		t.Error("InfoPanel should contain instrument type")
	}
	if !strings.Contains(text, "280.50") {
		t.Error("InfoPanel should contain last price")
	}
	if !strings.Contains(text, "MAIN") {
		t.Error("InfoPanel should contain schedule session type")
	}
}

func TestProfilePanel_UpdateNilProfile(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	panel.Update(nil)

	text := panel.InfoPanel.GetText(false)
	if !strings.Contains(text, "Loading") {
		t.Error("Nil profile should show Loading message")
	}
}

func TestProfilePanel_SetTimeframe(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	panel.SetTimeframe(0)
	if panel.GetTimeframe() != 0 {
		t.Errorf("Expected timeframe 0, got %d", panel.GetTimeframe())
	}

	panel.SetTimeframe(3)
	if panel.GetTimeframe() != 3 {
		t.Errorf("Expected timeframe 3, got %d", panel.GetTimeframe())
	}
}

func TestProfilePanel_EmptySchedule(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	profile := &models.InstrumentProfile{
		Symbol:   "TEST@XNAS",
		Schedule: nil,
	}

	panel.Update(profile)

	text := panel.InfoPanel.GetText(false)
	if !strings.Contains(text, "Schedule unavailable") {
		t.Error("Empty schedule should show 'Schedule unavailable'")
	}
}

func TestProfilePanel_FooterText(t *testing.T) {
	app := tview.NewApplication()
	panel := NewProfilePanel(app)

	text := panel.Footer.GetText(false)
	if !strings.Contains(text, "ESC") {
		t.Error("Footer should contain ESC hint")
	}
	if !strings.Contains(text, "M5") {
		t.Error("Footer should contain M5 timeframe hint")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a very long string", 10, "this is..."},
		{"ab", 2, "ab"},
		{"abc", 2, "ab"},
	}

	for _, tc := range tests {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
		}
	}
}

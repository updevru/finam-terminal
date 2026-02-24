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
	if !strings.Contains(text, "Main") {
		t.Error("InfoPanel should contain schedule session display name")
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
		{"Сбербанк", 10, "Сбербанк"},            // 8 runes, fits in 10
		{"Сбербанк России", 10, "Сбербан..."},    // 15 runes, truncated to 10
		{"Привет", 6, "Привет"},                  // exact fit
		{"Привет мир", 6, "При..."},              // 10 runes, truncated
	}

	for _, tc := range tests {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.maxLen, got, tc.want)
		}
	}
}

func TestActiveSessions(t *testing.T) {
	base := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)

	t.Run("derives trading windows from CLOSED gaps", func(t *testing.T) {
		sessions := []models.TradingSession{
			{Type: "CLOSED", StartTime: base, EndTime: base.Add(6*time.Hour + 50*time.Minute)},
			{Type: "CLOSED", StartTime: base.Add(9*time.Hour + 49*time.Minute), EndTime: base.Add(9*time.Hour + 50*time.Minute)},
			{Type: "CLOSED", StartTime: base.Add(18*time.Hour + 39*time.Minute), EndTime: base.Add(18*time.Hour + 40*time.Minute)},
			{Type: "CLOSED", StartTime: base.Add(18*time.Hour + 50*time.Minute), EndTime: base.Add(19 * time.Hour)},
		}

		result := activeSessions(sessions)

		if len(result) != 3 {
			t.Fatalf("expected 3 trading windows, got %d", len(result))
		}
		if result[0].StartTime != base.Add(6*time.Hour+50*time.Minute) || result[0].EndTime != base.Add(9*time.Hour+49*time.Minute) {
			t.Errorf("window 0: got %s-%s, want 06:50-09:49", result[0].StartTime.Format("15:04"), result[0].EndTime.Format("15:04"))
		}
		if result[1].StartTime != base.Add(9*time.Hour+50*time.Minute) || result[1].EndTime != base.Add(18*time.Hour+39*time.Minute) {
			t.Errorf("window 1: got %s-%s, want 09:50-18:39", result[1].StartTime.Format("15:04"), result[1].EndTime.Format("15:04"))
		}
		if result[2].StartTime != base.Add(18*time.Hour+40*time.Minute) || result[2].EndTime != base.Add(18*time.Hour+50*time.Minute) {
			t.Errorf("window 2: got %s-%s, want 18:40-18:50", result[2].StartTime.Format("15:04"), result[2].EndTime.Format("15:04"))
		}
	})

	t.Run("deduplicates non-CLOSED sessions", func(t *testing.T) {
		sessions := []models.TradingSession{
			{Type: "EARLY_TRADING", StartTime: base.Add(7 * time.Hour), EndTime: base.Add(9*time.Hour + 49*time.Minute)},
			{Type: "EARLY_TRADING", StartTime: base.Add(7 * time.Hour), EndTime: base.Add(9*time.Hour + 49*time.Minute)},
			{Type: "EARLY_TRADING", StartTime: base.Add(7 * time.Hour), EndTime: base.Add(9*time.Hour + 49*time.Minute)},
			{Type: "CORE_TRADING", StartTime: base.Add(10 * time.Hour), EndTime: base.Add(18*time.Hour + 39*time.Minute)},
			{Type: "CLOSED", StartTime: base, EndTime: base.Add(7 * time.Hour)},
		}
		result := activeSessions(sessions)
		if len(result) != 2 {
			t.Fatalf("expected 2 unique sessions, got %d", len(result))
		}
		if result[0].Type != "EARLY_TRADING" {
			t.Errorf("expected EARLY_TRADING, got %s", result[0].Type)
		}
		if result[1].Type != "CORE_TRADING" {
			t.Errorf("expected CORE_TRADING, got %s", result[1].Type)
		}
	})

	t.Run("deduplicates same type with different times", func(t *testing.T) {
		sessions := []models.TradingSession{
			{Type: "OPENING_AUCTION", StartTime: base.Add(6*time.Hour + 50*time.Minute), EndTime: base.Add(6*time.Hour + 59*time.Minute)},
			{Type: "OPENING_AUCTION", StartTime: base.Add(6*time.Hour + 59*time.Minute), EndTime: base.Add(7 * time.Hour)},
			{Type: "OPENING_AUCTION", StartTime: base.Add(9*time.Hour + 50*time.Minute), EndTime: base.Add(9*time.Hour + 59*time.Minute)},
			{Type: "EARLY_TRADING", StartTime: base.Add(7 * time.Hour), EndTime: base.Add(9*time.Hour + 49*time.Minute)},
		}
		result := activeSessions(sessions)
		if len(result) != 2 {
			t.Fatalf("expected 2 unique sessions (one per type), got %d", len(result))
		}
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		result := activeSessions(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("handles single CLOSED session", func(t *testing.T) {
		sessions := []models.TradingSession{
			{Type: "CLOSED", StartTime: base, EndTime: base.Add(6 * time.Hour)},
		}
		result := activeSessions(sessions)
		if len(result) != 0 {
			t.Errorf("expected 0 windows from single CLOSED, got %d", len(result))
		}
	})
}

func TestSessionDisplayName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"EARLY_TRADING", "Early"},
		{"CORE_TRADING", "Main"},
		{"LATE_TRADING", "Late"},
		{"AFTER_TRADING", "After-hours"},
		{"OPENING_AUCTION", "Opening"},
		{"CLOSING_AUCTION", "Closing"},
		{"MAIN", "Main"},
		{"CLOSED", "Closed"},
		{"UNKNOWN_TYPE", "UNKNOWN_TYPE"},
	}
	for _, tc := range tests {
		got := sessionDisplayName(tc.input)
		if got != tc.want {
			t.Errorf("sessionDisplayName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

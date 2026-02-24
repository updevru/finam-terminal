package ui

import (
	"fmt"
	"sort"
	"strings"

	"finam-terminal/models"

	"github.com/rivo/tview"
)

// ProfilePanel is the full-screen instrument profile overlay component.
type ProfilePanel struct {
	Layout    *tview.Flex
	InfoPanel *tview.TextView
	ChartView *tview.TextView
	Footer    *tview.TextView

	app       *tview.Application
	profile   *models.InstrumentProfile
	timeframe int // 0=M5, 1=H1, 2=D, 3=W
}

// GetProfile returns the current instrument profile (may be nil).
func (p *ProfilePanel) GetProfile() *models.InstrumentProfile {
	return p.profile
}

// NewProfilePanel creates a new ProfilePanel with the standard layout.
func NewProfilePanel(app *tview.Application) *ProfilePanel {
	p := &ProfilePanel{
		app:       app,
		timeframe: 2, // Default: Daily
	}

	p.InfoPanel = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true)
	p.InfoPanel.SetBorder(true).SetTitle(" Details ")

	p.ChartView = tview.NewTextView().
		SetDynamicColors(true)
	p.ChartView.SetBorder(true).SetTitle(" Chart ")

	p.Footer = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	p.Footer.SetText(profileFooterText)

	// Horizontal: InfoPanel (42 cols fixed) + ChartView (flex)
	contentRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(p.InfoPanel, 42, 0, false).
		AddItem(p.ChartView, 0, 1, false)

	// Vertical: content + footer
	p.Layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(contentRow, 0, 1, false).
		AddItem(p.Footer, 1, 0, false)

	return p
}

const profileFooterText = "[yellow]1[white] M5  [yellow]2[white] H1  [yellow]3[white] D  [yellow]4[white] W  │  [yellow]A[white] Order  [yellow]R[white] Refresh  [yellow]ESC[white] Back"

// RestoreFooter resets the footer to the default hint text.
func (p *ProfilePanel) RestoreFooter() {
	p.Footer.SetText(profileFooterText)
}

// Update performs a full refresh of both the info panel and chart.
func (p *ProfilePanel) Update(profile *models.InstrumentProfile) {
	p.profile = profile
	p.renderInfoPanel()
	p.renderChart()
}

// UpdateChart refreshes only the chart (used for timeframe switches).
func (p *ProfilePanel) UpdateChart(bars []models.Bar) {
	if p.profile != nil {
		p.profile.Bars = bars
	}
	p.renderChart()
}

// SetTimeframe sets the current timeframe index (0=M5, 1=H1, 2=D, 3=W).
func (p *ProfilePanel) SetTimeframe(idx int) {
	p.timeframe = idx
}

// GetTimeframe returns the current timeframe index.
func (p *ProfilePanel) GetTimeframe() int {
	return p.timeframe
}

// renderInfoPanel renders the left info panel with instrument details.
func (p *ProfilePanel) renderInfoPanel() {
	if p.profile == nil {
		p.InfoPanel.SetText("[gray]Loading...")
		return
	}

	var sb strings.Builder

	// Title
	symbol := p.profile.Symbol
	if p.profile.Details != nil && p.profile.Details.Name != "" {
		fmt.Fprintf(&sb, "[yellow::b]%s[-:-:-]\n", truncate(p.profile.Details.Name, 38))
		fmt.Fprintf(&sb, "[white]%s\n", symbol)
	} else {
		fmt.Fprintf(&sb, "[yellow::b]%s[-:-:-]\n", symbol)
	}
	sb.WriteString("\n")

	// Details section
	if d := p.profile.Details; d != nil {
		sb.WriteString("[cyan::b]─── Details ───[-:-:-]\n")
		writeField(&sb, "Type", d.Type)
		writeField(&sb, "Board", d.Board)
		writeField(&sb, "ISIN", d.ISIN)
		writeField(&sb, "Currency", d.QuoteCurrency)
		writeField(&sb, "Lot Size", d.LotSize)
		writeField(&sb, "Decimals", fmt.Sprintf("%d", d.Decimals))
		writeField(&sb, "Min Step", fmt.Sprintf("%d", d.MinStep))
		if d.ExpirationDate != "" {
			writeField(&sb, "Expiry", d.ExpirationDate)
		}
		sb.WriteString("\n")
	}

	// Quote section
	if q := p.profile.Quote; q != nil {
		sb.WriteString("[cyan::b]─── Quote ───[-:-:-]\n")
		writeField(&sb, "Last", q.Last)
		writeField(&sb, "Bid", fmt.Sprintf("%s (%s)", q.Bid, q.BidSize))
		writeField(&sb, "Ask", fmt.Sprintf("%s (%s)", q.Ask, q.AskSize))
		writeField(&sb, "Volume", q.Volume)
		writeField(&sb, "Open", q.Open)
		writeField(&sb, "High", q.High)
		writeField(&sb, "Low", q.Low)
		writeField(&sb, "Close", q.Close)
		sb.WriteString("\n")
	}

	// Trading params section
	if t := p.profile.Params; t != nil {
		sb.WriteString("[cyan::b]─── Trading ───[-:-:-]\n")
		tradable := "[red]No[-]"
		if t.IsTradable {
			tradable = "[green]Yes[-]"
		}
		writeField(&sb, "Tradable", tradable)
		writeField(&sb, "Long", t.Longable)
		writeField(&sb, "Short", t.Shortable)
		if t.LongRiskRate != "" && t.LongRiskRate != "N/A" {
			writeField(&sb, "Long Risk", t.LongRiskRate)
		}
		if t.ShortRiskRate != "" && t.ShortRiskRate != "N/A" {
			writeField(&sb, "Short Risk", t.ShortRiskRate)
		}
		if t.LongInitialMargin != "" {
			writeField(&sb, "Long Margin", t.LongInitialMargin)
		}
		if t.ShortInitialMargin != "" {
			writeField(&sb, "Short Margin", t.ShortInitialMargin)
		}
		sb.WriteString("\n")
	}

	// Schedule section
	if len(p.profile.Schedule) > 0 {
		sb.WriteString("[cyan::b]─── Schedule ───[-:-:-]\n")
		tradingSessions := activeSessions(p.profile.Schedule)
		for _, s := range tradingSessions {
			start := s.StartTime.Format("15:04")
			end := s.EndTime.Format("15:04")
			fmt.Fprintf(&sb, " [green]%-14s [white]%s - %s\n", sessionDisplayName(s.Type), start, end)
		}
	} else {
		sb.WriteString("[gray]Schedule unavailable\n")
	}

	p.InfoPanel.SetText(sb.String())
}

// renderChart renders the candlestick chart in the ChartView.
func (p *ProfilePanel) renderChart() {
	if p.profile == nil || len(p.profile.Bars) == 0 {
		p.ChartView.SetText("\n\n\n          [gray]No data[-]")
		return
	}

	// Get available dimensions from the ChartView
	_, _, width, height := p.ChartView.GetInnerRect()
	if width <= 0 || height <= 0 {
		// Fallback dimensions if not yet drawn
		width = 60
		height = 20
	}

	chart := RenderCandlestickChart(p.profile.Bars, width, height)
	p.ChartView.SetText(chart)
}

// writeField writes a label-value pair to the string builder.
func writeField(sb *strings.Builder, label, value string) {
	if value == "" {
		value = "N/A"
	}
	fmt.Fprintf(sb, " [white]%-12s [lightgray]%s\n", label, value)
}

// sessionDisplayNames maps API session type constants to human-readable names.
var sessionDisplayNames = map[string]string{
	"EARLY_TRADING":   "Early",
	"CORE_TRADING":    "Main",
	"LATE_TRADING":    "Late",
	"AFTER_TRADING":   "After-hours",
	"OPENING_AUCTION": "Opening",
	"CLOSING_AUCTION": "Closing",
	"EVENING":         "Evening",
	"MORNING":         "Morning",
	"MAIN":            "Main",
	"CLOSED":          "Closed",
}

// sessionDisplayName returns a human-readable name for a session type constant.
func sessionDisplayName(raw string) string {
	if name, ok := sessionDisplayNames[raw]; ok {
		return name
	}
	return raw
}

// activeSessions filters and deduplicates schedule sessions.
// If non-CLOSED sessions exist, returns unique ones.
// Otherwise, derives trading windows from CLOSED gaps.
func activeSessions(sessions []models.TradingSession) []models.TradingSession {
	var closed []models.TradingSession
	var active []models.TradingSession
	for _, s := range sessions {
		if strings.EqualFold(s.Type, "CLOSED") {
			closed = append(closed, s)
		} else {
			active = append(active, s)
		}
	}

	if len(active) > 0 {
		result := dedup(active)
		sort.Slice(result, func(i, j int) bool {
			return result[i].StartTime.Format("15:04") < result[j].StartTime.Format("15:04")
		})
		return result
	}

	if len(closed) == 0 {
		return nil
	}

	// Sort CLOSED sessions by start time
	sort.Slice(closed, func(i, j int) bool {
		return closed[i].StartTime.Before(closed[j].StartTime)
	})

	// Derive trading windows from gaps between consecutive CLOSED periods
	var result []models.TradingSession
	for i := 0; i < len(closed)-1; i++ {
		gapStart := closed[i].EndTime
		gapEnd := closed[i+1].StartTime
		if !gapEnd.After(gapStart) {
			continue
		}
		result = append(result, models.TradingSession{
			Type:      "Trading",
			StartTime: gapStart,
			EndTime:   gapEnd,
		})
	}

	return result
}

// dedup keeps only one entry per session type (the first occurrence).
func dedup(sessions []models.TradingSession) []models.TradingSession {
	seen := make(map[string]bool)
	var result []models.TradingSession
	for _, s := range sessions {
		if seen[s.Type] {
			continue
		}
		seen[s.Type] = true
		result = append(result, s)
	}
	return result
}

// truncate truncates a string to maxLen runes with ellipsis.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

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

// Instrument-type classification from AssetDetails markers. A bond has a face
// value; an equity has none of the derivative/bond markers (no contract size,
// strike, or face value).
func isBondDetails(d *models.AssetDetails) bool {
	return d != nil && d.BondFaceValue != ""
}

func isEquityDetails(d *models.AssetDetails) bool {
	return d != nil && d.ContractSize == "" && d.Strike == "" && d.BondFaceValue == ""
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

		// Instrument-type-specific section
		if d.ContractSize != "" && d.Strike != "" {
			// Options
			sb.WriteString("[cyan::b]─── Options ───[-:-:-]\n")
			writeField(&sb, "Contract", d.ContractSize)
			writeField(&sb, "Strike", d.Strike)
			sb.WriteString("\n")
		} else if d.ContractSize != "" {
			// Futures
			sb.WriteString("[cyan::b]─── Futures ───[-:-:-]\n")
			writeField(&sb, "Contract", d.ContractSize)
			sb.WriteString("\n")
		} else if d.BondFaceValue != "" {
			// Bonds
			sb.WriteString("[cyan::b]─── Bond ───[-:-:-]\n")
			faceVal := d.BondFaceValue
			if d.BondFaceCurrency != "" {
				faceVal += " " + d.BondFaceCurrency
			}
			writeField(&sb, "Face Value", faceVal)
			sb.WriteString("\n")

			// Bond corporate-action calendar (coupons/amortization/offers).
			p.renderBondEvents(&sb)
		}

		// Corporate-action calendars: dividends and splits for equities.
		if isEquityDetails(d) {
			p.renderDividends(&sb)
			p.renderSplits(&sb)
		}
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
		if q.OpenInterest != "" && q.OpenInterest != "0" {
			writeField(&sb, "Open Int.", q.OpenInterest)
		}
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

// calendarCap bounds how many past and future entries are shown per calendar
// section in the compact profile panel.
const calendarCap = 3

// capCalendar splits items (assumed sorted ascending by date) into past and
// future by isFuture, keeps at most calendarCap of the nearest entries on each
// side, and reports whether older/newer entries were hidden. The returned slice
// is [past…, future…] in chronological order.
func capCalendar[T any](items []T, isFuture func(T) bool) (rows []T, morePast, moreFuture bool) {
	var past, future []T
	for _, it := range items {
		if isFuture(it) {
			future = append(future, it)
		} else {
			past = append(past, it)
		}
	}
	if len(past) > calendarCap {
		past = past[len(past)-calendarCap:]
		morePast = true
	}
	if len(future) > calendarCap {
		future = future[:calendarCap]
		moreFuture = true
	}
	rows = append(append(rows, past...), future...)
	return rows, morePast, moreFuture
}

// renderDividends renders the compact Dividends section for an equity.
func (p *ProfilePanel) renderDividends(sb *strings.Builder) {
	divs := p.profile.Dividends
	if len(divs) == 0 {
		return
	}
	rows, morePast, moreFuture := capCalendar(divs, func(d models.Dividend) bool { return d.IsFuture })

	sb.WriteString("[cyan::b]─── Dividends ───[-:-:-]\n")
	if morePast {
		sb.WriteString(" [gray]…[-]\n")
	}
	for _, d := range rows {
		val := d.Amount
		if d.Currency != "" {
			val += " " + d.Currency
		}
		fmt.Fprintf(sb, " [white]%-11s [lightgray]%s\n", d.Date, val)
	}
	if moreFuture {
		sb.WriteString(" [gray]…[-]\n")
	}
	sb.WriteString("\n")
}

// renderSplits renders the compact Splits section for an equity.
func (p *ProfilePanel) renderSplits(sb *strings.Builder) {
	splits := p.profile.Splits
	if len(splits) == 0 {
		return
	}
	rows, morePast, moreFuture := capCalendar(splits, func(s models.Split) bool { return s.IsFuture })

	sb.WriteString("[cyan::b]─── Splits ───[-:-:-]\n")
	if morePast {
		sb.WriteString(" [gray]…[-]\n")
	}
	for _, s := range rows {
		val := s.OldRatio + "→" + s.NewRatio
		if s.NewLot != "" {
			val += "  lot " + s.NewLot
		}
		fmt.Fprintf(sb, " [white]%-11s [lightgray]%s\n", s.Date, val)
	}
	if moreFuture {
		sb.WriteString(" [gray]…[-]\n")
	}
	sb.WriteString("\n")
}

// renderBondEvents renders the bond corporate-action calendar, grouped into
// Coupons / Amortization / Offers sections by Kind. Each section is capped and
// hinted like the equity calendars.
func (p *ProfilePanel) renderBondEvents(sb *strings.Builder) {
	events := p.profile.BondEvents
	if len(events) == 0 {
		return
	}
	var coupons, amorts, offers []models.BondEvent
	for _, e := range events {
		switch e.Kind {
		case models.BondEventCoupon:
			coupons = append(coupons, e)
		case models.BondEventAmortization:
			amorts = append(amorts, e)
		case models.BondEventOffer:
			offers = append(offers, e)
		}
	}
	renderBondSection(sb, "Coupons", coupons, couponRow)
	renderBondSection(sb, "Amortization", amorts, amortizationRow)
	renderBondSection(sb, "Offers", offers, offerRow)
}

// renderBondSection renders one capped bond-event section using the given
// per-event (date, details) formatter.
func renderBondSection(sb *strings.Builder, title string, items []models.BondEvent, row func(models.BondEvent) (string, string)) {
	if len(items) == 0 {
		return
	}
	rows, morePast, moreFuture := capCalendar(items, func(e models.BondEvent) bool { return e.IsFuture })
	fmt.Fprintf(sb, "[cyan::b]─── %s ───[-:-:-]\n", title)
	if morePast {
		sb.WriteString(" [gray]…[-]\n")
	}
	for _, e := range rows {
		col1, col2 := row(e)
		fmt.Fprintf(sb, " [white]%-11s [lightgray]%s\n", col1, col2)
	}
	if moreFuture {
		sb.WriteString(" [gray]…[-]\n")
	}
	sb.WriteString("\n")
}

// couponRow: payment date + rate % + record date.
func couponRow(e models.BondEvent) (string, string) {
	var details string
	if e.Percent != "" {
		details += e.Percent + "%"
	}
	if e.RecordDate != "" {
		if details != "" {
			details += "  "
		}
		details += "rec " + e.RecordDate
	}
	return e.Date, details
}

// amortizationRow: date + percent + new face value.
func amortizationRow(e models.BondEvent) (string, string) {
	var details string
	if e.Percent != "" {
		details += e.Percent + "%"
	}
	if e.NewFaceValue != "" {
		if details != "" {
			details += "  "
		}
		details += "→ " + e.NewFaceValue
	}
	return e.Date, details
}

// offerRow: date window (Start…End) + type + price.
func offerRow(e models.BondEvent) (string, string) {
	window := e.Date
	if e.Start != "" && e.End != "" {
		window = e.Start + "…" + e.End
	}
	var details string
	if e.Type != "" {
		details += e.Type
	}
	if e.Price != "" {
		if details != "" {
			details += "  "
		}
		details += e.Price
	}
	return window, details
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

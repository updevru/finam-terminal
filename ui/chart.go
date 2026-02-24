package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"finam-terminal/models"
)

// RenderCandlestickChart renders a Unicode candlestick chart with tview color tags.
// It is a pure function: given bars and dimensions, it returns a tview-tagged string.
func RenderCandlestickChart(bars []models.Bar, width, height int) string {
	if len(bars) == 0 {
		return centerText("No data", width, height)
	}

	const gutterWidth = 9 // left Y-axis gutter (8 chars + 1 separator)
	chartWidth := width - gutterWidth
	if chartWidth < 2 {
		chartWidth = 2
	}

	// Each candle takes 2 columns (1 char candle + 1 space)
	maxCandles := chartWidth / 2
	if maxCandles < 1 {
		maxCandles = 1
	}

	// Reserve 2 rows for X-axis: separator line (└───) + labels row
	chartHeight := height - 2
	if chartHeight < 3 {
		chartHeight = 3
	}

	// Take only the last N bars that fit
	visibleBars := bars
	if len(visibleBars) > maxCandles {
		visibleBars = visibleBars[len(visibleBars)-maxCandles:]
	}

	// Find price range
	minPrice := math.MaxFloat64
	maxPrice := -math.MaxFloat64
	for _, b := range visibleBars {
		if b.Low < minPrice {
			minPrice = b.Low
		}
		if b.High > maxPrice {
			maxPrice = b.High
		}
	}

	// Add small padding to price range
	priceRange := maxPrice - minPrice
	if priceRange == 0 {
		priceRange = 1
		minPrice -= 0.5
		maxPrice += 0.5
	}
	padding := priceRange * 0.05
	minPrice -= padding
	maxPrice += padding
	priceRange = maxPrice - minPrice

	// Build the chart grid row by row
	var sb strings.Builder

	for row := 0; row < chartHeight; row++ {
		// Price at this row (top = maxPrice, bottom = minPrice)
		rowPriceHigh := maxPrice - (float64(row)/float64(chartHeight))*priceRange
		rowPriceLow := maxPrice - (float64(row+1)/float64(chartHeight))*priceRange

		// Y-axis label (every 4th row or first/last)
		if row == 0 || row == chartHeight-1 || row%(chartHeight/4+1) == 0 {
			label := formatPriceLabel(rowPriceHigh)
			sb.WriteString(fmt.Sprintf("%8s│", label))
		} else {
			sb.WriteString("        │")
		}

		// Draw each candle column
		for _, bar := range visibleBars {
			bodyTop := math.Max(bar.Open, bar.Close)
			bodyBot := math.Min(bar.Open, bar.Close)

			isBullish := bar.Close >= bar.Open
			color := "[red]"
			if isBullish {
				color = "[green]"
			}

			char := " "
			if bar.High >= rowPriceLow && bar.Low <= rowPriceHigh {
				// This row intersects the candle's range
				if bodyTop >= rowPriceLow && bodyBot <= rowPriceHigh {
					// Body intersects this row
					char = color + "█" + "[-]"
				} else {
					// Wick only
					char = color + "│" + "[-]"
				}
			}

			sb.WriteString(char)
			sb.WriteString(" ")
		}

		sb.WriteString("\n")
	}

	// X-axis separator
	sb.WriteString("        └")
	sb.WriteString(strings.Repeat("─", chartWidth))
	sb.WriteString("\n")

	// X-axis labels with smart formatting based on timeframe
	sb.WriteString("         ")
	labels := buildXAxisLabels(visibleBars)

	// Place labels using a cursor to prevent overlaps
	cursor := 0 // next column position we can write to
	for _, lbl := range labels {
		col := lbl.pos * 2 // each candle = 2 columns
		if col < cursor {
			continue // would overlap previous label
		}
		// Pad to reach this column
		if col > cursor {
			sb.WriteString(strings.Repeat(" ", col-cursor))
		}
		sb.WriteString("[gray]")
		sb.WriteString(lbl.text)
		sb.WriteString("[-]")
		cursor = col + len(lbl.text)
	}

	return sb.String()
}

// centerText returns a string with the message centered in the given dimensions
func centerText(msg string, width, height int) string {
	var sb strings.Builder
	topPad := height / 2
	for i := 0; i < topPad; i++ {
		sb.WriteString("\n")
	}
	leftPad := (width - len(msg)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(msg)
	return sb.String()
}

// xAxisLabel holds a positioned label for the X-axis.
type xAxisLabel struct {
	pos  int    // bar index
	text string // display text
}

// buildXAxisLabels creates positioned labels for the X-axis based on bar timestamps.
// It auto-detects intraday vs daily vs weekly timeframes and uses appropriate formatting.
func buildXAxisLabels(bars []models.Bar) []xAxisLabel {
	if len(bars) == 0 {
		return nil
	}

	// Detect timeframe from median interval between bars
	tf := detectTimeframe(bars)

	var labels []xAxisLabel
	n := len(bars)

	// Choose label interval so labels are ~8-12 chars apart (each candle = 2 cols)
	// Labels are 5-8 chars wide, need at least 2 char gap between them
	var labelWidth int
	switch tf {
	case tfMinutes:
		labelWidth = 5 // "HH:MM"
	case tfHours:
		labelWidth = 5 // "HH:MM"
	case tfDaily:
		labelWidth = 5 // "DD.MM"
	default:
		labelWidth = 8 // "DD.MM.YY"
	}
	minSpacing := labelWidth + 3 // label width + minimum gap
	minBarsBetween := minSpacing / 2
	if minBarsBetween < 1 {
		minBarsBetween = 1
	}

	// Desired ~5-7 labels across the chart
	desiredLabels := 6
	interval := n / desiredLabels
	if interval < minBarsBetween {
		interval = minBarsBetween
	}

	prevDay := -1
	for i := 0; i < n; i++ {
		bar := bars[i]
		isFirst := i == 0
		isLast := i == n-1

		switch tf {
		case tfMinutes, tfHours:
			// For intraday: show time, but mark day boundaries with date
			day := bar.Timestamp.YearDay()
			if isFirst || (day != prevDay && prevDay != -1) {
				// Day boundary — show date
				labels = append(labels, xAxisLabel{pos: i, text: bar.Timestamp.Format("02.01")})
				prevDay = day
				continue
			}
			prevDay = day
			if isFirst || isLast || i%interval == 0 {
				labels = append(labels, xAxisLabel{pos: i, text: bar.Timestamp.Format("15:04")})
			}

		case tfDaily:
			if isFirst || isLast || i%interval == 0 {
				labels = append(labels, xAxisLabel{pos: i, text: bar.Timestamp.Format("02.01")})
			}

		default: // weekly+
			if isFirst || isLast || i%interval == 0 {
				labels = append(labels, xAxisLabel{pos: i, text: bar.Timestamp.Format("02.01.06")})
			}
		}
	}

	return labels
}

type timeframeType int

const (
	tfMinutes timeframeType = iota
	tfHours
	tfDaily
	tfWeekly
)

// detectTimeframe determines the chart timeframe from bar intervals.
func detectTimeframe(bars []models.Bar) timeframeType {
	if len(bars) < 2 {
		return tfDaily
	}
	// Collect intervals, skip gaps (weekends, overnight)
	var intervals []time.Duration
	for i := 1; i < len(bars) && i < 10; i++ {
		d := bars[i].Timestamp.Sub(bars[i-1].Timestamp)
		if d > 0 {
			intervals = append(intervals, d)
		}
	}
	if len(intervals) == 0 {
		return tfDaily
	}
	// Use minimum interval (most representative, skips weekend gaps)
	minInterval := intervals[0]
	for _, d := range intervals[1:] {
		if d < minInterval {
			minInterval = d
		}
	}
	switch {
	case minInterval < 30*time.Minute:
		return tfMinutes
	case minInterval < 4*time.Hour:
		return tfHours
	case minInterval < 3*24*time.Hour:
		return tfDaily
	default:
		return tfWeekly
	}
}

// formatPriceLabel formats a price value for the Y-axis gutter (max 8 chars)
func formatPriceLabel(price float64) string {
	if price >= 10000 {
		return fmt.Sprintf("%.0f", price)
	}
	if price >= 100 {
		return fmt.Sprintf("%.1f", price)
	}
	if price >= 1 {
		return fmt.Sprintf("%.2f", price)
	}
	return fmt.Sprintf("%.4f", price)
}

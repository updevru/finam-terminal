package ui

import (
	"fmt"
	"math"
	"strings"

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

	// Reserve 1 row for X-axis labels
	chartHeight := height - 1
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

	// X-axis labels
	sb.WriteString("         ")
	labelInterval := len(visibleBars) / 4
	if labelInterval < 1 {
		labelInterval = 1
	}
	for i, bar := range visibleBars {
		if i%labelInterval == 0 || i == len(visibleBars)-1 {
			label := bar.Timestamp.Format("01/02")
			// Ensure label doesn't overflow
			available := (len(visibleBars) - i) * 2
			if len(label) <= available {
				sb.WriteString(label)
				// Pad remaining space for this label
				remaining := labelInterval*2 - len(label)
				if remaining > 0 && i != len(visibleBars)-1 {
					sb.WriteString(strings.Repeat(" ", remaining))
				}
				// Skip positions covered by the label
				for j := 1; j < len(label)/2+1 && i+j < len(visibleBars); j++ {
						_ = j // positions consumed by the label
				}
			} else {
				sb.WriteString("  ")
			}
		} else {
			sb.WriteString("  ")
		}
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

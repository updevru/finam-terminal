package ui

import (
	"strings"
	"testing"
	"time"

	"finam-terminal/models"
)

func TestRenderCandlestickChart_EmptyBars(t *testing.T) {
	result := RenderCandlestickChart(nil, 80, 20)
	if !strings.Contains(result, "No data") {
		t.Errorf("Expected 'No data' message for empty bars, got: %s", result)
	}
}

func TestRenderCandlestickChart_EmptySlice(t *testing.T) {
	result := RenderCandlestickChart([]models.Bar{}, 80, 20)
	if !strings.Contains(result, "No data") {
		t.Errorf("Expected 'No data' message for empty slice, got: %s", result)
	}
}

func TestRenderCandlestickChart_SingleBar(t *testing.T) {
	bars := []models.Bar{
		{
			Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local),
			Open:      100.0,
			High:      105.0,
			Low:       95.0,
			Close:     103.0,
			Volume:    1000,
		},
	}

	result := RenderCandlestickChart(bars, 80, 20)

	// Should not contain "No data"
	if strings.Contains(result, "No data") {
		t.Error("Single bar should render chart, not 'No data'")
	}

	// Should contain green color tag (bullish: close > open)
	if !strings.Contains(result, "[green]") {
		t.Error("Bullish bar should contain [green] color tag")
	}
}

func TestRenderCandlestickChart_BullishAndBearish(t *testing.T) {
	bars := []models.Bar{
		{
			Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local),
			Open:      100.0,
			High:      110.0,
			Low:       95.0,
			Close:     108.0, // bullish
			Volume:    1000,
		},
		{
			Timestamp: time.Date(2024, 1, 15, 11, 0, 0, 0, time.Local),
			Open:      108.0,
			High:      112.0,
			Low:       100.0,
			Close:     102.0, // bearish
			Volume:    1500,
		},
	}

	result := RenderCandlestickChart(bars, 80, 20)

	if !strings.Contains(result, "[green]") {
		t.Error("Should contain [green] for bullish bar")
	}
	if !strings.Contains(result, "[red]") {
		t.Error("Should contain [red] for bearish bar")
	}
}

func TestRenderCandlestickChart_ContainsPriceLabels(t *testing.T) {
	bars := []models.Bar{
		{
			Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local),
			Open:      100.0,
			High:      110.0,
			Low:       90.0,
			Close:     105.0,
			Volume:    1000,
		},
	}

	result := RenderCandlestickChart(bars, 80, 20)

	// Y-axis should have price labels
	lines := strings.Split(result, "\n")
	foundPriceLabel := false
	for _, line := range lines {
		// Price labels are right-aligned numbers in the left gutter
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && (strings.Contains(trimmed, "100") || strings.Contains(trimmed, "110") || strings.Contains(trimmed, "90")) {
			foundPriceLabel = true
			break
		}
	}
	if !foundPriceLabel {
		t.Error("Chart should contain price labels on Y-axis")
	}
}

func TestRenderCandlestickChart_SmallDimensions(t *testing.T) {
	bars := []models.Bar{
		{
			Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local),
			Open:      100.0,
			High:      105.0,
			Low:       95.0,
			Close:     103.0,
			Volume:    1000,
		},
	}

	// Should not panic with very small dimensions
	result := RenderCandlestickChart(bars, 15, 5)
	if result == "" {
		t.Error("Should return non-empty result even for small dimensions")
	}
}

func TestRenderCandlestickChart_ReturnsString(t *testing.T) {
	bars := make([]models.Bar, 50)
	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.Local)
	for i := range bars {
		bars[i] = models.Bar{
			Timestamp: base.Add(time.Duration(i) * time.Hour),
			Open:      100.0 + float64(i),
			High:      105.0 + float64(i),
			Low:       95.0 + float64(i),
			Close:     102.0 + float64(i),
			Volume:    float64(1000 + i*100),
		}
	}

	result := RenderCandlestickChart(bars, 80, 24)

	// Should be a non-empty string
	if len(result) == 0 {
		t.Error("Chart with 50 bars should produce non-empty output")
	}

	// Should have multiple lines
	lines := strings.Split(result, "\n")
	if len(lines) < 5 {
		t.Errorf("Expected at least 5 lines of output, got %d", len(lines))
	}
}

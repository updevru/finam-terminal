package ui

import "testing"

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		decimals int
		expected string
	}{
		{"positive large", 1234567.89, 2, "1 234 567.89"},
		{"positive medium", 12345.50, 2, "12 345.50"},
		{"positive small", 999.99, 2, "999.99"},
		{"positive tiny", 0.50, 2, "0.50"},
		{"negative large", -1234567.89, 2, "-1 234 567.89"},
		{"negative medium", -12345.50, 2, "-12 345.50"},
		{"negative small", -999.99, 2, "-999.99"},
		{"zero", 0, 2, "0.00"},
		{"negative zero", -0.0, 2, "0.00"},
		{"exactly thousand", 1000.00, 2, "1 000.00"},
		{"exactly million", 1000000.00, 2, "1 000 000.00"},
		{"no decimals", 1234567.0, 0, "1 234 567"},
		{"one decimal", 1234.5, 1, "1 234.5"},
		{"large number", 999999999.99, 2, "999 999 999.99"},
		{"three digits", 100.00, 2, "100.00"},
		{"two digits", 99.99, 2, "99.99"},
		{"one digit", 5.00, 2, "5.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNumber(tt.input, tt.decimals)
			if got != tt.expected {
				t.Errorf("formatNumber(%v, %d) = %q, want %q", tt.input, tt.decimals, got, tt.expected)
			}
		})
	}
}

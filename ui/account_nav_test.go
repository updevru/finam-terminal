package ui

import "testing"

func TestAccountIdxToRow(t *testing.T) {
	tests := []struct {
		idx      int
		expected int
	}{
		{0, 0},
		{1, 2},
		{2, 4},
		{5, 10},
	}
	for _, tt := range tests {
		got := accountIdxToRow(tt.idx)
		if got != tt.expected {
			t.Errorf("accountIdxToRow(%d) = %d, want %d", tt.idx, got, tt.expected)
		}
	}
}

func TestRowToAccountIdx(t *testing.T) {
	tests := []struct {
		row      int
		expected int
	}{
		{0, 0}, // ID row of account 0
		{1, 0}, // data row of account 0
		{2, 1}, // ID row of account 1
		{3, 1}, // data row of account 1
		{4, 2},
		{5, 2},
		{10, 5},
		{11, 5},
	}
	for _, tt := range tests {
		got := rowToAccountIdx(tt.row)
		if got != tt.expected {
			t.Errorf("rowToAccountIdx(%d) = %d, want %d", tt.row, got, tt.expected)
		}
	}
}

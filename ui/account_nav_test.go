package ui

import (
	"finam-terminal/models"
	"testing"
)

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
		{0, 0},
		{1, 0},
		{2, 1},
		{3, 1},
		{4, 2},
		{5, 2},
	}
	for _, tt := range tests {
		got := rowToAccountIdx(tt.row)
		if got != tt.expected {
			t.Errorf("rowToAccountIdx(%d) = %d, want %d", tt.row, got, tt.expected)
		}
	}
}

func TestAccountNavigation_SkipsByAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "0"},
		{ID: "ACC2", Equity: "2000.00", UnrealizedPnL: "0"},
		{ID: "ACC3", Equity: "3000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)

	app.selectedIdx = 0
	updateAccountList(app)
	row, _ := app.portfolioView.AccountTable.GetSelection()
	if row != 0 {
		t.Errorf("Initial: expected row 0, got %d", row)
	}

	app.selectedIdx = 1
	updateAccountList(app)
	row, _ = app.portfolioView.AccountTable.GetSelection()
	if row != 2 {
		t.Errorf("After down: expected row 2, got %d", row)
	}

	app.selectedIdx = 2
	updateAccountList(app)
	row, _ = app.portfolioView.AccountTable.GetSelection()
	if row != 4 {
		t.Errorf("After second down: expected row 4, got %d", row)
	}
}

func TestAccountNavigation_BoundsCheck(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	app.selectedIdx = 0
	updateAccountList(app)

	row, _ := app.portfolioView.AccountTable.GetSelection()
	if row != 0 {
		t.Errorf("Expected row 0, got %d", row)
	}
}

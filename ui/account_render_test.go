package ui

import (
	"finam-terminal/models"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// createTestAppWithAccounts creates a minimal App with the given accounts for render testing.
func createTestAppWithAccounts(accounts []models.AccountInfo) *App {
	app := NewApp(nil, accounts)
	return app
}

func TestUpdateAccountList_TwoRowPerAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "12345678", Equity: "1234567.89", UnrealizedPnL: "15000.00"},
		{ID: "87654321", Equity: "543210.00", UnrealizedPnL: "-2100.50"},
	}
	app := createTestAppWithAccounts(accounts)

	updateAccountList(app)

	// No header row anymore; 2 accounts × 2 rows = 4 rows total
	expectedRows := 4
	got := app.portfolioView.AccountTable.GetRowCount()
	if got != expectedRows {
		t.Errorf("Expected %d rows, got %d", expectedRows, got)
	}

	// Account 1, row 0: ID
	cell := app.portfolioView.AccountTable.GetCell(0, 0)
	if cell.Text != "12345678" {
		t.Errorf("Row 0 col 0: expected '12345678', got %q", cell.Text)
	}

	// Account 1, row 1: Equity + PnL
	equityCell := app.portfolioView.AccountTable.GetCell(1, 0)
	if equityCell.Text == "" {
		t.Error("Row 1 col 0: expected equity text, got empty")
	}
	pnlCell := app.portfolioView.AccountTable.GetCell(1, 1)
	if pnlCell.Text == "" {
		t.Error("Row 1 col 1: expected PnL text, got empty")
	}

	// Account 2, row 2: ID
	cell2 := app.portfolioView.AccountTable.GetCell(2, 0)
	if cell2.Text != "87654321" {
		t.Errorf("Row 2 col 0: expected '87654321', got %q", cell2.Text)
	}

	// Account 2, row 3: Equity + PnL
	equityCell2 := app.portfolioView.AccountTable.GetCell(3, 0)
	if equityCell2.Text == "" {
		t.Error("Row 3 col 0: expected equity text, got empty")
	}
}

func TestUpdateAccountList_PnLColors(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "500.00"},
		{ID: "ACC2", Equity: "2000.00", UnrealizedPnL: "-300.00"},
		{ID: "ACC3", Equity: "3000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	tests := []struct {
		name          string
		pnlRow        int
		pnlCol        int
		expectedColor tcell.Color
	}{
		{"positive PnL green", 1, 1, tcell.ColorGreen},
		{"negative PnL red", 3, 1, tcell.ColorRed},
		{"zero PnL gray", 5, 1, tcell.ColorGray},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cell := app.portfolioView.AccountTable.GetCell(tt.pnlRow, tt.pnlCol)
			_, fg, _ := cell.Style.Decompose()
			if fg != tt.expectedColor {
				t.Errorf("Expected color %v, got %v for cell text %q", tt.expectedColor, fg, cell.Text)
			}
		})
	}
}

func TestUpdateAccountList_PositivePnLHasPlusSign(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "500.00"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	pnlCell := app.portfolioView.AccountTable.GetCell(1, 1)
	if len(pnlCell.Text) == 0 || pnlCell.Text[0] != '+' {
		t.Errorf("Expected positive PnL to start with '+', got %q", pnlCell.Text)
	}
}

func TestUpdateAccountList_NegativePnLHasMinusSign(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "-300.50"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	pnlCell := app.portfolioView.AccountTable.GetCell(1, 1)
	if len(pnlCell.Text) == 0 || pnlCell.Text[0] != '-' {
		t.Errorf("Expected negative PnL to start with '-', got %q", pnlCell.Text)
	}
}

func TestUpdateAccountList_EquityFormatted(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1234567.89", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	equityCell := app.portfolioView.AccountTable.GetCell(1, 0)
	expected := "1 234 567.89"
	if equityCell.Text != expected {
		t.Errorf("Expected equity %q, got %q", expected, equityCell.Text)
	}
}

func TestUpdateAccountList_NoTypeColumn(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Type: "SomeType", Equity: "1000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	// Check that "Type" does not appear in any cell
	for row := 0; row < app.portfolioView.AccountTable.GetRowCount(); row++ {
		for col := 0; col < app.portfolioView.AccountTable.GetColumnCount(); col++ {
			cell := app.portfolioView.AccountTable.GetCell(row, col)
			if cell != nil && cell.Text == "SomeType" {
				t.Error("Type column should not be present in two-row account layout")
			}
		}
	}
}

func TestUpdateAccountList_SelectionCoversCorrectRow(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "0"},
		{ID: "ACC2", Equity: "2000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	app.selectedIdx = 1
	updateAccountList(app)

	// Selected account 1 → should select row 2 (account 1's ID row = row 2)
	selectedRow, _ := app.portfolioView.AccountTable.GetSelection()
	expectedRow := 2 // account index 1 * 2 rows per account
	if selectedRow != expectedRow {
		t.Errorf("Expected selected row %d, got %d", expectedRow, selectedRow)
	}
}

func TestUpdateAccountList_EmptyAccounts(t *testing.T) {
	app := createTestAppWithAccounts([]models.AccountInfo{})
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 0 {
		t.Errorf("Expected 0 rows for empty accounts, got %d", app.portfolioView.AccountTable.GetRowCount())
	}
}

func TestUpdateAccountList_SingleAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ONLY1", Equity: "999.99", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 2 {
		t.Errorf("Expected 2 rows for single account, got %d", app.portfolioView.AccountTable.GetRowCount())
	}
}

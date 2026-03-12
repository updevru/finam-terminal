package ui

import (
	"finam-terminal/models"
	"strings"
	"testing"
)

func TestAccountIntegration_ZeroAccounts(t *testing.T) {
	app := createTestAppWithAccounts([]models.AccountInfo{})
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 0 {
		t.Errorf("Expected 0 rows for 0 accounts, got %d", app.portfolioView.AccountTable.GetRowCount())
	}
}

func TestAccountIntegration_SingleAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC_SINGLE", Equity: "50000.00", UnrealizedPnL: "1234.56"},
	}
	app := createTestAppWithAccounts(accounts)
	app.selectedIdx = 0
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 1 {
		t.Fatalf("Expected 1 row, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	cell := app.portfolioView.AccountTable.GetCell(0, 0)
	lines := strings.Split(cell.Text, "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d: %q", len(lines), cell.Text)
	}
	if lines[0] != "ACC_SINGLE" {
		t.Errorf("Line 1: expected 'ACC_SINGLE', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "50 000.00") {
		t.Errorf("Line 2: expected '50 000.00', got %q", lines[1])
	}
	if !strings.Contains(lines[1], "+1 234.56") {
		t.Errorf("Line 2: expected '+1 234.56', got %q", lines[1])
	}

	selectedRow, _ := app.portfolioView.AccountTable.GetSelection()
	if selectedRow != 0 {
		t.Errorf("Expected selected row 0, got %d", selectedRow)
	}
}

func TestAccountIntegration_ThreeAccounts(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "100000.00", UnrealizedPnL: "5000.00"},
		{ID: "ACC2", Equity: "200000.00", UnrealizedPnL: "-3000.00"},
		{ID: "ACC3", LoadError: "connection timeout"},
	}
	app := createTestAppWithAccounts(accounts)
	app.selectedIdx = 1
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 3 {
		t.Fatalf("Expected 3 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	// Account 1: positive PnL
	cell0 := app.portfolioView.AccountTable.GetCell(0, 0)
	if !strings.Contains(cell0.Text, "+5 000.00") {
		t.Errorf("ACC1: expected '+5 000.00' in text, got %q", cell0.Text)
	}

	// Account 2: negative PnL
	cell1 := app.portfolioView.AccountTable.GetCell(1, 0)
	if !strings.Contains(cell1.Text, "-3 000.00") {
		t.Errorf("ACC2: expected '-3 000.00' in text, got %q", cell1.Text)
	}

	// Account 3: error
	cell2 := app.portfolioView.AccountTable.GetCell(2, 0)
	if !strings.Contains(cell2.Text, "[error]") {
		t.Errorf("ACC3: expected '[error]' in text, got %q", cell2.Text)
	}

	// Selection
	selectedRow, _ := app.portfolioView.AccountTable.GetSelection()
	if selectedRow != 1 {
		t.Errorf("Expected selected row 1, got %d", selectedRow)
	}
}

func TestAccountIntegration_SwitchSelection(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "0"},
		{ID: "ACC2", Equity: "2000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)

	app.selectedIdx = 0
	updateAccountList(app)
	row, _ := app.portfolioView.AccountTable.GetSelection()
	if row != 0 {
		t.Errorf("Expected selected row 0, got %d", row)
	}

	app.selectedIdx = 1
	updateAccountList(app)
	row, _ = app.portfolioView.AccountTable.GetSelection()
	if row != 1 {
		t.Errorf("Expected selected row 1 after switch, got %d", row)
	}
}

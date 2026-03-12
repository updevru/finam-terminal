package ui

import (
	"finam-terminal/models"
	"strings"
	"testing"
)

// createTestAppWithAccounts creates a minimal App with the given accounts for render testing.
func createTestAppWithAccounts(accounts []models.AccountInfo) *App {
	app := NewApp(nil, accounts)
	return app
}

func TestUpdateAccountList_OneRowPerAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "12345678", Equity: "1234567.89", UnrealizedPnL: "15000.00"},
		{ID: "87654321", Equity: "543210.00", UnrealizedPnL: "-2100.50"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	// 2 accounts = 2 rows (one multi-line row each)
	if app.portfolioView.AccountTable.GetRowCount() != 2 {
		t.Errorf("Expected 2 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	// Account 1: text contains ID on first line
	cell0 := app.portfolioView.AccountTable.GetCell(0, 0)
	if !strings.HasPrefix(cell0.Text, "12345678\n") {
		t.Errorf("Row 0: expected text starting with '12345678\\n', got %q", cell0.Text)
	}

	// Account 2: text contains ID on first line
	cell1 := app.portfolioView.AccountTable.GetCell(1, 0)
	if !strings.HasPrefix(cell1.Text, "87654321\n") {
		t.Errorf("Row 1: expected text starting with '87654321\\n', got %q", cell1.Text)
	}
}

func TestUpdateAccountList_EquityFormatted(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1234567.89", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	cell := app.portfolioView.AccountTable.GetCell(0, 0)
	if !strings.Contains(cell.Text, "1 234 567.89") {
		t.Errorf("Expected formatted equity in cell text, got %q", cell.Text)
	}
}

func TestUpdateAccountList_PositivePnLHasPlusSign(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "500.00"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	cell := app.portfolioView.AccountTable.GetCell(0, 0)
	if !strings.Contains(cell.Text, "+500.00") {
		t.Errorf("Expected '+500.00' in cell text, got %q", cell.Text)
	}
}

func TestUpdateAccountList_NegativePnLHasMinusSign(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "-300.50"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	cell := app.portfolioView.AccountTable.GetCell(0, 0)
	if !strings.Contains(cell.Text, "-300.50") {
		t.Errorf("Expected '-300.50' in cell text, got %q", cell.Text)
	}
}

func TestUpdateAccountList_NoTypeColumn(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Type: "SomeType", Equity: "1000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	cell := app.portfolioView.AccountTable.GetCell(0, 0)
	if strings.Contains(cell.Text, "SomeType") {
		t.Error("Type should not appear in account cell text")
	}
}

func TestUpdateAccountList_ErrorAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ERR_ACC", LoadError: "broker unavailable"},
		{ID: "OK_ACC", Equity: "5000.00", UnrealizedPnL: "100.00"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 2 {
		t.Fatalf("Expected 2 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	// Error account: text has ID + [error]
	errCell := app.portfolioView.AccountTable.GetCell(0, 0)
	if !strings.Contains(errCell.Text, "ERR_ACC") {
		t.Errorf("Expected error cell to contain 'ERR_ACC', got %q", errCell.Text)
	}
	if !strings.Contains(errCell.Text, "[error]") {
		t.Errorf("Expected error cell to contain '[error]', got %q", errCell.Text)
	}

	// Normal account renders correctly
	okCell := app.portfolioView.AccountTable.GetCell(1, 0)
	if !strings.Contains(okCell.Text, "OK_ACC") {
		t.Errorf("Expected normal cell to contain 'OK_ACC', got %q", okCell.Text)
	}
}

func TestUpdateAccountList_SelectionRow(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "0"},
		{ID: "ACC2", Equity: "2000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	app.selectedIdx = 1
	updateAccountList(app)

	selectedRow, _ := app.portfolioView.AccountTable.GetSelection()
	if selectedRow != 1 {
		t.Errorf("Expected selected row 1, got %d", selectedRow)
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

	if app.portfolioView.AccountTable.GetRowCount() != 1 {
		t.Errorf("Expected 1 row for single account, got %d", app.portfolioView.AccountTable.GetRowCount())
	}
}

func TestUpdateAccountList_MultiLineText(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "50000.00", UnrealizedPnL: "1234.56"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	cell := app.portfolioView.AccountTable.GetCell(0, 0)
	lines := strings.Split(cell.Text, "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines in cell, got %d: %q", len(lines), cell.Text)
	}
	if lines[0] != "ACC1" {
		t.Errorf("Line 1: expected 'ACC1', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "50 000.00") {
		t.Errorf("Line 2: expected equity '50 000.00', got %q", lines[1])
	}
	if !strings.Contains(lines[1], "+1 234.56") {
		t.Errorf("Line 2: expected PnL '+1 234.56', got %q", lines[1])
	}
}

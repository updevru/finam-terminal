package ui

import (
	"finam-terminal/models"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestAccountIntegration_ZeroAccounts(t *testing.T) {
	app := createTestAppWithAccounts([]models.AccountInfo{})
	updateAccountList(app)

	rowCount := app.portfolioView.AccountTable.GetRowCount()
	if rowCount != 0 {
		t.Errorf("Expected 0 rows for 0 accounts, got %d", rowCount)
	}
}

func TestAccountIntegration_SingleAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC_SINGLE", Equity: "50000.00", UnrealizedPnL: "1234.56"},
	}
	app := createTestAppWithAccounts(accounts)
	app.selectedIdx = 0
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 2 {
		t.Fatalf("Expected 2 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	// ID row
	idCell := app.portfolioView.AccountTable.GetCell(0, 0)
	if idCell.Text != "ACC_SINGLE" {
		t.Errorf("Expected ID 'ACC_SINGLE', got %q", idCell.Text)
	}

	// Equity row
	equityCell := app.portfolioView.AccountTable.GetCell(1, 0)
	if equityCell.Text != "50 000.00" {
		t.Errorf("Expected equity '50 000.00', got %q", equityCell.Text)
	}

	// PnL cell
	pnlCell := app.portfolioView.AccountTable.GetCell(1, 1)
	if pnlCell.Text != "+1 234.56" {
		t.Errorf("Expected PnL '+1 234.56', got %q", pnlCell.Text)
	}
	fg, _, _ := pnlCell.Style.Decompose()
	if fg != tcell.ColorGreen {
		t.Errorf("Expected green PnL, got %v", fg)
	}

	// Selected highlight on both rows
	_, bg0, _ := app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	_, bg1, _ := app.portfolioView.AccountTable.GetCell(1, 0).Style.Decompose()
	if bg0 != tcell.ColorDarkSlateGray || bg1 != tcell.ColorDarkSlateGray {
		t.Errorf("Expected highlight bg on selected account, got %v / %v", bg0, bg1)
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

	// Total rows: 3 accounts × 2 = 6
	if app.portfolioView.AccountTable.GetRowCount() != 6 {
		t.Fatalf("Expected 6 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	// Account 1 (not selected): normal bg
	_, bg0, _ := app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	if bg0 != tcell.ColorBlack {
		t.Errorf("ACC1 should have black bg (not selected), got %v", bg0)
	}

	// Account 2 (selected): highlight bg
	_, bg2, _ := app.portfolioView.AccountTable.GetCell(2, 0).Style.Decompose()
	_, bg3, _ := app.portfolioView.AccountTable.GetCell(3, 0).Style.Decompose()
	if bg2 != tcell.ColorDarkSlateGray || bg3 != tcell.ColorDarkSlateGray {
		t.Errorf("ACC2 should have highlight bg, got %v / %v", bg2, bg3)
	}

	// Account 2 PnL: negative, red
	pnl2 := app.portfolioView.AccountTable.GetCell(3, 1)
	if pnl2.Text != "-3 000.00" {
		t.Errorf("Expected PnL '-3 000.00', got %q", pnl2.Text)
	}
	fg2, _, _ := pnl2.Style.Decompose()
	if fg2 != tcell.ColorRed {
		t.Errorf("Expected red PnL, got %v", fg2)
	}

	// Account 3 (error): ID + [error]
	errID := app.portfolioView.AccountTable.GetCell(4, 0)
	if errID.Text != "ACC3" {
		t.Errorf("Expected error account ID 'ACC3', got %q", errID.Text)
	}
	errData := app.portfolioView.AccountTable.GetCell(5, 0)
	if errData.Text != "[error]" {
		t.Errorf("Expected '[error]', got %q", errData.Text)
	}

	// Selection row should be 2 (account idx 1)
	selectedRow, _ := app.portfolioView.AccountTable.GetSelection()
	if selectedRow != 2 {
		t.Errorf("Expected selected row 2, got %d", selectedRow)
	}
}

func TestAccountIntegration_SwitchSelection(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "0"},
		{ID: "ACC2", Equity: "2000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)

	// Select first account
	app.selectedIdx = 0
	updateAccountList(app)

	_, bg0a, _ := app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	_, bg2a, _ := app.portfolioView.AccountTable.GetCell(2, 0).Style.Decompose()
	if bg0a != tcell.ColorDarkSlateGray {
		t.Errorf("ACC1 should be highlighted, got bg %v", bg0a)
	}
	if bg2a != tcell.ColorBlack {
		t.Errorf("ACC2 should not be highlighted, got bg %v", bg2a)
	}

	// Switch to second account
	app.selectedIdx = 1
	updateAccountList(app)

	_, bg0b, _ := app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	_, bg2b, _ := app.portfolioView.AccountTable.GetCell(2, 0).Style.Decompose()
	if bg0b != tcell.ColorBlack {
		t.Errorf("ACC1 should not be highlighted after switch, got bg %v", bg0b)
	}
	if bg2b != tcell.ColorDarkSlateGray {
		t.Errorf("ACC2 should be highlighted after switch, got bg %v", bg2b)
	}
}

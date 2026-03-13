package ui

import (
	"finam-terminal/models"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestAccountIntegration_ZeroAccounts(t *testing.T) {
	app := createTestAppWithAccounts([]models.AccountInfo{})
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 0 {
		t.Errorf("Expected 0 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}
}

func TestAccountIntegration_SingleAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC_SINGLE", Equity: "50000.00"},
	}
	app := createTestAppWithAccounts(accounts)
	app.positions["ACC_SINGLE"] = []models.Position{{DailyPnL: "1234.56"}}
	app.selectedIdx = 0
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 2 {
		t.Fatalf("Expected 2 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	// ID row
	if app.portfolioView.AccountTable.GetCell(0, 0).Text != "ACC_SINGLE" {
		t.Errorf("Expected 'ACC_SINGLE', got %q", app.portfolioView.AccountTable.GetCell(0, 0).Text)
	}

	// Data row: equity + PnL
	dataText := app.portfolioView.AccountTable.GetCell(1, 0).Text
	if !strings.Contains(dataText, "50 000.00") {
		t.Errorf("Expected '50 000.00' in data, got %q", dataText)
	}
	if !strings.Contains(dataText, "+1 234.56") {
		t.Errorf("Expected '+1 234.56' in data, got %q", dataText)
	}

	// Highlight on both rows
	_, bg0, _ := app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	_, bg1, _ := app.portfolioView.AccountTable.GetCell(1, 0).Style.Decompose()
	if bg0 != tcell.ColorDarkSlateGray || bg1 != tcell.ColorDarkSlateGray {
		t.Errorf("Selected account should have highlight bg, got %v / %v", bg0, bg1)
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

	if app.portfolioView.AccountTable.GetRowCount() != 6 {
		t.Fatalf("Expected 6 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	// ACC1 (not selected): black bg
	_, bg0, _ := app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	if bg0 != tcell.ColorBlack {
		t.Errorf("ACC1 bg: expected Black, got %v", bg0)
	}

	// ACC2 (selected): highlight on both rows
	_, bg2, _ := app.portfolioView.AccountTable.GetCell(2, 0).Style.Decompose()
	_, bg3, _ := app.portfolioView.AccountTable.GetCell(3, 0).Style.Decompose()
	if bg2 != tcell.ColorDarkSlateGray || bg3 != tcell.ColorDarkSlateGray {
		t.Errorf("ACC2 bg: expected DarkSlateGray, got %v / %v", bg2, bg3)
	}

	// ACC3 error
	if app.portfolioView.AccountTable.GetCell(5, 0).Text != "[error]" {
		t.Errorf("ACC3 data: expected '[error]', got %q", app.portfolioView.AccountTable.GetCell(5, 0).Text)
	}

	// Selection on row 2
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

	app.selectedIdx = 0
	updateAccountList(app)
	_, bg0, _ := app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	_, bg2, _ := app.portfolioView.AccountTable.GetCell(2, 0).Style.Decompose()
	if bg0 != tcell.ColorDarkSlateGray {
		t.Errorf("ACC1 should be highlighted, got %v", bg0)
	}
	if bg2 != tcell.ColorBlack {
		t.Errorf("ACC2 should not be highlighted, got %v", bg2)
	}

	app.selectedIdx = 1
	updateAccountList(app)
	_, bg0, _ = app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	_, bg2, _ = app.portfolioView.AccountTable.GetCell(2, 0).Style.Decompose()
	if bg0 != tcell.ColorBlack {
		t.Errorf("ACC1 should not be highlighted after switch, got %v", bg0)
	}
	if bg2 != tcell.ColorDarkSlateGray {
		t.Errorf("ACC2 should be highlighted after switch, got %v", bg2)
	}
}

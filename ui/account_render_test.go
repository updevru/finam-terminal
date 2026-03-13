package ui

import (
	"finam-terminal/models"
	"strings"
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

	// 2 accounts × 2 rows = 4 rows total
	if app.portfolioView.AccountTable.GetRowCount() != 4 {
		t.Errorf("Expected 4 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	// Account 1, row 0: ID
	if app.portfolioView.AccountTable.GetCell(0, 0).Text != "12345678" {
		t.Errorf("Row 0: expected '12345678', got %q", app.portfolioView.AccountTable.GetCell(0, 0).Text)
	}

	// Account 1, row 1: equity + PnL
	dataText := app.portfolioView.AccountTable.GetCell(1, 0).Text
	if !strings.Contains(dataText, "1 234 567.89") {
		t.Errorf("Row 1: expected equity in text, got %q", dataText)
	}

	// Account 2, row 2: ID
	if app.portfolioView.AccountTable.GetCell(2, 0).Text != "87654321" {
		t.Errorf("Row 2: expected '87654321', got %q", app.portfolioView.AccountTable.GetCell(2, 0).Text)
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
		dataRow       int
		expectedColor tcell.Color
	}{
		{"positive PnL green", 1, tcell.ColorGreen},
		{"negative PnL red", 3, tcell.ColorRed},
		{"zero PnL gray", 5, tcell.ColorGray},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cell := app.portfolioView.AccountTable.GetCell(tt.dataRow, 0)
			fg, _, _ := cell.Style.Decompose()
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

	dataText := app.portfolioView.AccountTable.GetCell(1, 0).Text
	if !strings.Contains(dataText, "+500.00") {
		t.Errorf("Expected '+500.00' in data row, got %q", dataText)
	}
}

func TestUpdateAccountList_NegativePnLHasMinusSign(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "-300.50"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	dataText := app.portfolioView.AccountTable.GetCell(1, 0).Text
	if !strings.Contains(dataText, "-300.50") {
		t.Errorf("Expected '-300.50' in data row, got %q", dataText)
	}
}

func TestUpdateAccountList_EquityFormatted(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1234567.89", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	dataText := app.portfolioView.AccountTable.GetCell(1, 0).Text
	if !strings.Contains(dataText, "1 234 567.89") {
		t.Errorf("Expected formatted equity, got %q", dataText)
	}
}

func TestUpdateAccountList_NoTypeColumn(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Type: "SomeType", Equity: "1000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	for row := 0; row < app.portfolioView.AccountTable.GetRowCount(); row++ {
		cell := app.portfolioView.AccountTable.GetCell(row, 0)
		if cell != nil && strings.Contains(cell.Text, "SomeType") {
			t.Error("Type should not appear in account list")
		}
	}
}

func TestUpdateAccountList_ErrorAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ERR_ACC", LoadError: "broker unavailable"},
		{ID: "OK_ACC", Equity: "5000.00", UnrealizedPnL: "100.00"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 4 {
		t.Fatalf("Expected 4 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}

	if app.portfolioView.AccountTable.GetCell(0, 0).Text != "ERR_ACC" {
		t.Errorf("Expected 'ERR_ACC', got %q", app.portfolioView.AccountTable.GetCell(0, 0).Text)
	}
	if app.portfolioView.AccountTable.GetCell(1, 0).Text != "[error]" {
		t.Errorf("Expected '[error]', got %q", app.portfolioView.AccountTable.GetCell(1, 0).Text)
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
	if selectedRow != 2 {
		t.Errorf("Expected selected row 2, got %d", selectedRow)
	}
}

func TestUpdateAccountList_EmptyAccounts(t *testing.T) {
	app := createTestAppWithAccounts([]models.AccountInfo{})
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 0 {
		t.Errorf("Expected 0 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}
}

func TestUpdateAccountList_SingleAccount(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ONLY1", Equity: "999.99", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	if app.portfolioView.AccountTable.GetRowCount() != 2 {
		t.Errorf("Expected 2 rows, got %d", app.portfolioView.AccountTable.GetRowCount())
	}
}

func TestUpdateAccountList_HighlightBothRows(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "50.00"},
		{ID: "ACC2", Equity: "2000.00", UnrealizedPnL: "-10.00"},
	}
	app := createTestAppWithAccounts(accounts)
	app.selectedIdx = 0
	updateAccountList(app)

	// Selected account: both rows have highlight bg
	_, idBg, _ := app.portfolioView.AccountTable.GetCell(0, 0).Style.Decompose()
	_, dataBg, _ := app.portfolioView.AccountTable.GetCell(1, 0).Style.Decompose()
	if idBg != tcell.ColorDarkSlateGray {
		t.Errorf("Selected ID row bg: expected DarkSlateGray, got %v", idBg)
	}
	if dataBg != tcell.ColorDarkSlateGray {
		t.Errorf("Selected data row bg: expected DarkSlateGray, got %v", dataBg)
	}

	// Non-selected account: black bg
	_, nonBg, _ := app.portfolioView.AccountTable.GetCell(2, 0).Style.Decompose()
	if nonBg != tcell.ColorBlack {
		t.Errorf("Non-selected ID row bg: expected Black, got %v", nonBg)
	}
}

func TestUpdateAccountList_TransparentDisabled(t *testing.T) {
	accounts := []models.AccountInfo{
		{ID: "ACC1", Equity: "1000.00", UnrealizedPnL: "0"},
	}
	app := createTestAppWithAccounts(accounts)
	updateAccountList(app)

	idCell := app.portfolioView.AccountTable.GetCell(0, 0)
	if idCell.Transparent {
		t.Error("ID cell should have Transparent=false for custom bg to render")
	}
	dataCell := app.portfolioView.AccountTable.GetCell(1, 0)
	if dataCell.Transparent {
		t.Error("Data cell should have Transparent=false for custom bg to render")
	}
}

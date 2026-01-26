package ui

import (
	"finam-terminal/models"
	"github.com/rivo/tview"
	"testing"
)

func TestNewPortfolioView(t *testing.T) {
	app := tview.NewApplication()
	pv := NewPortfolioView(app)

	if pv == nil {
		t.Fatal("Expected PortfolioView to be not nil")
	}

	if pv.Layout == nil {
		t.Error("Expected PortfolioView Layout to be not nil")
	}

	if pv.AccountList == nil {
		t.Error("Expected PortfolioView AccountList to be not nil")
	}

	if pv.AccountTable == nil {
		t.Error("Expected PortfolioView AccountTable to be not nil")
	}

	if pv.PositionsTable == nil {
		t.Error("Expected PortfolioView PositionsTable to be not nil")
	}

	if pv.SummaryArea == nil {
		t.Error("Expected PortfolioView SummaryArea to be not nil")
	}
}

func TestPortfolioView_UpdateAccounts(t *testing.T) {
	app := tview.NewApplication()
	pv := NewPortfolioView(app)

	accounts := []models.AccountInfo{
		{ID: "ACC1", Type: "T1", Status: "S1"},
		{ID: "ACC2", Type: "T2", Status: "S2"},
	}

	// Mocking the app wrapper logic roughly for the test or just testing the component logic directly
	// Since updateAccountList is in render.go and uses *App, we can't test it directly here easily without *App.
	// But wait, UpdateAccounts in components.go was removed/unused in favor of updateAccountList in render.go?
	// Let's check components.go content. I modified updateAccountList in render.go.
	// But components.go still has UpdateAccounts method?
	// Let's check if I updated components.go UpdateAccounts method. I didn't.
	// But updateAccountList in render.go is what is used by the app.
	// The test TestPortfolioView_UpdateAccounts tests pv.UpdateAccounts.
	// I should probably remove UpdateAccounts from components.go if it's unused, or update it to match.
	// Let's update pv.UpdateAccounts to match the new design (no status) to keep the component consistent.

	pv.UpdateAccounts(accounts)

	if pv.AccountTable.GetRowCount() != 3 { // 1 header + 2 data
		t.Errorf("Expected 3 rows in table, got %d", pv.AccountTable.GetRowCount())
	}

	cell := pv.AccountTable.GetCell(1, 0)
	if cell.Text != "ACC1" {
		t.Errorf("Expected first account ID to be ACC1, got %s", cell.Text)
	}
}

func TestPortfolioView_UpdateSummary(t *testing.T) {
	app := tview.NewApplication()
	pv := NewPortfolioView(app)

	acc := models.AccountInfo{
		ID:            "ACC1",
		Equity:        "1000.50",
		UnrealizedPnL: "50.25",
	}

	pv.UpdateSummary(acc)

	text := pv.SummaryArea.GetText(false)
	if text == "" {
		t.Error("Expected SummaryArea text to be not empty")
	}
}

func TestPortfolioView_UpdatePositions(t *testing.T) {
	app := tview.NewApplication()
	pv := NewPortfolioView(app)

	positions := []models.Position{
		{Symbol: "S1", Quantity: "10", AveragePrice: "100", CurrentPrice: "110", UnrealizedPnL: "100"},
		{Symbol: "S2", Quantity: "5", AveragePrice: "200", CurrentPrice: "190", UnrealizedPnL: "-50"},
	}

	pv.UpdatePositions(positions)

	if pv.PositionsTable.GetRowCount() != 3 { // 1 header + 2 data
		t.Errorf("Expected 3 rows in positions table, got %d", pv.PositionsTable.GetRowCount())
	}

	cell := pv.PositionsTable.GetCell(1, 0)
	if cell.Text != "S1" {
		t.Errorf("Expected first position symbol to be S1, got %s", cell.Text)
	}
}

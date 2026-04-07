package ui

import (
	"reflect"
	"testing"

	"finam-terminal/models"
	"github.com/rivo/tview"
)

func TestPositionsTable_Expansion(t *testing.T) {
	// Setup
	tviewApp := tview.NewApplication()
	pv := NewPortfolioView(tviewApp)
	app := &App{
		app:           tviewApp,
		portfolioView: pv,
		positions:     make(map[string][]models.Position),
		quotes:        make(map[string]map[string]*models.Quote),
		accounts:      []models.AccountInfo{{ID: "test_acc"}},
		selectedIdx:   0,
	}
	app.positions["test_acc"] = []models.Position{}

	// Execute
	updatePositionsTable(app)

	table := pv.TabbedView.PositionsTable
	colCount := table.GetColumnCount()
	if colCount == 0 {
		t.Fatal("Table has no columns")
	}

	// Inspect first cell to find the field
	cell := table.GetCell(0, 0)
	val := reflect.ValueOf(cell).Elem()
	typeOfT := val.Type()
	expansionFieldName := ""

	for field := range typeOfT.Fields() {
		// Look for something that sounds like expansion
		if field.Name == "expansion" || field.Name == "Expansion" {
			expansionFieldName = field.Name
			break
		}
	}

	if expansionFieldName == "" {
		// List all fields for debugging
		for field := range typeOfT.Fields() {
			t.Logf("Field: %s", field.Name)
		}
		t.Fatal("Could not find expansion field")
	}

	for i := range colCount {
		cell := table.GetCell(0, i)
		val := reflect.ValueOf(cell).Elem()
		expansionField := val.FieldByName(expansionFieldName)

		expansion := expansionField.Int()
		if expansion != 1 {
			t.Errorf("Expected column %d to have expansion 1, got %d", i, expansion)
		}
	}
}

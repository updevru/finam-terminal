package ui

import (
	"strings"
	"testing"
	"time"

	"finam-terminal/models"

	"github.com/gdamore/tcell/v2"
)

// indexAppWithSelection builds a loaded Index tab with the given row selected.
// Row 1 is the first constituent (row 0 is the header).
func indexAppWithSelection(t *testing.T, row int) *App {
	t.Helper()
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	setupInputHandlers(app)
	app.portfolioView.TabbedView.SetTab(TabIndex)
	updateIndexTable(app)
	app.portfolioView.TabbedView.IndexTable.Select(row, 0)
	app.app.SetFocus(app.portfolioView.TabbedView.IndexTable)
	return app
}

// TestSelectedIndexSymbol_FollowsSortedOrder verifies the row→symbol mapping
// uses the rendered (sorted) order, not the order the API returned — otherwise
// Enter would open a different instrument than the one highlighted.
func TestSelectedIndexSymbol_FollowsSortedOrder(t *testing.T) {
	app := indexAppWithSelection(t, 1)

	// Rendered order is by weight: GAZP (0.0120), LKOH (0.0100), SBER (0.0080).
	if got := app.selectedIndexSymbol(); got != "GAZP@MISX" {
		t.Errorf("row 1 symbol = %q, want GAZP@MISX", got)
	}

	app.portfolioView.TabbedView.IndexTable.Select(3, 0)
	if got := app.selectedIndexSymbol(); got != "SBER@MISX" {
		t.Errorf("row 3 symbol = %q, want SBER@MISX", got)
	}
}

// TestSelectedIndexSymbol_HeaderAndOutOfRange verifies the header row and a
// stale selection resolve to nothing rather than to the wrong instrument.
func TestSelectedIndexSymbol_HeaderAndOutOfRange(t *testing.T) {
	app := indexAppWithSelection(t, 0)
	if got := app.selectedIndexSymbol(); got != "" {
		t.Errorf("header row symbol = %q, want empty", got)
	}

	app.portfolioView.TabbedView.IndexTable.Select(99, 0)
	if got := app.selectedIndexSymbol(); got != "" {
		t.Errorf("out-of-range row symbol = %q, want empty", got)
	}
}

// TestIndexTab_EnterOpensProfile verifies Enter opens the profile of the
// highlighted instrument.
func TestIndexTab_EnterOpensProfile(t *testing.T) {
	app := indexAppWithSelection(t, 2)

	capture := app.portfolioView.TabbedView.IndexTable.GetInputCapture()
	capture(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if !app.IsProfileOpen() {
		t.Fatal("Enter on the Index tab did not open the profile")
	}
	if app.profileSymbol != "LKOH@MISX" {
		t.Errorf("profile symbol = %q, want LKOH@MISX", app.profileSymbol)
	}
}

// TestIndexTab_EnterOnHeaderDoesNothing verifies the header row is inert.
func TestIndexTab_EnterOnHeaderDoesNothing(t *testing.T) {
	app := indexAppWithSelection(t, 0)

	capture := app.portfolioView.TabbedView.IndexTable.GetInputCapture()
	capture(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if app.IsProfileOpen() {
		t.Error("Enter on the header row opened a profile")
	}
}

// TestIndexTab_AOpensOrderModal verifies A opens the standard order modal with
// the full symbol, which is what the existing order path expects.
func TestIndexTab_AOpensOrderModal(t *testing.T) {
	app := indexAppWithSelection(t, 1)

	capture := app.portfolioView.TabbedView.IndexTable.GetInputCapture()
	capture(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))

	// The page itself is added in Run(), which tests do not start; what matters
	// here is that the modal was armed with the highlighted instrument.
	if got := app.orderModal.GetInstrument(); got != "GAZP@MISX" {
		t.Errorf("order modal instrument = %q, want GAZP@MISX", got)
	}
	if got := app.orderModal.GetQuantity(); got != 0 {
		t.Errorf("order modal quantity = %v, want 0 for a fresh order", got)
	}
}

// TestIndexTab_CyrillicAOpensOrderModal verifies the Russian keyboard layout
// works too, matching the other tabs.
func TestIndexTab_CyrillicAOpensOrderModal(t *testing.T) {
	app := indexAppWithSelection(t, 1)

	capture := app.portfolioView.TabbedView.IndexTable.GetInputCapture()
	capture(tcell.NewEventKey(tcell.KeyRune, 'ф', tcell.ModNone))

	if got := app.orderModal.GetInstrument(); got != "GAZP@MISX" {
		t.Errorf("order modal instrument = %q, want GAZP@MISX", got)
	}
}

// TestIndexTab_AWarmsTheLot verifies the lot is resolved through the existing
// warm-up path, so the modal sizes the order in real lots.
func TestIndexTab_AWarmsTheLot(t *testing.T) {
	ensured := make(chan string, 4)
	mock := &mockClient{
		EnsureLotSizeFunc: func(_, symbol string) float64 {
			ensured <- symbol
			return 10
		},
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	setupInputHandlers(app)
	app.portfolioView.TabbedView.SetTab(TabIndex)
	updateIndexTable(app)
	app.portfolioView.TabbedView.IndexTable.Select(1, 0)

	capture := app.portfolioView.TabbedView.IndexTable.GetInputCapture()
	capture(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))

	select {
	case got := <-ensured:
		if got != "GAZP@MISX" {
			t.Errorf("warmed the lot for %q, want GAZP@MISX", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the lot was never warmed for the selected instrument")
	}
}

// TestCloseProfile_ReturnsToTheIndexTab verifies leaving a profile opened from
// the Index tab puts focus back on that tab rather than on Positions.
func TestCloseProfile_ReturnsToTheIndexTab(t *testing.T) {
	app := indexAppWithSelection(t, 1)
	app.OpenProfileForSymbol("GAZP@MISX")

	app.CloseProfile()

	if app.portfolioView.TabbedView.ActiveTab != TabIndex {
		t.Errorf("active tab after closing the profile = %v, want TabIndex", app.portfolioView.TabbedView.ActiveTab)
	}
	if app.app.GetFocus() != app.portfolioView.TabbedView.IndexTable {
		t.Error("focus after closing the profile did not return to the Index table")
	}
	if row, _ := app.portfolioView.TabbedView.IndexTable.GetSelection(); row != 1 {
		t.Errorf("selection after closing the profile = row %d, want 1 (preserved)", row)
	}
}

// TestStatusBar_IndexTabHints verifies the status bar advertises the keys that
// actually work on this tab.
func TestStatusBar_IndexTabHints(t *testing.T) {
	app := indexAppWithSelection(t, 1)

	updateStatusBar(app)

	text := app.statusBar.GetText(false)
	for _, want := range []string{"A", "Buy", "R", "Refresh"} {
		if !strings.Contains(text, want) {
			t.Errorf("status bar %q does not mention %q", text, want)
		}
	}
}

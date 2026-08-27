package ui

import (
	"testing"

	"github.com/rivo/tview"

	"github.com/gdamore/tcell/v2"
)

// indexAppWithPages builds a loaded Index tab whose overlay pages are
// registered, so the global Escape handler sees the same front page it sees at
// runtime. Row 1 (the first constituent) is selected.
func indexAppWithPages(t *testing.T) *App {
	t.Helper()
	app := indexAppWithSelection(t, 1)
	app.pages.AddPage("main", tview.NewBox(), true, true)
	app.pages.AddPage("modal", app.orderModal.Layout, true, false)
	app.pages.AddPage("close_modal", app.closeModal.Layout, true, false)
	app.pages.AddPage("search_modal", app.searchModal.Layout, true, false)
	return app
}

// pressGlobal feeds a key through the application-level input capture, the way
// tview does before the focused primitive sees it.
func pressGlobal(app *App, key tcell.Key, r rune) {
	app.app.GetInputCapture()(tcell.NewEventKey(key, r, tcell.ModNone))
}

// arrowsStillMoveTheIndexSelection is the user-visible symptom: after the
// overlay is gone, Down must move the highlighted row of the Index table.
func arrowsStillMoveTheIndexSelection(t *testing.T, app *App) {
	t.Helper()
	table, ok := app.app.GetFocus().(*tview.Table)
	if !ok {
		t.Fatalf("focus is %T, not a table — arrow keys reach nothing", app.app.GetFocus())
	}
	if table != app.portfolioView.TabbedView.IndexTable {
		t.Fatal("focus landed on a table of another tab, so the Index selection cannot move")
	}
	before, _ := app.portfolioView.TabbedView.IndexTable.GetSelection()
	table.GetInputCapture()(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	after, _ := app.portfolioView.TabbedView.IndexTable.GetSelection()
	if after != before+1 {
		t.Errorf("Down moved the Index selection from row %d to %d, want %d", before, after, before+1)
	}
}

// TestIndexTab_EscapeFromOrderModalRestoresNavigation reproduces the report: A
// then Escape on the Index tab left focus on the hidden Positions table, so the
// arrow keys silently drove an invisible tab until the tab was switched.
func TestIndexTab_EscapeFromOrderModalRestoresNavigation(t *testing.T) {
	app := indexAppWithPages(t)

	app.portfolioView.TabbedView.IndexTable.GetInputCapture()(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	if !app.IsModalOpen() {
		t.Fatal("A on the Index tab did not open the order modal")
	}

	pressGlobal(app, tcell.KeyEscape, 0)

	if app.IsModalOpen() {
		t.Fatal("Escape did not close the order modal")
	}
	arrowsStillMoveTheIndexSelection(t, app)
}

// TestIndexTab_EscapeFromSearchModalRestoresNavigation covers the same focus
// restore for the search overlay, which is reachable from every tab.
func TestIndexTab_EscapeFromSearchModalRestoresNavigation(t *testing.T) {
	app := indexAppWithPages(t)

	app.portfolioView.TabbedView.IndexTable.GetInputCapture()(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	if !app.IsSearchModalOpen() {
		t.Fatal("S on the Index tab did not open the search modal")
	}

	pressGlobal(app, tcell.KeyEscape, 0)

	if app.IsSearchModalOpen() {
		t.Fatal("Escape did not close the search modal")
	}
	arrowsStillMoveTheIndexSelection(t, app)
}

// TestCloseModals_KeepPositionsFocusOnTheDefaultTab guards the original
// behaviour: on the Positions tab every overlay still hands focus back there.
func TestCloseModals_KeepPositionsFocusOnTheDefaultTab(t *testing.T) {
	app := indexAppWithPages(t)
	app.portfolioView.TabbedView.SetTab(TabPositions)

	for name, closeFn := range map[string]func(){
		"order":  app.CloseOrderModal,
		"close":  app.CloseCloseModal,
		"search": app.CloseSearchModal,
	} {
		app.app.SetFocus(app.orderModal.Form)
		closeFn()
		if app.app.GetFocus() != app.portfolioView.TabbedView.PositionsTable {
			t.Errorf("closing the %s modal did not return focus to the Positions table", name)
		}
	}
}

// TestTabKey_FocusesTheIndexTable covers the other place that enumerated tabs
// by hand: Tab from the account table has to reach the Index table too.
func TestTabKey_FocusesTheIndexTable(t *testing.T) {
	app := indexAppWithPages(t)
	app.app.SetFocus(app.portfolioView.AccountTable)

	pressGlobal(app, tcell.KeyTab, 0)

	if app.app.GetFocus() != app.portfolioView.TabbedView.IndexTable {
		t.Error("Tab from the account table did not focus the Index table")
	}
}

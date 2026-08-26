package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// TestTabbedView_IndexIsTheFourthTab verifies the Index tab exists as a page of
// its own and that switching to it shows the index table.
func TestTabbedView_IndexIsTheFourthTab(t *testing.T) {
	tv := NewTabbedView()

	if tv.IndexTable == nil {
		t.Fatal("TabbedView.IndexTable is nil")
	}

	tv.SetTab(TabIndex)

	if tv.ActiveTab != TabIndex {
		t.Errorf("ActiveTab = %v, want TabIndex", tv.ActiveTab)
	}
	name, _ := tv.Content.GetFrontPage()
	if name != "index" {
		t.Errorf("front page = %q, want \"index\"", name)
	}
}

// TestTabbedView_HeaderShowsIndexTab verifies the header renders all four tabs
// and highlights Index when it is active.
func TestTabbedView_HeaderShowsIndexTab(t *testing.T) {
	tv := NewTabbedView()
	tv.SetTab(TabIndex)

	header := tv.Header.GetText(false)
	for _, tab := range []string{" Positions ", " History ", " Orders ", " Index "} {
		if !strings.Contains(header, tab) {
			t.Errorf("header %q does not contain %q", header, tab)
		}
	}
	// The active tab is the one drawn on a yellow background.
	if !strings.Contains(header, "[black:yellow] Index [-]") {
		t.Errorf("header %q does not highlight the Index tab", header)
	}
}

// TestInputHandler_TabCycleIncludesIndex verifies the forward cycle visits all
// four tabs and wraps back to Positions, with focus following the tab.
func TestInputHandler_TabCycleIncludesIndex(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	setupInputHandlers(app)
	app.app.SetFocus(app.portfolioView.TabbedView.PositionsTable)
	capture := app.app.GetInputCapture()

	tv := app.portfolioView.TabbedView
	want := []struct {
		tab   TabType
		table interface{}
	}{
		{TabHistory, tv.HistoryTable},
		{TabOrders, tv.OrdersTable},
		{TabIndex, tv.IndexTable},
		{TabPositions, tv.PositionsTable},
	}

	for i, step := range want {
		capture(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
		if tv.ActiveTab != step.tab {
			t.Fatalf("step %d: ActiveTab = %v, want %v", i, tv.ActiveTab, step.tab)
		}
		if app.app.GetFocus() != step.table {
			t.Errorf("step %d: focus did not follow the %v tab", i, step.tab)
		}
	}
}

// TestInputHandler_PrevTabReachesIndexFromPositions verifies the backward cycle
// wraps to Index rather than to Orders.
func TestInputHandler_PrevTabReachesIndexFromPositions(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	setupInputHandlers(app)
	app.app.SetFocus(app.portfolioView.TabbedView.PositionsTable)
	capture := app.app.GetInputCapture()

	capture(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))

	if got := app.portfolioView.TabbedView.ActiveTab; got != TabIndex {
		t.Errorf("ActiveTab = %v, want TabIndex after wrapping backwards", got)
	}
}

// TestInputHandler_IndexTableNavigatesTabs verifies the Index table carries the
// same navigation capture as the other tables, so ←/→ keep working from it.
func TestInputHandler_IndexTableNavigatesTabs(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	setupInputHandlers(app)

	capture := app.portfolioView.TabbedView.IndexTable.GetInputCapture()
	if capture == nil {
		t.Fatal("IndexTable has no input capture — setupTableNavigation was not applied")
	}

	app.portfolioView.TabbedView.SetTab(TabIndex)
	capture(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))

	if got := app.portfolioView.TabbedView.ActiveTab; got != TabPositions {
		t.Errorf("ActiveTab = %v, want TabPositions after → from Index", got)
	}
}

package ui

import (
	"strings"
	"testing"
	"time"

	"finam-terminal/models"

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

// TestInputHandler_EnteringIndexTabLoadsOnce verifies opening the tab triggers
// exactly one composition load and that coming back does not trigger another.
func TestInputHandler_EnteringIndexTabLoadsOnce(t *testing.T) {
	loaded := make(chan struct{}, 4)
	mock := &mockClient{
		GetIndexConstituentsFunc: func(string) ([]models.IndexConstituent, error) {
			loaded <- struct{}{}
			return testConstituents(), nil
		},
	}
	app := NewApp(mock, nil)
	setupInputHandlers(app)
	app.app.SetFocus(app.portfolioView.TabbedView.PositionsTable)
	capture := app.app.GetInputCapture()

	// Positions → History → Orders → Index
	for range 3 {
		capture(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	}

	select {
	case <-loaded:
	case <-time.After(2 * time.Second):
		t.Fatal("entering the Index tab did not load the composition")
	}

	// Wait for the load to be recorded before leaving and re-entering.
	deadline := time.Now().Add(2 * time.Second)
	for {
		app.dataMutex.RLock()
		done := app.indexLoaded
		app.dataMutex.RUnlock()
		if done || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	capture(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)) // → Positions
	capture(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))  // → Index again

	if n := mock.GetIndexConstituentsCalls.Load(); n != 1 {
		t.Errorf("composition loaded %d times, want 1 — re-entering the tab must reuse it", n)
	}
}

// TestInputHandler_EnteringIndexTabShowsLoading verifies the tab says it is
// loading while the composition is being fetched. Drawing before starting the
// load painted "No constituents" and left it there for the whole fetch, which
// read as a broken tab.
func TestInputHandler_EnteringIndexTabShowsLoading(t *testing.T) {
	release := make(chan struct{})
	mock := &mockClient{
		GetIndexConstituentsFunc: func(string) ([]models.IndexConstituent, error) {
			<-release // hold the load open so the loading state is observable
			return testConstituents(), nil
		},
	}
	app := NewApp(mock, nil)
	setupInputHandlers(app)
	app.app.SetFocus(app.portfolioView.TabbedView.PositionsTable)
	capture := app.app.GetInputCapture()

	for range 3 {
		capture(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	}
	defer close(release)

	got := app.portfolioView.TabbedView.IndexTable.GetCell(1, 0).Text
	if !strings.Contains(got, "Loading") {
		t.Errorf("first row on entry = %q, want it to report loading", got)
	}
}

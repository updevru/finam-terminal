package ui

import (
	"slices"
	"sync"
	"testing"
	"time"

	"finam-terminal/models"
)

// TestShouldDisableIndexStream covers the detector that protects the positions
// stream: if the subscription stops working only after the index composition
// joined it, the composition is what gets dropped.
func TestShouldDisableIndexStream(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	longAgo := now.Add(-2 * time.Minute)
	recently := now.Add(-5 * time.Second)

	tests := []struct {
		name          string
		indexIncluded bool
		streamUp      bool
		includedAt    time.Time
		failures      int
		want          bool
	}{
		{name: "down past the window with the index included", indexIncluded: true, includedAt: longAgo, want: true},
		{name: "repeated failures with the index included", indexIncluded: true, includedAt: recently, failures: 3, want: true},
		{name: "still inside the window", indexIncluded: true, includedAt: recently, failures: 1},
		{name: "stream is up", indexIncluded: true, streamUp: true, includedAt: longAgo, failures: 5},
		{name: "index not in the subscription", indexIncluded: false, includedAt: longAgo, failures: 5},
		{name: "index never joined", indexIncluded: true, includedAt: time.Time{}, failures: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldDisableIndexStream(tt.indexIncluded, tt.streamUp, tt.includedAt, tt.failures, now)
			if got != tt.want {
				t.Errorf("shouldDisableIndexStream() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIndexStreamGuard_RestoresPositionSubscription verifies the recovery path
// end to end at the UI level: once the guard trips, the very next subscription
// carries the positions alone, so portfolio quotes come back.
func TestIndexStreamGuard_RestoresPositionSubscription(t *testing.T) {
	var declared []string
	mock := &mockClient{
		SetQuoteSymbolsFunc: func(symbols []string) { declared = symbols },
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{{Symbol: "SBER@MISX"}}
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)

	app.recomputeStreamSymbols()
	if len(declared) != 3 {
		t.Fatalf("initial subscription = %v, want the whole composition", declared)
	}

	// The stream never came up after the composition joined.
	app.dataMutex.Lock()
	app.indexStreamIncludedAt = time.Now().Add(-2 * time.Minute)
	app.dataMutex.Unlock()
	app.streamLive.Store(false)

	app.evaluateIndexStreamHealth()

	app.dataMutex.RLock()
	disabled := app.indexStreamDisabled
	app.dataMutex.RUnlock()
	if !disabled {
		t.Fatal("the guard did not trip after the window elapsed")
	}
	if !slices.Equal(declared, []string{"SBER@MISX"}) {
		t.Errorf("subscription after the guard tripped = %v, want the positions alone", declared)
	}

	// The exclusion is for the session: re-entering the tab must not re-add them.
	app.recomputeStreamSymbols()
	if !slices.Equal(declared, []string{"SBER@MISX"}) {
		t.Errorf("subscription after a recompute = %v, want the positions alone", declared)
	}
}

// TestIndexStreamGuard_LiveStreamIsLeftAlone verifies a working stream is never
// stripped of the composition, however long it has been subscribed.
func TestIndexStreamGuard_LiveStreamIsLeftAlone(t *testing.T) {
	var declared []string
	mock := &mockClient{
		SetQuoteSymbolsFunc: func(symbols []string) { declared = symbols },
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{{Symbol: "SBER@MISX"}}
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.recomputeStreamSymbols()

	app.dataMutex.Lock()
	app.indexStreamIncludedAt = time.Now().Add(-10 * time.Minute)
	app.dataMutex.Unlock()
	app.streamLive.Store(true)

	app.evaluateIndexStreamHealth()

	app.dataMutex.RLock()
	disabled := app.indexStreamDisabled
	app.dataMutex.RUnlock()
	if disabled {
		t.Error("the guard tripped on a healthy stream")
	}
	if len(declared) != 3 {
		t.Errorf("subscription = %v, want the composition still included", declared)
	}
}

// TestIndexStreamGuard_ClockStartsWhenTheIndexJoins verifies the window is
// measured from the moment the composition entered the subscription, not from
// application start — otherwise a long session would trip the guard instantly.
func TestIndexStreamGuard_ClockStartsWhenTheIndexJoins(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{{Symbol: "SBER@MISX"}}
	app.indexConstituents = testConstituents()
	app.indexLoaded = true

	// Index tab closed: the composition is not in the subscription.
	app.recomputeStreamSymbols()
	app.dataMutex.RLock()
	before := app.indexStreamIncludedAt
	app.dataMutex.RUnlock()
	if !before.IsZero() {
		t.Fatalf("clock started while the composition was not subscribed (%v)", before)
	}

	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.recomputeStreamSymbols()

	app.dataMutex.RLock()
	after := app.indexStreamIncludedAt
	app.dataMutex.RUnlock()
	if after.IsZero() {
		t.Error("clock did not start when the composition joined the subscription")
	}

	// Leaving the tab stops the clock again.
	app.portfolioView.TabbedView.SetTab(TabPositions)
	app.recomputeStreamSymbols()
	app.dataMutex.RLock()
	cleared := app.indexStreamIncludedAt
	app.dataMutex.RUnlock()
	if !cleared.IsZero() {
		t.Errorf("clock still running after leaving the tab (%v)", cleared)
	}
}

// TestOnStreamState_CountsFailuresWhileIndexSubscribed verifies a flapping
// stream is counted, so the guard also catches a subscription that connects and
// immediately drops rather than never connecting at all.
func TestOnStreamState_CountsFailuresWhileIndexSubscribed(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.recomputeStreamSymbols()

	app.onStreamState(false)
	app.onStreamState(false)

	app.dataMutex.RLock()
	failures := app.indexStreamFailures
	app.dataMutex.RUnlock()
	if failures != 2 {
		t.Errorf("failure count = %d, want 2", failures)
	}

	// A healthy stream clears the suspicion.
	app.onStreamState(true)
	app.dataMutex.RLock()
	failures = app.indexStreamFailures
	app.dataMutex.RUnlock()
	if failures != 0 {
		t.Errorf("failure count after the stream came up = %d, want 0", failures)
	}
}

// TestIndexState_ConcurrentAccess hammers every path that touches the Index tab
// state from several goroutines at once. It is the local stand-in for the race
// detector on machines without a C toolchain: Go's runtime still panics on a
// concurrent map write, which is the failure mode an unguarded indexQuotes or
// indexConstituents would produce.
//
// The Index tab is kept inactive and the inbox flush is left until the end,
// because both redraw tview widgets, which only ever run on the event loop. The
// point here is the shared state behind them, not the drawing.
func TestIndexState_ConcurrentAccess(t *testing.T) {
	mock := &mockClient{
		GetIndexConstituentsFunc: func(string) ([]models.IndexConstituent, error) {
			return testConstituents(), nil
		},
		GetQuotesFunc: func(_ string, symbols []string) (map[string]*models.Quote, error) {
			out := make(map[string]*models.Quote, len(symbols))
			for _, s := range symbols {
				out[s] = &models.Quote{Symbol: s, Last: "100.00", Change: "1.00"}
			}
			return out, nil
		},
		SetQuoteSymbolsFunc: func([]string) {},
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{{Symbol: "SBER@MISX"}}

	const workers = 8
	const rounds = 40

	var wg sync.WaitGroup
	run := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range rounds {
				f()
			}
		}()
	}

	for range workers {
		run(func() { app.loadIndexSync() })
		run(func() { app.pollIndexQuotesSync(true, false, 0) })
		run(func() { _ = app.indexSymbols() })
		run(func() { app.recomputeStreamSymbols() })
		run(func() { app.evaluateIndexStreamHealth() })
		run(func() { app.onStreamState(false) })
		run(func() { app.onStreamQuote(models.Quote{Symbol: "GAZP@MISX", Last: "290.00"}) })
	}

	wg.Wait()

	// The flush is serial on purpose: it repaints tview widgets, which are
	// owned by the event loop. Its shared-state bookkeeping is what the
	// concurrent writers above have been feeding.
	app.flushQuoteInbox()

	app.dataMutex.RLock()
	defer app.dataMutex.RUnlock()
	if len(app.indexConstituents) != 3 {
		t.Errorf("composition ended with %d constituents, want 3", len(app.indexConstituents))
	}
}

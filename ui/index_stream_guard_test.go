package ui

import (
	"slices"
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

package ui

import (
	"errors"
	"os"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"finam-terminal/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestShouldPollIndexQuotes covers the decision matrix for the automatic sweep.
//
// Stream liveness is deliberately absent: the broker caps a subscription at
// around ten symbols, so a live stream cannot cover a 46-name index and the
// sweep still has gaps to fill. What bounds the cost instead is the cooldown,
// the tab being on screen, and the rate-limit latch.
func TestShouldPollIndexQuotes(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	longAgo := now.Add(-5 * time.Minute)
	justNow := now.Add(-5 * time.Second)

	tests := []struct {
		name         string
		tabActive    bool
		autoDisabled bool
		lastPoll     time.Time
		want         bool
	}{
		{name: "tab open, cooled down", tabActive: true, lastPoll: longAgo, want: true},
		{name: "never swept yet", tabActive: true, lastPoll: time.Time{}, want: true},
		{name: "tab closed", tabActive: false, lastPoll: longAgo},
		{name: "still inside the cooldown", tabActive: true, lastPoll: justNow},
		{name: "disabled after a rate limit", tabActive: true, autoDisabled: true, lastPoll: longAgo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldPollIndexQuotes(tt.tabActive, tt.autoDisabled, tt.lastPoll, now)
			if got != tt.want {
				t.Errorf("shouldPollIndexQuotes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIndexPollCooldown documents the spacing of automatic sweeps: one pass over
// the composition per minute is 46 requests a minute against a 200/min budget.
func TestIndexPollCooldown(t *testing.T) {
	if indexPollCooldown != time.Minute {
		t.Errorf("indexPollCooldown = %v, want 1m", indexPollCooldown)
	}
}

// TestUncoveredIndexSymbols verifies the sweep asks only for the rows the stream
// is not already carrying — the whole point of pairing the two sources.
func TestUncoveredIndexSymbols(t *testing.T) {
	constituents := sortConstituents(testConstituents())

	got := uncoveredIndexSymbols(constituents, []string{"SBER@MISX", "IRRELEVANT@MISX"})
	want := []string{"GAZP@MISX", "LKOH@MISX"}
	if !slices.Equal(got, want) {
		t.Errorf("uncovered = %v, want %v", got, want)
	}

	all := []string{"SBER@MISX", "GAZP@MISX", "LKOH@MISX"}
	if got := uncoveredIndexSymbols(constituents, all); len(got) != 0 {
		t.Errorf("with everything subscribed, uncovered = %v, want none", got)
	}
	if got := uncoveredIndexSymbols(constituents, nil); len(got) != 3 {
		t.Errorf("with nothing subscribed, uncovered = %v, want all 3", got)
	}
}

// TestSweepIndexQuotes_FillsOnlyTheGaps verifies the sweep fetches exactly the
// rows the stream leaves out, and stores them.
func TestSweepIndexQuotes_FillsOnlyTheGaps(t *testing.T) {
	var asked []string
	mock := &mockClient{
		SubscribedSymbolsFunc: func() []string { return []string{"SBER@MISX"} },
		GetQuotesFunc: func(_ string, symbols []string) (map[string]*models.Quote, error) {
			asked = append(asked, symbols...)
			out := make(map[string]*models.Quote, len(symbols))
			for _, s := range symbols {
				out[s] = &models.Quote{Symbol: s, Last: "100.00", Change: "1.00"}
			}
			return out, nil
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)

	if !app.sweepIndexQuotes(true, true) {
		t.Fatal("sweep reported that it did nothing")
	}

	want := []string{"GAZP@MISX", "LKOH@MISX"}
	if !slices.Equal(asked, want) {
		t.Errorf("sweep asked for %v, want %v (SBER is already on the stream)", asked, want)
	}

	app.dataMutex.RLock()
	defer app.dataMutex.RUnlock()
	for _, s := range want {
		if app.indexQuotes[s] == nil {
			t.Errorf("no quote stored for %s", s)
		}
	}
	if app.indexQuotes["SBER@MISX"] != nil {
		t.Error("the sweep overwrote a symbol the stream owns")
	}
}

// TestSweepIndexQuotes_OneSymbolPerRequest verifies the sweep is paced: the
// broker refuses a burst long before the per-minute budget is reached, so the
// requests go out one at a time rather than as a single batch.
func TestSweepIndexQuotes_OneSymbolPerRequest(t *testing.T) {
	var sizes []int
	mock := &mockClient{
		GetQuotesFunc: func(_ string, symbols []string) (map[string]*models.Quote, error) {
			sizes = append(sizes, len(symbols))
			return nil, nil
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true

	app.sweepIndexQuotes(true, true)

	if len(sizes) != 3 {
		t.Fatalf("made %d requests, want one per uncovered symbol (3)", len(sizes))
	}
	for i, n := range sizes {
		if n != 1 {
			t.Errorf("request %d carried %d symbols, want 1", i, n)
		}
	}
}

// TestSweepIndexQuotes_NothingUncoveredCostsNothing verifies the promise that
// survives the broker's cap: when the stream happens to carry the whole
// composition, the tab makes no unary calls at all.
func TestSweepIndexQuotes_NothingUncoveredCostsNothing(t *testing.T) {
	var calls atomic.Int64
	mock := &mockClient{
		SubscribedSymbolsFunc: func() []string {
			return []string{"SBER@MISX", "GAZP@MISX", "LKOH@MISX"}
		},
		GetQuotesFunc: func(string, []string) (map[string]*models.Quote, error) {
			calls.Add(1)
			return nil, nil
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true

	if app.sweepIndexQuotes(true, true) {
		t.Error("sweep reported work with nothing to fetch")
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("made %d request(s) with the whole composition on the stream, want 0", n)
	}
}

// TestSweepIndexQuotes_RespectsCooldown verifies automatic sweeps are spaced and
// that a manual refresh ignores the spacing.
func TestSweepIndexQuotes_RespectsCooldown(t *testing.T) {
	var calls atomic.Int64
	mock := &mockClient{
		GetQuotesFunc: func(string, []string) (map[string]*models.Quote, error) {
			calls.Add(1)
			return nil, nil
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.indexLastPoll = time.Now()

	if app.sweepIndexQuotes(false, true) {
		t.Fatal("an automatic sweep ran inside the cooldown")
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("made %d request(s) inside the cooldown, want 0", n)
	}

	app.sweepIndexQuotes(true, true)
	if n := calls.Load(); n == 0 {
		t.Error("a manual refresh was blocked by the cooldown")
	}
}

// TestSweepIndexQuotes_RateLimitStopsTheSweep verifies a rate limit ends the
// current pass immediately rather than walking the remaining symbols, and
// latches automatic refresh off for the session.
func TestSweepIndexQuotes_RateLimitStopsTheSweep(t *testing.T) {
	var calls atomic.Int64
	mock := &mockClient{
		GetQuotesFunc: func(string, []string) (map[string]*models.Quote, error) {
			calls.Add(1)
			return nil, status.Error(codes.ResourceExhausted, "Too Many Requests")
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true

	app.sweepIndexQuotes(true, true)

	if n := calls.Load(); n != 1 {
		t.Errorf("made %d request(s) after a rate limit, want 1 — the sweep must stop", n)
	}

	app.dataMutex.RLock()
	disabled := app.indexPollDisabled
	app.dataMutex.RUnlock()
	if !disabled {
		t.Error("a rate limit must disable automatic sweeps for the session")
	}
	if shouldPollIndexQuotes(true, disabled, time.Time{}, time.Now()) {
		t.Error("automatic sweeps are still allowed after a rate limit")
	}
}

// TestSweepIndexQuotes_OrdinaryErrorKeepsGoing verifies one bad symbol does not
// abandon the rest of the pass, and does not cost the user the feature.
func TestSweepIndexQuotes_OrdinaryErrorKeepsGoing(t *testing.T) {
	var calls atomic.Int64
	mock := &mockClient{
		GetQuotesFunc: func(_ string, symbols []string) (map[string]*models.Quote, error) {
			calls.Add(1)
			if symbols[0] == "GAZP@MISX" {
				return nil, errors.New("temporary failure")
			}
			return map[string]*models.Quote{symbols[0]: {Symbol: symbols[0], Last: "1"}}, nil
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true

	app.sweepIndexQuotes(true, true)

	if n := calls.Load(); n != 3 {
		t.Errorf("made %d request(s), want all 3 — one failure must not end the pass", n)
	}

	app.dataMutex.RLock()
	defer app.dataMutex.RUnlock()
	if app.indexPollDisabled {
		t.Error("an ordinary error must not disable automatic sweeps")
	}
	if app.indexQuotes["LKOH@MISX"] == nil {
		t.Error("symbols after the failing one were not fetched")
	}
}

// TestSweepIndexQuotes_NoCompositionNoCall verifies nothing is requested before
// the composition is known.
func TestSweepIndexQuotes_NoCompositionNoCall(t *testing.T) {
	var calls atomic.Int64
	mock := &mockClient{
		GetQuotesFunc: func(string, []string) (map[string]*models.Quote, error) {
			calls.Add(1)
			return nil, nil
		},
	}
	app := NewApp(mock, nil)

	app.sweepIndexQuotes(true, true)

	if n := calls.Load(); n != 0 {
		t.Errorf("made %d request(s) with no composition, want 0", n)
	}
}

// TestMain drops the sweep pacing: the delay exists to be gentle on the broker,
// and waiting it out would make the suite take minutes.
func TestMain(m *testing.M) {
	indexSweepDelay = 0
	os.Exit(m.Run())
}

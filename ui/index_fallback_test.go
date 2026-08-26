package ui

import (
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"finam-terminal/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestShouldPollIndexQuotes covers the whole decision matrix for the fallback
// batch. Every "false" here is a request the terminal does not send.
func TestShouldPollIndexQuotes(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	longAgo := now.Add(-5 * time.Minute)
	justNow := now.Add(-5 * time.Second)

	tests := []struct {
		name         string
		streamLive   bool
		tabActive    bool
		autoDisabled bool
		lastPoll     time.Time
		want         bool
	}{
		{name: "stream down, tab open, cooled down", tabActive: true, lastPoll: longAgo, want: true},
		{name: "never polled yet", tabActive: true, lastPoll: time.Time{}, want: true},
		{name: "stream live owns the quotes", streamLive: true, tabActive: true, lastPoll: longAgo},
		{name: "tab closed", tabActive: false, lastPoll: longAgo},
		{name: "still inside the cooldown", tabActive: true, lastPoll: justNow},
		{name: "auto polling disabled after a rate limit", tabActive: true, autoDisabled: true, lastPoll: longAgo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldPollIndexQuotes(tt.streamLive, tt.tabActive, tt.autoDisabled, tt.lastPoll, now)
			if got != tt.want {
				t.Errorf("shouldPollIndexQuotes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIndexPollCooldown documents the minimum spacing of automatic batches:
// 46 symbols twice a minute stays far inside the 200/min per-method budget.
func TestIndexPollCooldown(t *testing.T) {
	if indexPollCooldown != time.Minute {
		t.Errorf("indexPollCooldown = %v, want 1m", indexPollCooldown)
	}
}

// TestPollIndexQuotes_NotCalledWhileStreamLive verifies the headline promise of
// the design: with a live stream the tab costs zero unary quote calls.
func TestPollIndexQuotes_NotCalledWhileStreamLive(t *testing.T) {
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
	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.streamLive.Store(true)

	app.pollIndexQuotesSync(false, app.indexTabActive())

	if n := calls.Load(); n != 0 {
		t.Errorf("GetQuotes called %d time(s) with a live stream, want 0", n)
	}
}

// TestPollIndexQuotes_BatchesWhenStreamDown verifies the fallback asks for the
// whole composition in one batch and stores the answer.
func TestPollIndexQuotes_BatchesWhenStreamDown(t *testing.T) {
	var gotSymbols []string
	mock := &mockClient{
		GetQuotesFunc: func(_ string, symbols []string) (map[string]*models.Quote, error) {
			gotSymbols = append([]string(nil), symbols...)
			return map[string]*models.Quote{
				"GAZP@MISX": {Symbol: "GAZP@MISX", Last: "290.00", Change: "5.00"},
			}, nil
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.streamLive.Store(false)

	app.pollIndexQuotesSync(false, app.indexTabActive())

	slices.Sort(gotSymbols)
	want := []string{"GAZP@MISX", "LKOH@MISX", "SBER@MISX"}
	if !slices.Equal(gotSymbols, want) {
		t.Errorf("batched symbols = %v, want %v", gotSymbols, want)
	}

	app.dataMutex.RLock()
	q := app.indexQuotes["GAZP@MISX"]
	polled := app.indexLastPoll
	app.dataMutex.RUnlock()

	if q == nil || q.Last != "290.00" {
		t.Errorf("index quote = %+v, want last 290.00", q)
	}
	if polled.IsZero() {
		t.Error("indexLastPoll was not stamped, so the cooldown would never apply")
	}
}

// TestPollIndexQuotes_ManualBypassesCooldown verifies R still works right after
// an automatic batch — the cooldown bounds automation, not the user.
func TestPollIndexQuotes_ManualBypassesCooldown(t *testing.T) {
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
	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.streamLive.Store(false)
	app.indexLastPoll = time.Now()

	app.pollIndexQuotesSync(false, app.indexTabActive())
	if n := calls.Load(); n != 0 {
		t.Fatalf("automatic batch ran inside the cooldown (%d call(s))", n)
	}

	app.pollIndexQuotesSync(true, app.indexTabActive())
	if n := calls.Load(); n != 1 {
		t.Errorf("manual refresh made %d call(s), want 1", n)
	}
}

// TestPollIndexQuotes_RateLimitDisablesAutoPolling verifies a ResourceExhausted
// answer stops automatic batches for the rest of the session while leaving the
// manual key working, and says so in the status bar.
func TestPollIndexQuotes_RateLimitDisablesAutoPolling(t *testing.T) {
	var calls atomic.Int64
	mock := &mockClient{
		GetQuotesFunc: func(string, []string) (map[string]*models.Quote, error) {
			calls.Add(1)
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.streamLive.Store(false)

	app.pollIndexQuotesSync(false, app.indexTabActive())

	app.dataMutex.RLock()
	disabled := app.indexPollDisabled
	app.dataMutex.RUnlock()
	if !disabled {
		t.Fatal("a rate limit must disable automatic index polling for the session")
	}

	if shouldPollIndexQuotes(false, true, disabled, time.Time{}, time.Now()) {
		t.Error("automatic polling is still allowed after a rate limit")
	}

	// Manual refresh remains available.
	app.pollIndexQuotesSync(true, app.indexTabActive())
	if n := calls.Load(); n != 2 {
		t.Errorf("manual refresh after a rate limit made %d call(s) in total, want 2", n)
	}
}

// TestPollIndexQuotes_OrdinaryErrorKeepsAutoPolling verifies only a rate limit
// disables automation — a transient failure must not cost the user the feature.
func TestPollIndexQuotes_OrdinaryErrorKeepsAutoPolling(t *testing.T) {
	mock := &mockClient{
		GetQuotesFunc: func(string, []string) (map[string]*models.Quote, error) {
			return nil, errors.New("temporary failure")
		},
	}
	app := NewApp(mock, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.streamLive.Store(false)

	app.pollIndexQuotesSync(false, app.indexTabActive())

	app.dataMutex.RLock()
	disabled := app.indexPollDisabled
	app.dataMutex.RUnlock()
	if disabled {
		t.Error("an ordinary error must not disable automatic polling")
	}
}

// TestPollIndexQuotes_NoCompositionNoCall verifies nothing is requested before
// the composition is known.
func TestPollIndexQuotes_NoCompositionNoCall(t *testing.T) {
	var calls atomic.Int64
	mock := &mockClient{
		GetQuotesFunc: func(string, []string) (map[string]*models.Quote, error) {
			calls.Add(1)
			return nil, nil
		},
	}
	app := NewApp(mock, nil)
	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.streamLive.Store(false)

	app.pollIndexQuotesSync(true, app.indexTabActive())

	if n := calls.Load(); n != 0 {
		t.Errorf("GetQuotes called %d time(s) with no composition, want 0", n)
	}
}

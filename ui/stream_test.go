package ui

import (
	"testing"
	"time"

	"finam-terminal/models"
)

func TestComputeStreamSymbols(t *testing.T) {
	tests := []struct {
		name          string
		positions     []models.Position
		profileOpen   bool
		profileSymbol string
		want          []string
	}{
		{
			name: "positions only, sorted",
			positions: []models.Position{
				{Symbol: "SBER@TQBR"},
				{Symbol: "GAZP@TQBR"},
			},
			want: []string{"GAZP@TQBR", "SBER@TQBR"},
		},
		{
			name:          "profile symbol is added",
			positions:     []models.Position{{Symbol: "SBER@TQBR"}},
			profileOpen:   true,
			profileSymbol: "LKOH@TQBR",
			want:          []string{"LKOH@TQBR", "SBER@TQBR"},
		},
		{
			name:          "profile symbol already held",
			positions:     []models.Position{{Symbol: "SBER@TQBR"}},
			profileOpen:   true,
			profileSymbol: "SBER@TQBR",
			want:          []string{"SBER@TQBR"},
		},
		{
			name:          "closed profile contributes nothing",
			positions:     []models.Position{{Symbol: "SBER@TQBR"}},
			profileOpen:   false,
			profileSymbol: "LKOH@TQBR",
			want:          []string{"SBER@TQBR"},
		},
		{
			name:      "symbols without a MIC are filtered",
			positions: []models.Position{{Symbol: "SBER"}, {Symbol: "GAZP@TQBR"}},
			want:      []string{"GAZP@TQBR"},
		},
		{
			name:      "no positions, no profile",
			positions: nil,
			want:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The Index tab is covered separately in index_stream_test.go.
			got := computeStreamSymbols(tt.positions, tt.profileOpen, tt.profileSymbol, false, nil)
			if len(got) != len(tt.want) {
				t.Fatalf("computeStreamSymbols() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("computeStreamSymbols() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// TestFlushQuoteInbox_UpsertsQuotes verifies that a flush merges streamed quotes
// into the active account without discarding quotes it did not carry.
func TestFlushQuoteInbox_UpsertsQuotes(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.quotes["acc1"] = map[string]*models.Quote{
		"SBER@TQBR": {Symbol: "SBER@TQBR", Last: "280"},
		"GAZP@TQBR": {Symbol: "GAZP@TQBR", Last: "160"},
	}

	app.onStreamQuote(models.Quote{Symbol: "SBER@TQBR", Last: "291"})
	app.onStreamQuote(models.Quote{Symbol: "LKOH@TQBR", Last: "7100"})

	app.flushQuoteInbox()

	quotes := app.quotes["acc1"]
	if got := quotes["SBER@TQBR"].Last; got != "291" {
		t.Errorf("SBER last = %q, want 291 (updated by the stream)", got)
	}
	if got := quotes["GAZP@TQBR"].Last; got != "160" {
		t.Errorf("GAZP last = %q, want 160 (untouched)", got)
	}
	if quotes["LKOH@TQBR"] == nil || quotes["LKOH@TQBR"].Last != "7100" {
		t.Error("a symbol seen only on the stream should be inserted")
	}

	app.quoteInboxMu.Lock()
	inboxLen := len(app.quoteInbox)
	queued := app.quoteFlushQueued
	app.quoteInboxMu.Unlock()

	if inboxLen != 0 {
		t.Errorf("inbox holds %d entries after a flush, want 0", inboxLen)
	}
	if queued {
		t.Error("flush flag should be cleared after a flush")
	}
}

// TestOnStreamQuote_CoalescesFlushes verifies that a burst of quotes schedules a
// single redraw.
func TestOnStreamQuote_CoalescesFlushes(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0

	app.onStreamQuote(models.Quote{Symbol: "SBER@TQBR", Last: "291"})

	app.quoteInboxMu.Lock()
	queuedAfterFirst := app.quoteFlushQueued
	app.quoteInboxMu.Unlock()
	if !queuedAfterFirst {
		t.Fatal("the first quote should schedule a flush")
	}

	app.onStreamQuote(models.Quote{Symbol: "GAZP@TQBR", Last: "161"})
	app.onStreamQuote(models.Quote{Symbol: "SBER@TQBR", Last: "292"})

	app.quoteInboxMu.Lock()
	inbox := len(app.quoteInbox)
	last := app.quoteInbox["SBER@TQBR"].Last
	app.quoteInboxMu.Unlock()

	if inbox != 2 {
		t.Errorf("inbox holds %d symbols, want 2", inbox)
	}
	if last != "292" {
		t.Errorf("inbox SBER last = %q, want 292 (latest wins)", last)
	}
}

// TestOnStreamQuote_DropsAfterStop verifies that quotes arriving after Stop are
// discarded — queueing a redraw on a stopped application blocks forever.
func TestOnStreamQuote_DropsAfterStop(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.Stop()

	app.onStreamQuote(models.Quote{Symbol: "SBER@TQBR", Last: "291"})

	app.quoteInboxMu.Lock()
	defer app.quoteInboxMu.Unlock()
	if len(app.quoteInbox) != 0 {
		t.Error("quotes arriving after Stop should be dropped")
	}
}

// TestShouldSkipQuotePolling covers the ownership matrix: while the stream is
// live it owns the active account's quotes, and nothing else.
func TestShouldSkipQuotePolling(t *testing.T) {
	tests := []struct {
		name       string
		streamLive bool
		accountID  string
		want       bool
	}{
		{name: "stream live, active account", streamLive: true, accountID: "acc1", want: true},
		{name: "stream live, other account", streamLive: true, accountID: "acc2", want: false},
		{name: "stream down, active account", streamLive: false, accountID: "acc1", want: false},
		{name: "stream down, other account", streamLive: false, accountID: "acc2", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}, {ID: "acc2"}})
			app.selectedIdx = 0
			app.streamLive.Store(tt.streamLive)

			if got := app.shouldSkipQuotePolling(tt.accountID); got != tt.want {
				t.Errorf("shouldSkipQuotePolling(%q) = %v, want %v", tt.accountID, got, tt.want)
			}
		})
	}
}

// TestRecomputeStreamSymbols_UsesActiveAccount verifies that switching accounts
// re-declares the subscription with the new account's symbols.
func TestRecomputeStreamSymbols_UsesActiveAccount(t *testing.T) {
	var captured []string
	mock := &mockClient{
		SetQuoteSymbolsFunc: func(symbols []string) {
			captured = append([]string(nil), symbols...)
		},
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}, {ID: "acc2"}})
	app.positions["acc1"] = []models.Position{{Symbol: "SBER@TQBR"}}
	app.positions["acc2"] = []models.Position{{Symbol: "LKOH@TQBR"}}

	app.selectedIdx = 1
	app.recomputeStreamSymbols()

	if len(captured) != 1 || captured[0] != "LKOH@TQBR" {
		t.Errorf("declared symbols = %v, want [LKOH@TQBR]", captured)
	}
}

// TestApplyAccountData_KeepsStreamQuotes verifies that a poll which skipped
// quotes does not wipe the quotes the stream delivered.
func TestApplyAccountData_KeepsStreamQuotes(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.quotes["acc1"] = map[string]*models.Quote{
		"SBER@TQBR": {Symbol: "SBER@TQBR", Last: "291"},
	}

	positions := []models.Position{{Symbol: "SBER@TQBR", Ticker: "SBER", Quantity: "100"}}
	app.applyAccountData("acc1", positions, nil, nil)

	quotes := app.quotes["acc1"]
	if quotes["SBER@TQBR"] == nil || quotes["SBER@TQBR"].Last != "291" {
		t.Error("skipped quote polling must not replace the streamed quotes")
	}
}

// TestApplyAccountData_PolledQuotesWin verifies that a normal poll still
// replaces the quote map wholesale.
func TestApplyAccountData_PolledQuotesWin(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.quotes["acc1"] = map[string]*models.Quote{
		"SBER@TQBR": {Symbol: "SBER@TQBR", Last: "291"},
	}

	polled := map[string]*models.Quote{"SBER@TQBR": {Symbol: "SBER@TQBR", Last: "285"}}
	app.applyAccountData("acc1", nil, polled, nil)

	if got := app.quotes["acc1"]["SBER@TQBR"].Last; got != "285" {
		t.Errorf("quote last = %q, want 285 from the poller", got)
	}
}

// TestLoadDataAsync_SkipsQuotesWhileStreamLive verifies the fallback switch: the
// poller leaves the active account's quotes alone while the stream is live, and
// resumes as soon as it drops.
func TestLoadDataAsync_SkipsQuotesWhileStreamLive(t *testing.T) {
	quoteCalls := make(chan string, 4)
	mock := &mockClient{
		GetAccountDetailsFunc: func(accountID string) (*models.AccountInfo, []models.Position, error) {
			return &models.AccountInfo{ID: accountID}, []models.Position{{Symbol: "SBER@TQBR"}}, nil
		},
		GetQuotesFunc: func(accountID string, symbols []string) (map[string]*models.Quote, error) {
			quoteCalls <- accountID
			return map[string]*models.Quote{}, nil
		},
	}

	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}, {ID: "acc2"}})
	app.selectedIdx = 0
	app.streamLive.Store(true)

	app.loadDataAsync("acc1")
	select {
	case got := <-quoteCalls:
		t.Fatalf("quotes polled for %q while the stream owns the active account", got)
	case <-time.After(500 * time.Millisecond):
	}

	// Inactive accounts keep polling.
	app.loadDataAsync("acc2")
	select {
	case got := <-quoteCalls:
		if got != "acc2" {
			t.Errorf("polled quotes for %q, want acc2", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the inactive account must keep polling quotes")
	}

	// Stream down: the next tick polls the active account again.
	app.streamLive.Store(false)
	app.loadDataAsync("acc1")
	select {
	case got := <-quoteCalls:
		if got != "acc1" {
			t.Errorf("polled quotes for %q, want acc1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("polling must resume for the active account once the stream is down")
	}
}

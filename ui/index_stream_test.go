package ui

import (
	"slices"
	"testing"

	"finam-terminal/models"
)

// TestComputeStreamSymbols_WithIndex verifies that the index composition joins
// the subscription only while its tab is active, and leaves it again when the
// user navigates away — the whole point of the "extend the existing stream"
// design is that idle tabs cost nothing.
func TestComputeStreamSymbols_WithIndex(t *testing.T) {
	positions := []models.Position{{Symbol: "SBER@MISX"}}
	indexSymbols := []string{"GAZP@MISX", "LKOH@MISX", "SBER@MISX"}

	tests := []struct {
		name          string
		indexActive   bool
		indexSymbols  []string
		profileOpen   bool
		profileSymbol string
		want          []string
	}{
		{
			name:         "index tab inactive contributes nothing",
			indexActive:  false,
			indexSymbols: indexSymbols,
			want:         []string{"SBER@MISX"},
		},
		{
			// Priority order: the position first, then the composition. The
			// broker caps subscription size and the client truncates from the
			// end, so portfolio symbols must never be the ones dropped.
			name:         "index tab active adds the composition after the positions",
			indexActive:  true,
			indexSymbols: indexSymbols,
			want:         []string{"SBER@MISX", "GAZP@MISX", "LKOH@MISX"},
		},
		{
			name:          "index tab active with an open profile",
			indexActive:   true,
			indexSymbols:  indexSymbols,
			profileOpen:   true,
			profileSymbol: "MOEX@MISX",
			want:          []string{"SBER@MISX", "MOEX@MISX", "GAZP@MISX", "LKOH@MISX"},
		},
		{
			name:         "index symbols without a MIC are filtered",
			indexActive:  true,
			indexSymbols: []string{"GAZP", "LKOH@MISX"},
			want:         []string{"SBER@MISX", "LKOH@MISX"},
		},
		{
			name:         "index tab active before the composition loads",
			indexActive:  true,
			indexSymbols: nil,
			want:         []string{"SBER@MISX"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeStreamSymbols(positions, tt.profileOpen, tt.profileSymbol, tt.indexActive, tt.indexSymbols)
			if !slices.Equal(got, tt.want) {
				t.Errorf("computeStreamSymbols() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRecomputeStreamSymbols_FollowsIndexTab verifies the live subscription
// actually grows when the Index tab opens and shrinks back when it closes.
func TestRecomputeStreamSymbols_FollowsIndexTab(t *testing.T) {
	var declared []string
	mock := &mockClient{
		SetQuoteSymbolsFunc: func(symbols []string) { declared = symbols },
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{{Symbol: "SBER@MISX"}}
	app.indexConstituents = testConstituents()
	app.indexLoaded = true

	app.recomputeStreamSymbols()
	if !slices.Equal(declared, []string{"SBER@MISX"}) {
		t.Fatalf("with the Index tab closed: %v, want only the position", declared)
	}

	app.portfolioView.TabbedView.SetTab(TabIndex)
	app.recomputeStreamSymbols()
	want := []string{"SBER@MISX", "GAZP@MISX", "LKOH@MISX"}
	if !slices.Equal(declared, want) {
		t.Fatalf("with the Index tab open: %v, want %v (position first)", declared, want)
	}

	app.portfolioView.TabbedView.SetTab(TabPositions)
	app.recomputeStreamSymbols()
	if !slices.Equal(declared, []string{"SBER@MISX"}) {
		t.Errorf("after leaving the Index tab: %v, want only the position back", declared)
	}
}

// TestFlushQuoteInbox_FeedsIndexQuotes verifies index quotes are stored
// independently of the selected account: the tab shows the same data whichever
// account is active, and works with no account at all.
func TestFlushQuoteInbox_FeedsIndexQuotes(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)

	app.onStreamQuote(models.Quote{Symbol: "GAZP@MISX", Last: "290.00", Change: "5.00"})
	app.onStreamQuote(models.Quote{Symbol: "SBER@MISX", Last: "310.00", Change: "-1.00"})
	app.flushQuoteInbox()

	app.dataMutex.RLock()
	gazp := app.indexQuotes["GAZP@MISX"]
	sber := app.indexQuotes["SBER@MISX"]
	app.dataMutex.RUnlock()

	if gazp == nil || gazp.Last != "290.00" || gazp.Change != "5.00" {
		t.Errorf("GAZP index quote = %+v, want last 290.00 / change 5.00", gazp)
	}
	if sber == nil || sber.Last != "310.00" {
		t.Errorf("SBER index quote = %+v, want last 310.00", sber)
	}

	// The rendered table must show them, not placeholders.
	updateIndexTable(app)
	if got := app.portfolioView.TabbedView.IndexTable.GetCell(1, 2).Text; got != "290.00" {
		t.Errorf("rendered price = %q, want 290.00", got)
	}
}

// TestFlushQuoteInbox_IndexQuotesIgnoreAccount verifies index quotes land even
// when no account is selected, since the tab does not belong to an account.
func TestFlushQuoteInbox_IndexQuotesIgnoreAccount(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true

	app.onStreamQuote(models.Quote{Symbol: "GAZP@MISX", Last: "290.00"})
	app.flushQuoteInbox()

	app.dataMutex.RLock()
	got := app.indexQuotes["GAZP@MISX"]
	app.dataMutex.RUnlock()

	if got == nil {
		t.Fatal("index quote was dropped because no account is selected")
	}
	if got.Last != "290.00" {
		t.Errorf("index quote last = %q, want 290.00", got.Last)
	}
}

// TestFlushQuoteInbox_OnlyKeepsIndexSymbols verifies the index map does not
// accumulate everything on the stream — position symbols outside the
// composition belong to the account map only.
func TestFlushQuoteInbox_OnlyKeepsIndexSymbols(t *testing.T) {
	app := NewApp(&mockClient{}, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.indexConstituents = testConstituents()
	app.indexLoaded = true

	app.onStreamQuote(models.Quote{Symbol: "YDEX@MISX", Last: "4000"})
	app.flushQuoteInbox()

	app.dataMutex.RLock()
	_, inIndex := app.indexQuotes["YDEX@MISX"]
	_, inAccount := app.quotes["acc1"]["YDEX@MISX"]
	app.dataMutex.RUnlock()

	if inIndex {
		t.Error("a symbol outside the composition must not enter the index quote map")
	}
	if !inAccount {
		t.Error("the account quote map should still receive it")
	}
}

// TestRecomputeStreamSymbols_CarriesTheWholeComposition verifies the tab now
// subscribes to every constituent rather than a scrolling window: the broker's
// limit is per subscription, so 46 symbols cost four parallel streams and the
// set no longer changes as the user scrolls.
func TestRecomputeStreamSymbols_CarriesTheWholeComposition(t *testing.T) {
	var declared []string
	mock := &mockClient{
		SetQuoteSymbolsFunc: func(symbols []string) { declared = append([]string(nil), symbols...) },
	}
	app := NewApp(mock, nil)
	app.indexConstituents = manyConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)

	app.recomputeStreamSymbols()

	if len(declared) != indexTestSize {
		t.Fatalf("declared %d symbols, want the whole composition (%d)", len(declared), indexTestSize)
	}

	// Scrolling must not change the set any more.
	before := append([]string(nil), declared...)
	app.portfolioView.TabbedView.IndexTable.SetOffset(25, 0)
	app.recomputeStreamSymbols()

	if !slices.Equal(declared, before) {
		t.Error("scrolling changed the subscribed set; it should be scroll-independent now")
	}
}

// TestRecomputeStreamSymbols_PositionsComeFirst verifies the priority order that
// protects portfolio quotes: the client shards the list in order and drops the
// tail if it ever runs out of streams.
func TestRecomputeStreamSymbols_PositionsComeFirst(t *testing.T) {
	var declared []string
	mock := &mockClient{
		SetQuoteSymbolsFunc: func(symbols []string) { declared = append([]string(nil), symbols...) },
	}
	app := NewApp(mock, []models.AccountInfo{{ID: "acc1"}})
	app.selectedIdx = 0
	app.positions["acc1"] = []models.Position{{Symbol: "MYPOS@MISX"}}
	app.indexConstituents = manyConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)

	app.recomputeStreamSymbols()

	if len(declared) == 0 || declared[0] != "MYPOS@MISX" {
		t.Errorf("subscription starts with %v, want the position first", declared[:min(3, len(declared))])
	}
	if len(declared) != indexTestSize+1 {
		t.Errorf("declared %d symbols, want the position plus the whole composition", len(declared))
	}
}

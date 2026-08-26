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

// TestIndexWindowSymbols verifies the tab asks only for a screenful of the
// composition, and that the window moves in quantised blocks so holding an
// arrow key does not resubscribe on every row.
func TestIndexWindowSymbols(t *testing.T) {
	constituents := manyConstituents()

	tests := []struct {
		name       string
		offset     int
		wantFirst  string
		wantLen    int
		wantSecond string
	}{
		{name: "top of the list", offset: 0, wantFirst: "TICK0@MISX", wantLen: indexStreamWindow},
		{name: "within the first block does not move", offset: 7, wantFirst: "TICK0@MISX", wantLen: indexStreamWindow},
		{name: "next block", offset: 10, wantFirst: "TICK10@MISX", wantLen: indexStreamWindow},
		{name: "still the same block", offset: 19, wantFirst: "TICK10@MISX", wantLen: indexStreamWindow},
		{name: "near the end is clamped", offset: 40, wantFirst: "TICK40@MISX", wantLen: 6},
		{name: "negative offset is treated as the top", offset: -5, wantFirst: "TICK0@MISX", wantLen: indexStreamWindow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indexWindowSymbols(constituents, tt.offset)
			if len(got) != tt.wantLen {
				t.Fatalf("window length = %d, want %d (%v)", len(got), tt.wantLen, got)
			}
			if got[0] != tt.wantFirst {
				t.Errorf("window starts at %q, want %q", got[0], tt.wantFirst)
			}
		})
	}

	if got := indexWindowSymbols(nil, 0); got != nil {
		t.Errorf("empty composition returned %v, want nil", got)
	}
}

// TestIndexWindowSymbols_NeverExceedsTheWindow guards the budget: whatever the
// offset, the tab must not ask for the whole 46-symbol composition again.
func TestIndexWindowSymbols_NeverExceedsTheWindow(t *testing.T) {
	constituents := manyConstituents()

	for offset := range 60 {
		if got := indexWindowSymbols(constituents, offset); len(got) > indexStreamWindow {
			t.Fatalf("offset %d produced %d symbols, want at most %d", offset, len(got), indexStreamWindow)
		}
	}
}

// TestRecomputeStreamSymbols_WindowFollowsScrolling verifies the live
// subscription tracks the part of the index the user is actually looking at.
func TestRecomputeStreamSymbols_WindowFollowsScrolling(t *testing.T) {
	var declared []string
	mock := &mockClient{
		SetQuoteSymbolsFunc: func(symbols []string) { declared = append([]string(nil), symbols...) },
	}
	app := NewApp(mock, nil)
	app.indexConstituents = manyConstituents()
	app.indexLoaded = true
	app.portfolioView.TabbedView.SetTab(TabIndex)

	app.recomputeStreamSymbols()
	if len(declared) != indexStreamWindow || declared[0] != "TICK0@MISX" {
		t.Fatalf("initial window = %v, want %d symbols from the top", declared, indexStreamWindow)
	}

	// Scroll down: tview stores the viewport offset on the table.
	app.portfolioView.TabbedView.IndexTable.SetOffset(25, 0)
	app.recomputeStreamSymbols()

	if declared[0] != "TICK20@MISX" {
		t.Errorf("window after scrolling starts at %q, want TICK20@MISX", declared[0])
	}
	if len(declared) > indexStreamWindow {
		t.Errorf("window grew to %d symbols, want at most %d", len(declared), indexStreamWindow)
	}
}

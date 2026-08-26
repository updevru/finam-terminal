package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"finam-terminal/models"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// testConstituents is a small composition with weights deliberately out of
// order, mirroring what GetConstituents returns.
func testConstituents() []models.IndexConstituent {
	return []models.IndexConstituent{
		{Symbol: "SBER@MISX", Ticker: "SBER", Name: "Сбербанк", Sector: "Финансы", Weight: 0.0080},
		{Symbol: "GAZP@MISX", Ticker: "GAZP", Name: "Газпром", Sector: "Нефть и газ", Weight: 0.0120},
		{Symbol: "LKOH@MISX", Ticker: "LKOH", Name: "ЛУКОЙЛ", Sector: "Нефть и газ", Weight: 0.0100},
	}
}

func loadedIndexApp(t *testing.T, quotes map[string]*models.Quote) *App {
	t.Helper()
	app := NewApp(&mockClient{}, nil)
	app.indexConstituents = testConstituents()
	app.indexLoaded = true
	if quotes != nil {
		app.indexQuotes = quotes
	}
	return app
}

// TestIndexTable_HeadersAndTitle verifies the column set and that the table is
// titled with the index name rather than a generic label.
func TestIndexTable_HeadersAndTitle(t *testing.T) {
	app := loadedIndexApp(t, nil)

	updateIndexTable(app)

	table := app.portfolioView.TabbedView.IndexTable
	want := []string{"Ticker", "Name", "Price", "Chg", "Chg%", "Weight", "Volume"}
	for i, h := range want {
		if got := table.GetCell(0, i).Text; got != h {
			t.Errorf("header[%d] = %q, want %q", i, got, h)
		}
	}

	if title := table.GetTitle(); !strings.Contains(title, activeIndex().Name) {
		t.Errorf("table title = %q, want it to contain %q", title, activeIndex().Name)
	}
}

// TestIndexTable_SortedByWeightDesc verifies the heaviest component comes first,
// regardless of the order the API returned.
func TestIndexTable_SortedByWeightDesc(t *testing.T) {
	app := loadedIndexApp(t, nil)

	updateIndexTable(app)

	table := app.portfolioView.TabbedView.IndexTable
	want := []string{"GAZP", "LKOH", "SBER"}
	for i, ticker := range want {
		if got := table.GetCell(i+1, 0).Text; got != ticker {
			t.Errorf("row %d ticker = %q, want %q", i+1, got, ticker)
		}
	}
}

// TestIndexTable_SortedAlphabeticallyWithoutWeights verifies the fallback order
// when the API sends no weights at all.
func TestIndexTable_SortedAlphabeticallyWithoutWeights(t *testing.T) {
	app := loadedIndexApp(t, nil)
	for i := range app.indexConstituents {
		app.indexConstituents[i].Weight = 0
	}

	updateIndexTable(app)

	table := app.portfolioView.TabbedView.IndexTable
	want := []string{"GAZP", "LKOH", "SBER"}
	for i, ticker := range want {
		if got := table.GetCell(i+1, 0).Text; got != ticker {
			t.Errorf("row %d ticker = %q, want %q (alphabetical fallback)", i+1, got, ticker)
		}
	}
}

// TestIndexTable_RendersQuoteAndChange verifies price, absolute change, percent
// change and volume, and that a rise is green while a fall is red.
func TestIndexTable_RendersQuoteAndChange(t *testing.T) {
	app := loadedIndexApp(t, map[string]*models.Quote{
		// close = 290 - 5 = 285 → +1.75%
		"GAZP@MISX": {Symbol: "GAZP@MISX", Last: "290.00", Change: "5.00", Volume: "1500000"},
		// close = 100 + 2.5 = 102.5 → -2.44%
		"LKOH@MISX": {Symbol: "LKOH@MISX", Last: "100.00", Change: "-2.50", Volume: "900"},
	})

	updateIndexTable(app)

	table := app.portfolioView.TabbedView.IndexTable

	// Row 1: GAZP, rising.
	if got := table.GetCell(1, 2).Text; got != "290.00" {
		t.Errorf("GAZP price = %q, want 290.00", got)
	}
	if got := table.GetCell(1, 3).Text; got != "+5.00" {
		t.Errorf("GAZP Chg = %q, want +5.00", got)
	}
	if got := table.GetCell(1, 4).Text; got != "+1.75%" {
		t.Errorf("GAZP Chg%% = %q, want +1.75%%", got)
	}
	if got := table.GetCell(1, 6).Text; got != "1 500 000" {
		t.Errorf("GAZP Volume = %q, want \"1 500 000\"", got)
	}
	if fg, _, _ := table.GetCell(1, 3).Style.Decompose(); fg != tcell.ColorGreen {
		t.Errorf("GAZP Chg colour = %v, want green", fg)
	}

	// Row 2: LKOH, falling.
	if got := table.GetCell(2, 3).Text; got != "-2.50" {
		t.Errorf("LKOH Chg = %q, want -2.50", got)
	}
	if got := table.GetCell(2, 4).Text; got != "-2.44%" {
		t.Errorf("LKOH Chg%% = %q, want -2.44%%", got)
	}
	if fg, _, _ := table.GetCell(2, 3).Style.Decompose(); fg != tcell.ColorRed {
		t.Errorf("LKOH Chg colour = %v, want red", fg)
	}
}

// TestIndexTable_MissingQuoteShowsDash verifies that a component without a quote
// renders placeholders rather than zeros or "N/A".
func TestIndexTable_MissingQuoteShowsDash(t *testing.T) {
	app := loadedIndexApp(t, nil)

	updateIndexTable(app)

	table := app.portfolioView.TabbedView.IndexTable
	for _, col := range []int{2, 3, 4, 6} {
		if got := table.GetCell(1, col).Text; got != "—" {
			t.Errorf("column %d without a quote = %q, want —", col, got)
		}
	}
	// The name column stays populated: it comes from the composition, not the quote.
	if got := table.GetCell(1, 1).Text; got != "Газпром" {
		t.Errorf("name = %q, want Газпром", got)
	}
}

// TestIndexTable_ZeroCloseHasNoPercent verifies the division guard: when the
// derived close is 0 the percent column is a dash, never NaN or Inf.
func TestIndexTable_ZeroCloseHasNoPercent(t *testing.T) {
	app := loadedIndexApp(t, map[string]*models.Quote{
		// last == change → close = 0
		"GAZP@MISX": {Symbol: "GAZP@MISX", Last: "5.00", Change: "5.00"},
	})

	updateIndexTable(app)

	got := app.portfolioView.TabbedView.IndexTable.GetCell(1, 4).Text
	if got != "—" {
		t.Errorf("Chg%% with a zero close = %q, want —", got)
	}
	if strings.Contains(got, "NaN") || strings.Contains(got, "Inf") {
		t.Errorf("Chg%% = %q, must never contain NaN/Inf", got)
	}
}

// TestIndexTable_FlatChangeIsNeutral verifies a zero change renders as 0.00 in
// the neutral colour rather than being coloured as a rise.
func TestIndexTable_FlatChangeIsNeutral(t *testing.T) {
	app := loadedIndexApp(t, map[string]*models.Quote{
		"GAZP@MISX": {Symbol: "GAZP@MISX", Last: "290.00", Change: "0"},
	})

	updateIndexTable(app)

	table := app.portfolioView.TabbedView.IndexTable
	if got := table.GetCell(1, 3).Text; got != "0.00" {
		t.Errorf("flat Chg = %q, want 0.00", got)
	}
	if fg, _, _ := table.GetCell(1, 3).Style.Decompose(); fg == tcell.ColorGreen || fg == tcell.ColorRed {
		t.Errorf("flat Chg colour = %v, want a neutral colour", fg)
	}
}

// TestIndexTable_LoadingState verifies the tab says it is loading before the
// composition arrives.
func TestIndexTable_LoadingState(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	app.indexLoading = true

	updateIndexTable(app)

	if got := app.portfolioView.TabbedView.IndexTable.GetCell(1, 0).Text; !strings.Contains(got, "Loading") {
		t.Errorf("loading row = %q, want it to mention Loading", got)
	}
}

// TestIndexTable_ErrorStateOffersRetry verifies a failed load explains itself
// and points at the manual retry key, since nothing retries automatically.
func TestIndexTable_ErrorStateOffersRetry(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	app.indexLoadErr = "backend down"

	updateIndexTable(app)

	got := app.portfolioView.TabbedView.IndexTable.GetCell(1, 0).Text
	if !strings.Contains(got, "backend down") {
		t.Errorf("error row = %q, want it to carry the error text", got)
	}
	if !strings.Contains(got, "R") {
		t.Errorf("error row = %q, want it to point at the R retry key", got)
	}
}

// TestEnsureIndexLoaded_LoadsOnceAndCaches verifies entering the tab loads the
// composition exactly once: a second entry must not spawn another request.
func TestEnsureIndexLoaded_LoadsOnceAndCaches(t *testing.T) {
	done := make(chan struct{}, 1)
	mock := &mockClient{
		GetIndexConstituentsFunc: func(string) ([]models.IndexConstituent, error) {
			select {
			case done <- struct{}{}:
			default:
			}
			return testConstituents(), nil
		},
	}
	app := NewApp(mock, nil)

	app.loadIndexSync()
	if n := mock.GetIndexConstituentsCalls.Load(); n != 1 {
		t.Fatalf("first load made %d call(s), want 1", n)
	}

	app.ensureIndexLoaded()
	if n := mock.GetIndexConstituentsCalls.Load(); n != 1 {
		t.Errorf("re-entering the tab made %d call(s) in total, want 1 — the loaded composition must be reused", n)
	}

	app.dataMutex.RLock()
	got := len(app.indexConstituents)
	app.dataMutex.RUnlock()
	if got != 3 {
		t.Errorf("stored %d constituents, want 3", got)
	}
}

// TestLoadIndexSync_ErrorIsRecorded verifies a failed load leaves a message for
// the tab and does not mark the composition as loaded.
func TestLoadIndexSync_ErrorIsRecorded(t *testing.T) {
	mock := &mockClient{
		GetIndexConstituentsFunc: func(string) ([]models.IndexConstituent, error) {
			return nil, errors.New("Constituents not found")
		},
	}
	app := NewApp(mock, nil)

	app.loadIndexSync()

	app.dataMutex.RLock()
	loaded, errMsg := app.indexLoaded, app.indexLoadErr
	app.dataMutex.RUnlock()

	if loaded {
		t.Error("indexLoaded = true after a failed load")
	}
	if !strings.Contains(errMsg, "Constituents not found") {
		t.Errorf("indexLoadErr = %q, want it to carry the API error", errMsg)
	}
}

// TestIndexSymbols_ReturnsFullSymbols verifies the helper the quote subscription
// will use hands back full ticker@mic symbols.
func TestIndexSymbols_ReturnsFullSymbols(t *testing.T) {
	app := loadedIndexApp(t, nil)

	got := app.indexSymbols()

	if len(got) != 3 {
		t.Fatalf("got %d symbols, want 3", len(got))
	}
	for _, s := range got {
		if !strings.Contains(s, "@") {
			t.Errorf("symbol %q is not a full ticker@mic symbol", s)
		}
	}
}

// renderTable draws a table onto a simulation screen and returns the visible
// lines. Rendering for real is the only way to check what the first smoke test
// complained about: tview derives both the visible rows and the column widths
// at draw time.
func renderTable(t *testing.T, table *tview.Table, width, height int) []string {
	t.Helper()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("simulation screen init: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(width, height)

	table.SetRect(0, 0, width, height)
	// Twice: tview adjusts the scroll offset to reveal the selection while
	// drawing, so the first pass renders the pre-scroll viewport.
	table.Draw(screen)
	table.Draw(screen)

	lines := make([]string, height)
	for y := range height {
		var sb strings.Builder
		for x := range width {
			cell, _, _ := screen.Get(x, y)
			sb.WriteString(cell)
		}
		lines[y] = strings.TrimRight(sb.String(), " ")
	}
	return lines
}

// TestIndexTable_HeaderStaysVisibleWhenScrolled guards the defect the first
// smoke test found: with 46 rows the header scrolled off the top and the column
// labels disappeared.
func TestIndexTable_HeaderStaysVisibleWhenScrolled(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	app.indexConstituents = manyConstituents()
	app.indexLoaded = true
	updateIndexTable(app)

	table := app.portfolioView.TabbedView.IndexTable
	// Select a row far below the fold, as a user scrolling the list would.
	table.Select(40, 0)

	rendered := strings.Join(renderTable(t, table, 120, 20), "\n")

	for _, header := range []string{"Ticker", "Name", "Price", "Chg", "Weight", "Volume"} {
		if !strings.Contains(rendered, header) {
			t.Errorf("column header %q is not on screen after scrolling to row 40:\n%s", header, rendered)
		}
	}
	// And the list really did scroll: the first row is gone, so the header is
	// on screen because it is pinned, not because nothing moved.
	if strings.Contains(rendered, "TICK0 ") {
		t.Errorf("the list did not scroll, so the header test proves nothing:\n%s", rendered)
	}
}

// TestIndexTable_FillsAvailableWidth guards the other half of the same defect:
// column widths are derived from the visible rows, so with expansion only on
// the header the table collapsed to its content width once the header scrolled
// away.
func TestIndexTable_FillsAvailableWidth(t *testing.T) {
	app := NewApp(&mockClient{}, nil)
	app.indexConstituents = manyConstituents()
	app.indexLoaded = true
	updateIndexTable(app)

	table := app.portfolioView.TabbedView.IndexTable
	table.Select(40, 0)

	const width = 120
	lines := renderTable(t, table, width, 20)

	widest := 0
	for _, line := range lines {
		if len(line) > widest {
			widest = len(line)
		}
	}

	// Content alone is far narrower than 120 columns; only expansion can get
	// the table anywhere near the edge.
	if widest < width*3/4 {
		t.Errorf("widest rendered line is %d of %d columns — the table is not using the available width:\n%s",
			widest, width, strings.Join(lines, "\n"))
	}
}

// indexTestSize is the size of the real IMOEX composition, which is what makes
// the tab long enough to scroll and too large for one subscription.
const indexTestSize = 46

// manyConstituents builds a composition the size of the real index.
func manyConstituents() []models.IndexConstituent {
	const n = indexTestSize
	out := make([]models.IndexConstituent, n)
	for i := range out {
		out[i] = models.IndexConstituent{
			Symbol: fmt.Sprintf("TICK%d@MISX", i),
			Ticker: fmt.Sprintf("TICK%d", i),
			Name:   fmt.Sprintf("Компания номер %d", i),
			Sector: "Финансы",
			Weight: float64(n-i) / 1000,
		}
	}
	return out
}

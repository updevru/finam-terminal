package ui

import (
	"log"
	"sort"
	"time"

	"finam-terminal/api"
	"finam-terminal/models"
)

// indexPollCooldown is the minimum spacing between automatic fallback quote
// batches. One batch is one LastQuote per component (46 for IMOEX), so at most
// two batches a minute stays far inside the broker's 200-per-minute budget for
// that method — and the batch only runs at all when the stream is down and the
// tab is on screen.
const indexPollCooldown = time.Minute

// IndexInfo names one index the tab can show.
//
// The list is a constant rather than an API lookup on purpose: the index symbol
// itself does not appear in the bulk asset list, so it cannot be discovered, and
// GetConstituents only answers for a handful of symbols. Adding another index
// (the API also serves NDX@_SCI and SPX@_SP) is one more element here.
type IndexInfo struct {
	Symbol string
	Name   string
}

var indexList = []IndexInfo{
	{Symbol: "IMOEX@RTSX", Name: "Индекс МосБиржи"},
}

// activeIndex returns the index the tab currently shows. There is exactly one
// for now; a picker arrives together with a second index.
func activeIndex() IndexInfo { return indexList[0] }

// ensureIndexLoaded loads the composition the first time the tab is opened.
// Re-entering the tab costs nothing: an already loaded (or in-flight) load is
// left alone, so the whole session stays within the 1-2 request budget.
func (a *App) ensureIndexLoaded() {
	a.dataMutex.RLock()
	skip := a.indexLoaded || a.indexLoading
	a.dataMutex.RUnlock()
	if skip {
		return
	}
	a.loadIndexAsync()
}

// loadIndexAsync fetches the index composition off the event loop and redraws
// the tab when it lands.
func (a *App) loadIndexAsync() {
	a.dataMutex.Lock()
	if a.indexLoading {
		a.dataMutex.Unlock()
		return
	}
	a.indexLoading = true
	a.dataMutex.Unlock()

	go func() {
		a.loadIndexSync()
		a.app.QueueUpdateDraw(func() {
			updateIndexTable(a)
			a.recomputeStreamSymbols()
			// With the stream live this is vetoed by the predicate; with it
			// down, waiting for the next background tick would leave the tab
			// blank for several seconds after the composition already arrived.
			a.pollIndexQuotesAsync(false)
		})
	}()
}

// loadIndexSync performs the composition fetch and records the outcome. It
// blocks, so it belongs off the event loop; it is separate from loadIndexAsync
// so tests can drive it without a running application.
func (a *App) loadIndexSync() {
	a.dataMutex.Lock()
	a.indexLoading = true
	a.indexLoadErr = ""
	a.dataMutex.Unlock()

	index := activeIndex()
	constituents, err := a.client.GetIndexConstituents(index.Symbol)

	a.dataMutex.Lock()
	defer a.dataMutex.Unlock()
	a.indexLoading = false

	if err != nil {
		log.Printf("[WARN] Failed to load the composition of %s: %v", index.Symbol, err)
		a.indexLoadErr = extractUserMessage(err)
		return
	}

	a.indexConstituents = constituents
	a.indexLoaded = true
}

// indexSymbols returns the full ticker@mic symbols of the loaded composition,
// which is what the quote subscription and the fallback batch need.
func (a *App) indexSymbols() []string {
	a.dataMutex.RLock()
	defer a.dataMutex.RUnlock()

	symbols := make([]string, 0, len(a.indexConstituents))
	for _, c := range a.indexConstituents {
		symbols = append(symbols, c.Symbol)
	}
	return symbols
}

// sortConstituents orders the tab: heaviest first, which puts the names that
// actually move the index at the top. With no weights at all (the API may send
// none) it falls back to alphabetical order, and ties break by ticker so the
// order never flickers between redraws.
func sortConstituents(constituents []models.IndexConstituent) []models.IndexConstituent {
	sorted := append([]models.IndexConstituent(nil), constituents...)

	hasWeights := false
	for _, c := range sorted {
		if c.Weight != 0 {
			hasWeights = true
			break
		}
	}

	sort.SliceStable(sorted, func(i, j int) bool {
		if hasWeights && sorted[i].Weight != sorted[j].Weight {
			return sorted[i].Weight > sorted[j].Weight
		}
		return sorted[i].Ticker < sorted[j].Ticker
	})
	return sorted
}

// shouldPollIndexQuotes decides whether an automatic sweep may start.
//
// Note what is *not* here: stream liveness. The broker caps a subscription at
// roughly ten symbols, so a live stream can never cover a 46-name index — the
// sweep fills whatever the subscription leaves out and skips the rest. When the
// stream happens to cover everything, the symbol list comes out empty and no
// request is made anyway.
func shouldPollIndexQuotes(tabActive, autoDisabled bool, lastPoll, now time.Time) bool {
	if !tabActive || autoDisabled {
		return false
	}
	return now.Sub(lastPoll) >= indexPollCooldown
}

// indexSweepDelay paces the individual quote requests of a sweep.
//
// The broker's per-method budget is 200 requests a minute, but a burst is
// refused long before that: firing 46 LastQuote calls back to back earned an
// immediate "Too Many Requests". At this spacing a full 46-name sweep takes
// about seven seconds and costs 46 requests a minute — comfortably inside the
// budget at a rate the broker tolerates.
// A var rather than a const so tests can drop the pacing; nothing in production
// writes to it.
var indexSweepDelay = 150 * time.Millisecond

// indexSweepRedrawEvery is how often a running sweep repaints, so rows appear
// progressively instead of all at the end.
const indexSweepRedrawEvery = 8

// uncoveredIndexSymbols returns the composition symbols the stream is not
// carrying, in display order, so the sweep fills exactly the gaps.
func uncoveredIndexSymbols(constituents []models.IndexConstituent, subscribed []string) []string {
	live := make(map[string]struct{}, len(subscribed))
	for _, s := range subscribed {
		live[s] = struct{}{}
	}

	out := make([]string, 0, len(constituents))
	for _, c := range constituents {
		if _, ok := live[c.Symbol]; ok {
			continue
		}
		out = append(out, c.Symbol)
	}
	return out
}

// pollIndexQuotesAsync fills the rows the stream cannot carry, off the event
// loop. manual marks a user-initiated refresh (R), which ignores the cooldown
// and the rate-limit latch.
func (a *App) pollIndexQuotesAsync(manual bool) {
	// The active tab lives in the tview widget tree, which belongs to the event
	// loop: it is read here, on the caller's thread, and handed to the worker
	// rather than read from it.
	tabActive := a.indexTabActive()

	go a.sweepIndexQuotes(manual, tabActive)
}

// sweepIndexQuotes walks the uncovered symbols one request at a time, pacing
// them so the burst never trips the broker's rate limit, and repaints as it
// goes. It reports whether it made any request at all.
//
// It blocks, so it belongs off the event loop; it is separate from
// pollIndexQuotesAsync so tests can drive it without a running application.
func (a *App) sweepIndexQuotes(manual, tabActive bool) bool {
	if a.client == nil {
		return false
	}

	a.dataMutex.RLock()
	constituents := sortConstituents(a.indexConstituents)
	lastPoll, autoDisabled := a.indexLastPoll, a.indexPollDisabled
	a.dataMutex.RUnlock()

	if len(constituents) == 0 {
		return false
	}
	if !manual && !shouldPollIndexQuotes(tabActive, autoDisabled, lastPoll, time.Now()) {
		return false
	}

	symbols := uncoveredIndexSymbols(constituents, a.client.SubscribedSymbols())
	if len(symbols) == 0 {
		// The stream covers the whole composition: nothing to ask for.
		return false
	}

	a.dataMutex.Lock()
	a.indexLastPoll = time.Now()
	a.dataMutex.Unlock()

	fetched := 0
	for i, symbol := range symbols {
		if a.stopped() {
			break
		}
		if i > 0 && !sleepUnlessStopped(a.stopChan, indexSweepDelay) {
			break
		}

		// accountID is empty on purpose: the symbols are already full
		// ticker@mic, so no account context is needed to resolve them.
		quotes, err := a.client.GetQuotes("", []string{symbol})
		if err != nil {
			if api.IsRateLimited(err) {
				a.dataMutex.Lock()
				a.indexPollDisabled = true
				a.dataMutex.Unlock()
				log.Printf("[WARN] Index quote sweep was rate limited after %d request(s); "+
					"automatic refresh is off for this session: %v", fetched, err)
				a.SetStatus("Index quotes rate limited — press R to refresh manually", StatusError)
				break
			}
			log.Printf("[WARN] Index quote for %s failed: %v", symbol, err)
			continue
		}

		a.dataMutex.Lock()
		for s, q := range quotes {
			if a.indexQuotes == nil {
				a.indexQuotes = make(map[string]*models.Quote)
			}
			a.indexQuotes[s] = q
		}
		a.dataMutex.Unlock()
		fetched++

		if fetched%indexSweepRedrawEvery == 0 {
			a.redrawIndexTab()
		}
	}

	if fetched > 0 {
		a.redrawIndexTab()
	}
	return fetched > 0
}

// redrawIndexTab repaints the tab from a worker goroutine, if it is still the
// one on screen and the application is still running.
func (a *App) redrawIndexTab() {
	if a.stopped() {
		return
	}
	// Queued from a goroutine: QueueUpdateDraw blocks until the event loop picks
	// it up, and a slow loop must not stall the sweep's pacing.
	go a.app.QueueUpdateDraw(func() {
		if a.indexTabActive() {
			updateIndexTable(a)
		}
	})
}

// stopped reports whether the application is shutting down. A QueueUpdateDraw
// after Stop() blocks its goroutine forever.
func (a *App) stopped() bool {
	select {
	case <-a.stopChan:
		return true
	default:
		return false
	}
}

// sleepUnlessStopped waits for d, returning false if the application stopped
// first.
func sleepUnlessStopped(stop <-chan struct{}, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-stop:
		return false
	case <-timer.C:
		return true
	}
}

// selectedIndexSymbol returns the full symbol of the highlighted Index row, or
// "" for the header row and a stale selection. It resolves through the rendered
// (sorted) order, so the instrument that opens is always the one the user is
// looking at.
func (a *App) selectedIndexSymbol() string {
	row, _ := a.portfolioView.TabbedView.IndexTable.GetSelection()
	if row <= 0 {
		return ""
	}

	a.dataMutex.RLock()
	constituents := sortConstituents(a.indexConstituents)
	a.dataMutex.RUnlock()

	idx := row - 1
	if idx >= len(constituents) {
		return ""
	}
	return constituents[idx].Symbol
}

// indexStreamWindow is how many composition symbols the tab asks the stream for,
// and indexWindowStep is how coarsely that window moves.
//
// The broker caps subscription size (undocumented — 46 symbols were refused
// outright), so the tab asks only for a screenful rather than the whole index,
// and the client narrows it further if even that is too much. The window start
// is quantised so that holding an arrow key scrolls without resubscribing on
// every row.
const (
	indexStreamWindow = 20
	indexWindowStep   = 10
)

// indexWindowSymbols returns the slice of the composition the stream should
// carry, given the table's current scroll offset. The offset is quantised, so
// the set changes once per block of rows instead of once per keypress.
func indexWindowSymbols(constituents []models.IndexConstituent, offset int) []string {
	if len(constituents) == 0 {
		return nil
	}

	start := max(offset, 0) / indexWindowStep * indexWindowStep
	if start >= len(constituents) {
		start = 0
	}
	end := min(start+indexStreamWindow, len(constituents))

	symbols := make([]string, 0, end-start)
	for _, c := range constituents[start:end] {
		symbols = append(symbols, c.Symbol)
	}
	return symbols
}

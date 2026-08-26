package ui

import (
	"log"
	"sort"

	"finam-terminal/models"
)

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

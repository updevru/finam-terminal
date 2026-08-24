package ui

import (
	"log"
	"slices"
	"strings"

	"finam-terminal/models"
)

// startQuoteStream wires the realtime quote stream (Trade API 2.19.0) into the
// UI and declares the initial subscription.
func (a *App) startQuoteStream() {
	if a.client == nil {
		return
	}

	a.client.StartQuoteStream(a.onStreamQuote, a.onStreamState)
	a.recomputeStreamSymbols()
}

// onStreamQuote runs on the API stream goroutine. It parks the quote in an inbox
// and schedules at most one redraw at a time, so a burst of updates costs one
// draw instead of one per quote.
func (a *App) onStreamQuote(q models.Quote) {
	// A redraw queued after Stop() blocks its goroutine forever.
	select {
	case <-a.stopChan:
		return
	default:
	}

	quote := q

	a.quoteInboxMu.Lock()
	if a.quoteInbox == nil {
		a.quoteInbox = make(map[string]*models.Quote)
	}
	a.quoteInbox[quote.Symbol] = &quote
	alreadyQueued := a.quoteFlushQueued
	a.quoteFlushQueued = true
	a.quoteInboxMu.Unlock()

	if alreadyQueued {
		return
	}

	// Queued from a goroutine: QueueUpdateDraw blocks until the event loop picks
	// it up, and this callback must not stall the stream.
	go a.app.QueueUpdateDraw(a.flushQuoteInbox)
}

// flushQuoteInbox runs on the event loop: it merges the collected quotes into
// the active account and repaints whatever is showing them.
func (a *App) flushQuoteInbox() {
	a.quoteInboxMu.Lock()
	inbox := a.quoteInbox
	a.quoteInbox = nil
	a.quoteFlushQueued = false
	a.quoteInboxMu.Unlock()

	if len(inbox) == 0 {
		return
	}

	a.dataMutex.Lock()
	accountID := ""
	if a.selectedIdx >= 0 && a.selectedIdx < len(a.accounts) {
		accountID = a.accounts[a.selectedIdx].ID
	}
	if accountID != "" {
		quotes := a.quotes[accountID]
		if quotes == nil {
			quotes = make(map[string]*models.Quote)
			a.quotes[accountID] = quotes
		}
		for symbol, q := range inbox {
			quotes[symbol] = q
		}
	}
	a.dataMutex.Unlock()

	if accountID == "" {
		return
	}

	if a.profileOpen {
		if q, ok := inbox[a.profileSymbol]; ok {
			if p := a.profilePanel.GetProfile(); p != nil {
				p.Quote = q
				a.profilePanel.Update(p)
			}
		}
		return
	}

	updatePositionsTable(a)
	updateInfoPanel(a)
}

// onStreamState records stream liveness. While the stream is live it owns the
// quotes of the active account; when it drops, the 5-second poller takes over
// again on the next tick.
func (a *App) onStreamState(up bool) {
	a.streamLive.Store(up)
	if up {
		log.Printf("[INFO] Quote stream is live, quote polling paused for the active account")
	} else {
		log.Printf("[INFO] Quote stream is down, falling back to quote polling")
	}
}

// shouldSkipQuotePolling reports whether the poller must leave this account's
// quotes to the stream.
func (a *App) shouldSkipQuotePolling(accountID string) bool {
	if !a.streamLive.Load() {
		return false
	}

	a.dataMutex.RLock()
	defer a.dataMutex.RUnlock()

	return a.selectedIdx >= 0 && a.selectedIdx < len(a.accounts) &&
		a.accounts[a.selectedIdx].ID == accountID
}

// computeStreamSymbols returns the symbols the stream should carry: the active
// account's positions plus the symbol of an open profile. Symbols without a MIC
// are dropped (the stream only accepts ticker@mic), and the result is deduped
// and sorted so an unchanged set compares equal.
func computeStreamSymbols(positions []models.Position, profileOpen bool, profileSymbol string) []string {
	symbols := make([]string, 0, len(positions)+1)
	seen := make(map[string]struct{}, len(positions)+1)

	add := func(symbol string) {
		if !strings.Contains(symbol, "@") {
			return
		}
		if _, dup := seen[symbol]; dup {
			return
		}
		seen[symbol] = struct{}{}
		symbols = append(symbols, symbol)
	}

	for _, p := range positions {
		add(p.Symbol)
	}
	if profileOpen {
		add(profileSymbol)
	}

	slices.Sort(symbols)
	return symbols
}

// recomputeStreamSymbols re-declares the subscription from the current UI state.
// Safe to call from the event loop: SetQuoteSymbols never blocks.
func (a *App) recomputeStreamSymbols() {
	if a.client == nil {
		return
	}

	a.dataMutex.RLock()
	var positions []models.Position
	if a.selectedIdx >= 0 && a.selectedIdx < len(a.accounts) {
		positions = a.positions[a.accounts[a.selectedIdx].ID]
	}
	a.dataMutex.RUnlock()

	a.client.SetQuoteSymbols(computeStreamSymbols(positions, a.profileOpen, a.profileSymbol))
}

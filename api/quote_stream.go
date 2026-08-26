package api

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"finam-terminal/models"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// quoteToModel converts an SDK quote into the model the UI renders. Absent
// fields become "N/A" via formatDecimal.
func quoteToModel(symbol string, q *marketdata.Quote) *models.Quote {
	return &models.Quote{
		Symbol:       symbol,
		Bid:          formatDecimal(q.Bid),
		BidSize:      formatDecimal(q.BidSize),
		Ask:          formatDecimal(q.Ask),
		AskSize:      formatDecimal(q.AskSize),
		Last:         formatDecimal(q.Last),
		LastSize:     formatDecimal(q.LastSize),
		Volume:       formatDecimal(q.Volume),
		Open:         formatDecimal(q.Open),
		High:         formatDecimal(q.High),
		Low:          formatDecimal(q.Low),
		Close:        formatDecimal(q.Close),
		Change:       formatDecimal(q.Change),
		OpenInterest: formatDecimal(q.OpenInterest),
		Timestamp:    q.Timestamp.AsTime().Local(),
	}
}

// mergeQuote folds a streamed quote into the previously known state.
//
// A snapshot (Trade API 2.19.0 is_data_snapshot) is authoritative: it replaces
// the state wholesale, so a field it omits is genuinely absent. An incremental
// update carries only what changed, so every non-nil field overwrites the
// previous one and everything else is kept. The first message is treated as a
// full state even when the flag is not set — there is nothing to merge into.
//
// prev is never mutated: the merge works on a copy, so a consumer holding the
// previous quote keeps seeing consistent data.
func mergeQuote(prev, next *marketdata.Quote) *marketdata.Quote {
	if prev == nil || next == nil || next.IsDataSnapshot {
		return next
	}

	// proto.Clone rather than a struct copy: protobuf messages carry internal
	// state that must not be copied by value.
	merged := proto.Clone(prev).(*marketdata.Quote)
	merged.Symbol = next.Symbol
	merged.IsDataSnapshot = next.IsDataSnapshot

	if next.Timestamp != nil {
		merged.Timestamp = next.Timestamp
	}
	if next.Ask != nil {
		merged.Ask = next.Ask
	}
	if next.AskSize != nil {
		merged.AskSize = next.AskSize
	}
	if next.Bid != nil {
		merged.Bid = next.Bid
	}
	if next.BidSize != nil {
		merged.BidSize = next.BidSize
	}
	if next.Last != nil {
		merged.Last = next.Last
	}
	if next.LastSize != nil {
		merged.LastSize = next.LastSize
	}
	if next.Volume != nil {
		merged.Volume = next.Volume
	}
	if next.Turnover != nil {
		merged.Turnover = next.Turnover
	}
	if next.Open != nil {
		merged.Open = next.Open
	}
	if next.High != nil {
		merged.High = next.High
	}
	if next.Low != nil {
		merged.Low = next.Low
	}
	if next.Close != nil {
		merged.Close = next.Close
	}
	if next.Change != nil {
		merged.Change = next.Change
	}
	if next.OpenInterest != nil {
		merged.OpenInterest = next.OpenInterest
	}

	return merged
}

// quoteStreamInitialBackoff and quoteStreamMaxBackoff bound the exponential
// backoff used to reopen the SubscribeQuote stream after a drop. They mirror the
// JWT renewal stream.
const (
	quoteStreamInitialBackoff = 1 * time.Second
	quoteStreamMaxBackoff     = 30 * time.Second
)

// StartQuoteStream starts the realtime quote manager (Trade API 2.19.0
// SubscribeQuote). onQuote is called for every quote the stream delivers, with
// incremental updates already merged into the last known state; onState reports
// stream liveness and fires only on an actual change. Both run on the manager
// goroutine, so they must not block.
//
// Calling it more than once is a no-op: the manager is a singleton per client.
// It stays idle until SetQuoteSymbols provides something to subscribe to.
func (c *Client) StartQuoteStream(onQuote func(models.Quote), onState func(up bool)) {
	c.quoteMu.Lock()
	if c.quoteStarted {
		c.quoteMu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.quoteStarted = true
	c.quoteOnQuote = onQuote
	c.quoteOnState = onState
	c.quoteCancel = cancel
	c.quoteWake = make(chan struct{}, 1)
	c.lastStreamQuotes = make(map[string]*marketdata.Quote)
	c.quoteMu.Unlock()

	go c.runQuoteStream(ctx)
}

// SetQuoteSymbols declares the set of symbols the stream should carry. It is
// safe to call from the UI thread: it never blocks and never performs I/O. When
// the set actually changes, the current subscription is cancelled and the
// manager resubscribes.
func (c *Client) SetQuoteSymbols(symbols []string) {
	normalized := normalizeSymbols(symbols)

	c.quoteMu.Lock()
	// Compare what would actually be subscribed, not what was asked for: with a
	// cap in force the tail of the list never reaches the broker, and reordering
	// within the cap is not a change worth dropping a live stream over.
	if sameSymbolSet(applySymbolCap(c.quoteSymbols, c.quoteSymbolCap), applySymbolCap(normalized, c.quoteSymbolCap)) {
		c.quoteMu.Unlock()
		return
	}
	c.quoteSymbols = normalized
	subCancel := c.quoteSubCancel
	wake := c.quoteWake
	c.quoteMu.Unlock()

	if subCancel != nil {
		subCancel()
	}
	if wake != nil {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

// normalizeSymbols keeps only full symbols (ticker@mic — the stream rejects bare
// tickers) and removes duplicates, keeping the first occurrence.
//
// The caller's order is preserved because it is priority order: the broker caps
// how many symbols one subscription may carry, and applySymbolCap truncates from
// the end, so whatever the caller puts first is what survives.
func normalizeSymbols(symbols []string) []string {
	seen := make(map[string]struct{}, len(symbols))
	out := make([]string, 0, len(symbols))
	for _, s := range symbols {
		if !strings.Contains(s, "@") {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// IsSymbolLimitError reports whether the broker refused a subscription because
// it carried too many symbols.
//
// Finam documents no limit and returns no dedicated status code for it — the
// stream simply ends with InvalidArgument and the message "Maximum number of
// symbols exceeded." Matching on the phrase is unpleasant but it is the only
// signal available, and the alternative (treating every InvalidArgument as a
// symbol limit) would shrink the subscription for unrelated reasons.
func IsSymbolLimitError(err error) bool {
	if err == nil || status.Code(err) != codes.InvalidArgument {
		return false
	}
	return strings.Contains(strings.ToLower(status.Convert(err).Message()), "maximum number of symbols")
}

// reducedSymbolCap halves the count that was just refused, so the client
// converges on the broker's real limit in a handful of attempts (46 → 23 → 11 →
// 5 → 2 → 1) without ever being told what it is. It never goes below one.
func reducedSymbolCap(attempted int) int {
	if attempted <= 2 {
		return 1
	}
	return attempted / 2
}

// applySymbolCap truncates the subscription to what the broker accepts. A cap of
// 0 means no limit has been discovered yet. Truncation is from the end, so the
// caller's leading (highest priority) symbols are the ones that stay.
func applySymbolCap(symbols []string, cap int) []string {
	if cap <= 0 || len(symbols) <= cap {
		return symbols
	}
	return symbols[:cap]
}

// getStreamContext returns a context for a long-lived stream: unlike
// getContext it carries no timeout, but it does carry the current session
// token, which is refreshed on every (re)subscribe.
func (c *Client) getStreamContext(parent context.Context) (context.Context, context.CancelFunc) {
	c.tokenMutex.RLock()
	token := c.token
	c.tokenMutex.RUnlock()

	ctx, cancel := context.WithCancel(parent)
	ctx = metadata.AppendToOutgoingContext(ctx, "Authorization", token)
	return ctx, cancel
}

// runQuoteStream owns the SubscribeQuote subscription: it (re)subscribes when
// the symbol set changes, reconnects with exponential backoff after a drop, and
// stops when ctx is cancelled (Close).
func (c *Client) runQuoteStream(ctx context.Context) {
	log.Printf("[INFO] Quote stream manager started")
	backoff := quoteStreamInitialBackoff

	for {
		if ctx.Err() != nil {
			log.Printf("[INFO] Quote stream manager stopped")
			return
		}

		desired := c.currentQuoteSymbols()
		symbols := applySymbolCap(desired, c.currentSymbolCap())
		if len(symbols) == 0 {
			// Nothing to subscribe to: wait for a symbol change.
			if !c.waitForQuoteWake(ctx) {
				log.Printf("[INFO] Quote stream manager stopped")
				return
			}
			continue
		}

		voluntary, dropped := c.runQuoteSubscription(ctx, desired, symbols)
		if ctx.Err() != nil {
			log.Printf("[INFO] Quote stream manager stopped")
			return
		}
		if voluntary {
			// Symbol change: resubscribe immediately, no backoff, no outage.
			backoff = quoteStreamInitialBackoff
			continue
		}
		if !dropped {
			continue
		}

		if !sleepOrDone(ctx, backoff) {
			log.Printf("[INFO] Quote stream manager stopped")
			return
		}
		backoff = nextBackoff(backoff)
	}
}

// runQuoteSubscription opens one subscription and consumes it until it ends.
// It reports whether the end was voluntary (a symbol change cancelled it) and
// whether the stream was dropped (so the caller backs off before retrying).
func (c *Client) runQuoteSubscription(ctx context.Context, desired, symbols []string) (voluntary, dropped bool) {
	subCtx, subCancel := c.getStreamContext(ctx)
	defer subCancel()

	c.quoteMu.Lock()
	c.quoteSubCancel = subCancel
	c.quoteMu.Unlock()

	stream, err := c.marketDataClient.SubscribeQuote(subCtx, &marketdata.SubscribeQuoteRequest{
		Symbols: symbols,
	})
	if err != nil {
		if c.quoteSymbolsChanged(desired) {
			return true, false
		}
		if ctx.Err() != nil {
			return false, false
		}
		if c.reduceSymbolCap(err, len(symbols)) {
			// Too many symbols: resubscribe immediately with fewer. This is a
			// negotiation, not an outage, so liveness is left untouched.
			return true, false
		}
		c.logGRPCError("MarketDataService", "SubscribeQuote", err, fmt.Sprintf("Symbols: %v", symbols))
		c.setQuoteStreamState(false)
		return false, true
	}

	log.Printf("[DEBUG] Quote stream subscribed to %d symbol(s): %v", len(symbols), symbols)
	c.trimStreamQuotes(symbols)

	for {
		resp, recvErr := stream.Recv()
		if recvErr != nil {
			if c.quoteSymbolsChanged(desired) {
				return true, false
			}
			if ctx.Err() != nil {
				return false, false
			}
			if c.reduceSymbolCap(recvErr, len(symbols)) {
				return true, false
			}
			log.Printf("[WARN] Quote stream disconnected: %v. Reconnecting...", recvErr)
			c.setQuoteStreamState(false)
			return false, true
		}

		// The stream is proven alive only by data: gRPC opens streams lazily.
		c.setQuoteStreamState(true)

		if resp.Error != nil {
			log.Printf("[WARN] Quote stream reported an error: code=%d %s", resp.Error.Code, resp.Error.Description)
		}

		for _, q := range resp.Quote {
			c.deliverStreamQuote(q)
		}
	}
}

// reduceSymbolCap shrinks the subscription when the broker says it carried too
// many symbols, and reports whether it did. The broker never states the limit,
// so the client halves what was just refused until a subscription survives.
func (c *Client) reduceSymbolCap(err error, attempted int) bool {
	if !IsSymbolLimitError(err) {
		return false
	}

	next := reducedSymbolCap(attempted)

	c.quoteMu.Lock()
	if c.quoteSymbolCap > 0 && c.quoteSymbolCap <= next {
		// Already at or below this size; nothing more to learn here.
		c.quoteMu.Unlock()
		return false
	}
	c.quoteSymbolCap = next
	c.quoteMu.Unlock()

	log.Printf("[WARN] Broker refused a %d-symbol quote subscription (%v); retrying with at most %d",
		attempted, err, next)
	return true
}

// currentSymbolCap returns the largest subscription size known to work, or 0
// while no limit has been discovered.
func (c *Client) currentSymbolCap() int {
	c.quoteMu.Lock()
	defer c.quoteMu.Unlock()
	return c.quoteSymbolCap
}

// QuoteSymbolCap exposes the discovered subscription limit (0 = none hit yet) so
// the UI can size what it asks for.
func (c *Client) QuoteSymbolCap() int { return c.currentSymbolCap() }

// currentQuoteSymbols returns a copy of the desired symbol set.
func (c *Client) currentQuoteSymbols() []string {
	c.quoteMu.Lock()
	defer c.quoteMu.Unlock()
	return append([]string(nil), c.quoteSymbols...)
}

// quoteSymbolsChanged reports whether the set that should be subscribed has
// moved away from the one the current subscription was opened with.
func (c *Client) quoteSymbolsChanged(symbols []string) bool {
	c.quoteMu.Lock()
	defer c.quoteMu.Unlock()
	return !sameSymbolSet(applySymbolCap(c.quoteSymbols, c.quoteSymbolCap), applySymbolCap(symbols, c.quoteSymbolCap))
}

// sameSymbolSet compares two symbol lists ignoring order, so a reshuffle that
// subscribes to exactly the same instruments does not cost a reconnect.
func sameSymbolSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := append([]string(nil), a...)
	sortedB := append([]string(nil), b...)
	slices.Sort(sortedA)
	slices.Sort(sortedB)
	return slices.Equal(sortedA, sortedB)
}

// waitForQuoteWake blocks until the symbol set changes or the manager is
// stopped. It returns false once the manager must stop.
func (c *Client) waitForQuoteWake(ctx context.Context) bool {
	c.quoteMu.Lock()
	wake := c.quoteWake
	c.quoteMu.Unlock()

	select {
	case <-ctx.Done():
		return false
	case <-wake:
		return true
	}
}

// setQuoteStreamState reports a liveness change, deduplicated: the callback only
// fires when the state actually flips.
func (c *Client) setQuoteStreamState(up bool) {
	c.quoteMu.Lock()
	if c.quoteUp == up {
		c.quoteMu.Unlock()
		return
	}
	c.quoteUp = up
	onState := c.quoteOnState
	c.quoteMu.Unlock()

	if up {
		log.Printf("[INFO] Quote stream up")
	} else {
		log.Printf("[INFO] Quote stream down, falling back to polling")
	}

	if onState != nil {
		onState(up)
	}
}

// deliverStreamQuote merges one streamed quote into the last known state and
// hands the result to the consumer.
func (c *Client) deliverStreamQuote(q *marketdata.Quote) {
	if q == nil || q.Symbol == "" {
		return
	}

	c.quoteMu.Lock()
	merged := mergeQuote(c.lastStreamQuotes[q.Symbol], q)
	c.lastStreamQuotes[q.Symbol] = merged
	onQuote := c.quoteOnQuote
	c.quoteMu.Unlock()

	if onQuote != nil {
		onQuote(*quoteToModel(q.Symbol, merged))
	}
}

// trimStreamQuotes drops remembered state for symbols that are no longer part of
// the subscription, so a symbol re-added later starts from a fresh snapshot.
func (c *Client) trimStreamQuotes(symbols []string) {
	c.quoteMu.Lock()
	defer c.quoteMu.Unlock()

	for symbol := range c.lastStreamQuotes {
		if !slices.Contains(symbols, symbol) {
			delete(c.lastStreamQuotes, symbol)
		}
	}
}

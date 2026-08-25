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
	"google.golang.org/grpc/metadata"
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
	if slices.Equal(c.quoteSymbols, normalized) {
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
// tickers), removes duplicates and sorts, so two equal sets compare equal.
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
	slices.Sort(out)
	return out
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

		symbols := c.currentQuoteSymbols()
		if len(symbols) == 0 {
			// Nothing to subscribe to: wait for a symbol change.
			if !c.waitForQuoteWake(ctx) {
				log.Printf("[INFO] Quote stream manager stopped")
				return
			}
			continue
		}

		voluntary, dropped := c.runQuoteSubscription(ctx, symbols)
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
func (c *Client) runQuoteSubscription(ctx context.Context, symbols []string) (voluntary, dropped bool) {
	subCtx, subCancel := c.getStreamContext(ctx)
	defer subCancel()

	c.quoteMu.Lock()
	c.quoteSubCancel = subCancel
	c.quoteMu.Unlock()

	stream, err := c.marketDataClient.SubscribeQuote(subCtx, &marketdata.SubscribeQuoteRequest{
		Symbols: symbols,
	})
	if err != nil {
		if c.quoteSymbolsChanged(symbols) {
			return true, false
		}
		if ctx.Err() != nil {
			return false, false
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
			if c.quoteSymbolsChanged(symbols) {
				return true, false
			}
			if ctx.Err() != nil {
				return false, false
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

// currentQuoteSymbols returns a copy of the desired symbol set.
func (c *Client) currentQuoteSymbols() []string {
	c.quoteMu.Lock()
	defer c.quoteMu.Unlock()
	return append([]string(nil), c.quoteSymbols...)
}

// quoteSymbolsChanged reports whether the desired set has moved away from the
// one the current subscription was opened with.
func (c *Client) quoteSymbolsChanged(symbols []string) bool {
	c.quoteMu.Lock()
	defer c.quoteMu.Unlock()
	return !slices.Equal(c.quoteSymbols, symbols)
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

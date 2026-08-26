package api

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync/atomic"
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
	// Order-insensitive: the same instruments split across streams differently
	// are still the same coverage, and a reshuffle is not worth dropping live
	// subscriptions over.
	if sameSymbolSet(c.quoteSymbols, normalized) {
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

// defaultQuoteShardSize is how many symbols one subscription may carry.
//
// Measured against the real API on 2026-08-26: 15 symbols are accepted, 16 are
// refused with InvalidArgument "Maximum number of symbols exceeded". The limit
// applies **per subscription, not per connection** — three parallel 15-symbol
// streams deliver 45 symbols at once — so a long symbol list becomes several
// streams rather than a truncated one.
const defaultQuoteShardSize = 15

// maxQuoteShards bounds how many parallel subscriptions the client will open.
// Five concurrent streams were verified to work; the cap leaves headroom while
// stopping a pathological symbol list from spawning streams without limit. The
// tail beyond it is dropped, and since the caller's order is priority order,
// what survives is what matters most.
const maxQuoteShards = 8

// shardSymbols splits the desired set into subscriptions the broker accepts,
// preserving order so the leading (highest priority) symbols land in the first
// shard.
func shardSymbols(symbols []string, size int) [][]string {
	if size <= 0 {
		size = defaultQuoteShardSize
	}

	var shards [][]string
	for start := 0; start < len(symbols) && len(shards) < maxQuoteShards; start += size {
		end := min(start+size, len(symbols))
		shards = append(shards, symbols[start:end])
	}

	if dropped := len(symbols) - shardedCount(shards); dropped > 0 {
		log.Printf("[WARN] Quote subscription capped at %d streams; dropping %d lowest-priority symbol(s)",
			maxQuoteShards, dropped)
	}
	return shards
}

// shardedCount totals the symbols actually covered by the shards.
func shardedCount(shards [][]string) int {
	n := 0
	for _, shard := range shards {
		n += len(shard)
	}
	return n
}

// runQuoteStream supervises the subscriptions. It splits the desired symbol set
// into shards the broker will accept and keeps one worker per shard, restarting
// only the shards whose contents actually changed. It stops when ctx is
// cancelled (Close).
func (c *Client) runQuoteStream(ctx context.Context) {
	log.Printf("[INFO] Quote stream manager started")
	defer func() {
		c.stopAllShards()
		log.Printf("[INFO] Quote stream manager stopped")
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		desired := c.currentQuoteSymbols()
		c.reconcileShards(ctx, shardSymbols(desired, c.shardSize()))

		if !c.waitForQuoteWake(ctx) {
			return
		}
	}
}

// reconcileShards brings the running workers in line with the wanted shards:
// unchanged shards are left alone, so a symbol change costs a reconnect only for
// the subscriptions it actually affects.
func (c *Client) reconcileShards(ctx context.Context, wanted [][]string) {
	c.quoteMu.Lock()
	running := c.quoteShards
	c.quoteMu.Unlock()

	keep := make([]*quoteShard, 0, len(wanted))
	used := make([]bool, len(running))

	for _, symbols := range wanted {
		matched := false
		for i, shard := range running {
			if used[i] || !slices.Equal(shard.symbols, symbols) {
				continue
			}
			used[i] = true
			keep = append(keep, shard)
			matched = true
			break
		}
		if matched {
			continue
		}
		keep = append(keep, c.startShard(ctx, symbols))
	}

	c.quoteMu.Lock()
	c.quoteShards = keep
	c.quoteMu.Unlock()

	for i, shard := range running {
		if !used[i] {
			shard.stop()
		}
	}

	// Forget state for symbols no longer carried, so one re-added later starts
	// from a fresh snapshot instead of merging into stale fields.
	kept := make([]string, 0, shardedCount(wanted))
	for _, shard := range keep {
		kept = append(kept, shard.symbols...)
	}
	c.trimStreamQuotes(kept)

	// Reconciliation is not itself a liveness event: a freshly started shard has
	// not received anything yet, and reporting an outage for that would turn
	// every symbol change into a false alarm. The exception is subscribing to
	// nothing at all, which genuinely means no quotes are coming.
	if len(keep) == 0 {
		c.refreshStreamState()
	}
}

// quoteShard is one subscription: a fixed symbol set, its own worker goroutine
// and its own reconnect cycle, so one failing shard cannot take the others down.
type quoteShard struct {
	symbols []string
	cancel  context.CancelFunc
	live    atomic.Bool
}

func (s *quoteShard) stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// startShard launches the worker for one subscription.
func (c *Client) startShard(parent context.Context, symbols []string) *quoteShard {
	ctx, cancel := context.WithCancel(parent)
	shard := &quoteShard{symbols: symbols, cancel: cancel}

	go c.runShard(ctx, shard)
	return shard
}

// runShard keeps one subscription alive, reconnecting with exponential backoff
// after a drop and renegotiating the shard size if the broker refuses it.
func (c *Client) runShard(ctx context.Context, shard *quoteShard) {
	backoff := quoteStreamInitialBackoff

	for {
		// A cancelled shard was retired by the supervisor, not dropped by the
		// broker. It goes quiet without reporting an outage; liveness is left to
		// the shard that replaced it.
		if ctx.Err() != nil {
			shard.live.Store(false)
			return
		}

		voluntary, dropped := c.runQuoteSubscription(ctx, shard)
		if ctx.Err() != nil {
			shard.live.Store(false)
			return
		}
		if voluntary {
			// The shard size was renegotiated; the supervisor re-shards.
			return
		}
		if !dropped {
			continue
		}

		if !sleepOrDone(ctx, backoff) {
			shard.live.Store(false)
			return
		}
		backoff = nextBackoff(backoff)
	}
}

// stopAllShards cancels every worker.
func (c *Client) stopAllShards() {
	c.quoteMu.Lock()
	shards := c.quoteShards
	c.quoteShards = nil
	c.quoteMu.Unlock()

	for _, shard := range shards {
		shard.stop()
	}
}

// shardSize is the current per-subscription symbol budget: the measured default
// unless the broker has refused it, in which case the negotiated cap wins.
func (c *Client) shardSize() int {
	if cap := c.currentSymbolCap(); cap > 0 {
		return cap
	}
	return defaultQuoteShardSize
}

// runQuoteSubscription opens one shard's subscription and consumes it until it
// ends. It reports whether the end was voluntary (the shard size was
// renegotiated, so the supervisor should re-shard) and whether the stream was
// dropped (so the caller backs off before retrying).
func (c *Client) runQuoteSubscription(ctx context.Context, shard *quoteShard) (voluntary, dropped bool) {
	symbols := shard.symbols

	subCtx, subCancel := c.getStreamContext(ctx)
	defer subCancel()

	stream, err := c.marketDataClient.SubscribeQuote(subCtx, &marketdata.SubscribeQuoteRequest{
		Symbols: symbols,
	})
	if err != nil {
		if ctx.Err() != nil {
			return false, false
		}
		if c.reduceSymbolCap(err, len(symbols)) {
			// Too many symbols for one subscription: the supervisor re-shards
			// into smaller pieces. A negotiation, not an outage.
			c.wakeQuoteManager()
			return true, false
		}
		c.logGRPCError("MarketDataService", "SubscribeQuote", err, fmt.Sprintf("Symbols: %v", symbols))
		shard.live.Store(false)
		c.refreshStreamState()
		return false, true
	}

	log.Printf("[DEBUG] Quote stream subscribed to %d symbol(s): %v", len(symbols), symbols)

	for {
		resp, recvErr := stream.Recv()
		if recvErr != nil {
			if ctx.Err() != nil {
				return false, false
			}
			if c.reduceSymbolCap(recvErr, len(symbols)) {
				c.wakeQuoteManager()
				return true, false
			}
			log.Printf("[WARN] Quote stream (%d symbols) disconnected: %v. Reconnecting...", len(symbols), recvErr)
			shard.live.Store(false)
			c.refreshStreamState()
			return false, true
		}

		// The shard is proven alive only by data: gRPC opens streams lazily.
		if !shard.live.Swap(true) {
			c.refreshStreamState()
		}

		if resp.Error != nil {
			log.Printf("[WARN] Quote stream reported an error: code=%d %s", resp.Error.Code, resp.Error.Description)
		}

		for _, q := range resp.Quote {
			c.deliverStreamQuote(q)
		}
	}
}

// wakeQuoteManager nudges the supervisor to re-evaluate the sharding.
func (c *Client) wakeQuoteManager() {
	c.quoteMu.Lock()
	wake := c.quoteWake
	c.quoteMu.Unlock()

	if wake != nil {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

// refreshStreamState recomputes overall liveness from the shards. The stream
// counts as up while any shard is delivering; consumers that need to know
// whether a particular instrument is covered ask SubscribedSymbols instead.
func (c *Client) refreshStreamState() {
	c.quoteMu.Lock()
	shards := append([]*quoteShard(nil), c.quoteShards...)
	c.quoteMu.Unlock()

	up := false
	for _, shard := range shards {
		if shard.live.Load() {
			up = true
			break
		}
	}
	c.setQuoteStreamState(up)
}

// reduceSymbolCap shrinks the per-subscription budget when the broker says a
// shard carried too many symbols, and reports whether it did. The measured
// limit is 15; this exists so an undocumented change to it cannot break the
// terminal — the client halves what was refused until a subscription survives.
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

	log.Printf("[WARN] Broker refused a %d-symbol quote subscription (%v); resharding with at most %d per stream",
		attempted, err, next)
	return true
}

// currentSymbolCap returns the negotiated per-subscription limit, or 0 while the
// measured default has not been contradicted.
func (c *Client) currentSymbolCap() int {
	c.quoteMu.Lock()
	defer c.quoteMu.Unlock()
	return c.quoteSymbolCap
}

// QuoteSymbolCap exposes the negotiated per-subscription limit (0 = the measured
// default still holds).
func (c *Client) QuoteSymbolCap() int { return c.currentSymbolCap() }

// currentQuoteSymbols returns a copy of the desired symbol set.
func (c *Client) currentQuoteSymbols() []string {
	c.quoteMu.Lock()
	defer c.quoteMu.Unlock()
	return append([]string(nil), c.quoteSymbols...)
}

// SubscribedSymbols returns the symbols the stream is actually delivering right
// now: the union of the shards that are live. A shard that has not come up, or
// has dropped and is reconnecting, contributes nothing, so a caller can treat
// the complement as "not covered" whatever the reason.
func (c *Client) SubscribedSymbols() []string {
	c.quoteMu.Lock()
	shards := append([]*quoteShard(nil), c.quoteShards...)
	c.quoteMu.Unlock()

	var out []string
	for _, shard := range shards {
		if shard.live.Load() {
			out = append(out, shard.symbols...)
		}
	}
	return out
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

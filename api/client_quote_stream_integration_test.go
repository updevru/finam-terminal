//go:build integration

package api

import (
	"testing"
	"time"

	"finam-terminal/api/testserver"
	"finam-terminal/models"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// quoteStreamSink collects stream callbacks for assertions.
type quoteStreamSink struct {
	quotes chan models.Quote
	states chan bool
}

func newQuoteStreamSink() *quoteStreamSink {
	return &quoteStreamSink{
		quotes: make(chan models.Quote, 64),
		states: make(chan bool, 64),
	}
}

func (s *quoteStreamSink) start(c *Client) {
	c.StartQuoteStream(
		func(q models.Quote) { s.quotes <- q },
		func(up bool) { s.states <- up },
	)
}

func (s *quoteStreamSink) waitQuote(t *testing.T, deadline time.Duration) models.Quote {
	t.Helper()
	select {
	case q := <-s.quotes:
		return q
	case <-time.After(deadline):
		t.Fatal("timed out waiting for a streamed quote")
		return models.Quote{}
	}
}

func (s *quoteStreamSink) waitState(t *testing.T, want bool, deadline time.Duration) {
	t.Helper()
	timeout := time.After(deadline)
	for {
		select {
		case got := <-s.states:
			if got == want {
				return
			}
		case <-timeout:
			t.Fatalf("timed out waiting for stream state %v", want)
		}
	}
}

func (s *quoteStreamSink) expectNoState(t *testing.T, within time.Duration) {
	t.Helper()
	select {
	case got := <-s.states:
		t.Fatalf("unexpected stream state change to %v", got)
	case <-time.After(within):
	}
}

// waitQuoteStreamCalls blocks until the mock server has opened at least n quote
// streams, or fails the test.
func waitQuoteStreamCalls(t *testing.T, ts *testserver.TestServer, n int64, deadline time.Duration) {
	t.Helper()

	timeout := time.After(deadline)
	for {
		if ts.MarketData.QuoteStreamCallCount.Load() >= n {
			return
		}
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for %d SubscribeQuote calls; got %d", n, ts.MarketData.QuoteStreamCallCount.Load())
		case <-ts.MarketData.QuoteStreamCalled:
		}
	}
}

// TestIntegration_QuoteStream_DeliversSnapshot verifies the happy path: the
// requested symbols reach the server, the snapshot is delivered as a model, and
// the stream is reported up only after real data arrives.
func TestIntegration_QuoteStream_DeliversSnapshot(t *testing.T) {
	client, ts := setupTestServer(t)
	sink := newQuoteStreamSink()

	sink.start(client)
	client.SetQuoteSymbols([]string{"SBER@TQBR"})
	waitQuoteStreamCalls(t, ts, 1, 3*time.Second)

	// Opening a stream proves nothing: gRPC opens lazily, so no up event yet.
	sink.expectNoState(t, 200*time.Millisecond)

	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		Quotes: []*marketdata.Quote{testserver.DefaultStreamQuote("SBER@TQBR", true)},
	}

	q := sink.waitQuote(t, 3*time.Second)
	if q.Symbol != "SBER@TQBR" {
		t.Errorf("quote symbol = %q, want SBER@TQBR", q.Symbol)
	}
	if q.Last != "290.00" {
		t.Errorf("quote Last = %q, want 290.00", q.Last)
	}
	if q.Bid != "289.90" {
		t.Errorf("quote Bid = %q, want 289.90", q.Bid)
	}
	sink.waitState(t, true, 3*time.Second)

	symbols, _ := ts.MarketData.LastQuoteStreamSymbols.Load().([]string)
	if len(symbols) != 1 || symbols[0] != "SBER@TQBR" {
		t.Errorf("subscribed symbols = %v, want [SBER@TQBR]", symbols)
	}
}

// TestIntegration_QuoteStream_MergesIncrement verifies that an incremental
// update keeps the fields the snapshot established.
func TestIntegration_QuoteStream_MergesIncrement(t *testing.T) {
	client, ts := setupTestServer(t)
	sink := newQuoteStreamSink()

	sink.start(client)
	client.SetQuoteSymbols([]string{"SBER@TQBR"})
	waitQuoteStreamCalls(t, ts, 1, 3*time.Second)

	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		Quotes: []*marketdata.Quote{testserver.DefaultStreamQuote("SBER@TQBR", true)},
	}
	sink.waitQuote(t, 3*time.Second)

	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		Quotes: []*marketdata.Quote{testserver.DefaultStreamQuote("SBER@TQBR", false)},
	}

	q := sink.waitQuote(t, 3*time.Second)
	if q.Last != "291.00" {
		t.Errorf("quote Last = %q, want 291.00 from the increment", q.Last)
	}
	if q.Bid != "289.90" {
		t.Errorf("quote Bid = %q, want 289.90 kept from the snapshot", q.Bid)
	}
	if q.Volume != "1500000" {
		t.Errorf("quote Volume = %q, want 1500000 kept from the snapshot", q.Volume)
	}
}

// TestIntegration_QuoteStream_ResubscribesOnSymbolChange verifies that changing
// the symbol set reopens the stream without reporting the stream as down.
func TestIntegration_QuoteStream_ResubscribesOnSymbolChange(t *testing.T) {
	client, ts := setupTestServer(t)
	sink := newQuoteStreamSink()

	sink.start(client)
	client.SetQuoteSymbols([]string{"SBER@TQBR"})
	waitQuoteStreamCalls(t, ts, 1, 3*time.Second)

	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		Quotes: []*marketdata.Quote{testserver.DefaultStreamQuote("SBER@TQBR", true)},
	}
	sink.waitQuote(t, 3*time.Second)
	sink.waitState(t, true, 3*time.Second)

	client.SetQuoteSymbols([]string{"SBER@TQBR", "GAZP@TQBR"})
	waitQuoteStreamCalls(t, ts, 2, 3*time.Second)

	// A deliberate resubscribe is not an outage.
	sink.expectNoState(t, 300*time.Millisecond)

	// Caller order is preserved — it is priority order for the broker's symbol
	// cap — so the assertion is on the set, not the sequence.
	symbols, _ := ts.MarketData.LastQuoteStreamSymbols.Load().([]string)
	if !sameSymbolSet(symbols, []string{"GAZP@TQBR", "SBER@TQBR"}) {
		t.Errorf("subscribed symbols = %v, want both GAZP@TQBR and SBER@TQBR", symbols)
	}
	if got := ts.MarketData.QuoteStreamCallCount.Load(); got != 2 {
		t.Errorf("SubscribeQuote call count = %d, want 2", got)
	}
}

// TestIntegration_QuoteStream_ReconnectsAfterDrop verifies that a dropped stream
// reports down and is reopened after the backoff.
func TestIntegration_QuoteStream_ReconnectsAfterDrop(t *testing.T) {
	client, ts := setupTestServer(t)
	sink := newQuoteStreamSink()

	sink.start(client)
	client.SetQuoteSymbols([]string{"SBER@TQBR"})
	waitQuoteStreamCalls(t, ts, 1, 3*time.Second)

	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		Quotes: []*marketdata.Quote{testserver.DefaultStreamQuote("SBER@TQBR", true)},
	}
	sink.waitQuote(t, 3*time.Second)
	sink.waitState(t, true, 3*time.Second)

	// Drop the stream from the server side.
	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		Err: status.Error(codes.Unavailable, "stream dropped"),
	}

	sink.waitState(t, false, 3*time.Second)
	// The client retries after the initial backoff of about a second.
	waitQuoteStreamCalls(t, ts, 2, 5*time.Second)

	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		Quotes: []*marketdata.Quote{testserver.DefaultStreamQuote("SBER@TQBR", true)},
	}
	sink.waitQuote(t, 3*time.Second)
	sink.waitState(t, true, 3*time.Second)
}

// TestIntegration_QuoteStream_EmptySymbolsDoesNotSubscribe verifies that the
// manager stays idle until it has something to subscribe to.
func TestIntegration_QuoteStream_EmptySymbolsDoesNotSubscribe(t *testing.T) {
	client, ts := setupTestServer(t)
	sink := newQuoteStreamSink()

	sink.start(client)
	client.SetQuoteSymbols(nil)

	time.Sleep(300 * time.Millisecond)
	if got := ts.MarketData.QuoteStreamCallCount.Load(); got != 0 {
		t.Errorf("SubscribeQuote call count = %d, want 0 for an empty symbol set", got)
	}

	// Filtered-out symbols (no MIC) count as empty too.
	client.SetQuoteSymbols([]string{"SBER"})
	time.Sleep(300 * time.Millisecond)
	if got := ts.MarketData.QuoteStreamCallCount.Load(); got != 0 {
		t.Errorf("SubscribeQuote call count = %d, want 0 for symbols without a MIC", got)
	}
}

// TestIntegration_QuoteStream_CloseStopsManager verifies that Close ends the
// stream manager instead of leaving it reconnecting forever.
func TestIntegration_QuoteStream_CloseStopsManager(t *testing.T) {
	ts := testserver.NewTestServer()
	ts.Start()
	defer ts.Stop()

	conn, err := ts.Dial(t.Context())
	if err != nil {
		t.Fatalf("failed to dial test server: %v", err)
	}
	client, err := newClientFromConn(conn, "test-api-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	sink := newQuoteStreamSink()
	sink.start(client)
	client.SetQuoteSymbols([]string{"SBER@TQBR"})
	waitQuoteStreamCalls(t, ts, 1, 3*time.Second)

	if err := client.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	before := ts.MarketData.QuoteStreamCallCount.Load()
	time.Sleep(1500 * time.Millisecond) // longer than the initial backoff
	if got := ts.MarketData.QuoteStreamCallCount.Load(); got != before {
		t.Errorf("SubscribeQuote call count = %d after Close, want %d", got, before)
	}
}

// TestIntegration_QuoteStream_InBandErrorKeepsStream verifies that an error
// reported inside a response does not tear the stream down.
func TestIntegration_QuoteStream_InBandErrorKeepsStream(t *testing.T) {
	client, ts := setupTestServer(t)
	sink := newQuoteStreamSink()

	sink.start(client)
	client.SetQuoteSymbols([]string{"SBER@TQBR"})
	waitQuoteStreamCalls(t, ts, 1, 3*time.Second)

	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		StreamErr: &marketdata.StreamError{Code: 42, Description: "subscription warning"},
	}
	ts.MarketData.QuoteStreamQueue <- testserver.QuoteStreamItem{
		Quotes: []*marketdata.Quote{testserver.DefaultStreamQuote("SBER@TQBR", true)},
	}

	q := sink.waitQuote(t, 3*time.Second)
	if q.Symbol != "SBER@TQBR" {
		t.Errorf("quote symbol = %q, want SBER@TQBR", q.Symbol)
	}
	if got := ts.MarketData.QuoteStreamCallCount.Load(); got != 1 {
		t.Errorf("SubscribeQuote call count = %d, want 1: the stream must survive an in-band error", got)
	}
}

// TestIntegration_QuoteStream_RecoversAfterNarrowingSymbols simulates a server
// that refuses subscriptions above a symbol count — the undocumented risk of
// adding the whole index composition to the existing stream. It proves the
// recovery the UI guard relies on: once the wide set is replaced with the
// narrow one, the positions stream comes back with no manual intervention.
func TestIntegration_QuoteStream_RecoversAfterNarrowingSymbols(t *testing.T) {
	client, ts := setupTestServer(t)
	sink := newQuoteStreamSink()

	const maxSymbols = 2
	ts.MarketData.SubscribeQuoteOverride = func(req *marketdata.SubscribeQuoteRequest, stream marketdata.MarketDataService_SubscribeQuoteServer) error {
		if len(req.Symbols) > maxSymbols {
			return status.Errorf(codes.ResourceExhausted, "too many symbols: %d", len(req.Symbols))
		}
		return stream.Send(&marketdata.SubscribeQuoteResponse{
			Quote: []*marketdata.Quote{testserver.DefaultStreamQuote(req.Symbols[0], true)},
		})
	}

	sink.start(client)

	// The wide set (positions + index composition) is rejected outright. The
	// stream never reports "down" here because it was never up — the client
	// starts in the down state and only reports changes.
	client.SetQuoteSymbols([]string{"SBER@MISX", "GAZP@MISX", "LKOH@MISX", "MOEX@MISX"})
	waitQuoteStreamCalls(t, ts, 1, 5*time.Second)
	sink.expectNoState(t, 500*time.Millisecond)

	// The guard drops the index symbols; the positions alone must work again.
	client.SetQuoteSymbols([]string{"SBER@MISX"})

	q := sink.waitQuote(t, 5*time.Second)
	if q.Symbol != "SBER@MISX" {
		t.Errorf("quote symbol = %q, want SBER@MISX", q.Symbol)
	}
	sink.waitState(t, true, 5*time.Second)

	symbols, _ := ts.MarketData.LastQuoteStreamSymbols.Load().([]string)
	if len(symbols) != 1 || symbols[0] != "SBER@MISX" {
		t.Errorf("subscribed symbols after recovery = %v, want [SBER@MISX]", symbols)
	}
}

// TestIntegration_QuoteStream_AdaptsToTheBrokerSymbolLimit reproduces what the
// real broker did during the first smoke test: it accepted the subscription and
// then killed the stream with InvalidArgument "Maximum number of symbols
// exceeded". The limit is undocumented, so the client has to find it by halving
// what was refused until a subscription survives — and the caller's leading
// symbols must be the ones that stay.
func TestIntegration_QuoteStream_AdaptsToTheBrokerSymbolLimit(t *testing.T) {
	client, ts := setupTestServer(t)
	sink := newQuoteStreamSink()

	const brokerLimit = 3
	ts.MarketData.SubscribeQuoteOverride = func(req *marketdata.SubscribeQuoteRequest, stream marketdata.MarketDataService_SubscribeQuoteServer) error {
		if len(req.Symbols) > brokerLimit {
			return status.Error(codes.InvalidArgument, "Maximum number of symbols exceeded.")
		}
		return stream.Send(&marketdata.SubscribeQuoteResponse{
			Quote: []*marketdata.Quote{testserver.DefaultStreamQuote(req.Symbols[0], true)},
		})
	}

	sink.start(client)

	// Priority order: the position first, then the index composition.
	client.SetQuoteSymbols([]string{
		"SBER@MISX",
		"GAZP@MISX", "LKOH@MISX", "MOEX@MISX", "PLZL@MISX",
		"ROSN@MISX", "TATN@MISX", "VTBR@MISX", "YDEX@MISX",
	})

	// No manual intervention: the client must negotiate its way down on its own.
	q := sink.waitQuote(t, 10*time.Second)
	if q.Symbol != "SBER@MISX" {
		t.Errorf("first quote is for %q, want SBER@MISX — the leading symbol must survive truncation", q.Symbol)
	}
	sink.waitState(t, true, 10*time.Second)

	symbols, _ := ts.MarketData.LastQuoteStreamSymbols.Load().([]string)
	if len(symbols) > brokerLimit {
		t.Errorf("settled on %d symbols, want at most %d", len(symbols), brokerLimit)
	}
	if len(symbols) == 0 || symbols[0] != "SBER@MISX" {
		t.Errorf("settled subscription = %v, want it to start with the highest-priority symbol", symbols)
	}
	if cap := client.QuoteSymbolCap(); cap <= 0 || cap > brokerLimit {
		t.Errorf("discovered cap = %d, want a positive value no greater than %d", cap, brokerLimit)
	}
}

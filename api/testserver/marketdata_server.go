package testserver

import (
	"context"
	"sync/atomic"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// QuoteStreamItem is one item fed into MockMarketDataServer.QuoteStreamQueue:
// quotes to send over the SubscribeQuote stream, an in-band stream error to
// report inside a response, or a transport error that ends the stream
// (simulating a dropped connection).
type QuoteStreamItem struct {
	Quotes    []*marketdata.Quote
	StreamErr *marketdata.StreamError
	Err       error
}

// MockMarketDataServer implements marketdata.MarketDataServiceServer for testing.
type MockMarketDataServer struct {
	marketdata.UnimplementedMarketDataServiceServer

	// QuoteOverride, if set, is called instead of the default behavior.
	QuoteOverride func(ctx context.Context, req *marketdata.QuoteRequest) (*marketdata.QuoteResponse, error)

	// QuoteStreamCallCount tracks the number of SubscribeQuote calls (each call
	// is one stream open, i.e. an initial subscribe, a resubscribe after a
	// symbol change, or a reconnect).
	QuoteStreamCallCount atomic.Int64

	// QuoteStreamCalled is sent to (non-blocking) on every SubscribeQuote call.
	QuoteStreamCalled chan struct{}

	// QuoteStreamQueue feeds the active SubscribeQuote stream. The stream also
	// ends cleanly (nil) when its context is cancelled.
	QuoteStreamQueue chan QuoteStreamItem

	// LastQuoteStreamSymbols stores the symbol list ([]string) of the most
	// recent SubscribeQuote request.
	LastQuoteStreamSymbols atomic.Value

	// SubscribeQuoteOverride, if set, is called instead of the default stream
	// behavior.
	SubscribeQuoteOverride func(req *marketdata.SubscribeQuoteRequest, stream marketdata.MarketDataService_SubscribeQuoteServer) error
}

// NewMockMarketDataServer creates a MockMarketDataServer with defaults.
func NewMockMarketDataServer() *MockMarketDataServer {
	return &MockMarketDataServer{
		QuoteStreamCalled: make(chan struct{}, 100),
		QuoteStreamQueue:  make(chan QuoteStreamItem, 100),
	}
}

// SubscribeQuote streams whatever the test feeds into QuoteStreamQueue until the
// stream context is cancelled or an item carries a transport error.
func (m *MockMarketDataServer) SubscribeQuote(req *marketdata.SubscribeQuoteRequest, stream marketdata.MarketDataService_SubscribeQuoteServer) error {
	m.QuoteStreamCallCount.Add(1)
	m.LastQuoteStreamSymbols.Store(append([]string(nil), req.Symbols...))

	// Non-blocking notification
	select {
	case m.QuoteStreamCalled <- struct{}{}:
	default:
	}

	if m.SubscribeQuoteOverride != nil {
		return m.SubscribeQuoteOverride(req, stream)
	}

	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case item, ok := <-m.QuoteStreamQueue:
			if !ok {
				return nil
			}
			if item.Err != nil {
				return item.Err
			}
			if err := stream.Send(&marketdata.SubscribeQuoteResponse{
				Quote: item.Quotes,
				Error: item.StreamErr,
			}); err != nil {
				return err
			}
		}
	}
}

// LastQuote returns a quote for the requested symbol.
func (m *MockMarketDataServer) LastQuote(ctx context.Context, req *marketdata.QuoteRequest) (*marketdata.QuoteResponse, error) {
	if m.QuoteOverride != nil {
		return m.QuoteOverride(ctx, req)
	}

	q := DefaultQuote(req.Symbol)
	if q == nil {
		return nil, status.Errorf(codes.NotFound, "quote not found for %s", req.Symbol)
	}
	return &marketdata.QuoteResponse{
		Symbol: req.Symbol,
		Quote:  q,
	}, nil
}

// Bars returns candlestick data for the requested symbol.
func (m *MockMarketDataServer) Bars(_ context.Context, req *marketdata.BarsRequest) (*marketdata.BarsResponse, error) {
	bars := DefaultBars(req.Symbol)
	return &marketdata.BarsResponse{
		Symbol: req.Symbol,
		Bars:   bars,
	}, nil
}

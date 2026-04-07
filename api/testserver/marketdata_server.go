package testserver

import (
	"context"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockMarketDataServer implements marketdata.MarketDataServiceServer for testing.
type MockMarketDataServer struct {
	marketdata.UnimplementedMarketDataServiceServer

	// QuoteOverride, if set, is called instead of the default behavior.
	QuoteOverride func(ctx context.Context, req *marketdata.QuoteRequest) (*marketdata.QuoteResponse, error)
}

// NewMockMarketDataServer creates a MockMarketDataServer with defaults.
func NewMockMarketDataServer() *MockMarketDataServer {
	return &MockMarketDataServer{}
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

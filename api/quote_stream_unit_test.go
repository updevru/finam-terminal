package api

import (
	"context"
	"testing"
	"time"

	"finam-terminal/models"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/grpc"
)

// fakeQuoteStream is a manual fake for
// grpc.ServerStreamingClient[marketdata.SubscribeQuoteResponse], mirroring
// fakeJwtRenewalStream.
type fakeQuoteStream struct {
	grpc.ClientStream
	ctx      context.Context
	recvFunc func() (*marketdata.SubscribeQuoteResponse, error)
}

func (f *fakeQuoteStream) Recv() (*marketdata.SubscribeQuoteResponse, error) {
	return f.recvFunc()
}

func (f *fakeQuoteStream) Context() context.Context {
	return f.ctx
}

// blockingQuoteStream returns a stream whose Recv blocks until its context is
// cancelled — the shape of a healthy but quiet subscription.
func blockingQuoteStream(ctx context.Context) *fakeQuoteStream {
	return &fakeQuoteStream{
		ctx: ctx,
		recvFunc: func() (*marketdata.SubscribeQuoteResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
}

func TestNormalizeSymbols(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "filters bare tickers",
			input: []string{"SBER", "SBER@TQBR"},
			want:  []string{"SBER@TQBR"},
		},
		{
			name:  "deduplicates",
			input: []string{"SBER@TQBR", "SBER@TQBR"},
			want:  []string{"SBER@TQBR"},
		},
		{
			name:  "sorts",
			input: []string{"SBER@TQBR", "GAZP@TQBR", "LKOH@TQBR"},
			want:  []string{"GAZP@TQBR", "LKOH@TQBR", "SBER@TQBR"},
		},
		{
			name:  "empty stays empty",
			input: nil,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSymbols(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("normalizeSymbols(%v) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("normalizeSymbols(%v) = %v, want %v", tt.input, got, tt.want)
				}
			}
		})
	}
}

// TestQuoteStream_ResubscribesOnSymbolChange verifies the manager loop without a
// server: a changed symbol set cancels the open subscription and reopens one
// with the new symbols.
func TestQuoteStream_ResubscribesOnSymbolChange(t *testing.T) {
	subscribed := make(chan []string, 4)

	mockMarketData := &mockMarketDataServiceClient{
		SubscribeQuoteFunc: func(ctx context.Context, in *marketdata.SubscribeQuoteRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[marketdata.SubscribeQuoteResponse], error) {
			subscribed <- append([]string(nil), in.Symbols...)
			streamCtx, cancel := context.WithCancel(ctx)
			go func() {
				<-ctx.Done()
				cancel()
			}()
			return blockingQuoteStream(streamCtx), nil
		},
	}

	client := &Client{marketDataClient: mockMarketData}
	t.Cleanup(func() {
		client.quoteMu.Lock()
		cancel := client.quoteCancel
		client.quoteMu.Unlock()
		if cancel != nil {
			cancel()
		}
	})

	client.StartQuoteStream(func(models.Quote) {}, func(bool) {})

	client.SetQuoteSymbols([]string{"SBER@TQBR"})
	first := waitSymbols(t, subscribed)
	if len(first) != 1 || first[0] != "SBER@TQBR" {
		t.Fatalf("first subscription = %v, want [SBER@TQBR]", first)
	}

	client.SetQuoteSymbols([]string{"SBER@TQBR", "GAZP@TQBR"})
	second := waitSymbols(t, subscribed)
	if len(second) != 2 || second[0] != "GAZP@TQBR" || second[1] != "SBER@TQBR" {
		t.Fatalf("second subscription = %v, want sorted [GAZP@TQBR SBER@TQBR]", second)
	}
}

// TestQuoteStream_SameSymbolsDoNotResubscribe verifies that repeating the
// current set (in any order) leaves the open subscription alone.
func TestQuoteStream_SameSymbolsDoNotResubscribe(t *testing.T) {
	subscribed := make(chan []string, 4)

	mockMarketData := &mockMarketDataServiceClient{
		SubscribeQuoteFunc: func(ctx context.Context, in *marketdata.SubscribeQuoteRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[marketdata.SubscribeQuoteResponse], error) {
			subscribed <- append([]string(nil), in.Symbols...)
			return blockingQuoteStream(ctx), nil
		},
	}

	client := &Client{marketDataClient: mockMarketData}
	t.Cleanup(func() {
		client.quoteMu.Lock()
		cancel := client.quoteCancel
		client.quoteMu.Unlock()
		if cancel != nil {
			cancel()
		}
	})

	client.StartQuoteStream(func(models.Quote) {}, func(bool) {})

	client.SetQuoteSymbols([]string{"GAZP@TQBR", "SBER@TQBR"})
	waitSymbols(t, subscribed)

	// Same set, different order, plus a bare ticker that is filtered out.
	client.SetQuoteSymbols([]string{"SBER@TQBR", "GAZP@TQBR", "LKOH"})

	select {
	case got := <-subscribed:
		t.Fatalf("unexpected resubscription with %v", got)
	case <-time.After(300 * time.Millisecond):
	}
}

func waitSymbols(t *testing.T, ch chan []string) []string {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a subscription")
		return nil
	}
}

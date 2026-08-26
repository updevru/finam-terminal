package api

import (
	"context"
	"testing"
	"time"

	"finam-terminal/models"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			// Caller order is priority order: the broker caps how many symbols
			// a subscription may carry, and the client truncates from the end,
			// so whatever the caller puts first must survive.
			name:  "preserves caller priority order",
			input: []string{"SBER@TQBR", "GAZP@TQBR", "LKOH@TQBR"},
			want:  []string{"SBER@TQBR", "GAZP@TQBR", "LKOH@TQBR"},
		},
		{
			name:  "keeps the first occurrence when deduplicating",
			input: []string{"SBER@TQBR", "GAZP@TQBR", "SBER@TQBR"},
			want:  []string{"SBER@TQBR", "GAZP@TQBR"},
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
	// Caller order is preserved (it is priority order); what matters here is
	// that both instruments are subscribed.
	if !sameSymbolSet(second, []string{"SBER@TQBR", "GAZP@TQBR"}) {
		t.Fatalf("second subscription = %v, want both symbols", second)
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

// TestIsSymbolLimitError recognises the broker's undocumented cap on
// SubscribeQuote symbol counts. There is no status code for it — the server
// answers InvalidArgument with a message — so the phrase is what identifies it.
func TestIsSymbolLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "the real message from the broker",
			err:  status.Error(codes.InvalidArgument, "Maximum number of symbols exceeded."),
			want: true,
		},
		{
			name: "case and wording tolerance",
			err:  status.Error(codes.InvalidArgument, "maximum number of symbols is 20"),
			want: true,
		},
		{
			name: "another InvalidArgument is not a symbol limit",
			err:  status.Error(codes.InvalidArgument, "Token is invalid or malformed"),
		},
		{
			name: "a rate limit is not a symbol limit",
			err:  status.Error(codes.ResourceExhausted, "Too Many Requests"),
		},
		{name: "no error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSymbolLimitError(tt.err); got != tt.want {
				t.Errorf("IsSymbolLimitError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestReducedSymbolCap halves the attempted count so the client converges on the
// broker's real limit within a few attempts, and never reduces below one symbol.
func TestReducedSymbolCap(t *testing.T) {
	tests := []struct {
		attempted int
		want      int
	}{
		{attempted: 46, want: 23},
		{attempted: 23, want: 11},
		{attempted: 11, want: 5},
		{attempted: 5, want: 2},
		{attempted: 2, want: 1},
		{attempted: 1, want: 1},
		{attempted: 0, want: 1},
	}

	for _, tt := range tests {
		if got := reducedSymbolCap(tt.attempted); got != tt.want {
			t.Errorf("reducedSymbolCap(%d) = %d, want %d", tt.attempted, got, tt.want)
		}
	}
}

// TestApplySymbolCap truncates from the end, so the caller's highest-priority
// symbols survive.
func TestApplySymbolCap(t *testing.T) {
	symbols := []string{"SBER@TQBR", "GAZP@TQBR", "LKOH@TQBR", "MOEX@TQBR"}

	if got := applySymbolCap(symbols, 0); len(got) != 4 {
		t.Errorf("cap 0 (unknown) truncated to %d symbols, want all 4", len(got))
	}
	got := applySymbolCap(symbols, 2)
	if len(got) != 2 || got[0] != "SBER@TQBR" || got[1] != "GAZP@TQBR" {
		t.Errorf("applySymbolCap(.., 2) = %v, want the first two", got)
	}
	if got := applySymbolCap(symbols, 10); len(got) != 4 {
		t.Errorf("a cap above the list length truncated to %d, want 4", len(got))
	}
}

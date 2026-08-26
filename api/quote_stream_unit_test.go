package api

import (
	"context"
	"fmt"
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

// TestShardSymbols verifies the desired set is split into subscriptions the
// broker will accept. The measured limit is 15 symbols per subscription, and it
// applies per subscription rather than per connection, so a long list becomes
// several parallel streams instead of a truncated one.
func TestShardSymbols(t *testing.T) {
	symbols := make([]string, 46)
	for i := range symbols {
		symbols[i] = fmt.Sprintf("T%d@MISX", i)
	}

	shards := shardSymbols(symbols, defaultQuoteShardSize)

	if len(shards) != 4 {
		t.Fatalf("46 symbols produced %d shards, want 4", len(shards))
	}
	total := 0
	for i, shard := range shards {
		if len(shard) > defaultQuoteShardSize {
			t.Errorf("shard %d carries %d symbols, want at most %d", i, len(shard), defaultQuoteShardSize)
		}
		total += len(shard)
	}
	if total != len(symbols) {
		t.Errorf("shards cover %d symbols, want all %d", total, len(symbols))
	}

	// Priority order is preserved across the split: the first shard holds the
	// leading symbols, which is where positions live.
	if shards[0][0] != "T0@MISX" {
		t.Errorf("first shard starts with %q, want T0@MISX", shards[0][0])
	}

	if got := shardSymbols(nil, defaultQuoteShardSize); len(got) != 0 {
		t.Errorf("no symbols produced %d shards, want none", len(got))
	}
	if got := shardSymbols(symbols[:5], defaultQuoteShardSize); len(got) != 1 || len(got[0]) != 5 {
		t.Errorf("a short list produced %v, want a single shard of 5", got)
	}
}

// TestShardSymbols_BoundsTheStreamCount verifies a very long list cannot spawn
// unbounded parallel streams; the tail is dropped, and priority order means the
// symbols that matter most are the ones kept.
func TestShardSymbols_BoundsTheStreamCount(t *testing.T) {
	symbols := make([]string, defaultQuoteShardSize*(maxQuoteShards+5))
	for i := range symbols {
		symbols[i] = fmt.Sprintf("T%d@MISX", i)
	}

	shards := shardSymbols(symbols, defaultQuoteShardSize)

	if len(shards) != maxQuoteShards {
		t.Errorf("produced %d shards, want the cap of %d", len(shards), maxQuoteShards)
	}
	if shards[0][0] != "T0@MISX" {
		t.Errorf("first shard starts with %q, want the highest-priority symbol", shards[0][0])
	}
}

// TestDefaultQuoteShardSize pins the measured limit. 15 symbols are accepted and
// 16 are refused with "Maximum number of symbols exceeded" (measured against the
// real API on 2026-08-26).
func TestDefaultQuoteShardSize(t *testing.T) {
	if defaultQuoteShardSize != 15 {
		t.Errorf("defaultQuoteShardSize = %d, want 15 (the measured broker limit)", defaultQuoteShardSize)
	}
}

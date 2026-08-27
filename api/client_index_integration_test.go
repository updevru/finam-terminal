//go:build integration

package api

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"finam-terminal/api/testserver"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const testIndexSymbol = "IMOEX@RTSX"

// expireIndexCache backdates the cached composition so the next call is forced
// to refetch. It lives with the tests because nothing in production ages the
// cache by hand.
func (c *Client) expireIndexCache(indexSymbol string) {
	c.indexMu.Lock()
	defer c.indexMu.Unlock()

	entry := c.indexCache[indexSymbol]
	entry.fetchedAt = time.Now().Add(-2 * indexCacheTTL)
	c.indexCache[indexSymbol] = entry
}

// TestIntegration_GetIndexConstituents_Pagination verifies the full composition
// is collected across pages, in API order, with the ticker derived from the
// symbol and a nil weight tolerated.
func TestIntegration_GetIndexConstituents_Pagination(t *testing.T) {
	client, ts := setupTestServer(t)

	got, err := client.GetIndexConstituents(testIndexSymbol)
	if err != nil {
		t.Fatalf("GetIndexConstituents error: %v", err)
	}

	if len(got) != 4 {
		t.Fatalf("expected 4 constituents across 2 pages, got %d", len(got))
	}
	if n := ts.Assets.GetConstituentsCallCount.Load(); n != 2 {
		t.Errorf("expected 2 RPCs (one per page), got %d", n)
	}

	// API order must be preserved: sorting is the UI's job.
	wantSymbols := []string{"SBER@MISX", "GAZP@MISX", "LKOH@MISX", "MOEX@MISX"}
	for i, want := range wantSymbols {
		if got[i].Symbol != want {
			t.Errorf("constituent[%d].Symbol = %q, want %q", i, got[i].Symbol, want)
		}
	}

	if got[0].Ticker != "SBER" {
		t.Errorf("constituent[0].Ticker = %q, want SBER (symbol truncated at @)", got[0].Ticker)
	}
	if got[0].Name != "Сбербанк" {
		t.Errorf("constituent[0].Name = %q, want Сбербанк", got[0].Name)
	}
	if got[0].Sector != "Финансы" {
		t.Errorf("constituent[0].Sector = %q, want Финансы", got[0].Sector)
	}
	if got[0].Weight != 0.008 {
		t.Errorf("constituent[0].Weight = %v, want 0.008", got[0].Weight)
	}
	if got[3].Weight != 0 {
		t.Errorf("constituent[3].Weight = %v, want 0 for a nil weight wrapper", got[3].Weight)
	}
}

// TestIntegration_GetIndexConstituents_CachedWithinTTL verifies a second call
// inside the TTL costs zero RPCs — the composition budget for the whole session
// is 1-2 requests.
func TestIntegration_GetIndexConstituents_CachedWithinTTL(t *testing.T) {
	client, ts := setupTestServer(t)

	first, err := client.GetIndexConstituents(testIndexSymbol)
	if err != nil {
		t.Fatalf("first GetIndexConstituents error: %v", err)
	}
	callsAfterFirst := ts.Assets.GetConstituentsCallCount.Load()

	second, err := client.GetIndexConstituents(testIndexSymbol)
	if err != nil {
		t.Fatalf("second GetIndexConstituents error: %v", err)
	}

	if n := ts.Assets.GetConstituentsCallCount.Load(); n != callsAfterFirst {
		t.Errorf("cached call performed %d extra RPC(s), want 0", n-callsAfterFirst)
	}
	if len(second) != len(first) {
		t.Errorf("cached result has %d constituents, want %d", len(second), len(first))
	}
}

// TestIntegration_GetIndexConstituents_StaleOnError verifies that a failed
// refetch after the TTL expires keeps serving the previous composition instead
// of blanking the tab.
func TestIntegration_GetIndexConstituents_StaleOnError(t *testing.T) {
	client, ts := setupTestServer(t)

	first, err := client.GetIndexConstituents(testIndexSymbol)
	if err != nil {
		t.Fatalf("first GetIndexConstituents error: %v", err)
	}

	client.expireIndexCache(testIndexSymbol)
	ts.Assets.GetConstituentsError = status.Error(codes.Unavailable, "backend down")

	stale, err := client.GetIndexConstituents(testIndexSymbol)
	if err != nil {
		t.Fatalf("expected the stale composition, got error: %v", err)
	}
	if len(stale) != len(first) {
		t.Errorf("stale result has %d constituents, want %d", len(stale), len(first))
	}
}

// TestIntegration_GetIndexConstituents_FirstLoadError verifies that a failure
// with nothing cached is reported, so the tab can show a message and offer R.
func TestIntegration_GetIndexConstituents_FirstLoadError(t *testing.T) {
	client, ts := setupTestServer(t)
	ts.Assets.GetConstituentsError = status.Error(codes.NotFound, "Constituents not found")

	got, err := client.GetIndexConstituents(testIndexSymbol)
	if err == nil {
		t.Fatalf("expected an error on the first load, got %d constituents", len(got))
	}
	if got != nil {
		t.Errorf("expected no constituents on error, got %d", len(got))
	}
}

// TestIntegration_GetIndexConstituents_EmptyIsError verifies an empty response
// is treated as a failed load and does not poison the cache: once the API
// recovers, the next call fetches the real composition.
func TestIntegration_GetIndexConstituents_EmptyIsError(t *testing.T) {
	client, ts := setupTestServer(t)
	ts.Assets.EmptyConstituents = true

	if _, err := client.GetIndexConstituents(testIndexSymbol); err == nil {
		t.Fatal("expected an error for an empty composition")
	}

	ts.Assets.EmptyConstituents = false
	got, err := client.GetIndexConstituents(testIndexSymbol)
	if err != nil {
		t.Fatalf("GetIndexConstituents after recovery: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("expected 4 constituents after recovery, got %d — the empty response poisoned the cache", len(got))
	}
}

// TestIntegration_GetIndexConstituents_PageLimit verifies the guard against a
// server that never stops handing out cursors.
func TestIntegration_GetIndexConstituents_PageLimit(t *testing.T) {
	client, ts := setupTestServer(t)

	// Every page reports another page after it.
	ts.Assets.EndlessConstituents = true

	got, err := client.GetIndexConstituents(testIndexSymbol)
	if err != nil {
		t.Fatalf("GetIndexConstituents error: %v", err)
	}
	if n := ts.Assets.GetConstituentsCallCount.Load(); n != int64(maxConstituentPages) {
		t.Errorf("made %d RPCs, want the page guard to stop at %d", n, maxConstituentPages)
	}
	if len(got) == 0 {
		t.Error("expected the pages collected so far to be returned")
	}
}

// TestIntegration_GetIndexConstituents_TTL documents the cache window used for
// the composition.
func TestIntegration_GetIndexConstituents_TTL(t *testing.T) {
	if indexCacheTTL != 24*time.Hour {
		t.Errorf("indexCacheTTL = %v, want 24h", indexCacheTTL)
	}
}

// TestIntegration_GetQuotes_RateLimitAbortsBatch verifies a ResourceExhausted
// answer ends the batch instead of walking through the remaining symbols. The
// Index tab's fallback asks for the whole composition at once, so continuing
// past a rate limit would turn one refused call into dozens.
func TestIntegration_GetQuotes_RateLimitAbortsBatch(t *testing.T) {
	client, ts := setupTestServer(t)

	var calls atomic.Int64
	ts.MarketData.QuoteOverride = func(context.Context, *marketdata.QuoteRequest) (*marketdata.QuoteResponse, error) {
		calls.Add(1)
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	symbols := []string{"SBER@MISX", "GAZP@MISX", "LKOH@MISX", "MOEX@MISX"}
	quotes, err := client.GetQuotes("", symbols)

	if err == nil {
		t.Fatal("expected an error when the broker rate-limits the batch")
	}
	if !IsRateLimited(err) {
		t.Errorf("IsRateLimited(%v) = false, want true", err)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("made %d LastQuote call(s) after a rate limit, want 1 (the batch must stop)", n)
	}
	if len(quotes) != 0 {
		t.Errorf("got %d quotes, want none", len(quotes))
	}
}

// TestIntegration_GetQuotes_OrdinaryErrorSkipsSymbol verifies an ordinary
// per-symbol failure still skips just that symbol, as before.
func TestIntegration_GetQuotes_OrdinaryErrorSkipsSymbol(t *testing.T) {
	client, ts := setupTestServer(t)

	ts.MarketData.QuoteOverride = func(_ context.Context, req *marketdata.QuoteRequest) (*marketdata.QuoteResponse, error) {
		if req.Symbol == "GAZP@TQBR" {
			return nil, status.Error(codes.NotFound, "no such symbol")
		}
		return &marketdata.QuoteResponse{Symbol: req.Symbol, Quote: testserver.DefaultQuote("SBER@TQBR")}, nil
	}

	quotes, err := client.GetQuotes("", []string{"GAZP@TQBR", "SBER@TQBR"})
	if err != nil {
		t.Fatalf("GetQuotes error: %v", err)
	}
	if len(quotes) != 1 {
		t.Errorf("got %d quotes, want 1 (the failing symbol is skipped, the other survives)", len(quotes))
	}
}

// TestIntegration_GetQuotes_WithoutAccountSkipsLotResolution guards a defect the
// log exposed: the Index tab sweep asks for quotes with no account context, and
// getFullSymbol still tried to warm the lot caches, so every swept symbol cost a
// GetAsset and a GetAssetParams call that the broker rejected outright with
// "Invalid arguments:account_id". They failed, so nothing was cached, so it
// repeated on every pass — 494 guaranteed failures in one session.
func TestIntegration_GetQuotes_WithoutAccountSkipsLotResolution(t *testing.T) {
	client, ts := setupTestServer(t)

	// The asset cache is warmed at startup; start counting from here.
	assetsBefore := ts.Assets.GetAssetCallCount.Load()
	paramsBefore := ts.Assets.GetAssetParamsCallCount.Load()

	quotes, err := client.GetQuotes("", []string{"SBER@TQBR"})
	if err != nil {
		t.Fatalf("GetQuotes error: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("got %d quotes, want 1", len(quotes))
	}

	if n := ts.Assets.GetAssetCallCount.Load() - assetsBefore; n != 0 {
		t.Errorf("made %d GetAsset call(s) with no account, want 0 — they can only fail", n)
	}
	if n := ts.Assets.GetAssetParamsCallCount.Load() - paramsBefore; n != 0 {
		t.Errorf("made %d GetAssetParams call(s) with no account, want 0 — they can only fail", n)
	}
}

// TestIntegration_GetQuotes_WithAccountStillWarmsLots verifies the fix is
// narrow: with a real account the lot caches are still warmed as a side effect,
// which is what makes the positions view cheap.
func TestIntegration_GetQuotes_WithAccountStillWarmsLots(t *testing.T) {
	client, ts := setupTestServer(t)

	// Use a symbol the startup cache did not already resolve.
	assetsBefore := ts.Assets.GetAssetCallCount.Load()

	if _, err := client.GetQuotes("ACC-1", []string{"LKOH@TQBR"}); err != nil {
		t.Fatalf("GetQuotes error: %v", err)
	}

	if n := ts.Assets.GetAssetCallCount.Load() - assetsBefore; n == 0 {
		t.Error("with an account, the lot cache should still be warmed")
	}
}

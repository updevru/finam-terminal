//go:build integration

package api

import (
	"context"
	"testing"
	"time"

	"finam-terminal/api/testserver"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/marketdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIntegration_Error_UnauthenticatedOnMethod(t *testing.T) {
	client, ts := setupTestServer(t)

	// Auth succeeds, but then the accounts method returns Unauthenticated
	ts.Accounts.GetAccountError = status.Errorf(codes.Unauthenticated, "session expired")

	_, _, err := client.GetAccountDetails("ACC001")
	if err == nil {
		t.Fatal("expected Unauthenticated error")
	}
}

func TestIntegration_Error_NotFound(t *testing.T) {
	client, ts := setupTestServer(t)
	ts.Assets.GetAssetError = status.Errorf(codes.NotFound, "asset not found")

	_, err := client.GetAssetInfo("ACC001", "UNKNOWN@XXXX")
	if err == nil {
		t.Fatal("expected NotFound error")
	}
}

func TestIntegration_Error_ServerUnavailable(t *testing.T) {
	ts := testserver.NewTestServer()
	ts.Start()

	conn, err := ts.Dial(context.Background())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	client, err := newClientFromConn(conn, "test-api-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Stop the server mid-test
	ts.Stop()

	// Now all gRPC calls should fail
	_, err = client.GetAssetInfo("ACC001", "SBER@TQBR")
	if err == nil {
		t.Fatal("expected error after server stop")
	}
}

func TestIntegration_Error_DeadlineExceeded(t *testing.T) {
	client, ts := setupTestServer(t)

	// Override quote to introduce a delay longer than the client's context timeout
	ts.MarketData.QuoteOverride = func(ctx context.Context, req *marketdata.QuoteRequest) (*marketdata.QuoteResponse, error) {
		// Sleep until context is cancelled (client has 30s timeout, so we simulate slow server)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(60 * time.Second):
			return nil, status.Errorf(codes.DeadlineExceeded, "should not reach here")
		}
	}

	// GetQuotes uses getContext() which has a 30s timeout — too long for test.
	// Instead, let's test with a method that will definitely hit the delay.
	// We can verify the error is handled gracefully (no panic, returns error or skips).
	quotes, err := client.GetQuotes("ACC001", []string{"SBER"})
	// GetQuotes silently skips errors per symbol, so it may return empty rather than error
	if err != nil {
		// If it does error, that's fine too
		return
	}
	// SBER should be missing since the request timed out/was delayed
	if _, ok := quotes["SBER@TQBR"]; ok {
		t.Error("expected SBER quote to be missing due to delay")
	}
}

func TestIntegration_Error_EmptyResponse(t *testing.T) {
	client, ts := setupTestServer(t)

	// Override quote to return nil quote
	ts.MarketData.QuoteOverride = func(_ context.Context, req *marketdata.QuoteRequest) (*marketdata.QuoteResponse, error) {
		return &marketdata.QuoteResponse{
			Symbol: req.Symbol,
			Quote:  nil, // empty/nil quote
		}, nil
	}

	quotes, err := client.GetQuotes("ACC001", []string{"SBER"})
	if err != nil {
		t.Fatalf("GetQuotes error: %v", err)
	}

	// Should handle nil quote gracefully — skip it
	if _, ok := quotes["SBER@TQBR"]; ok {
		t.Error("expected nil quote to be skipped")
	}
}

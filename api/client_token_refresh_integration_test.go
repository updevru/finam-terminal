//go:build integration

package api

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"finam-terminal/api/testserver"

	"github.com/FinamWeb/finam-trade-api/go/grpc/tradeapi/v1/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// setupShortLivedClient creates a client with a very short JWT expiry
// so the refresh goroutine triggers quickly.
func setupShortLivedClient(t *testing.T, expiry time.Duration) (*Client, *testserver.TestServer) {
	t.Helper()

	ts := testserver.NewTestServer()
	ts.Auth.TokenExpiry = expiry
	ts.Start()

	conn, err := ts.Dial(context.Background())
	if err != nil {
		t.Fatalf("failed to dial test server: %v", err)
	}

	client, err := newClientFromConn(conn, "test-api-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		ts.Stop()
	})

	return client, ts
}

func TestIntegration_TokenRefresh_BeforeExpiry(t *testing.T) {
	// Token expires in 5 seconds; refresh should happen ~3s in (5s - 2min buffer => immediate/1s)
	client, ts := setupShortLivedClient(t, 5*time.Second)
	_ = client // keep alive

	// Initial auth call = 1. Wait for at least one refresh call (= 2+).
	deadline := time.After(10 * time.Second)
	for {
		count := ts.Auth.AuthCallCount.Load()
		if count >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for refresh; auth calls: %d", count)
		case <-ts.Auth.AuthCalled:
			// Got a notification, check count again
		}
	}
}

func TestIntegration_TokenRefresh_RetryOnFailure(t *testing.T) {
	ts := testserver.NewTestServer()
	// Short expiry so refresh triggers quickly
	ts.Auth.TokenExpiry = 5 * time.Second
	ts.Start()

	conn, err := ts.Dial(context.Background())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	client, err := newClientFromConn(conn, "test-api-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		ts.Stop()
	})

	// After initial auth succeeds, make subsequent auth calls fail then succeed
	var callNum atomic.Int64
	ts.Auth.AuthOverride = func(_ context.Context, req *auth.AuthRequest) (*auth.AuthResponse, error) {
		n := callNum.Add(1)
		if n == 1 {
			// First refresh attempt: fail
			return nil, status.Errorf(codes.Unavailable, "temporary failure")
		}
		// Second attempt: succeed
		jwt := testserver.MakeJWT(time.Now().Add(1 * time.Hour))
		return &auth.AuthResponse{Token: jwt}, nil
	}

	// Wait for at least 3 total auth calls (initial + fail + retry succeed)
	deadline := time.After(45 * time.Second)
	for {
		count := ts.Auth.AuthCallCount.Load()
		if count >= 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for retry; auth calls: %d, override calls: %d", count, callNum.Load())
		case <-ts.Auth.AuthCalled:
		}
	}
}

func TestIntegration_TokenRefresh_StopsOnClose(t *testing.T) {
	ts := testserver.NewTestServer()
	ts.Auth.TokenExpiry = 5 * time.Second
	ts.Start()
	defer ts.Stop()

	conn, err := ts.Dial(context.Background())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	client, err := newClientFromConn(conn, "test-api-token")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Close the client — this should stop the refresh goroutine
	if err := client.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}

	// Record count after close
	countAfterClose := ts.Auth.AuthCallCount.Load()

	// Wait a bit and verify no more auth calls happen
	time.Sleep(3 * time.Second)
	countLater := ts.Auth.AuthCallCount.Load()

	if countLater > countAfterClose {
		t.Errorf("expected no more auth calls after Close, but count went from %d to %d", countAfterClose, countLater)
	}
}

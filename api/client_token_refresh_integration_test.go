//go:build integration

package api

import (
	"context"
	"testing"
	"time"

	"finam-terminal/api/testserver"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// setupStreamClient creates a client backed by the mock test server, ready to
// exercise the SubscribeJwtRenewal streaming token refresh.
func setupStreamClient(t *testing.T) (*Client, *testserver.TestServer) {
	t.Helper()

	ts := testserver.NewTestServer()
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

// waitForJwtRenewalCall blocks until the mock server observes at least n
// SubscribeJwtRenewal calls (initial subscribe + any reconnects), or fails
// the test after deadline.
func waitForJwtRenewalCall(t *testing.T, ts *testserver.TestServer, n int64, deadline time.Duration) {
	t.Helper()

	timeout := time.After(deadline)
	for {
		if ts.Auth.JwtRenewalCallCount.Load() >= n {
			return
		}
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for %d SubscribeJwtRenewal call(s); got %d", n, ts.Auth.JwtRenewalCallCount.Load())
		case <-ts.Auth.JwtRenewalCalled:
		}
	}
}

// currentToken safely reads the client's current in-memory JWT.
func currentToken(c *Client) string {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.token
}

func TestIntegration_TokenRefresh_UpdatesFromStream(t *testing.T) {
	client, ts := setupStreamClient(t)

	// Wait for the client to open the SubscribeJwtRenewal stream.
	waitForJwtRenewalCall(t, ts, 1, 5*time.Second)

	initialToken := currentToken(client)

	newToken := testserver.MakeJWT(time.Now().Add(1 * time.Hour))
	ts.Auth.JwtRenewalQueue <- testserver.JwtRenewalItem{Token: newToken}

	deadline := time.After(5 * time.Second)
	for currentToken(client) != newToken {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for token to update via stream; initial=%q current=%q want=%q",
				initialToken, currentToken(client), newToken)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func TestIntegration_TokenRefresh_ReconnectsAfterDrop(t *testing.T) {
	client, ts := setupStreamClient(t)

	waitForJwtRenewalCall(t, ts, 1, 5*time.Second)

	// Drop the stream — the client must reconnect (open a new stream).
	ts.Auth.JwtRenewalQueue <- testserver.JwtRenewalItem{Err: status.Errorf(codes.Unavailable, "stream dropped")}

	waitForJwtRenewalCall(t, ts, 2, 10*time.Second)

	// The reconnected stream should still work: push a token and confirm it
	// reaches the client.
	newToken := testserver.MakeJWT(time.Now().Add(1 * time.Hour))
	ts.Auth.JwtRenewalQueue <- testserver.JwtRenewalItem{Token: newToken}

	deadline := time.After(5 * time.Second)
	for currentToken(client) != newToken {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for token to update after reconnect; current=%q want=%q",
				currentToken(client), newToken)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func TestIntegration_TokenRefresh_StopsOnClose(t *testing.T) {
	ts := testserver.NewTestServer()
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

	waitForJwtRenewalCall(t, ts, 1, 5*time.Second)

	// Close the client — this should stop the renewal stream/goroutine.
	if err := client.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}

	countAfterClose := ts.Auth.JwtRenewalCallCount.Load()

	// Wait a bit and verify no reconnect attempts happen after Close.
	time.Sleep(3 * time.Second)
	countLater := ts.Auth.JwtRenewalCallCount.Load()

	if countLater > countAfterClose {
		t.Errorf("expected no more SubscribeJwtRenewal calls after Close, but count went from %d to %d", countAfterClose, countLater)
	}
}

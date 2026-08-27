//go:build integration

package api

import (
	"context"
	"testing"
	"time"

	"finam-terminal/api/testserver"
)

// TestIntegration_TokenWatch_RenewsWhileTheStreamIsSilent proves the client
// keeps its session valid on its own.
//
// The SubscribeJwtRenewal stream is subscribed and healthy here but sends
// nothing — exactly the state the real client is in after a reconnect, because
// the broker only delivers the next token on its own ~14 minute schedule
// counted from the subscribe. The token's expiry keeps running regardless, so
// without an independent renewal the session dies and every RPC starts failing
// with Unauthenticated.
func TestIntegration_TokenWatch_RenewsWhileTheStreamIsSilent(t *testing.T) {
	restore := setTokenWatchTimingForTest(20*time.Millisecond, 1*time.Second)
	defer restore()

	ts := testserver.NewTestServer()
	// A session far shorter than the real 15 minutes, so the test does not wait.
	ts.Auth.TokenExpiry = 2 * time.Second
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

	firstExpiry := client.TokenExpiry()
	if firstExpiry.IsZero() {
		t.Fatal("startup did not record a token expiry")
	}
	authCallsAfterStartup := ts.Auth.AuthCallCount.Load()

	// The renewal must happen before the current token expires, not after.
	deadline := time.After(time.Until(firstExpiry))
	for {
		if ts.Auth.AuthCallCount.Load() > authCallsAfterStartup && client.TokenExpiry().After(firstExpiry) {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("token was not renewed before it expired: Auth calls %d -> %d, expiry %v -> %v",
				authCallsAfterStartup, ts.Auth.AuthCallCount.Load(), firstExpiry, client.TokenExpiry())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// setTokenWatchTimingForTest shortens the expiry watchdog so a test does not
// have to wait out a real session, and restores the production values.
func setTokenWatchTimingForTest(interval, lead time.Duration) func() {
	prevInterval, prevLead := tokenWatchInterval, tokenRenewLead
	tokenWatchInterval, tokenRenewLead = interval, lead
	return func() {
		tokenWatchInterval, tokenRenewLead = prevInterval, prevLead
	}
}

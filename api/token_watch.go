package api

import (
	"context"
	"log"
	"time"
)

// shouldReauthenticate reports whether the session token must be renewed now:
// it is inside the lead window before expiry, or already past it.
//
// A zero expiry means TokenDetails never answered, so nothing is known about
// the session. Renewing blindly would mean re-authenticating on every tick, so
// the unknown case is left to the renewal stream.
func shouldReauthenticate(expiry, now time.Time, lead time.Duration) bool {
	if expiry.IsZero() {
		return false
	}
	return !now.Before(expiry.Add(-lead))
}

// tokenRenewLead is how long before expiry the client renews the session on its
// own, and tokenWatchInterval is how often it checks. They are vars so tests can
// shorten them.
//
// The interval is deliberately short relative to the ~15 minute session: after
// the machine wakes from sleep the token is already dead, and the terminal must
// not stay blind for longer than one tick.
var (
	tokenRenewLead     = 2 * time.Minute
	tokenWatchInterval = 30 * time.Second
)

// watchTokenExpiry renews the session token before it expires, independently of
// the SubscribeJwtRenewal stream.
//
// The stream alone is not enough: the broker sends the next token on a ~14
// minute schedule counted from the subscribe, and sends nothing when a stream is
// (re)opened. Every reconnect therefore pushes the next delivery back while the
// current token's expiry keeps running, and a machine that slept through the
// session wakes with a token that is already dead. In that gap every RPC —
// account data and every quote subscription alike — fails with Unauthenticated
// until the broker's own schedule comes round.
//
// It stops silently once ctx is cancelled (Close).
func (c *Client) watchTokenExpiry(ctx context.Context) {
	log.Printf("[INFO] Session expiry watchdog started")
	ticker := time.NewTicker(tokenWatchInterval)
	defer func() {
		ticker.Stop()
		log.Printf("[INFO] Session expiry watchdog stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !shouldReauthenticate(c.TokenExpiry(), time.Now(), tokenRenewLead) {
				continue
			}
			if err := c.authenticate(c.apiToken); err != nil {
				log.Printf("[ERROR] Session renewal failed: %v. Retrying in %v...", err, tokenWatchInterval)
				continue
			}
			log.Printf("[INFO] Session renewed before expiry")
		}
	}
}

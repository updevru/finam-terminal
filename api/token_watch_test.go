package api

import (
	"testing"
	"time"
)

// TestShouldReauthenticate covers the decision that keeps the session alive
// independently of the SubscribeJwtRenewal stream. The stream delivers a new
// token roughly every 14 minutes counted from the moment it was subscribed, so
// any drop shifts the next delivery while the current token's expiry keeps
// running — the client has to renew on its own before that gap opens.
func TestShouldReauthenticate(t *testing.T) {
	now := time.Date(2026, 8, 27, 20, 25, 0, 0, time.UTC)
	const lead = 2 * time.Minute

	tests := []struct {
		name   string
		expiry time.Time
		want   bool
	}{
		{
			name:   "fresh token is left alone",
			expiry: now.Add(15 * time.Minute),
			want:   false,
		},
		{
			name:   "just outside the lead window",
			expiry: now.Add(lead + time.Second),
			want:   false,
		},
		{
			name:   "inside the lead window",
			expiry: now.Add(lead - time.Second),
			want:   true,
		},
		{
			name:   "already expired while the machine slept",
			expiry: now.Add(-3 * time.Hour),
			want:   true,
		},
		{
			name:   "unknown expiry is not a reason to renew",
			expiry: time.Time{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldReauthenticate(tt.expiry, now, lead); got != tt.want {
				t.Errorf("shouldReauthenticate(%v, %v, %v) = %v; want %v",
					tt.expiry, now, lead, got, tt.want)
			}
		})
	}
}

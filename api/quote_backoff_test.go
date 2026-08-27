package api

import (
	"testing"
	"time"
)

// TestNextShardBackoff covers how long a shard waits before reconnecting.
//
// The delay must start over once a subscription has actually delivered
// something: a shard that ran for hours and then hit a network blip is not the
// same case as one that cannot connect at all, and making it wait out the cap
// leaves the terminal without quotes for half a minute for no reason.
func TestNextShardBackoff(t *testing.T) {
	tests := []struct {
		name      string
		current   time.Duration
		delivered bool
		want      time.Duration
	}{
		{
			name:    "the first retry waits the initial backoff",
			current: 0,
			want:    quoteStreamInitialBackoff,
		},
		{
			name:    "a shard that never connected backs off further",
			current: quoteStreamInitialBackoff,
			want:    2 * quoteStreamInitialBackoff,
		},
		{
			name:    "the backoff is capped",
			current: quoteStreamMaxBackoff,
			want:    quoteStreamMaxBackoff,
		},
		{
			name:      "a shard that delivered data starts over",
			current:   quoteStreamMaxBackoff,
			delivered: true,
			want:      quoteStreamInitialBackoff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextShardBackoff(tt.current, tt.delivered); got != tt.want {
				t.Errorf("nextShardBackoff(%v, %v) = %v; want %v",
					tt.current, tt.delivered, got, tt.want)
			}
		})
	}
}

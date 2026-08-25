package main

import (
	"testing"

	"finam-terminal/version"
)

// TestOfferPendingUpdateSkipsDevBuild verifies a development build never even
// looks at the update cache: no dialog, no file, no network. This is the
// guarantee that `go run main.go` and the Docker image behave exactly as they
// did before the updater existed.
func TestOfferPendingUpdateSkipsDevBuild(t *testing.T) {
	for _, v := range []string{"dev", "dev (a1b2c3d)", "dev (a1b2c3d, dirty)", ""} {
		t.Run(v, func(t *testing.T) {
			prev := version.Version
			version.Version = v
			t.Cleanup(func() { version.Version = prev })

			if offerPendingUpdate() {
				t.Errorf("offerPendingUpdate() = true for version %q, want false on a dev build", v)
			}
		})
	}
}

// TestOfferPendingUpdateSkipsWhenCurrentIsNewer verifies a release build that
// is ahead of whatever the cache holds is never offered a downgrade, so no
// dialog can appear.
func TestOfferPendingUpdateSkipsWhenCurrentIsNewer(t *testing.T) {
	prev := version.Version
	version.Version = "v999.0.0"
	t.Cleanup(func() { version.Version = prev })

	if offerPendingUpdate() {
		t.Error("offerPendingUpdate() = true while running a version newer than any release")
	}
}

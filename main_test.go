package main

import (
	"testing"

	"finam-terminal/version"
)

// TestPendingUpdateSkipsDevBuild verifies a development build never even looks
// at the update cache: no dialog, no file, no network. This is the guarantee
// that `go run main.go` and the Docker image behave exactly as they did before
// the updater existed.
func TestPendingUpdateSkipsDevBuild(t *testing.T) {
	for _, v := range []string{"dev", "dev (a1b2c3d)", "dev (a1b2c3d, dirty)", ""} {
		t.Run(v, func(t *testing.T) {
			prev := version.Version
			version.Version = v
			t.Cleanup(func() { version.Version = prev })

			if got := pendingUpdate(); got != "" {
				t.Errorf("pendingUpdate() = %q for version %q, want empty on a dev build", got, v)
			}
		})
	}
}

// TestPendingUpdateSkipsWhenCurrentIsNewer verifies a release build that is
// ahead of whatever the cache holds is never offered a downgrade.
func TestPendingUpdateSkipsWhenCurrentIsNewer(t *testing.T) {
	prev := version.Version
	version.Version = "v999.0.0"
	t.Cleanup(func() { version.Version = prev })

	if got := pendingUpdate(); got != "" {
		t.Errorf("pendingUpdate() = %q while running a version newer than any release, want empty", got)
	}
}

// TestOfferPendingUpdateIgnoresEmptyVersion verifies no dialog is raised when
// there is nothing to offer — the empty version must short-circuit before any
// tview application is created.
func TestOfferPendingUpdateIgnoresEmptyVersion(t *testing.T) {
	if offerPendingUpdate("") {
		t.Error("offerPendingUpdate(\"\") = true, want false with nothing to install")
	}
}

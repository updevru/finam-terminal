package ui

import (
	"strings"
	"testing"

	"finam-terminal/version"
)

// testReleaseTag is the version the indicator tests pretend to be running.
const testReleaseTag = "v0.13.0"

// asReleaseBuild makes version.String() report testReleaseTag for the duration
// of a test, so the update logic is exercised the way it behaves on a real
// release build.
func asReleaseBuild(t *testing.T) {
	t.Helper()
	prev := version.Version
	version.Version = testReleaseTag
	t.Cleanup(func() { version.Version = prev })
}

// TestHeaderLabelIsVersionOnly verifies the header shows nothing but the name
// and the running version, including the "v" prefix rule for bare numeric
// versions. An available update must never change this text — the user is told
// about updates by the startup dialog, not by the header.
func TestHeaderLabelIsVersionOnly(t *testing.T) {
	tests := []struct {
		name    string
		current string
		want    string
	}{
		{name: "release tag", current: "v0.13.0", want: " Finam Terminal v0.13.0 "},
		{name: "bare numeric version gets a v", current: "0.13.0", want: " Finam Terminal v0.13.0 "},
		{name: "dev build", current: "dev (a1b2c3d)", want: " Finam Terminal dev (a1b2c3d) "},
		{name: "empty version", current: "", want: " Finam Terminal  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := headerLabel(tt.current); got != tt.want {
				t.Errorf("headerLabel(%q) = %q, want %q", tt.current, got, tt.want)
			}
		})
	}
}

// TestHeaderHasNoUpdateIndicator is the regression guard for the removal: no
// matter what the application knows about a newer release, the header text
// must stay clean.
func TestHeaderHasNoUpdateIndicator(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)

	before := app.header.GetText(true)

	app.SetUpdateAvailable("v99.0.0")

	after := app.header.GetText(true)
	if after != before {
		t.Errorf("header changed after SetUpdateAvailable: %q -> %q, want it untouched", before, after)
	}
	for _, unwanted := range []string{"⚡", "v99.0.0", "U"} {
		if strings.Contains(after, unwanted) {
			t.Errorf("header = %q, must not contain %q", after, unwanted)
		}
	}
}

// TestSetUpdateAvailableStoresVersion verifies the setter still records the
// version — the U hotkey and the next startup dialog both read it.
func TestSetUpdateAvailableStoresVersion(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)

	app.SetUpdateAvailable("v99.0.0")

	if got := app.LatestVersion(); got != "v99.0.0" {
		t.Errorf("LatestVersion() = %q, want v99.0.0", got)
	}
}

// TestSetUpdateAvailableIgnoresOlderVersion verifies a stale or bogus version
// is never recorded.
func TestSetUpdateAvailableIgnoresOlderVersion(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)

	app.SetUpdateAvailable("v0.0.1")

	if got := app.LatestVersion(); got != "" {
		t.Errorf("LatestVersion() = %q, want it left empty", got)
	}
}

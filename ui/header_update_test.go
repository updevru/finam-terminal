package ui

import (
	"strings"
	"testing"

	"finam-terminal/version"
)

// testReleaseTag is the version the indicator tests pretend to be running.
const testReleaseTag = "v0.13.0"

// asReleaseBuild makes version.String() report testReleaseTag for the duration
// of a test, so the update indicator logic is exercised the way it behaves on
// a real release build.
func asReleaseBuild(t *testing.T) {
	t.Helper()
	prev := version.Version
	version.Version = testReleaseTag
	t.Cleanup(func() { version.Version = prev })
}

// TestHeaderLabelWithoutUpdate verifies the header is unchanged when no newer
// release is known, including the "v" prefix rule for bare numeric versions.
func TestHeaderLabelWithoutUpdate(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    string
	}{
		{name: "release tag", current: "v0.13.0", latest: "", want: " Finam Terminal v0.13.0 "},
		{name: "bare numeric version gets a v", current: "0.13.0", latest: "", want: " Finam Terminal v0.13.0 "},
		{name: "dev build", current: "dev (a1b2c3d)", latest: "", want: " Finam Terminal dev (a1b2c3d) "},
		{name: "same version is not an update", current: "v0.13.0", latest: "v0.13.0", want: " Finam Terminal v0.13.0 "},
		{name: "older release is not an update", current: "v0.15.0", latest: "v0.14.0", want: " Finam Terminal v0.15.0 "},
		{name: "dev build ignores a newer release", current: "dev", latest: "v0.14.0", want: " Finam Terminal dev "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := headerLabel(tt.current, tt.latest); got != tt.want {
				t.Errorf("headerLabel(%q, %q) = %q, want %q", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

// TestHeaderLabelWithUpdate verifies the indicator appears with the new
// version number and the hint about the U key.
func TestHeaderLabelWithUpdate(t *testing.T) {
	got := headerLabel("v0.13.0", "v0.14.0")

	for _, want := range []string{"Finam Terminal", "v0.13.0", "v0.14.0", "⚡", "U"} {
		if !strings.Contains(got, want) {
			t.Errorf("headerLabel() = %q, want it to contain %q", got, want)
		}
	}
	if !strings.Contains(got, "[yellow]") || !strings.Contains(got, "[-]") {
		t.Errorf("headerLabel() = %q, want the indicator wrapped in dynamic colour tags", got)
	}
}

// TestSetUpdateAvailableRedrawsHeader verifies the setter stores the version
// and repaints the header text.
func TestSetUpdateAvailableRedrawsHeader(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)

	before := app.header.GetText(true)
	if strings.Contains(before, "⚡") {
		t.Fatalf("header already shows an update indicator: %q", before)
	}

	app.SetUpdateAvailable("v99.0.0")

	after := app.header.GetText(true)
	if !strings.Contains(after, "⚡") || !strings.Contains(after, "v99.0.0") {
		t.Errorf("header after SetUpdateAvailable = %q, want the indicator and v99.0.0", after)
	}
	if got := app.LatestVersion(); got != "v99.0.0" {
		t.Errorf("LatestVersion() = %q, want v99.0.0", got)
	}
}

// TestSetUpdateAvailableIgnoresOlderVersion verifies a stale or bogus version
// never lights the indicator.
func TestSetUpdateAvailableIgnoresOlderVersion(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)

	app.SetUpdateAvailable("v0.0.1")

	if strings.Contains(app.header.GetText(true), "⚡") {
		t.Errorf("header = %q, want no indicator for an older version", app.header.GetText(true))
	}
	if got := app.LatestVersion(); got != "" {
		t.Errorf("LatestVersion() = %q, want it left empty", got)
	}
}

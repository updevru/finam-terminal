package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

// TestDefaults verifies the package-level variables have their expected
// fallback values when no ldflags injection has occurred.
func TestDefaults(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version default = %q, want %q", Version, "dev")
	}
	if Commit != "unknown" {
		t.Errorf("Commit default = %q, want %q", Commit, "unknown")
	}
	if BuildDate != "unknown" {
		t.Errorf("BuildDate default = %q, want %q", BuildDate, "unknown")
	}
}

// TestFormatVersion exercises the pure formatter for every meaningful
// combination of (version, commit, dirty).
func TestFormatVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		dirty   bool
		want    string
	}{
		{
			name:    "released tag with v prefix",
			version: "v1.2.3",
			commit:  "unknown",
			dirty:   false,
			want:    "v1.2.3",
		},
		{
			name:    "released tag ignores commit and dirty",
			version: "v1.2.3",
			commit:  "abcdef0",
			dirty:   true,
			want:    "v1.2.3",
		},
		{
			name:    "released tag without v prefix",
			version: "1.2.3",
			commit:  "unknown",
			dirty:   false,
			want:    "1.2.3",
		},
		{
			name:    "released semver pre-release",
			version: "v2.0.0-rc1",
			commit:  "unknown",
			dirty:   false,
			want:    "v2.0.0-rc1",
		},
		{
			name:    "dev with no commit",
			version: "dev",
			commit:  "unknown",
			dirty:   false,
			want:    "dev",
		},
		{
			name:    "dev with empty commit",
			version: "dev",
			commit:  "",
			dirty:   false,
			want:    "dev",
		},
		{
			name:    "dev with unknown commit but dirty",
			version: "dev",
			commit:  "unknown",
			dirty:   true,
			want:    "dev",
		},
		{
			name:    "dev with commit",
			version: "dev",
			commit:  "abc1234",
			dirty:   false,
			want:    "dev (abc1234)",
		},
		{
			name:    "dev with commit and dirty",
			version: "dev",
			commit:  "abc1234",
			dirty:   true,
			want:    "dev (abc1234, dirty)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersion(tt.version, tt.commit, tt.dirty)
			if got != tt.want {
				t.Errorf("formatVersion(%q, %q, %v) = %q, want %q",
					tt.version, tt.commit, tt.dirty, got, tt.want)
			}
		})
	}
}

// TestStringInjected verifies String() returns the injected version
// verbatim, bypassing any VCS fallback.
func TestStringInjected(t *testing.T) {
	restore := overrideVars(t, "v9.9.9", "deadbee", "2026-04-06T12:00:00Z")
	defer restore()

	got := String()
	if got != "v9.9.9" {
		t.Errorf("String() = %q, want %q", got, "v9.9.9")
	}
}

// TestStringDevWithInjectedCommit verifies that when Commit is set explicitly
// (e.g. via ldflags) but Version stays as "dev", String() includes the commit
// without consulting runtime/debug.
func TestStringDevWithInjectedCommit(t *testing.T) {
	restore := overrideVars(t, "dev", "fa1afe1", "unknown")
	defer restore()

	// Force the readBuildInfo hook to return false to prove we don't need it.
	restoreHook := overrideReadBuildInfo(t, func() (*debug.BuildInfo, bool) {
		return nil, false
	})
	defer restoreHook()

	got := String()
	if got != "dev (fa1afe1)" {
		t.Errorf("String() = %q, want %q", got, "dev (fa1afe1)")
	}
}

// TestStringDevFallbackToVCS verifies that when Version=="dev" and
// Commit=="unknown", String() falls back to runtime/debug.ReadBuildInfo
// and constructs the display string from vcs.revision/vcs.modified.
func TestStringDevFallbackToVCS(t *testing.T) {
	restore := overrideVars(t, "dev", "unknown", "unknown")
	defer restore()

	restoreHook := overrideReadBuildInfo(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "0123456789abcdef"},
				{Key: "vcs.modified", Value: "false"},
			},
		}, true
	})
	defer restoreHook()

	got := String()
	if got != "dev (0123456)" {
		t.Errorf("String() = %q, want %q", got, "dev (0123456)")
	}
}

// TestStringDevFallbackDirty mirrors the previous case but flips
// vcs.modified to true.
func TestStringDevFallbackDirty(t *testing.T) {
	restore := overrideVars(t, "dev", "unknown", "unknown")
	defer restore()

	restoreHook := overrideReadBuildInfo(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef1234567890"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	})
	defer restoreHook()

	got := String()
	if got != "dev (abcdef1, dirty)" {
		t.Errorf("String() = %q, want %q", got, "dev (abcdef1, dirty)")
	}
}

// TestStringDevFallbackNoBuildInfo verifies that when ReadBuildInfo returns
// ok=false (e.g. binary built outside a module context), String() degrades
// gracefully to "dev".
func TestStringDevFallbackNoBuildInfo(t *testing.T) {
	restore := overrideVars(t, "dev", "unknown", "unknown")
	defer restore()

	restoreHook := overrideReadBuildInfo(t, func() (*debug.BuildInfo, bool) {
		return nil, false
	})
	defer restoreHook()

	got := String()
	if got != "dev" {
		t.Errorf("String() = %q, want %q", got, "dev")
	}
}

// TestStringDevFallbackEmptyVCS verifies that when ReadBuildInfo returns ok
// but no vcs.revision setting (e.g. tarball build), String() returns "dev".
func TestStringDevFallbackEmptyVCS(t *testing.T) {
	restore := overrideVars(t, "dev", "unknown", "unknown")
	defer restore()

	restoreHook := overrideReadBuildInfo(t, func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "GOOS", Value: "linux"},
			},
		}, true
	})
	defer restoreHook()

	got := String()
	if got != "dev" {
		t.Errorf("String() = %q, want %q", got, "dev")
	}
}

// TestInfo verifies Info() returns the raw package variables verbatim.
func TestInfo(t *testing.T) {
	restore := overrideVars(t, "v1.0.0", "deadbeef", "2026-04-06T12:00:00Z")
	defer restore()

	v, c, d := Info()
	if v != "v1.0.0" {
		t.Errorf("Info() version = %q, want %q", v, "v1.0.0")
	}
	if c != "deadbeef" {
		t.Errorf("Info() commit = %q, want %q", c, "deadbeef")
	}
	if d != "2026-04-06T12:00:00Z" {
		t.Errorf("Info() buildDate = %q, want %q", d, "2026-04-06T12:00:00Z")
	}
}

// TestStringRealBuildInfo is a smoke test that exercises the real
// runtime/debug path without overriding the hook. We can't assert the
// exact value (it depends on the test runner environment), but we can
// require that it doesn't panic and starts with "dev".
func TestStringRealBuildInfo(t *testing.T) {
	restore := overrideVars(t, "dev", "unknown", "unknown")
	defer restore()

	got := String()
	if !strings.HasPrefix(got, "dev") {
		t.Errorf("String() = %q, want prefix %q", got, "dev")
	}
}

// overrideVars replaces the package-level variables for the duration of a
// test and returns a restore function.
func overrideVars(t *testing.T, version, commit, buildDate string) func() {
	t.Helper()
	prevV, prevC, prevD := Version, Commit, BuildDate
	Version, Commit, BuildDate = version, commit, buildDate
	return func() {
		Version, Commit, BuildDate = prevV, prevC, prevD
	}
}

// overrideReadBuildInfo replaces the readBuildInfo hook with a stub for the
// duration of a test and returns a restore function.
func overrideReadBuildInfo(t *testing.T, fn func() (*debug.BuildInfo, bool)) func() {
	t.Helper()
	prev := readBuildInfo
	readBuildInfo = fn
	return func() {
		readBuildInfo = prev
	}
}

package updater

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withStateDir points the state file at a temporary directory for the
// duration of a test and restores the production resolver afterwards.
func withStateDir(t *testing.T, dir string) {
	t.Helper()
	prev := stateDirFunc
	stateDirFunc = func() (string, error) { return dir, nil }
	t.Cleanup(func() { stateDirFunc = prev })
}

// TestLoadStateMissingFile verifies a missing cache is not an error: the very
// first run must behave as "never checked" without disturbing startup.
func TestLoadStateMissingFile(t *testing.T) {
	withStateDir(t, t.TempDir())

	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v, want nil", err)
	}
	if !state.LastCheck.IsZero() || state.LatestVersion != "" || state.ReleaseURL != "" {
		t.Errorf("LoadState() = %+v, want zero state", state)
	}
}

// TestLoadStateCorruptJSON verifies a damaged cache degrades to "never
// checked" instead of failing the launch.
func TestLoadStateCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	withStateDir(t, dir)

	if err := os.WriteFile(filepath.Join(dir, stateFileName), []byte("{not json"), 0644); err != nil {
		t.Fatalf("seed corrupt state: %v", err)
	}

	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v, want nil", err)
	}
	if !state.LastCheck.IsZero() || state.LatestVersion != "" {
		t.Errorf("LoadState() = %+v, want zero state", state)
	}
}

// TestLoadStateEmptyFile verifies a zero-length file is treated like a
// missing one (a crash mid-write must not wedge the checker).
func TestLoadStateEmptyFile(t *testing.T) {
	dir := t.TempDir()
	withStateDir(t, dir)

	if err := os.WriteFile(filepath.Join(dir, stateFileName), nil, 0644); err != nil {
		t.Fatalf("seed empty state: %v", err)
	}

	state, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v, want nil", err)
	}
	if !state.LastCheck.IsZero() {
		t.Errorf("LoadState() = %+v, want zero state", state)
	}
}

// TestSaveLoadRoundTrip verifies every field survives a write/read cycle.
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	withStateDir(t, dir)

	want := State{
		LastCheck:     time.Date(2026, 8, 25, 9, 12, 33, 0, time.UTC),
		LatestVersion: "v0.14.0",
		ReleaseURL:    "https://github.com/updevru/finam-terminal/releases/tag/v0.14.0",
		PublishedAt:   time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC),
	}

	if err := SaveState(want); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	got, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if !got.LastCheck.Equal(want.LastCheck) {
		t.Errorf("LastCheck = %v, want %v", got.LastCheck, want.LastCheck)
	}
	if !got.PublishedAt.Equal(want.PublishedAt) {
		t.Errorf("PublishedAt = %v, want %v", got.PublishedAt, want.PublishedAt)
	}
	if got.LatestVersion != want.LatestVersion {
		t.Errorf("LatestVersion = %q, want %q", got.LatestVersion, want.LatestVersion)
	}
	if got.ReleaseURL != want.ReleaseURL {
		t.Errorf("ReleaseURL = %q, want %q", got.ReleaseURL, want.ReleaseURL)
	}
}

// TestSaveStateIsAtomic verifies the temp+rename write leaves no leftovers
// and that a second save overwrites cleanly.
func TestSaveStateIsAtomic(t *testing.T) {
	dir := t.TempDir()
	withStateDir(t, dir)

	if err := SaveState(State{LatestVersion: "v0.14.0", LastCheck: time.Now()}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	if err := SaveState(State{LatestVersion: "v0.15.0", LastCheck: time.Now()}); err != nil {
		t.Fatalf("second SaveState() error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temporary file %q left behind after SaveState", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("directory contains %d entries, want exactly 1 (%s)", len(entries), stateFileName)
	}

	got, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if got.LatestVersion != "v0.15.0" {
		t.Errorf("LatestVersion = %q, want %q", got.LatestVersion, "v0.15.0")
	}
}

// TestSaveStateCreatesDirectory verifies the config directory is created on
// demand — the updater may run before any token was ever saved.
func TestSaveStateCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", ".finam-cli")
	withStateDir(t, dir)

	if err := SaveState(State{LatestVersion: "v0.14.0"}); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, stateFileName)); err != nil {
		t.Fatalf("state file not created: %v", err)
	}
}

// TestStateDirUnavailable verifies both operations fail softly when the home
// directory cannot be resolved: no panic, LoadState still yields zero state.
func TestStateDirUnavailable(t *testing.T) {
	prev := stateDirFunc
	stateDirFunc = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { stateDirFunc = prev })

	state, err := LoadState()
	if err == nil {
		t.Error("LoadState() error = nil, want an error when the config dir is unavailable")
	}
	if !state.LastCheck.IsZero() {
		t.Errorf("LoadState() = %+v, want zero state", state)
	}
	if err := SaveState(State{LatestVersion: "v0.14.0"}); err == nil {
		t.Error("SaveState() error = nil, want an error when the config dir is unavailable")
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUserConfigDir verifies the exported config directory resolves to
// ~/.finam-cli — the same directory saveTokenInternal writes .env into.
func TestUserConfigDir(t *testing.T) {
	dir, err := UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error = %v, want nil", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory available: %v", err)
	}

	want := filepath.Join(home, ".finam-cli")
	if dir != want {
		t.Errorf("UserConfigDir() = %q, want %q", dir, want)
	}
}

// TestUserConfigDirMatchesTokenLocation guards the single-source-of-truth
// rule: the token .env and any other config file must share one directory.
func TestUserConfigDirMatchesTokenLocation(t *testing.T) {
	home := t.TempDir()

	if err := saveTokenInternal(home, "test-token"); err != nil {
		t.Fatalf("saveTokenInternal() error = %v", err)
	}

	envPath := filepath.Join(userConfigDirIn(home), ".env")
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf("token .env not found at %s: %v", envPath, err)
	}
}

package updater

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeExecutable writes a stand-in binary and points executablePath at it.
func fakeExecutable(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "finam-terminal")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	if err := os.WriteFile(exe, []byte(content), 0755); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}

	prev := executablePath
	executablePath = func() (string, error) { return exe, nil }
	t.Cleanup(func() { executablePath = prev })
	return exe
}

// TestReplaceExecutableSwapsBinary verifies the two-step rename installs the
// new binary in place of the old one.
func TestReplaceExecutableSwapsBinary(t *testing.T) {
	exe := fakeExecutable(t, "old binary")
	tmp := filepath.Join(filepath.Dir(exe), "update.tmp")
	if err := os.WriteFile(tmp, []byte("new binary"), 0755); err != nil {
		t.Fatalf("write replacement: %v", err)
	}

	if err := replaceExecutable(exe, tmp); err != nil {
		t.Fatalf("replaceExecutable() error = %v", err)
	}

	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read replaced executable: %v", err)
	}
	if string(got) != "new binary" {
		t.Errorf("executable content = %q, want %q", got, "new binary")
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("temporary file still present after a successful swap (err %v)", err)
	}

	backup := exe + backupSuffix
	_, backupErr := os.Stat(backup)
	if runtime.GOOS == "windows" {
		if backupErr != nil {
			t.Errorf("backup %s missing on windows: %v", backup, backupErr)
		}
	} else if !os.IsNotExist(backupErr) {
		t.Errorf("backup %s should be removed on unix (err %v)", backup, backupErr)
	}
}

// TestReplaceExecutableRollsBack verifies a failure of the second rename
// restores the original binary byte for byte.
func TestReplaceExecutableRollsBack(t *testing.T) {
	exe := fakeExecutable(t, "old binary")
	missing := filepath.Join(filepath.Dir(exe), "does-not-exist.tmp")

	err := replaceExecutable(exe, missing)
	if err == nil {
		t.Fatal("replaceExecutable() error = nil, want a failure for a missing replacement")
	}

	got, readErr := os.ReadFile(exe)
	if readErr != nil {
		t.Fatalf("original executable is gone after a failed swap: %v", readErr)
	}
	if string(got) != "old binary" {
		t.Errorf("executable content = %q, want the original %q", got, "old binary")
	}
	if _, statErr := os.Stat(exe + backupSuffix); !os.IsNotExist(statErr) {
		t.Errorf("backup left behind after rollback (err %v)", statErr)
	}
}

// TestCleanupStaleBackup verifies the leftover .old file from a previous
// Windows update is removed on the next launch.
func TestCleanupStaleBackup(t *testing.T) {
	exe := fakeExecutable(t, "current binary")
	backup := exe + backupSuffix
	if err := os.WriteFile(backup, []byte("previous binary"), 0755); err != nil {
		t.Fatalf("write stale backup: %v", err)
	}

	CleanupStaleBackup()

	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Errorf("stale backup still present (err %v)", err)
	}
	if _, err := os.Stat(exe); err != nil {
		t.Errorf("cleanup removed the running executable: %v", err)
	}
}

// TestCleanupStaleBackupNoop verifies cleanup is silent when there is nothing
// to clean and when the executable path cannot be resolved.
func TestCleanupStaleBackupNoop(t *testing.T) {
	fakeExecutable(t, "current binary")
	CleanupStaleBackup() // no backup exists — must not panic or log noise

	prev := executablePath
	executablePath = func() (string, error) { return "", errors.New("no executable") }
	t.Cleanup(func() { executablePath = prev })
	CleanupStaleBackup()
}

// TestEnsureWritableRejectsUnwritableDir verifies an install directory we
// cannot write to is reported as ErrNotWritable rather than failing halfway
// through the update.
func TestEnsureWritableRejectsUnwritableDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir")

	err := ensureWritable(missing)
	if err == nil {
		t.Fatal("ensureWritable() error = nil, want ErrNotWritable")
	}
	if !errors.Is(err, ErrNotWritable) {
		t.Errorf("ensureWritable() error = %v, want it to wrap ErrNotWritable", err)
	}
}

// TestEnsureWritableAcceptsTempDir verifies the happy path leaves no probe
// file behind.
func TestEnsureWritableAcceptsTempDir(t *testing.T) {
	dir := t.TempDir()
	if err := ensureWritable(dir); err != nil {
		t.Fatalf("ensureWritable() error = %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("write probe left %d entries behind, want none", len(entries))
	}
}

// TestManualUpdateCommand verifies the fallback instruction names the right
// installer for the running platform.
func TestManualUpdateCommand(t *testing.T) {
	got := ManualUpdateCommand()
	if got == "" {
		t.Fatal("ManualUpdateCommand() = empty, want an installer command")
	}
	want := "install.sh"
	if runtime.GOOS == "windows" {
		want = "install.ps1"
	}
	if !strings.Contains(got, want) {
		t.Errorf("ManualUpdateCommand() = %q, want it to mention %s", got, want)
	}
}

// TestSelfUpdateReplacesRunningBinary drives the whole flow against a local
// server: download, verify, swap.
func TestSelfUpdateReplacesRunningBinary(t *testing.T) {
	assetName, err := CurrentAssetName()
	if err != nil {
		t.Skipf("no published build for this platform: %v", err)
	}

	exe := fakeExecutable(t, "old binary")
	payload := []byte("brand new binary payload")

	mux := http.NewServeMux()
	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})
	mux.HandleFunc("/sums", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sha256Hex(payload) + "  " + assetName + "\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := &Release{TagName: "v0.14.0", Assets: []Asset{
		{Name: assetName, DownloadURL: srv.URL + "/binary", Size: int64(len(payload))},
		{Name: checksumsAssetName, DownloadURL: srv.URL + "/sums"},
	}}

	var lastDone int64
	if err := SelfUpdate(context.Background(), rel, func(done, total int64) { lastDone = done }); err != nil {
		t.Fatalf("SelfUpdate() error = %v", err)
	}

	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read updated executable: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("executable content = %q, want %q", got, payload)
	}
	if lastDone != int64(len(payload)) {
		t.Errorf("final progress = %d, want %d", lastDone, len(payload))
	}

	assertNoTempFiles(t, filepath.Dir(exe))
}

// TestSelfUpdateRejectsCorruptDownload verifies a checksum mismatch leaves the
// running binary untouched.
func TestSelfUpdateRejectsCorruptDownload(t *testing.T) {
	assetName, err := CurrentAssetName()
	if err != nil {
		t.Skipf("no published build for this platform: %v", err)
	}

	exe := fakeExecutable(t, "old binary")

	mux := http.NewServeMux()
	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("corrupted payload"))
	})
	mux.HandleFunc("/sums", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sha256Hex([]byte("the expected payload")) + "  " + assetName + "\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rel := &Release{TagName: "v0.14.0", Assets: []Asset{
		{Name: assetName, DownloadURL: srv.URL + "/binary", Size: 17},
		{Name: checksumsAssetName, DownloadURL: srv.URL + "/sums"},
	}}

	if err := SelfUpdate(context.Background(), rel, nil); err == nil {
		t.Fatal("SelfUpdate() error = nil, want a checksum failure")
	}

	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read executable after a failed update: %v", err)
	}
	if string(got) != "old binary" {
		t.Errorf("executable content = %q, want the original %q", got, "old binary")
	}
	assertNoTempFiles(t, filepath.Dir(exe))
}

// TestSelfUpdateUnsupportedPlatformAsset verifies a release without an asset
// for this platform fails before touching anything on disk.
func TestSelfUpdateUnsupportedPlatformAsset(t *testing.T) {
	exe := fakeExecutable(t, "old binary")
	rel := &Release{TagName: "v0.14.0", Assets: []Asset{{Name: "finam-terminal-plan9-mips"}}}

	err := SelfUpdate(context.Background(), rel, nil)
	if err == nil {
		t.Fatal("SelfUpdate() error = nil, want a missing asset error")
	}

	got, readErr := os.ReadFile(exe)
	if readErr != nil || string(got) != "old binary" {
		t.Errorf("executable changed after a failed update: %q (err %v)", got, readErr)
	}
	assertNoTempFiles(t, filepath.Dir(exe))
}

// assertNoTempFiles fails when an update left a .tmp file in the install
// directory.
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temporary file %q left in the install directory", e.Name())
		}
	}
}

package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// binaryPayload is the stand-in for a released binary in the download tests.
var binaryPayload = []byte(strings.Repeat("finam-terminal-binary-payload\n", 64))

// sha256Hex is the checksum helper the tests compare against.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// downloadFixture wires a release whose binary and checksums assets are served
// by a local test server. Handlers may be overridden per test.
type downloadFixture struct {
	release   *Release
	asset     *Asset
	binaryURL string
	sumsURL   string
}

// newDownloadFixture starts a server serving the payload at /binary and,
// when withSums is true, a checksums.txt at /sums.
func newDownloadFixture(t *testing.T, payload []byte, withSums bool, sums string) *downloadFixture {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = w.Write(payload)
	})
	if withSums {
		mux.HandleFunc("/sums", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(sums))
		})
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	f := &downloadFixture{binaryURL: srv.URL + "/binary", sumsURL: srv.URL + "/sums"}
	assets := []Asset{{
		Name:        "finam-terminal-linux-amd64",
		DownloadURL: f.binaryURL,
		Size:        int64(len(binaryPayload)),
	}}
	if withSums {
		assets = append(assets, Asset{Name: checksumsAssetName, DownloadURL: f.sumsURL})
	}
	f.release = &Release{TagName: "v0.14.0", Assets: assets}
	f.asset = &f.release.Assets[0]
	return f
}

// defaultSums is a sha256sum-style checksums.txt covering the payload plus an
// unrelated entry, to prove the parser picks the right line.
func defaultSums() string {
	return "0000000000000000000000000000000000000000000000000000000000000000  finam-terminal-darwin-arm64\n" +
		sha256Hex(binaryPayload) + "  finam-terminal-linux-amd64\n"
}

// TestDownloadAssetWritesFileAndReportsProgress verifies the happy path: the
// payload lands at the destination and progress is reported monotonically.
func TestDownloadAssetWritesFileAndReportsProgress(t *testing.T) {
	f := newDownloadFixture(t, binaryPayload, true, defaultSums())
	dest := filepath.Join(t.TempDir(), "download.tmp")

	var updates []int64
	var lastTotal int64
	err := downloadAsset(context.Background(), f.release, f.asset, dest, func(done, total int64) {
		updates = append(updates, done)
		lastTotal = total
	})
	if err != nil {
		t.Fatalf("downloadAsset() error = %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != string(binaryPayload) {
		t.Errorf("downloaded %d bytes, want the %d-byte payload", len(got), len(binaryPayload))
	}

	if len(updates) == 0 {
		t.Fatal("progress callback was never invoked")
	}
	for i := 1; i < len(updates); i++ {
		if updates[i] < updates[i-1] {
			t.Errorf("progress went backwards: %v", updates)
			break
		}
	}
	if updates[len(updates)-1] != int64(len(binaryPayload)) {
		t.Errorf("final progress = %d, want %d", updates[len(updates)-1], len(binaryPayload))
	}
	if lastTotal != int64(len(binaryPayload)) {
		t.Errorf("reported total = %d, want %d", lastTotal, len(binaryPayload))
	}
}

// TestDownloadAssetRejectsTamperedPayload verifies a checksum mismatch fails
// the download and leaves nothing behind.
func TestDownloadAssetRejectsTamperedPayload(t *testing.T) {
	tampered := append([]byte("tampered"), binaryPayload...)
	f := newDownloadFixture(t, tampered, true, defaultSums())
	dir := t.TempDir()
	dest := filepath.Join(dir, "download.tmp")

	err := downloadAsset(context.Background(), f.release, f.asset, dest, nil)
	if err == nil {
		t.Fatal("downloadAsset() error = nil, want a checksum mismatch")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "sha256") &&
		!strings.Contains(strings.ToLower(err.Error()), "контрольн") {
		t.Errorf("error %q does not mention the checksum", err)
	}
	assertNoLeftovers(t, dir)
}

// TestDownloadAssetFallsBackToSize verifies releases published before
// checksums.txt existed are still verifiable by asset size.
func TestDownloadAssetFallsBackToSize(t *testing.T) {
	f := newDownloadFixture(t, binaryPayload, false, "")
	dest := filepath.Join(t.TempDir(), "download.tmp")

	if err := downloadAsset(context.Background(), f.release, f.asset, dest, nil); err != nil {
		t.Fatalf("downloadAsset() error = %v, want the size fallback to accept the payload", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if len(got) != len(binaryPayload) {
		t.Errorf("downloaded %d bytes, want %d", len(got), len(binaryPayload))
	}
}

// TestDownloadAssetSizeMismatch verifies a truncated download is rejected when
// no checksums.txt is available.
func TestDownloadAssetSizeMismatch(t *testing.T) {
	f := newDownloadFixture(t, binaryPayload[:10], false, "")
	dir := t.TempDir()
	dest := filepath.Join(dir, "download.tmp")

	err := downloadAsset(context.Background(), f.release, f.asset, dest, nil)
	if err == nil {
		t.Fatal("downloadAsset() error = nil, want a size mismatch")
	}
	assertNoLeftovers(t, dir)
}

// TestDownloadAssetMissingChecksumEntry verifies a checksums.txt without a
// line for our asset falls back to the size check rather than failing.
func TestDownloadAssetMissingChecksumEntry(t *testing.T) {
	sums := "0000000000000000000000000000000000000000000000000000000000000000  finam-terminal-darwin-arm64\n"
	f := newDownloadFixture(t, binaryPayload, true, sums)
	dest := filepath.Join(t.TempDir(), "download.tmp")

	if err := downloadAsset(context.Background(), f.release, f.asset, dest, nil); err != nil {
		t.Fatalf("downloadAsset() error = %v, want the size fallback", err)
	}
}

// TestDownloadAssetContextCancelled verifies aborting mid-body reports an
// error and removes the partial file.
func TestDownloadAssetContextCancelled(t *testing.T) {
	blocked := make(chan struct{})
	served := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(binaryPayload)*2))
		_, _ = w.Write(binaryPayload)
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		close(served)
		<-blocked
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	defer close(blocked)

	rel := &Release{TagName: "v0.14.0", Assets: []Asset{{
		Name:        "finam-terminal-linux-amd64",
		DownloadURL: srv.URL + "/binary",
		Size:        int64(len(binaryPayload)) * 2,
	}}}

	dir := t.TempDir()
	dest := filepath.Join(dir, "download.tmp")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-served
		cancel()
	}()

	if err := downloadAsset(ctx, rel, &rel.Assets[0], dest, nil); err == nil {
		t.Fatal("downloadAsset() error = nil, want a context error")
	}
	assertNoLeftovers(t, dir)
}

// TestDownloadAssetHTTPError verifies a non-200 response is an error.
func TestDownloadAssetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	rel := &Release{TagName: "v0.14.0", Assets: []Asset{{
		Name:        "finam-terminal-linux-amd64",
		DownloadURL: srv.URL,
		Size:        10,
	}}}
	dir := t.TempDir()

	if err := downloadAsset(context.Background(), rel, &rel.Assets[0], filepath.Join(dir, "download.tmp"), nil); err == nil {
		t.Fatal("downloadAsset() error = nil, want an HTTP error")
	}
	assertNoLeftovers(t, dir)
}

// TestParseChecksums covers the sha256sum output format, including the binary
// marker, blank lines and malformed rows.
func TestParseChecksums(t *testing.T) {
	input := "\n" +
		"aaaa  finam-terminal-linux-amd64\n" +
		"bbbb *finam-terminal-windows-amd64.exe\n" +
		"garbage-line-without-a-name\n" +
		"cccc  ./dist/finam-terminal-darwin-arm64\n"

	got := parseChecksums([]byte(input))

	want := map[string]string{
		"finam-terminal-linux-amd64":       "aaaa",
		"finam-terminal-windows-amd64.exe": "bbbb",
		"finam-terminal-darwin-arm64":      "cccc",
	}
	if len(got) != len(want) {
		t.Fatalf("parseChecksums() = %v, want %v", got, want)
	}
	for name, sum := range want {
		if got[name] != sum {
			t.Errorf("parseChecksums()[%q] = %q, want %q", name, got[name], sum)
		}
	}
}

// TestParseChecksumsMatchesReleaseWorkflowFormat pins the parser to the exact
// output of `sha256sum finam-terminal-*` as run by the release workflow.
//
// GNU coreutils separates the digest from the name with two spaces in text
// mode and with " *" in binary mode (which is what it defaults to on Windows);
// the release job may produce either, so both must parse.
func TestParseChecksumsMatchesReleaseWorkflowFormat(t *testing.T) {
	const (
		sumA = "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"
		sumB = "3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"
	)

	for _, sep := range []string{"  ", " *"} {
		input := sumA + sep + "finam-terminal-linux-amd64\n" +
			sumB + sep + "finam-terminal-windows-amd64.exe\n"

		got := parseChecksums([]byte(input))

		if got["finam-terminal-linux-amd64"] != sumA {
			t.Errorf("separator %q: linux sum = %q, want %q", sep, got["finam-terminal-linux-amd64"], sumA)
		}
		if got["finam-terminal-windows-amd64.exe"] != sumB {
			t.Errorf("separator %q: windows sum = %q, want %q", sep, got["finam-terminal-windows-amd64.exe"], sumB)
		}
	}
}

// assertNoLeftovers fails when a failed download left a file behind.
func assertNoLeftovers(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", dir, err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("failed download left %v behind, want the directory empty", names)
	}
}

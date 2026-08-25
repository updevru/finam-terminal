package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// latestReleaseJSON is a trimmed but faithful sample of the GitHub
// /releases/latest payload.
const latestReleaseJSON = `{
  "tag_name": "v0.14.0",
  "html_url": "https://github.com/updevru/finam-terminal/releases/tag/v0.14.0",
  "published_at": "2026-08-25T08:00:00Z",
  "assets": [
    {"name": "finam-terminal-linux-amd64", "browser_download_url": "https://example.test/linux", "size": 12345},
    {"name": "finam-terminal-windows-amd64.exe", "browser_download_url": "https://example.test/win", "size": 23456},
    {"name": "checksums.txt", "browser_download_url": "https://example.test/sums", "size": 321}
  ]
}`

// withAPIBase points the GitHub client at a test server for one test.
func withAPIBase(t *testing.T, base string) {
	t.Helper()
	prev := apiBaseURL
	apiBaseURL = base
	t.Cleanup(func() { apiBaseURL = prev })
}

// TestFetchLatestReleaseParsesPayload verifies every field the updater relies
// on is decoded from the API response.
func TestFetchLatestReleaseParsesPayload(t *testing.T) {
	var gotPath, gotAccept, gotUserAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(latestReleaseJSON))
	}))
	defer srv.Close()
	withAPIBase(t, srv.URL)

	rel, err := FetchLatestRelease(context.Background())
	if err != nil {
		t.Fatalf("FetchLatestRelease() error = %v", err)
	}

	if gotPath != "/repos/updevru/finam-terminal/releases/latest" {
		t.Errorf("request path = %q, want the updevru/finam-terminal latest release", gotPath)
	}
	if gotAccept != "application/vnd.github+json" {
		t.Errorf("Accept header = %q, want application/vnd.github+json", gotAccept)
	}
	if !strings.HasPrefix(gotUserAgent, "finam-terminal/") {
		t.Errorf("User-Agent = %q, want a finam-terminal/<version> prefix", gotUserAgent)
	}

	if rel.TagName != "v0.14.0" {
		t.Errorf("TagName = %q, want v0.14.0", rel.TagName)
	}
	if rel.HTMLURL != "https://github.com/updevru/finam-terminal/releases/tag/v0.14.0" {
		t.Errorf("HTMLURL = %q", rel.HTMLURL)
	}
	want := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	if !rel.PublishedAt.Equal(want) {
		t.Errorf("PublishedAt = %v, want %v", rel.PublishedAt, want)
	}
	if len(rel.Assets) != 3 {
		t.Fatalf("len(Assets) = %d, want 3", len(rel.Assets))
	}
	if rel.Assets[0].Name != "finam-terminal-linux-amd64" ||
		rel.Assets[0].DownloadURL != "https://example.test/linux" ||
		rel.Assets[0].Size != 12345 {
		t.Errorf("Assets[0] = %+v, want the linux/amd64 asset fully decoded", rel.Assets[0])
	}
}

// TestReleaseAssetByName covers asset lookup, including the miss case.
func TestReleaseAssetByName(t *testing.T) {
	rel := &Release{Assets: []Asset{
		{Name: "finam-terminal-linux-amd64", DownloadURL: "https://example.test/linux", Size: 1},
		{Name: "checksums.txt", DownloadURL: "https://example.test/sums", Size: 2},
	}}

	if a := rel.AssetByName("checksums.txt"); a == nil || a.DownloadURL != "https://example.test/sums" {
		t.Errorf("AssetByName(checksums.txt) = %+v, want the checksums asset", a)
	}
	if a := rel.AssetByName("finam-terminal-darwin-arm64"); a != nil {
		t.Errorf("AssetByName(missing) = %+v, want nil", a)
	}
}

// TestFetchLatestReleaseErrors verifies every failure mode surfaces an error
// instead of a half-filled release.
func TestFetchLatestReleaseErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
			},
		},
		{
			name: "rate limited",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, `{"message":"API rate limit exceeded"}`, http.StatusForbidden)
			},
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
		},
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("{not json"))
			},
		},
		{
			name: "empty tag",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"tag_name": "", "assets": []}`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			withAPIBase(t, srv.URL)

			rel, err := FetchLatestRelease(context.Background())
			if err == nil {
				t.Fatalf("FetchLatestRelease() error = nil, want an error (got %+v)", rel)
			}
			if rel != nil {
				t.Errorf("FetchLatestRelease() release = %+v, want nil on error", rel)
			}
		})
	}
}

// TestFetchLatestReleaseHonoursContext verifies an already-cancelled context
// aborts the request rather than hanging on a slow server.
func TestFetchLatestReleaseHonoursContext(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		_, _ = w.Write([]byte(latestReleaseJSON))
	}))
	defer srv.Close()
	defer close(release)
	withAPIBase(t, srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := FetchLatestRelease(ctx); err == nil {
		t.Fatal("FetchLatestRelease() error = nil, want a context error")
	}
}

// TestFetchTimeoutIsBounded documents the 10s ceiling on an update check: it
// must never be long enough to be felt at startup.
func TestFetchTimeoutIsBounded(t *testing.T) {
	if fetchTimeout <= 0 || fetchTimeout > 10*time.Second {
		t.Errorf("fetchTimeout = %v, want a positive value no greater than 10s", fetchTimeout)
	}
}

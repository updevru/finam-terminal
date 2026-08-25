package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"finam-terminal/version"
)

const (
	// repoOwner and repoName identify the public repository whose releases
	// the terminal updates itself from.
	repoOwner = "updevru"
	repoName  = "finam-terminal"

	// acceptHeader is the media type GitHub recommends pinning to.
	acceptHeader = "application/vnd.github+json"

	// fetchTimeout bounds a single update check. The check runs in the
	// background, but the ceiling keeps a hung connection from holding a
	// goroutine (and the shutdown path) indefinitely.
	fetchTimeout = 10 * time.Second

	// maxResponseBytes caps how much of a response body is read, so a
	// misbehaving endpoint cannot exhaust memory.
	maxResponseBytes = 4 << 20 // 4 MiB
)

// apiBaseURL is the GitHub API root. It is a package variable so tests can
// redirect it at an httptest server; production code never changes it.
var apiBaseURL = "https://api.github.com"

// Asset is a single downloadable file attached to a GitHub release.
type Asset struct {
	// Name is the file name, e.g. "finam-terminal-linux-amd64".
	Name string `json:"name"`
	// DownloadURL is the direct download link for the asset.
	DownloadURL string `json:"browser_download_url"`
	// Size is the asset size in bytes, used as an integrity fallback when a
	// release predates checksums.txt.
	Size int64 `json:"size"`
}

// Release is the subset of a GitHub release the updater needs.
type Release struct {
	// TagName is the release tag, e.g. "v0.14.0".
	TagName string `json:"tag_name"`
	// HTMLURL is the human-readable release page.
	HTMLURL string `json:"html_url"`
	// PublishedAt is when the release was published.
	PublishedAt time.Time `json:"published_at"`
	// Assets are the files attached to the release.
	Assets []Asset `json:"assets"`
}

// AssetByName returns the asset with the given file name, or nil when the
// release does not carry it.
func (r *Release) AssetByName(name string) *Asset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
}

// FetchLatestRelease queries the GitHub API for the newest published release
// of the terminal.
//
// The /releases/latest endpoint never returns drafts or pre-releases, so no
// filtering is needed on our side. The request carries no credentials — the
// unauthenticated limit of 60 requests per hour per IP leaves a 60x margin for
// one check per day.
//
// Every failure (network, 403 rate limit, 5xx, malformed body, empty tag)
// returns a nil release and an error; the caller is expected to log it as a
// warning and try again on the normal schedule.
func FetchLatestRelease(ctx context.Context) (*Release, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBaseURL, repoOwner, repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build release request: %w", err)
	}
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("User-Agent", userAgent())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned %s for the latest release", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read release response: %w", err)
	}

	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, fmt.Errorf("decode release response: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release response has no tag_name")
	}
	return &rel, nil
}

// userAgent builds the User-Agent GitHub asks API clients to identify
// themselves with.
func userAgent() string {
	return "finam-terminal/" + version.String()
}

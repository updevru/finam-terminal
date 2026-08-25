package updater

import (
	"strings"
	"testing"
)

// TestAssetNameSupportedPlatforms pins the asset names to the artifacts the
// release workflow actually publishes.
func TestAssetNameSupportedPlatforms(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{goos: "linux", goarch: "amd64", want: "finam-terminal-linux-amd64"},
		{goos: "darwin", goarch: "amd64", want: "finam-terminal-darwin-amd64"},
		{goos: "darwin", goarch: "arm64", want: "finam-terminal-darwin-arm64"},
		{goos: "windows", goarch: "amd64", want: "finam-terminal-windows-amd64.exe"},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got, err := AssetName(tt.goos, tt.goarch)
			if err != nil {
				t.Fatalf("AssetName(%q, %q) error = %v, want nil", tt.goos, tt.goarch, err)
			}
			if got != tt.want {
				t.Errorf("AssetName(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

// TestAssetNameUnsupportedPlatforms verifies platforms without a published
// build fail with a message naming the platform, so the user knows why.
func TestAssetNameUnsupportedPlatforms(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
	}{
		{goos: "linux", goarch: "arm64"},
		{goos: "windows", goarch: "arm64"},
		{goos: "linux", goarch: "386"},
		{goos: "freebsd", goarch: "amd64"},
		{goos: "", goarch: ""},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got, err := AssetName(tt.goos, tt.goarch)
			if err == nil {
				t.Fatalf("AssetName(%q, %q) = %q, want an error", tt.goos, tt.goarch, got)
			}
			if got != "" {
				t.Errorf("AssetName(%q, %q) = %q, want an empty name on error", tt.goos, tt.goarch, got)
			}
			if !strings.Contains(err.Error(), tt.goos+"/"+tt.goarch) {
				t.Errorf("error %q does not name the platform %s/%s", err, tt.goos, tt.goarch)
			}
		})
	}
}

// TestCurrentAssetName verifies the convenience wrapper resolves the running
// platform (or explains why it cannot).
func TestCurrentAssetName(t *testing.T) {
	name, err := CurrentAssetName()
	if err != nil {
		t.Skipf("no published build for this platform: %v", err)
	}
	if !strings.HasPrefix(name, "finam-terminal-") {
		t.Errorf("CurrentAssetName() = %q, want a finam-terminal- prefix", name)
	}
}

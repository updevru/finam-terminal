package updater

import (
	"fmt"
	"runtime"
)

// binaryPrefix is the common prefix of every released binary asset. It must
// stay in sync with .github/workflows/release.yml, which builds
// dist/finam-terminal-${os}-${arch}${ext}.
const binaryPrefix = "finam-terminal-"

// checksumsAssetName is the release asset listing the SHA256 sum of every
// binary, in the `sha256␠␠filename` format produced by sha256sum(1).
const checksumsAssetName = "checksums.txt"

// supportedPlatforms mirrors the build matrix of the release workflow. A
// platform absent from this map has no published binary, so the terminal
// cannot update itself there even though the update check still works.
var supportedPlatforms = map[string]string{
	"linux/amd64":   binaryPrefix + "linux-amd64",
	"darwin/amd64":  binaryPrefix + "darwin-amd64",
	"darwin/arm64":  binaryPrefix + "darwin-arm64",
	"windows/amd64": binaryPrefix + "windows-amd64.exe",
}

// AssetName returns the name of the release asset holding the binary for the
// given platform, e.g. "finam-terminal-windows-amd64.exe".
//
// Platforms outside the release matrix (linux/arm64 and windows/arm64 among
// them) return an error naming the platform — the caller shows it to the user,
// who can still build from source.
func AssetName(goos, goarch string) (string, error) {
	platform := goos + "/" + goarch
	name, ok := supportedPlatforms[platform]
	if !ok {
		return "", fmt.Errorf("нет готовой сборки для платформы %s — соберите из исходников", platform)
	}
	return name, nil
}

// CurrentAssetName returns the release asset name for the running platform.
func CurrentAssetName() (string, error) {
	return AssetName(runtime.GOOS, runtime.GOARCH)
}

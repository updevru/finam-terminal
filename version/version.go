// Package version exposes build-time metadata about the finam-terminal binary.
//
// The Version, Commit, and BuildDate variables are designed to be overridden
// at link time via:
//
//	go build -ldflags "-X finam-terminal/version.Version=v1.2.3 \
//	                   -X finam-terminal/version.Commit=$(git rev-parse HEAD) \
//	                   -X finam-terminal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
//
// They are package-level vars (not consts) so the linker can replace them.
//
// When no ldflags injection occurred (e.g. plain `go build main.go`), String()
// falls back to runtime/debug.ReadBuildInfo() to extract the VCS revision and
// dirty flag stamped by the Go toolchain. The result for a development build is
// rendered as "dev (a1b2c3d)" or "dev (a1b2c3d, dirty)" — and just "dev" when
// no VCS info is available (e.g. when building from a tarball outside a repo).
package version

import (
	"fmt"
	"runtime/debug"
)

// Version is the human-readable release identifier (e.g. "v1.2.3"). Defaults
// to "dev" when not injected via ldflags.
var Version = "dev"

// Commit is the git commit SHA the binary was built from. Defaults to
// "unknown" when not injected via ldflags.
var Commit = "unknown"

// BuildDate is the UTC timestamp at which the binary was built (RFC3339).
// Defaults to "unknown" when not injected via ldflags.
var BuildDate = "unknown"

// readBuildInfo is a package-level hook around debug.ReadBuildInfo so tests
// can swap in deterministic build metadata. Production code uses the real
// implementation.
var readBuildInfo = debug.ReadBuildInfo

// String returns a display string suitable for the UI header.
//
// Behaviour:
//   - If Version was injected to anything other than "dev", it is returned
//     verbatim. No "v" prefix is added — the caller is expected to inject the
//     full tag (e.g. "v1.2.3").
//   - Otherwise (a development build), an attempt is made to enrich the
//     output with the VCS revision: either the injected Commit or, failing
//     that, vcs.revision from runtime/debug.ReadBuildInfo. The result is
//     "dev (sha)" or "dev (sha, dirty)".
//   - If no commit can be resolved at all, "dev" is returned.
func String() string {
	return formatVersion(resolveVersion())
}

// Info returns the raw values of the Version, Commit, and BuildDate variables
// without any formatting or fallback resolution. Useful for diagnostics or a
// future --version CLI flag.
func Info() (version, commit, buildDate string) {
	return Version, Commit, BuildDate
}

// resolveVersion produces the effective (version, commit, dirty) tuple used
// by String. It applies the runtime/debug fallback only when Version=="dev"
// and the commit is missing.
func resolveVersion() (version, commit string, dirty bool) {
	version = Version
	commit = Commit

	// A real release tag short-circuits — never look at VCS info.
	if version != "dev" {
		return version, commit, false
	}

	// Dev build with an explicitly injected commit (e.g. `make build` from a
	// clean tree) — trust the injection and skip the VCS fallback.
	if commit != "" && commit != "unknown" {
		return version, commit, false
	}

	// No commit injected: try to recover one from the build info stamped by
	// the Go toolchain. This works for `go build` inside a git checkout but
	// not for tarball builds.
	info, ok := readBuildInfo()
	if !ok || info == nil {
		return version, "", false
	}

	var revision string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if revision == "" {
		return version, "", false
	}
	return version, shortenSHA(revision), dirty
}

// formatVersion is the pure rendering helper. It performs no I/O and never
// touches package-level state, which makes it trivially testable.
func formatVersion(version, commit string, dirty bool) string {
	if version != "dev" {
		return version
	}
	if commit == "" || commit == "unknown" {
		return "dev"
	}
	if dirty {
		return fmt.Sprintf("dev (%s, dirty)", commit)
	}
	return fmt.Sprintf("dev (%s)", commit)
}

// shortenSHA truncates a git revision to the canonical 7-character short form
// used in display strings. Inputs shorter than 7 characters are returned as-is.
func shortenSHA(sha string) string {
	const shortLen = 7
	if len(sha) <= shortLen {
		return sha
	}
	return sha[:shortLen]
}

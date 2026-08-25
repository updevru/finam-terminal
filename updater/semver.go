// Package updater keeps the terminal up to date: it checks GitHub Releases in
// the background, tells the UI when a newer version is published, and can
// replace the running binary with the released asset.
//
// The package depends on the standard library only. Every layer is
// independent — version comparison (this file), the on-disk state cache, the
// GitHub client, the scheduler, and the self-update applier — so each can be
// tested in isolation without network access.
package updater

import (
	"strconv"
	"strings"
)

// semver is a parsed semantic version. Build metadata is deliberately dropped
// during parsing: per the semver spec it never affects precedence.
type semver struct {
	major      int
	minor      int
	patch      int
	preRelease string // without the leading "-"; empty for a final release
}

// IsRelease reports whether v is a release version string, i.e. a parsable
// semver tag such as "v0.14.0" or "0.14.0".
//
// Development builds ("dev", "dev (a1b2c3d)", "dev (a1b2c3d, dirty)") and any
// other unparsable input return false. The whole update mechanism is gated on
// this: a binary that cannot name its own release must never offer to replace
// itself.
func IsRelease(v string) bool {
	_, ok := parseSemver(v)
	return ok
}

// Compare orders two version strings and returns -1 when a < b, 0 when they
// are equal, and 1 when a > b.
//
// The leading "v" is optional on either side. MAJOR, MINOR and PATCH are
// compared numerically, in that order. A pre-release version sorts below the
// release with the same numeric triple ("v1.0.0-rc1" < "v1.0.0"); two
// pre-releases are compared lexicographically. Build metadata is ignored.
//
// Unparsable input is treated as the zero version, so Compare never panics;
// callers that care about validity should use IsRelease first (IsNewer does).
func Compare(a, b string) int {
	va, _ := parseSemver(a)
	vb, _ := parseSemver(b)
	return compareSemver(va, vb)
}

// IsNewer reports whether latest is a strictly newer release than current.
//
// It returns false unless both sides are release versions — a dev build is
// never told to update, and a locally built binary that is ahead of the last
// published release is never asked to downgrade.
func IsNewer(current, latest string) bool {
	vCur, okCur := parseSemver(current)
	if !okCur {
		return false
	}
	vLatest, okLatest := parseSemver(latest)
	if !okLatest {
		return false
	}
	return compareSemver(vCur, vLatest) < 0
}

// parseSemver parses "[v]MAJOR.MINOR.PATCH[-prerelease][+build]" and reports
// whether the input was a well-formed version.
func parseSemver(v string) (semver, bool) {
	s := strings.TrimSpace(v)
	if s == "" {
		return semver{}, false
	}
	s = strings.TrimPrefix(s, "v")

	// Build metadata does not affect precedence — drop it before parsing.
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}

	var pre string
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return semver{}, false
	}

	nums := make([]int, 3)
	for i, p := range parts {
		if p == "" {
			return semver{}, false
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return semver{}, false
		}
		nums[i] = n
	}

	return semver{major: nums[0], minor: nums[1], patch: nums[2], preRelease: pre}, true
}

// compareSemver implements the precedence rules for two parsed versions.
func compareSemver(a, b semver) int {
	if c := compareInt(a.major, b.major); c != 0 {
		return c
	}
	if c := compareInt(a.minor, b.minor); c != 0 {
		return c
	}
	if c := compareInt(a.patch, b.patch); c != 0 {
		return c
	}
	return comparePreRelease(a.preRelease, b.preRelease)
}

// comparePreRelease orders the pre-release segment: an empty segment (a final
// release) outranks any pre-release, otherwise the comparison is textual.
func comparePreRelease(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	case a < b:
		return -1
	default:
		return 1
	}
}

// compareInt is the three-way comparison for ints.
func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

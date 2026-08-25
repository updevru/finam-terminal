package updater

import "testing"

// TestIsRelease verifies that only parsable semver tags are treated as
// release builds. Development builds must never trigger the update machinery.
func TestIsRelease(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "tag with v prefix", input: "v0.14.0", want: true},
		{name: "tag without v prefix", input: "0.14.0", want: true},
		{name: "pre-release tag", input: "v1.0.0-rc1", want: true},
		{name: "tag with build metadata", input: "v1.0.0+20260825", want: true},
		{name: "dev build", input: "dev", want: false},
		{name: "dev build with commit", input: "dev (a1b2c3d)", want: false},
		{name: "dev build dirty", input: "dev (a1b2c3d, dirty)", want: false},
		{name: "empty string", input: "", want: false},
		{name: "garbage", input: "not-a-version", want: false},
		{name: "two components only", input: "v1.2", want: false},
		{name: "four components", input: "v1.2.3.4", want: false},
		{name: "non numeric component", input: "v1.x.0", want: false},
		{name: "whitespace padded tag", input: "  v1.2.3  ", want: true},
		{name: "v prefix only", input: "v", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRelease(tt.input); got != tt.want {
				t.Errorf("IsRelease(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestCompare exercises the ordering rules: MAJOR/MINOR/PATCH precedence, an
// optional "v" prefix, and pre-release versions sorting below their release.
func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal with prefix", a: "v1.2.3", b: "v1.2.3", want: 0},
		{name: "equal mixed prefix", a: "1.2.3", b: "v1.2.3", want: 0},
		{name: "major lower", a: "v1.9.9", b: "v2.0.0", want: -1},
		{name: "major higher", a: "v2.0.0", b: "v1.9.9", want: 1},
		{name: "minor lower", a: "v1.2.9", b: "v1.3.0", want: -1},
		{name: "minor higher", a: "v1.3.0", b: "v1.2.9", want: 1},
		{name: "patch lower", a: "v1.2.3", b: "v1.2.4", want: -1},
		{name: "patch higher", a: "v1.2.4", b: "v1.2.3", want: 1},
		{name: "pre-release below release", a: "v1.0.0-rc1", b: "v1.0.0", want: -1},
		{name: "release above pre-release", a: "v1.0.0", b: "v1.0.0-rc1", want: 1},
		{name: "pre-release alphabetical", a: "v1.0.0-alpha", b: "v1.0.0-beta", want: -1},
		{name: "identical pre-release", a: "v1.0.0-rc1", b: "v1.0.0-rc1", want: 0},
		{name: "build metadata ignored", a: "v1.0.0+aaa", b: "v1.0.0+bbb", want: 0},
		{name: "double digit minor", a: "v0.9.0", b: "v0.14.0", want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Compare(tt.a, tt.b); got != tt.want {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestIsNewer covers the guard used before offering an update: both sides must
// be release versions and latest must be strictly greater than current.
func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{name: "newer patch", current: "v0.13.0", latest: "v0.13.1", want: true},
		{name: "newer minor", current: "v0.13.0", latest: "v0.14.0", want: true},
		{name: "equal versions", current: "v0.14.0", latest: "v0.14.0", want: false},
		{name: "downgrade", current: "v0.15.0", latest: "v0.14.0", want: false},
		{name: "current is dev", current: "dev", latest: "v0.14.0", want: false},
		{name: "current is dev with commit", current: "dev (a1b2c3d)", latest: "v0.14.0", want: false},
		{name: "latest is garbage", current: "v0.13.0", latest: "latest", want: false},
		{name: "latest empty", current: "v0.13.0", latest: "", want: false},
		{name: "both empty", current: "", latest: "", want: false},
		{name: "mixed prefixes", current: "0.13.0", latest: "v0.13.1", want: true},
		{name: "release over pre-release", current: "v1.0.0-rc1", latest: "v1.0.0", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNewer(tt.current, tt.latest); got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

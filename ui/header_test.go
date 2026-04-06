package ui

import (
	"strings"
	"testing"

	"finam-terminal/version"
)

// TestCreateHeader_RendersInjectedReleaseTag verifies that when version.Version
// is overridden (simulating an ldflags release build), the header text
// contains the injected tag verbatim and does not double the leading "v".
func TestCreateHeader_RendersInjectedReleaseTag(t *testing.T) {
	prev := version.Version
	version.Version = "v1.2.3"
	defer func() { version.Version = prev }()

	header := createHeader()
	text := header.GetText(true)

	if !strings.Contains(text, "Finam Terminal") {
		t.Errorf("header text = %q, want to contain %q", text, "Finam Terminal")
	}
	if !strings.Contains(text, "v1.2.3") {
		t.Errorf("header text = %q, want to contain %q", text, "v1.2.3")
	}
	if strings.Contains(text, "vv1.2.3") {
		t.Errorf("header text = %q, must not contain double 'v' prefix", text)
	}
	if strings.Contains(text, "1.0.0") {
		t.Errorf("header text = %q, must not contain hardcoded legacy version", text)
	}
}

// TestCreateHeader_RendersDevBuild verifies the dev path: when Version is the
// default "dev" sentinel and no commit was injected, the header still renders
// without panic and the text starts with the "dev" prefix from version.String().
func TestCreateHeader_RendersDevBuild(t *testing.T) {
	prevV, prevC := version.Version, version.Commit
	version.Version = "dev"
	version.Commit = "unknown"
	defer func() {
		version.Version = prevV
		version.Commit = prevC
	}()

	header := createHeader()
	text := header.GetText(true)

	if !strings.Contains(text, "Finam Terminal") {
		t.Errorf("header text = %q, want to contain %q", text, "Finam Terminal")
	}
	if !strings.Contains(text, "dev") {
		t.Errorf("header text = %q, want to contain %q", text, "dev")
	}
	if strings.Contains(text, "vdev") {
		t.Errorf("header text = %q, must not prefix dev with 'v'", text)
	}
	if strings.Contains(text, "1.0.0") {
		t.Errorf("header text = %q, must not contain hardcoded legacy version", text)
	}
}

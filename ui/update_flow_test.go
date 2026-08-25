package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"finam-terminal/updater"
)

// TestRenderProgressBar covers the pure bar renderer at both ends of the
// range, including an unknown total and an over-long download.
func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		done     int64
		total    int64
		contains string
	}{
		{name: "start", done: 0, total: 1000, contains: "0%"},
		{name: "half", done: 500, total: 1000, contains: "50%"},
		{name: "complete", done: 1000, total: 1000, contains: "100%"},
		{name: "over total is clamped", done: 1200, total: 1000, contains: "100%"},
		{name: "unknown total falls back to megabytes", done: 512 * 1024, total: 0, contains: "0.5 МБ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderProgressBar(tt.done, tt.total)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("renderProgressBar(%d, %d) = %q, want it to contain %q",
					tt.done, tt.total, got, tt.contains)
			}
			if strings.Contains(got, "\n") {
				t.Errorf("renderProgressBar(%d, %d) = %q, want a single line", tt.done, tt.total, got)
			}
		})
	}
}

// TestRenderProgressBarIsMonotonic verifies the filled portion never shrinks
// as the download advances.
func TestRenderProgressBarIsMonotonic(t *testing.T) {
	prev := -1
	for done := int64(0); done <= 1000; done += 100 {
		filled := strings.Count(renderProgressBar(done, 1000), "█")
		if filled < prev {
			t.Fatalf("filled blocks went from %d to %d at done=%d", prev, filled, done)
		}
		prev = filled
	}
	if prev == 0 {
		t.Error("the bar never filled up, want a full bar at 100%")
	}
}

// TestRunUpdateFlowSuccess verifies a successful update reports the installed
// version and returns the path to restart.
func TestRunUpdateFlowSuccess(t *testing.T) {
	var out bytes.Buffer
	restoreOutput := swapUpdateOutput(t, &out)
	defer restoreOutput()

	prevUpdate := selfUpdateFunc
	selfUpdateFunc = func(ctx context.Context, rel *updater.Release, progress func(done, total int64)) error {
		progress(50, 100)
		progress(100, 100)
		return nil
	}
	prevPath := executablePathFunc
	executablePathFunc = func() (string, error) { return "/opt/finam/finam-terminal", nil }
	t.Cleanup(func() {
		selfUpdateFunc = prevUpdate
		executablePathFunc = prevPath
	})

	exePath, err := RunUpdateFlow(&updater.Release{TagName: "v0.14.0"})
	if err != nil {
		t.Fatalf("RunUpdateFlow() error = %v", err)
	}
	if exePath != "/opt/finam/finam-terminal" {
		t.Errorf("RunUpdateFlow() path = %q, want the executable path", exePath)
	}

	printed := out.String()
	if !strings.Contains(printed, "v0.14.0") {
		t.Errorf("output %q does not mention the new version", printed)
	}
	if !strings.Contains(printed, "100%") {
		t.Errorf("output %q never reached 100%%", printed)
	}
	if !strings.HasSuffix(printed, "\n") {
		t.Errorf("output %q does not end with a newline, the console would be left mid-line", printed)
	}
}

// TestRunUpdateFlowError verifies a failed update prints a readable message
// and ends the line cleanly.
func TestRunUpdateFlowError(t *testing.T) {
	var out bytes.Buffer
	restoreOutput := swapUpdateOutput(t, &out)
	defer restoreOutput()

	prevUpdate := selfUpdateFunc
	selfUpdateFunc = func(ctx context.Context, rel *updater.Release, progress func(done, total int64)) error {
		progress(10, 100)
		return errors.New("контрольная сумма SHA256 не совпала")
	}
	t.Cleanup(func() { selfUpdateFunc = prevUpdate })

	if _, err := RunUpdateFlow(&updater.Release{TagName: "v0.14.0"}); err == nil {
		t.Fatal("RunUpdateFlow() error = nil, want the failure to propagate")
	}

	printed := out.String()
	if !strings.Contains(printed, "контрольная сумма") {
		t.Errorf("output %q does not explain the failure", printed)
	}
	if !strings.HasSuffix(printed, "\n") {
		t.Errorf("output %q does not end with a newline", printed)
	}
}

// TestRunUpdateFlowNotWritableShowsManualCommand verifies a read-only install
// directory tells the user exactly how to update by hand.
func TestRunUpdateFlowNotWritableShowsManualCommand(t *testing.T) {
	var out bytes.Buffer
	restoreOutput := swapUpdateOutput(t, &out)
	defer restoreOutput()

	prevUpdate := selfUpdateFunc
	selfUpdateFunc = func(ctx context.Context, rel *updater.Release, progress func(done, total int64)) error {
		return fmt.Errorf("%w: /usr/local/bin", updater.ErrNotWritable)
	}
	t.Cleanup(func() { selfUpdateFunc = prevUpdate })

	if _, err := RunUpdateFlow(&updater.Release{TagName: "v0.14.0"}); err == nil {
		t.Fatal("RunUpdateFlow() error = nil, want the permission failure to propagate")
	}

	printed := out.String()
	if !strings.Contains(printed, updater.ManualUpdateCommand()) {
		t.Errorf("output %q does not contain the manual update command %q",
			printed, updater.ManualUpdateCommand())
	}
}

// swapUpdateOutput redirects the console writer used by the update flow.
func swapUpdateOutput(t *testing.T, buf *bytes.Buffer) func() {
	t.Helper()
	prev := updateOutput
	updateOutput = buf
	return func() { updateOutput = prev }
}

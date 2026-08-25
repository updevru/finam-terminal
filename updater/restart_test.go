package updater

import (
	"errors"
	"os"
	"testing"
)

// TestRestartPassesCurrentProcessState verifies the replacement process is
// started with the same arguments and environment as the running one.
func TestRestartPassesCurrentProcessState(t *testing.T) {
	var gotPath string
	var gotArgs, gotEnv []string

	prev := execRestart
	execRestart = func(exePath string, args, env []string) error {
		gotPath, gotArgs, gotEnv = exePath, args, env
		return nil
	}
	t.Cleanup(func() { execRestart = prev })

	if err := Restart("/opt/finam/finam-terminal"); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}

	if gotPath != "/opt/finam/finam-terminal" {
		t.Errorf("exec path = %q, want /opt/finam/finam-terminal", gotPath)
	}
	if len(gotArgs) != len(os.Args) {
		t.Fatalf("args = %v, want the current os.Args %v", gotArgs, os.Args)
	}
	for i := range os.Args {
		if gotArgs[i] != os.Args[i] {
			t.Errorf("args[%d] = %q, want %q", i, gotArgs[i], os.Args[i])
		}
	}
	if len(gotEnv) != len(os.Environ()) {
		t.Errorf("env has %d entries, want the current environment (%d)", len(gotEnv), len(os.Environ()))
	}
}

// TestRestartRejectsEmptyPath verifies a missing executable path is reported
// instead of handed to exec.
func TestRestartRejectsEmptyPath(t *testing.T) {
	called := false
	prev := execRestart
	execRestart = func(string, []string, []string) error {
		called = true
		return nil
	}
	t.Cleanup(func() { execRestart = prev })

	if err := Restart(""); err == nil {
		t.Error("Restart(\"\") error = nil, want an error")
	}
	if called {
		t.Error("Restart(\"\") reached exec, want it rejected up front")
	}
}

// TestRestartPropagatesError verifies a failed exec surfaces to the caller so
// the user is told the update landed but the restart did not.
func TestRestartPropagatesError(t *testing.T) {
	sentinel := errors.New("exec failed")
	prev := execRestart
	execRestart = func(string, []string, []string) error { return sentinel }
	t.Cleanup(func() { execRestart = prev })

	if err := Restart("/opt/finam/finam-terminal"); !errors.Is(err, sentinel) {
		t.Errorf("Restart() error = %v, want it to wrap %v", err, sentinel)
	}
}

// TestExecutablePathForRestart verifies the applier and the restart agree on
// which file to launch.
func TestExecutablePathForRestart(t *testing.T) {
	exe := fakeExecutable(t, "binary")

	got, err := ExecutablePath()
	if err != nil {
		t.Fatalf("ExecutablePath() error = %v", err)
	}
	if got != exe {
		t.Errorf("ExecutablePath() = %q, want %q", got, exe)
	}
}

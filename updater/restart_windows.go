//go:build windows

package updater

import (
	"os"
	"os/exec"
)

// exitProcess is os.Exit behind a variable so tests never terminate the test
// binary.
var exitProcess = os.Exit

// platformRestart starts a fresh process and exits the current one. Windows
// has no execve: the new binary is launched as a child that inherits the
// console and the standard streams, and the parent quits immediately so the
// two never compete for input.
func platformRestart(exePath string, args, env []string) error {
	cmd := exec.Command(exePath, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	if err := cmd.Start(); err != nil {
		return err
	}

	exitProcess(0)
	return nil
}

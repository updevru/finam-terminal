//go:build !windows

package updater

import "syscall"

// platformRestart replaces the current process image with a fresh one via
// execve. The PID, the controlling terminal and the standard streams are all
// preserved, so the user sees the new version come up in the same window.
//
// On success this function does not return.
func platformRestart(exePath string, args, env []string) error {
	return syscall.Exec(exePath, args, env)
}

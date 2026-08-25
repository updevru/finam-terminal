package updater

import (
	"errors"
	"fmt"
	"os"
)

// execRestart is the platform-specific restart implementation, provided by
// restart_unix.go or restart_windows.go. It is a package variable so tests can
// assert what would be executed without actually replacing the process.
var execRestart = platformRestart

// ExecutablePath returns the real path of the running binary — the same path
// SelfUpdate replaces, and therefore the one to relaunch afterwards.
func ExecutablePath() (string, error) {
	return resolveExecutable()
}

// Restart relaunches the terminal from exePath with the current arguments and
// environment.
//
// On Unix the process image is replaced in place (same PID, same terminal), so
// a successful call never returns. On Windows a child process is started with
// the inherited standard streams and the current one exits.
func Restart(exePath string) error {
	if exePath == "" {
		return errors.New("перезапуск невозможен: путь к программе неизвестен")
	}
	if err := execRestart(exePath, os.Args, os.Environ()); err != nil {
		return fmt.Errorf("перезапустить программу: %w", err)
	}
	return nil
}

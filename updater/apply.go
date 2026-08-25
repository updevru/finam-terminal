package updater

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// backupSuffix is appended to the running binary while it is being replaced.
// On Windows the backup survives the update — a running .exe cannot be
// deleted — and is cleaned up by CleanupStaleBackup on the next launch.
const backupSuffix = ".old"

// ErrNotWritable reports that the directory holding the binary cannot be
// written to, so the terminal cannot replace itself. Callers should show
// ManualUpdateCommand() instead of a raw error.
var ErrNotWritable = errors.New("нет прав на запись в каталог установки")

// executablePath resolves the path of the running binary. It is a package
// variable so tests can point the applier at a scratch file instead of the
// real test binary.
var executablePath = os.Executable

// SelfUpdate downloads the release asset for the running platform, verifies
// it and replaces the running binary with it.
//
// The download lands in the same directory as the executable so the final
// rename stays on one filesystem and is therefore atomic; the temporary name
// carries the PID so two terminals updating at once cannot collide. The
// running binary is only touched after the download has been fully verified,
// and a failed swap is rolled back — on every error path the existing binary
// is left byte for byte intact.
//
// progress, when not nil, receives the running byte count during the download.
//
// The caller is expected to restart the process afterwards (see Restart).
func SelfUpdate(ctx context.Context, rel *Release, progress func(done, total int64)) error {
	exePath, err := resolveExecutable()
	if err != nil {
		return err
	}

	assetName, err := CurrentAssetName()
	if err != nil {
		return err
	}
	asset := rel.AssetByName(assetName)
	if asset == nil {
		return fmt.Errorf("в релизе %s нет файла %s", rel.TagName, assetName)
	}

	dir := filepath.Dir(exePath)
	if err := ensureWritable(dir); err != nil {
		return err
	}

	tmpPath := filepath.Join(dir, fmt.Sprintf(".finam-terminal-update-%d.tmp", os.Getpid()))
	// Guarantees the temporary file never outlives a failed update; after a
	// successful rename this is a no-op.
	defer func() { _ = os.Remove(tmpPath) }()

	if err := downloadAsset(ctx, rel, asset, tmpPath, progress); err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			return fmt.Errorf("сделать файл исполняемым: %w", err)
		}
	}

	return replaceExecutable(exePath, tmpPath)
}

// CleanupStaleBackup removes the .old backup left by a previous update. It is
// meant to be called once at startup and is deliberately silent: a leftover
// backup is cosmetic, and failing to remove it must never disturb the launch.
func CleanupStaleBackup() {
	exePath, err := resolveExecutable()
	if err != nil {
		return
	}
	_ = os.Remove(exePath + backupSuffix)
}

// ManualUpdateCommand returns the install-script command for the running
// platform, shown when the terminal cannot update itself.
func ManualUpdateCommand() string {
	if runtime.GOOS == "windows" {
		return "irm https://fcli.ru/install.ps1 | iex"
	}
	return "curl -fsSL https://fcli.ru/install.sh | bash"
}

// replaceExecutable swaps newPath in for exePath using two renames.
//
// The running binary is first moved aside to exePath+".old" (which is legal
// even on Windows, where deleting it would not be), then the new file is moved
// into place. If the second rename fails the first is undone, leaving the
// original binary exactly where it was. On Unix the backup is deleted right
// away; on Windows it stays until the next launch.
func replaceExecutable(exePath, newPath string) error {
	backupPath := exePath + backupSuffix
	// A leftover backup from an earlier update would make the first rename
	// fail on Windows.
	_ = os.Remove(backupPath)

	if err := os.Rename(exePath, backupPath); err != nil {
		return fmt.Errorf("переименовать текущий файл: %w", err)
	}

	if err := os.Rename(newPath, exePath); err != nil {
		// Put the original back before reporting the failure.
		if rollbackErr := os.Rename(backupPath, exePath); rollbackErr != nil {
			return fmt.Errorf("установить новую версию не удалось (%w), и откат тоже не удался: %v — "+
				"исходный файл лежит в %s", err, rollbackErr, backupPath)
		}
		return fmt.Errorf("установить новую версию: %w", err)
	}

	if runtime.GOOS != "windows" {
		_ = os.Remove(backupPath)
	}
	return nil
}

// resolveExecutable returns the real path of the running binary, following
// symlinks so an update replaces the binary itself rather than a link to it.
func resolveExecutable() (string, error) {
	exePath, err := executablePath()
	if err != nil {
		return "", fmt.Errorf("определить путь к программе: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		// A path we cannot resolve is still worth trying — EvalSymlinks fails
		// on some exotic filesystems where a plain rename works fine.
		log.Printf("[WARN] Executable path not resolved, using %s as is: %v", exePath, err)
		return exePath, nil
	}
	return resolved, nil
}

// ensureWritable probes the install directory by creating and removing a
// temporary file. Checking up front turns "permission denied" into a clear
// message before anything is downloaded.
func ensureWritable(dir string) error {
	probe, err := os.CreateTemp(dir, ".finam-terminal-write-probe-*")
	if err != nil {
		return fmt.Errorf("%w: %s", ErrNotWritable, dir)
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return nil
}

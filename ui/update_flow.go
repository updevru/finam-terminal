package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"finam-terminal/updater"
)

// progressBarWidth matches the startup progress bar in RunStartupSteps, so
// the update shares the visual language of the launch sequence.
const progressBarWidth = 20

// updateOutput is where the update flow draws. It is a variable so tests can
// capture the output instead of writing to the terminal.
var updateOutput io.Writer = os.Stdout

// selfUpdateFunc and executablePathFunc indirect the updater package so the
// flow can be tested without downloading or replacing anything.
var (
	selfUpdateFunc     = updater.SelfUpdate
	executablePathFunc = updater.ExecutablePath
)

// RunUpdateFlow downloads and installs the given release, drawing a console
// progress bar in the style of the startup sequence.
//
// It returns the path of the updated executable, which the caller passes to
// updater.Restart. On failure the error is both printed in readable Russian —
// including the manual install command when the install directory is not
// writable — and returned, so the caller can decide to carry on starting the
// terminal as usual.
func RunUpdateFlow(rel *updater.Release) (string, error) {
	_, _ = fmt.Fprintf(updateOutput, "\n\x1b[36mОбновление до %s\x1b[0m\n", rel.TagName)

	err := selfUpdateFunc(context.Background(), rel, func(done, total int64) {
		_, _ = fmt.Fprintf(updateOutput, "\r%s ", renderProgressBar(done, total))
	})

	// Always close the transient progress line before printing the outcome.
	_, _ = fmt.Fprint(updateOutput, "\r\033[K")

	if err != nil {
		printUpdateError(err)
		return "", err
	}

	exePath, pathErr := executablePathFunc()
	if pathErr != nil {
		printUpdateError(pathErr)
		return "", pathErr
	}

	_, _ = fmt.Fprintf(updateOutput, "\x1b[32m[OK]\x1b[0m Установлена версия %s, перезапуск...\n", rel.TagName)
	return exePath, nil
}

// printUpdateError explains a failed update without dumping a raw Go error at
// the user, and points at the install script when self-update is impossible.
func printUpdateError(err error) {
	_, _ = fmt.Fprintf(updateOutput, "\x1b[31m[ОШИБКА]\x1b[0m Обновление не выполнено: %v\n", err)

	if errors.Is(err, updater.ErrNotWritable) {
		_, _ = fmt.Fprintf(updateOutput, "         Обновите вручную: %s\n", updater.ManualUpdateCommand())
	}
	_, _ = fmt.Fprintf(updateOutput, "         Программа продолжит работу в текущей версии.\n")
}

// renderProgressBar draws a single-line download progress bar. With a known
// total it shows a percentage; when the size is unknown it falls back to the
// number of megabytes received.
func renderProgressBar(done, total int64) string {
	if total <= 0 {
		return fmt.Sprintf("\x1b[36m%s\x1b[0m %.1f МБ",
			strings.Repeat("░", progressBarWidth), float64(done)/(1024*1024))
	}

	ratio := float64(done) / float64(total)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(progressBarWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
	return fmt.Sprintf("\x1b[36m%s\x1b[0m %3.0f%%", bar, ratio*100)
}

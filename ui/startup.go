package ui

import (
	"fmt"
	"strings"
	"time"
)

// StartupStep represents a single initialization step.
type StartupStep struct {
	Name   string
	Action func() error
}

// RunStartupSteps executes steps with a console progress bar.
func RunStartupSteps(steps []StartupStep) error {
	total := len(steps)
	barWidth := 20

	fmt.Println() // Spacer

	for i, step := range steps {
		// Display Progress Bar for current state
		percent := float64(i) / float64(total)
		filled := int(percent * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		// Print Status Line (Transient)
		fmt.Printf("\r\x1b[36m%s\x1b[0m %s ", bar, step.Name) // Cyan bar

		start := time.Now()
		err := step.Action()

		// Artificial delay for UX if action is too fast
		if time.Since(start) < 300*time.Millisecond {
			time.Sleep(300 * time.Millisecond)
		}

		if err != nil {
			fmt.Printf("\r\033[K") // Clear line
			fmt.Printf("\x1b[31m[FAILED]\x1b[0m %s: %v\n", step.Name, err)
			return err
		}

		// On success, leave a permanent log line?
		// "After outputting inscription show log of program startup..."
		// So yes, we want a history.

		fmt.Printf("\r\033[K") // Clear line
		fmt.Printf("\x1b[32m[OK]\x1b[0m %s\n", step.Name)
	}

	// Final 100% bar
	bar := strings.Repeat("█", barWidth)
	fmt.Printf("\x1b[36m%s\x1b[0m Ready!\n", bar)
	time.Sleep(500 * time.Millisecond)

	return nil
}

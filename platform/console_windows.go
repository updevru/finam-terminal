//go:build windows

package platform

import (
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

// EnableUTF8 forces UTF-8 and use the native console driver for tcell on Windows.
func EnableUTF8() {
	// Force UTF-8 and use the native console driver for tcell.
	// This driver handles keyboard layouts much better than the VT driver on Windows.
	_ = os.Setenv("TCELL_UTF8", "1")
	_ = os.Setenv("TCELL_DRIVER", "console")

	// Set console code pages using both syscall and command line for maximum compatibility
	_ = windows.SetConsoleCP(65001)
	_ = windows.SetConsoleOutputCP(65001)
	_ = exec.Command("chcp", "65001").Run()
}

//go:build !windows

package platform

// EnableUTF8 is a no-op for non-Windows platforms.
func EnableUTF8() {
	// No-op for non-Windows platforms
}

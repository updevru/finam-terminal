package ui

import (
	"finam-terminal/updater"
	"finam-terminal/version"
)

// SetUpdateAvailable records that a newer release is available and repaints
// the header so the ⚡ indicator lights up.
//
// It is safe to call from any goroutine, but the caller is responsible for
// running it on the tview event loop (QueueUpdateDraw) when the application is
// already running — this method only guards its own state.
//
// A version that is not genuinely newer than the running one (a stale cache, a
// dev build, a malformed tag) is ignored, so the indicator can never lie.
func (a *App) SetUpdateAvailable(latest string) {
	if !updater.IsNewer(version.String(), latest) {
		return
	}

	a.updateMu.Lock()
	a.latestVersion = latest
	a.updateMu.Unlock()

	a.refreshHeader()
}

// LatestVersion returns the newest release the application knows about, or an
// empty string when the running version is up to date.
func (a *App) LatestVersion() string {
	a.updateMu.RLock()
	defer a.updateMu.RUnlock()
	return a.latestVersion
}

// UpdateRequested reports whether the user asked to install the update before
// the application stopped. main.go checks it after Run returns and performs
// the update outside of tview, since the process cannot replace itself while
// the TUI owns the terminal.
func (a *App) UpdateRequested() bool {
	return a.updateRequested.Load()
}

// refreshHeader repaints the header from the current version state.
func (a *App) refreshHeader() {
	if a.header == nil {
		return
	}
	a.header.SetText(headerLabel(version.String(), a.LatestVersion()))
}

package ui

import (
	"fmt"

	"finam-terminal/updater"
	"finam-terminal/version"

	"github.com/rivo/tview"
)

// updateModalPage is the tview page name of the in-app update modal.
const updateModalPage = "update_modal"

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

// OpenUpdateModal shows the in-app update dialog. It is a no-op when no newer
// version is known, so the caller never has to check first.
func (a *App) OpenUpdateModal() {
	latest := a.LatestVersion()
	if latest == "" {
		return
	}

	modal := tview.NewModal().
		SetText(fmt.Sprintf("⚡ Доступна новая версия\n\n    Установлена:  %s\n    Актуальная:   %s",
			version.String(), latest)).
		AddButtons([]string{updateButtonLabel, "Отмена"}).
		SetDoneFunc(func(_ int, label string) {
			if label == updateButtonLabel {
				a.ConfirmUpdate()
				return
			}
			a.CloseUpdateModal()
		})

	a.pages.AddPage(updateModalPage, modal, true, true)
	a.app.SetFocus(modal)
}

// CloseUpdateModal dismisses the update dialog and returns focus to the
// portfolio.
func (a *App) CloseUpdateModal() {
	a.pages.RemovePage(updateModalPage)
	a.app.SetFocus(a.portfolioView.AccountTable)
}

// IsUpdateModalOpen reports whether the update dialog is currently shown.
func (a *App) IsUpdateModalOpen() bool {
	name, _ := a.pages.GetFrontPage()
	return name == updateModalPage
}

// ConfirmUpdate records the user's decision to install the update and stops
// the TUI. The update itself runs in main.go after Run returns: the process
// cannot replace and restart itself while tview owns the terminal.
func (a *App) ConfirmUpdate() {
	a.updateRequested.Store(true)
	a.CloseUpdateModal()
	a.Stop()
}

// HandleUpdateKey reacts to the U hotkey: it opens the update dialog when an
// update is available and otherwise says so in the status bar.
func (a *App) HandleUpdateKey() {
	if a.LatestVersion() == "" {
		a.SetStatus("Установлена последняя версия", StatusInfo)
		return
	}
	a.OpenUpdateModal()
}

package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Button labels of the update dialog, shared with the tests and the in-app
// modal so the wording stays in one place.
const (
	updateButtonLabel   = "Обновить и перезапустить"
	continueButtonLabel = "Продолжить"
)

// UpdatePromptApp is the short-lived tview application shown before the main
// TUI when a newer release is already known from the update cache.
//
// It is modelled on SetupApp: its own tview.Application, run to completion,
// with the answer read from the struct afterwards.
type UpdatePromptApp struct {
	app      *tview.Application
	modal    *tview.Modal
	current  string
	latest   string
	accepted bool
}

// NewUpdatePromptApp builds the "a new version is available" dialog for the
// running version and the newly published one.
//
// The safe choice — continuing to the terminal — is the default: it is the
// pre-selected button, it is what Esc does, and it is what the zero value of
// the answer means.
func NewUpdatePromptApp(current, latest string) *UpdatePromptApp {
	p := &UpdatePromptApp{
		app:     tview.NewApplication(),
		current: current,
		latest:  latest,
	}

	p.modal = tview.NewModal().
		SetText(p.promptText()).
		AddButtons([]string{updateButtonLabel, continueButtonLabel}).
		SetDoneFunc(p.choose)
	// Focus "Продолжить" so a stray Enter never starts an unwanted download.
	p.modal.SetFocus(1)

	p.app.SetInputCapture(p.handleKey)
	return p
}

// Run shows the dialog and blocks until the user answers. It reports whether
// the user chose to update; any failure to draw the dialog is treated as
// "continue", because a broken prompt must not block the terminal.
func (p *UpdatePromptApp) Run() bool {
	if err := p.app.SetRoot(p.modal, true).Run(); err != nil {
		return false
	}
	return p.accepted
}

// promptText renders the dialog body with both versions.
func (p *UpdatePromptApp) promptText() string {
	return fmt.Sprintf("⚡ Доступна новая версия Finam Terminal\n\n"+
		"    Установлена:  %s\n"+
		"    Актуальная:   %s", p.current, p.latest)
}

// choose records the answer and closes the dialog.
func (p *UpdatePromptApp) choose(index int, label string) {
	p.accepted = label == updateButtonLabel
	p.app.Stop()
}

// handleKey makes Esc equivalent to "Продолжить" and lets every other key
// through to the modal.
func (p *UpdatePromptApp) handleKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		p.accepted = false
		p.app.Stop()
		return nil
	}
	return event
}

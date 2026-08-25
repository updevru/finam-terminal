package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// pressRune feeds a rune through the global input handler and returns what the
// handler did with the event.
func pressRune(app *App, r rune) *tcell.EventKey {
	setupInputHandlers(app)
	capture := app.app.GetInputCapture()
	return capture(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
}

// TestUpdateKeyOpensModalWhenUpdateAvailable verifies U opens the update modal
// once a newer release is known, for both Latin and Cyrillic layouts.
func TestUpdateKeyOpensModalWhenUpdateAvailable(t *testing.T) {
	for _, r := range []rune{'u', 'U', 'г', 'Г'} {
		t.Run(string(r), func(t *testing.T) {
			asReleaseBuild(t)
			app := NewApp(&mockClient{}, nil)
			app.SetUpdateAvailable("v0.14.0")

			if res := pressRune(app, r); res != nil {
				t.Errorf("pressing %q returned %v, want the event consumed", r, res)
			}
			if !app.IsUpdateModalOpen() {
				t.Errorf("pressing %q did not open the update modal", r)
			}
		})
	}
}

// TestUpdateKeyWithoutUpdateShowsStatus verifies U reports "up to date" in the
// status bar instead of opening an empty dialog.
func TestUpdateKeyWithoutUpdateShowsStatus(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)

	if res := pressRune(app, 'u'); res != nil {
		t.Errorf("pressing u returned %v, want the event consumed", res)
	}
	if app.IsUpdateModalOpen() {
		t.Error("update modal opened without an available update")
	}
	if !strings.Contains(app.statusMessage, "последняя версия") {
		t.Errorf("status message = %q, want it to say the version is current", app.statusMessage)
	}
}

// TestUpdateModalConfirmSetsFlagAndStops verifies confirming the modal records
// the request so main.go can act on it after the TUI exits.
func TestUpdateModalConfirmSetsFlagAndStops(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)
	app.SetUpdateAvailable("v0.14.0")
	app.OpenUpdateModal()

	if app.UpdateRequested() {
		t.Fatal("UpdateRequested() is true before the user confirmed")
	}

	app.ConfirmUpdate()

	if !app.UpdateRequested() {
		t.Error("UpdateRequested() = false after confirming, want true")
	}
}

// TestUpdateModalCancelKeepsRunning verifies dismissing the modal changes
// nothing but the visible page.
func TestUpdateModalCancelKeepsRunning(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)
	app.SetUpdateAvailable("v0.14.0")
	app.OpenUpdateModal()

	app.CloseUpdateModal()

	if app.IsUpdateModalOpen() {
		t.Error("update modal is still open after cancelling")
	}
	if app.UpdateRequested() {
		t.Error("UpdateRequested() = true after cancelling, want false")
	}
}

// TestUpdateModalEscapeCloses verifies Esc dismisses the update modal instead
// of quitting the application.
func TestUpdateModalEscapeCloses(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)
	app.SetUpdateAvailable("v0.14.0")
	app.OpenUpdateModal()

	setupInputHandlers(app)
	capture := app.app.GetInputCapture()
	if res := capture(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)); res != nil {
		t.Errorf("Esc returned %v, want the event consumed", res)
	}

	if app.IsUpdateModalOpen() {
		t.Error("update modal is still open after Esc")
	}
	if app.UpdateRequested() {
		t.Error("Esc requested an update, want it to cancel")
	}
}

// TestUpdateKeyDoesNotShadowExistingBindings guards the keys already bound on
// the main screen against a regression from the new binding.
func TestUpdateKeyDoesNotShadowExistingBindings(t *testing.T) {
	asReleaseBuild(t)
	app := NewApp(&mockClient{}, nil)
	app.SetUpdateAvailable("v0.14.0")

	// Every rune already bound on the main screen must keep being consumed by
	// its own handler and must never reach the update dialog. (The search page
	// itself is only registered by the full layout, so this asserts routing
	// rather than the modal's visibility.)
	for _, r := range []rune{'s', 'S', 'ы', 'Ы', 'r', 'R', 'к', 'К'} {
		if res := pressRune(app, r); res != nil {
			t.Errorf("pressing %q returned %v, want its existing handler to consume it", r, res)
		}
		if app.IsUpdateModalOpen() {
			t.Fatalf("pressing %q opened the update modal", r)
		}
	}
}

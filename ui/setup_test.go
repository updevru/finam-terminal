package ui

import (
	"testing"
)

func TestNewSetupApp(t *testing.T) {
	app := NewSetupApp("test-addr")
	if app == nil {
		t.Fatal("NewSetupApp returned nil")
	}
	if app.grpcAddr != "test-addr" {
		t.Errorf("Expected grpcAddr test-addr, got %s", app.grpcAddr)
	}
	if app.app == nil {
		t.Error("SetupApp.app (tview.Application) is nil")
	}
	if app.inputField == nil {
		t.Error("SetupApp.inputField is nil")
	}
	if app.statusText == nil {
		t.Error("SetupApp.statusText is nil")
	}
}

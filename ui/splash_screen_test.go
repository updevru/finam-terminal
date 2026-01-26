package ui

import (
	"testing"
)

func TestNewSplashScreen(t *testing.T) {
	ss := NewSplashScreen()
	if ss == nil {
		t.Fatal("NewSplashScreen should not return nil")
	}

	if ss.Layout == nil {
		t.Error("SplashScreen should have a Layout")
	}

	if ss.Logo == nil {
		t.Error("SplashScreen should have a Logo")
	}
}

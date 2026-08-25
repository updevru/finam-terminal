package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// TestNewUpdatePromptApp verifies the dialog is fully built and shows both
// versions plus both choices.
func TestNewUpdatePromptApp(t *testing.T) {
	p := NewUpdatePromptApp("v0.13.0", "v0.14.0")

	if p == nil {
		t.Fatal("NewUpdatePromptApp returned nil")
	}
	if p.app == nil {
		t.Fatal("UpdatePromptApp.app is nil")
	}
	if p.modal == nil {
		t.Fatal("UpdatePromptApp.modal is nil")
	}

	text := p.promptText()
	for _, want := range []string{"v0.13.0", "v0.14.0"} {
		if !strings.Contains(text, want) {
			t.Errorf("prompt text = %q, want it to contain %q", text, want)
		}
	}

	if p.accepted {
		t.Error("prompt starts as accepted, want the safe default of continuing")
	}
}

// TestUpdatePromptChoices verifies each button records the right answer.
func TestUpdatePromptChoices(t *testing.T) {
	tests := []struct {
		name  string
		index int
		label string
		want  bool
	}{
		{name: "update", index: 0, label: updateButtonLabel, want: true},
		{name: "continue", index: 1, label: continueButtonLabel, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUpdatePromptApp("v0.13.0", "v0.14.0")
			p.choose(tt.index, tt.label)

			if p.accepted != tt.want {
				t.Errorf("after choosing %q accepted = %v, want %v", tt.label, p.accepted, tt.want)
			}
		})
	}
}

// TestUpdatePromptEscapeContinues verifies Esc is treated as "Continue" — the
// dialog must never be able to trap a user who just wants to start trading.
func TestUpdatePromptEscapeContinues(t *testing.T) {
	p := NewUpdatePromptApp("v0.13.0", "v0.14.0")
	p.accepted = true // make sure the handler actively resets the answer

	if got := p.handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)); got != nil {
		t.Errorf("handleKey(Esc) returned %v, want nil (event consumed)", got)
	}
	if p.accepted {
		t.Error("Esc left the prompt accepted, want it to mean Continue")
	}
}

// TestUpdatePromptPassesOtherKeys verifies unrelated keys still reach the
// modal so the arrow keys and Enter keep working.
func TestUpdatePromptPassesOtherKeys(t *testing.T) {
	p := NewUpdatePromptApp("v0.13.0", "v0.14.0")

	ev := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	if got := p.handleKey(ev); got != ev {
		t.Errorf("handleKey(Enter) = %v, want the event passed through", got)
	}
}

package ui

import (
	"strings"
	"testing"
)

func TestFinamLogo(t *testing.T) {
	if len(FinamLogo) == 0 {
		t.Error("FinamLogo constant should not be empty")
	}
	if !strings.Contains(FinamLogo, "█") {
		t.Error("FinamLogo should contain the block character █")
	}
}

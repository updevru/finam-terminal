package ui

import (
	"errors"
	"testing"
)

func TestRunStartupSteps(t *testing.T) {
	steps := []StartupStep{
		{
			Name: "Test Step 1",
			Action: func() error { return nil },
		},
		{
			Name: "Test Step 2",
			Action: func() error { return nil },
		},
	}
	err := RunStartupSteps(steps)
	if err != nil {
		t.Errorf("RunStartupSteps failed: %v", err)
	}
}

func TestRunStartupSteps_Failure(t *testing.T) {
	steps := []StartupStep{
		{
			Name: "Fail Step",
			Action: func() error { return errors.New("boom") },
		},
	}
	err := RunStartupSteps(steps)
	if err == nil {
		t.Error("RunStartupSteps should fail")
	}
}

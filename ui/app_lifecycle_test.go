package ui

import (
	"finam-terminal/api"
	"finam-terminal/models"
	"testing"
	"time"
)

func TestApp_Stop_Concurrency(t *testing.T) {
	// Setup minimal app dependencies
	mockClient := &api.Client{}
	accounts := []models.AccountInfo{{ID: "test-acc", Type: "Test", Equity: "100.00"}}

	app := NewApp(mockClient, accounts)

	// Start a goroutine that simulates the background refresh or other activity
	// checking stopChan, similar to how the real app works
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-app.stopChan:
				return
			case <-ticker.C:
				// Work
			}
		}
	}()

	// Simulate concurrent Stop calls which might happen from multiple UI events
	// or from main main() defer and UI exit key
	done := make(chan bool)
	go func() {
		app.Stop()
		done <- true
	}()
	go func() {
		app.Stop()
		done <- true
	}()

	// Wait for both to finish
	<-done
	<-done

	// If we didn't panic, we passed.
	// The panic usually happens because closing a closed channel panics.
}

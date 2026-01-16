package ui

import (
	"finam-terminal/models"
	"log"
	"strings"
	"time"
)

// loadData loads account data from API
func (a *App) loadData(accountID string) {
	a.dataMutex.Lock()
	defer a.dataMutex.Unlock()

	_, pos, err := a.client.GetAccountDetails(accountID)
	if err != nil {
		log.Printf("[WARN] Failed to load positions: %v", err)
		a.positions[accountID] = []models.Position{}
		a.quotes[accountID] = make(map[string]*models.Quote)
		return
	}

	a.positions[accountID] = pos
	a.quotes[accountID] = make(map[string]*models.Quote)

	if len(pos) > 0 {
		symbols := make([]string, len(pos))
		for i, p := range pos {
			symbols[i] = p.Symbol
		}
		if q, err := a.client.GetQuotes(accountID, symbols); err == nil {
			a.quotes[accountID] = q
		}
	}
}

// loadDataAsync loads account data from API asynchronously, preventing UI blocking.
func (a *App) loadDataAsync(accountID string) {
	// Skip "Updating..." status to avoid UI lockups on start
	// a.SetStatus("Updating...", StatusLoading)

	go func() {
		// Fetch data in a separate goroutine
		_, pos, err := a.client.GetAccountDetails(accountID)
		if err != nil {
			log.Printf("[WARN] Failed to load positions for %s: %v", accountID, err)
			
			errMsg := "Error loading data"
			if err != nil && (err.Error() == "context deadline exceeded" || strings.Contains(err.Error(), "DeadlineExceeded")) {
				errMsg = "Connection Timeout"
			}
			// Update status only on error or completion
			a.SetStatus(errMsg, StatusError)
			
			// On error, we can schedule a UI update to clear data
			a.app.QueueUpdateDraw(func() {
				a.dataMutex.Lock()
				a.positions[accountID] = []models.Position{}
				a.quotes[accountID] = make(map[string]*models.Quote)
				a.dataMutex.Unlock()
				if a.selectedIdx < len(a.accounts) && a.accounts[a.selectedIdx].ID == accountID {
					updatePositionsTable(a)
					updateInfoPanel(a)
				}
			})
			return
		}

		var finalQuotes map[string]*models.Quote
		if len(pos) > 0 {
			symbols := make([]string, len(pos))
			for i, p := range pos {
				symbols[i] = p.Symbol
			}
			quotes, err := a.client.GetQuotes(accountID, symbols)
			if err != nil {
				log.Printf("[WARN] Failed to get quotes for %s: %v", accountID, err)
				a.SetStatus("Data loaded (Quotes failed)", StatusSuccess)
			} else {
				finalQuotes = quotes
				a.SetStatus("Data updated", StatusSuccess)
			}
		} else {
			a.SetStatus("Data updated", StatusSuccess)
		}

		// Clear status after some time
		time.AfterFunc(3*time.Second, func() {
			a.dataMutex.RLock()
			currentMsg := a.statusMessage
			a.dataMutex.RUnlock()
			if currentMsg == "Data updated" || currentMsg == "Data loaded (Quotes failed)" {
				a.SetStatus("", StatusInfo)
			}
		})

		// Schedule a UI update on the main thread
		a.app.QueueUpdateDraw(func() {
			a.dataMutex.Lock()
			a.positions[accountID] = pos
			if finalQuotes != nil {
				a.quotes[accountID] = finalQuotes
			} else {
				a.quotes[accountID] = make(map[string]*models.Quote)
			}
			a.dataMutex.Unlock()

			// If the data for the currently viewed account is updated, refresh the view.
			if a.selectedIdx < len(a.accounts) && a.accounts[a.selectedIdx].ID == accountID {
				updatePositionsTable(a)
				updateInfoPanel(a)
				updateStatusBar(a)
			}
		})
	}()
}

// backgroundRefresh runs periodic data refresh
func (a *App) backgroundRefresh(app *App) {
	// Initial refresh immediately
	time.Sleep(500 * time.Millisecond) // Give more time for UI start
	a.app.QueueUpdateDraw(func() {
		if a.selectedIdx >= 0 && a.selectedIdx < len(a.accounts) {
			a.loadDataAsync(a.accounts[a.selectedIdx].ID)
		}
	})

	ticker := time.NewTicker(refreshPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			// access UI state (selectedIdx) safely on the UI thread
			a.app.QueueUpdateDraw(func() {
				// Prioritize the active account
				if a.selectedIdx >= 0 && a.selectedIdx < len(a.accounts) {
					activeID := a.accounts[a.selectedIdx].ID
					a.loadDataAsync(activeID)

					// Refresh others
					for i, acc := range a.accounts {
						if i != a.selectedIdx {
							a.loadDataAsync(acc.ID)
						}
					}
				}
			})
		}
	}
}

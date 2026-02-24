package ui

import (
	"finam-terminal/models"
	"log"
	"strings"
	"sync"
	"time"
)

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

// loadHistoryAsync loads trade history from API asynchronously
func (a *App) loadHistoryAsync(accountID string) {
	a.SetStatus("Loading History...", StatusLoading)
	go func() {
		history, err := a.client.GetTradeHistory(accountID)
		if err != nil {
			log.Printf("[WARN] Failed to load history for %s: %v", accountID, err)
			a.SetStatus("Error loading history", StatusError)
			return
		}

		a.dataMutex.Lock()
		a.history[accountID] = history
		a.dataMutex.Unlock()

		a.app.QueueUpdateDraw(func() {
			if a.selectedIdx < len(a.accounts) && a.accounts[a.selectedIdx].ID == accountID {
				updateHistoryTable(a)
				a.SetStatus("History updated", StatusSuccess)
			}
		})
	}()
}

// loadOrdersAsync loads active orders from API asynchronously
func (a *App) loadOrdersAsync(accountID string) {
	a.SetStatus("Loading Orders...", StatusLoading)
	go func() {
		orders, err := a.client.GetActiveOrders(accountID)
		if err != nil {
			log.Printf("[WARN] Failed to load orders for %s: %v", accountID, err)
			a.SetStatus("Error loading orders", StatusError)
			return
		}

		a.dataMutex.Lock()
		a.activeOrders[accountID] = orders
		a.dataMutex.Unlock()

		a.app.QueueUpdateDraw(func() {
			if a.selectedIdx < len(a.accounts) && a.accounts[a.selectedIdx].ID == accountID {
				updateOrdersTable(a)
				a.SetStatus("Orders updated", StatusSuccess)
			}
		})
	}()
}

// loadProfileAsync loads all profile data in parallel goroutines.
func (a *App) loadProfileAsync(accountID, symbol string, timeframeIdx int) {
	go func() {
		profile := &models.InstrumentProfile{Symbol: symbol}
		var mu sync.Mutex
		var wg sync.WaitGroup

		// 1. GetAssetInfo
		wg.Add(1)
		go func() {
			defer wg.Done()
			details, err := a.client.GetAssetInfo(accountID, symbol)
			if err != nil {
				log.Printf("[WARN] GetAssetInfo failed for %s: %v", symbol, err)
				return
			}
			mu.Lock()
			profile.Details = details
			mu.Unlock()
		}()

		// 2. GetAssetParams
		wg.Add(1)
		go func() {
			defer wg.Done()
			params, err := a.client.GetAssetParams(accountID, symbol)
			if err != nil {
				log.Printf("[WARN] GetAssetParams failed for %s: %v", symbol, err)
				return
			}
			mu.Lock()
			profile.Params = params
			mu.Unlock()
		}()

		// 3. GetQuotes
		wg.Add(1)
		go func() {
			defer wg.Done()
			quotes, err := a.client.GetQuotes(accountID, []string{symbol})
			if err != nil {
				log.Printf("[WARN] GetQuotes failed for %s: %v", symbol, err)
				return
			}
			mu.Lock()
			for _, q := range quotes {
				profile.Quote = q
				break
			}
			mu.Unlock()
		}()

		// 4. GetSchedule
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions, err := a.client.GetSchedule(symbol)
			if err != nil {
				log.Printf("[WARN] GetSchedule failed for %s: %v", symbol, err)
				return
			}
			mu.Lock()
			profile.Schedule = sessions
			mu.Unlock()
		}()

		// 5. GetBars
		wg.Add(1)
		go func() {
			defer wg.Done()
			now := time.Now()
			tf := profileTimeframeEnums[timeframeIdx]
			from := now.Add(-profileTimeframeDurations[timeframeIdx])
			bars, err := a.client.GetBars(accountID, symbol, tf, from, now)
			if err != nil {
				log.Printf("[WARN] GetBars failed for %s: %v", symbol, err)
				return
			}
			mu.Lock()
			profile.Bars = bars
			mu.Unlock()
		}()

		wg.Wait()

		a.app.QueueUpdateDraw(func() {
			if a.profileOpen && a.profileSymbol == symbol {
				a.profilePanel.Update(profile)
				a.profilePanel.RestoreFooter()
			}
		})
	}()
}

// loadProfileBarsAsync reloads only bars for a timeframe switch.
func (a *App) loadProfileBarsAsync(accountID, symbol string, timeframeIdx int) {
	go func() {
		now := time.Now()
		tf := profileTimeframeEnums[timeframeIdx]
		from := now.Add(-profileTimeframeDurations[timeframeIdx])

		bars, err := a.client.GetBars(accountID, symbol, tf, from, now)
		if err != nil {
			log.Printf("[WARN] GetBars failed for %s (timeframe switch): %v", symbol, err)
			return
		}

		a.app.QueueUpdateDraw(func() {
			if a.profileOpen && a.profileSymbol == symbol {
				a.profilePanel.UpdateChart(bars)
			}
		})
	}()
}

// refreshProfileQuoteAndBars refreshes only quote and bars for an open profile.
func (a *App) refreshProfileQuoteAndBars(accountID, symbol string, timeframeIdx int) {
	go func() {
		var wg sync.WaitGroup
		var newQuote *models.Quote
		var newBars []models.Bar
		var mu sync.Mutex

		wg.Add(1)
		go func() {
			defer wg.Done()
			quotes, err := a.client.GetQuotes(accountID, []string{symbol})
			if err != nil {
				return
			}
			mu.Lock()
			for _, q := range quotes {
				newQuote = q
				break
			}
			mu.Unlock()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			now := time.Now()
			tf := profileTimeframeEnums[timeframeIdx]
			from := now.Add(-profileTimeframeDurations[timeframeIdx])
			bars, err := a.client.GetBars(accountID, symbol, tf, from, now)
			if err != nil {
				return
			}
			mu.Lock()
			newBars = bars
			mu.Unlock()
		}()

		wg.Wait()

		a.app.QueueUpdateDraw(func() {
			if a.profileOpen && a.profileSymbol == symbol {
				if a.profilePanel.profile != nil {
					if newQuote != nil {
						a.profilePanel.profile.Quote = newQuote
					}
					if newBars != nil {
						a.profilePanel.profile.Bars = newBars
					}
					a.profilePanel.Update(a.profilePanel.profile)
				}
			}
		})
	}()
}

// backgroundRefresh runs periodic data refresh
func (a *App) backgroundRefresh() {
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

					// Refresh profile if open
					if a.profileOpen && a.profileSymbol != "" {
						a.refreshProfileQuoteAndBars(activeID, a.profileSymbol, a.profileTimeframe)
					}

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

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
	// Check if this account has a load error — skip API calls
	for _, acc := range a.accounts {
		if acc.ID == accountID && acc.LoadError != "" {
			a.SetStatus("Account unavailable: "+acc.LoadError, StatusError)
			return
		}
	}

	go func() {
		// Fetch data in a separate goroutine
		accInfo, pos, err := a.client.GetAccountDetails(accountID)
		if err != nil {
			log.Printf("[WARN] Failed to load positions for %s: %v", accountID, err)

			errMsg := "Error loading data"
			if err != nil && (err.Error() == "context deadline exceeded" || strings.Contains(err.Error(), "DeadlineExceeded")) {
				errMsg = "Connection Timeout"
			}
			// Update status only on error or completion — keep cached data visible
			a.SetStatus(errMsg, StatusError)
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
			// Update account info (Equity, UnrealizedPnL) with fresh data from API
			if accInfo != nil {
				for i := range a.accounts {
					if a.accounts[i].ID == accountID {
						a.accounts[i].Equity = accInfo.Equity
						a.accounts[i].UnrealizedPnL = accInfo.UnrealizedPnL
						break
					}
				}
			}
			a.dataMutex.Unlock()

			// If the data for the currently viewed account is updated, refresh the view.
			if a.selectedIdx < len(a.accounts) && a.accounts[a.selectedIdx].ID == accountID {
				updateAccountList(a)
				updatePositionsTable(a)
				updateInfoPanel(a)
				updateStatusBar(a)
			}
		})
	}()
}

// loadHistoryAsync loads trade history from API asynchronously
func (a *App) loadHistoryAsync(accountID string) {
	for _, acc := range a.accounts {
		if acc.ID == accountID && acc.LoadError != "" {
			return
		}
	}
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
	for _, acc := range a.accounts {
		if acc.ID == accountID && acc.LoadError != "" {
			return
		}
	}
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

		// 1. GetAssetInfo first — the instrument type it reveals gates which
		// corporate-action calendars (if any) are fetched below.
		details, err := a.client.GetAssetInfo(accountID, symbol)
		if err != nil {
			log.Printf("[WARN] GetAssetInfo failed for %s: %v", symbol, err)
		} else {
			profile.Details = details
		}

		// 2. GetAssetParams
		wg.Go(func() {
			params, err := a.client.GetAssetParams(accountID, symbol)
			if err != nil {
				log.Printf("[WARN] GetAssetParams failed for %s: %v", symbol, err)
				return
			}
			mu.Lock()
			profile.Params = params
			mu.Unlock()
		})

		// 3. GetQuotes
		wg.Go(func() {
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
		})

		// 4. GetSchedule
		wg.Go(func() {
			sessions, err := a.client.GetSchedule(symbol)
			if err != nil {
				log.Printf("[WARN] GetSchedule failed for %s: %v", symbol, err)
				return
			}
			mu.Lock()
			profile.Schedule = sessions
			mu.Unlock()
		})

		// 5. GetBars
		wg.Go(func() {
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
		})

		// 6. Type-gated corporate-action calendars. For an equity, fetch the
		// dividend and split calendars in parallel; each is non-fatal — a failed
		// request just leaves its section empty. Non-equities (futures/options/
		// bonds) skip these entirely (bonds get their own events in Phase 4).
		if isEquityDetails(details) {
			wg.Go(func() {
				dividends, err := a.client.GetDividends(symbol)
				if err != nil {
					log.Printf("[WARN] GetDividends failed for %s: %v", symbol, err)
					return
				}
				mu.Lock()
				profile.Dividends = dividends
				mu.Unlock()
			})

			wg.Go(func() {
				splits, err := a.client.GetSplits(symbol)
				if err != nil {
					log.Printf("[WARN] GetSplits failed for %s: %v", symbol, err)
					return
				}
				mu.Lock()
				profile.Splits = splits
				mu.Unlock()
			})
		} else if isBondDetails(details) {
			// For a bond, fetch the coupon/amortization/offer calendar; non-fatal.
			wg.Go(func() {
				events, err := a.client.GetBondEvents(symbol)
				if err != nil {
					log.Printf("[WARN] GetBondEvents failed for %s: %v", symbol, err)
					return
				}
				mu.Lock()
				profile.BondEvents = events
				mu.Unlock()
			})
		}

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

		wg.Go(func() {
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
		})

		wg.Go(func() {
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
		})

		wg.Wait()

		a.app.QueueUpdateDraw(func() {
			if a.profileOpen && a.profileSymbol == symbol {
				if p := a.profilePanel.GetProfile(); p != nil {
					if newQuote != nil {
						p.Quote = newQuote
					}
					if newBars != nil {
						p.Bars = newBars
					}
					a.profilePanel.Update(p)
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

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// setupInputHandlers configures keyboard input handling
func setupInputHandlers(app *App) {
	quit := func() {
		app.Stop()
	}

	refresh := func() {
		// The Index tab is account-independent, so it refreshes before the
		// account guard below — it works even with no account selected.
		if app.portfolioView.TabbedView.ActiveTab == TabIndex {
			app.portfolioView.TabbedView.IndexTable.Clear()
			app.loadIndexAsync()
			// A manual refresh always fetches quotes, even inside the cooldown
			// and after a rate limit turned automation off.
			app.pollIndexQuotesAsync(true)
			return
		}
		if app.selectedIdx < len(app.accounts) {
			accountID := app.accounts[app.selectedIdx].ID
			switch app.portfolioView.TabbedView.ActiveTab {
			case TabPositions:
				app.portfolioView.TabbedView.PositionsTable.Clear()
				app.loadDataAsync(accountID)
			case TabHistory:
				app.portfolioView.TabbedView.HistoryTable.Clear()
				app.loadHistoryAsync(accountID)
			case TabOrders:
				app.portfolioView.TabbedView.OrdersTable.Clear()
				app.loadOrdersAsync(accountID)
			}
		}
	}

	switchAccount := func(idx int) {
		if idx >= 0 && idx < len(app.accounts) {
			// Written under the lock: background loaders read selectedIdx to
			// decide whether the quote stream owns this account.
			app.dataMutex.Lock()
			app.selectedIdx = idx
			app.dataMutex.Unlock()

			updateAccountList(app)

			// Update view immediately with cached data
			updatePositionsTable(app)
			updateInfoPanel(app)
			updateStatusBar(app)

			// The stream follows the active account
			app.recomputeStreamSymbols()

			// Trigger fresh data load for active tab
			accountID := app.accounts[idx].ID
			switch app.portfolioView.TabbedView.ActiveTab {
			case TabPositions:
				app.loadDataAsync(accountID)
			case TabHistory:
				app.loadHistoryAsync(accountID)
			case TabOrders:
				app.loadOrdersAsync(accountID)
			}
		}
	}

	switchToTab := func(tab TabType) {
		app.portfolioView.TabbedView.SetTab(tab)
		// Always update focus to the newly visible table
		switch tab {
		case TabPositions:
			app.app.SetFocus(app.portfolioView.TabbedView.PositionsTable)
		case TabHistory:
			app.app.SetFocus(app.portfolioView.TabbedView.HistoryTable)
		case TabOrders:
			app.app.SetFocus(app.portfolioView.TabbedView.OrdersTable)
		case TabIndex:
			app.app.SetFocus(app.portfolioView.TabbedView.IndexTable)
		}

		// Same reason as in refresh: the Index tab has no account to wait for.
		if tab == TabIndex {
			// Start the load first, then draw: the other order paints
			// "No constituents" and leaves it there for the whole fetch.
			app.ensureIndexLoaded()
			updateIndexTable(app)
			// With a live stream this is a no-op; with a dead one it fills the
			// tab from a single bounded batch.
			app.pollIndexQuotesAsync(false)
		}

		// The composition joins the subscription on entry and leaves it on exit,
		// so an unwatched tab costs nothing.
		app.recomputeStreamSymbols()

		if app.selectedIdx >= len(app.accounts) {
			return
		}
		accountID := app.accounts[app.selectedIdx].ID
		switch tab {
		case TabPositions:
			// Positions use cached data; background refresh keeps them fresh
			app.dataMutex.RLock()
			_, loaded := app.positions[accountID]
			app.dataMutex.RUnlock()
			if !loaded {
				app.loadDataAsync(accountID)
			}
		case TabHistory:
			// Always reload — trades may come from other terminals
			app.loadHistoryAsync(accountID)
		case TabOrders:
			// Always reload — orders may change from other terminals
			app.loadOrdersAsync(accountID)
		}
	}

	nextTab := func() {
		next := (int(app.portfolioView.TabbedView.ActiveTab) + 1) % TabCount()
		switchToTab(TabType(next))
	}

	prevTab := func() {
		prev := (int(app.portfolioView.TabbedView.ActiveTab) - 1 + TabCount()) % TabCount()
		switchToTab(TabType(prev))
	}

	setupTableNavigation := func(table *tview.Table) {
		table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyRight:
				nextTab()
				return nil
			case tcell.KeyLeft:
				prevTab()
				return nil
			case tcell.KeyDown, tcell.KeyCtrlN:
				row, _ := table.GetSelection()
				if row < table.GetRowCount()-1 {
					table.Select(row+1, 0)
				}
				return nil
			case tcell.KeyUp, tcell.KeyCtrlP:
				row, _ := table.GetSelection()
				if row > 1 {
					table.Select(row-1, 0)
				}
				return nil
			}
			switch event.Key() {
			case tcell.KeyEnter:
				if table == app.portfolioView.TabbedView.PositionsTable {
					app.OpenProfile()
					return nil
				}
				if table == app.portfolioView.TabbedView.IndexTable {
					if symbol := app.selectedIndexSymbol(); symbol != "" {
						app.OpenProfileForSymbol(symbol)
					}
					return nil
				}
			case tcell.KeyDelete:
				if table == app.portfolioView.TabbedView.OrdersTable {
					app.ShowCancelConfirmation()
					return nil
				}
			}
			switch event.Rune() {
			case 'q', 'Q', 'й', 'Й':
				quit()
				return nil
			case 'r', 'R', 'к', 'К':
				refresh()
				return nil
			case 'a', 'A', 'ф', 'Ф':
				if table == app.portfolioView.TabbedView.PositionsTable {
					app.OpenOrderModal()
				}
				if table == app.portfolioView.TabbedView.IndexTable {
					// The composition carries full ticker@mic symbols, so the
					// instrument goes through the existing order path unchanged.
					if symbol := app.selectedIndexSymbol(); symbol != "" {
						app.OpenOrderModalWithTicker(symbol)
					}
				}
				return nil
			case 'x', 'X', 'ч', 'Ч':
				if table == app.portfolioView.TabbedView.OrdersTable {
					app.ShowCancelConfirmation()
				}
				return nil
			case 'e', 'E', 'у', 'У':
				if table == app.portfolioView.TabbedView.OrdersTable {
					app.ShowModifyOrderModal()
				}
				return nil
			case 'c', 'C', 'с', 'С':
				if table == app.portfolioView.TabbedView.PositionsTable {
					app.OpenCloseModal()
				}
				return nil
			case 's', 'S', 'ы', 'Ы':
				app.OpenSearchModal()
				return nil
			}
			return event
		})
	}

	setupTableNavigation(app.portfolioView.TabbedView.PositionsTable)
	setupTableNavigation(app.portfolioView.TabbedView.HistoryTable)
	setupTableNavigation(app.portfolioView.TabbedView.OrdersTable)
	setupTableNavigation(app.portfolioView.TabbedView.IndexTable)

	// The stream carries a window of the composition, so scrolling has to move
	// the window with it. Runs on the event loop, and the window is quantised,
	// so this only resubscribes once per block of rows.
	app.portfolioView.TabbedView.IndexTable.SetSelectionChangedFunc(func(int, int) {
		app.recomputeStreamSymbols()
	})

	app.portfolioView.AccountTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown:
			switchAccount(app.selectedIdx + 1)
			return nil
		case tcell.KeyUp:
			switchAccount(app.selectedIdx - 1)
			return nil
		case tcell.KeyEnter:
			// Ignore Enter key to prevent freezing issues and accidental refreshes
			return nil
		}
		switch event.Rune() {
		case 'q', 'Q', 'й', 'Й':
			quit()
			return nil
		case 'r', 'R', 'к', 'К':
			refresh()
			return nil
		}
		return event
	})

	app.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If profile overlay is open, handle profile keys at global level
		// to avoid focus-related issues with ChartView inside Flex/Pages
		if app.IsProfileOpen() {
			// Alert modal (error dialog) on top — let all events pass through to it
			if app.IsAlertOpen() {
				return event
			}
			// Cancel confirmation on top of profile
			if app.IsCancelConfirmOpen() {
				if event.Key() == tcell.KeyEscape {
					app.pages.RemovePage("cancel_confirm")
					app.app.SetFocus(app.profilePanel.ChartView)
					return nil
				}
				return event
			}
			// Modals on top of profile: only handle Escape to close them
			if app.IsModalOpen() || app.IsCloseModalOpen() || app.IsSearchModalOpen() {
				if event.Key() == tcell.KeyEscape {
					if app.IsModalOpen() {
						app.CloseOrderModal()
						app.app.SetFocus(app.profilePanel.ChartView)
						return nil
					}
					if app.IsCloseModalOpen() {
						app.CloseCloseModal()
						app.app.SetFocus(app.profilePanel.ChartView)
						return nil
					}
					if app.IsSearchModalOpen() {
						app.CloseSearchModal()
						app.app.SetFocus(app.profilePanel.ChartView)
						return nil
					}
				}
				return event
			}
			// Profile keyboard shortcuts (handled globally for reliability)
			switch event.Key() {
			case tcell.KeyEscape:
				app.CloseProfile()
				return nil
			}
			switch event.Rune() {
			case '1':
				app.switchProfileTimeframe(0)
				return nil
			case '2':
				app.switchProfileTimeframe(1)
				return nil
			case '3':
				app.switchProfileTimeframe(2)
				return nil
			case '4':
				app.switchProfileTimeframe(3)
				return nil
			case 'a', 'A', 'ф', 'Ф':
				app.OpenOrderModalWithTicker(app.profileSymbol)
				return nil
			case 'r', 'R', 'к', 'К':
				if app.selectedIdx >= 0 && app.selectedIdx < len(app.accounts) {
					app.profilePanel.Footer.SetText("[yellow]Refreshing...[-]")
					app.loadProfileAsync(app.accounts[app.selectedIdx].ID, app.profileSymbol, app.profileTimeframe)
				}
				return nil
			case 's', 'S', 'ы', 'Ы':
				app.OpenSearchModal()
				return nil
			case 'q', 'Q', 'й', 'Й':
				quit()
				return nil
			}
			return nil // Consume unhandled keys to prevent them from reaching ChartView
		}

		// Cancel confirmation modal — pass all events through (Tab, Enter work natively)
		if app.IsCancelConfirmOpen() {
			if event.Key() == tcell.KeyEscape {
				app.pages.RemovePage("cancel_confirm")
				app.app.SetFocus(app.portfolioView.TabbedView.OrdersTable)
				return nil
			}
			return event
		}

		// If any modal is open, only handle Escape globally (if needed) or pass to focused widget
		if app.IsModalOpen() || app.IsCloseModalOpen() || app.IsSearchModalOpen() {
			if event.Key() == tcell.KeyEscape {
				if app.IsModalOpen() {
					app.CloseOrderModal()
					return nil
				}
				if app.IsCloseModalOpen() {
					app.CloseCloseModal()
					return nil
				}
				if app.IsSearchModalOpen() {
					app.CloseSearchModal()
					return nil
				}
			}
			// Return event to be handled by the modal's internal components (e.g. InputField)
			return event
		}

		// Update modal — Esc dismisses it, other keys reach its buttons
		if app.IsUpdateModalOpen() {
			if event.Key() == tcell.KeyEscape {
				app.CloseUpdateModal()
				return nil
			}
			return event
		}

		switch event.Key() {
		case tcell.KeyF1:
			// Switch to PortfolioView (already there, but for consistency)
			app.app.SetFocus(app.portfolioView.AccountTable)
			updateStatusBar(app)
			return nil
		case tcell.KeyF2:
			refresh()
			return nil
		case tcell.KeyTab, tcell.KeyBacktab:
			if app.app.GetFocus() == app.portfolioView.AccountTable {
				// Switch to the active tab's table
				switch app.portfolioView.TabbedView.ActiveTab {
				case TabPositions:
					app.app.SetFocus(app.portfolioView.TabbedView.PositionsTable)
				case TabHistory:
					app.app.SetFocus(app.portfolioView.TabbedView.HistoryTable)
				case TabOrders:
					app.app.SetFocus(app.portfolioView.TabbedView.OrdersTable)
				}
			} else {
				// Switch back to Account Table
				app.app.SetFocus(app.portfolioView.AccountTable)
			}
			updateStatusBar(app)
			return nil
		case tcell.KeyLeft:
			prevTab()
			return nil
		case tcell.KeyRight:
			nextTab()
			return nil
		case tcell.KeyCtrlR:
			refresh()
			return nil
		case tcell.KeyCtrlC, tcell.KeyEscape:
			quit()
			return nil
		}
		switch event.Rune() {
		case 'q', 'Q', 'й', 'Й':
			quit()
			return nil
		case 'r', 'R', 'к', 'К':
			refresh()
			return nil
		case 's', 'S', 'ы', 'Ы':
			app.OpenSearchModal()
			return nil
		case 'u', 'U', 'г', 'Г':
			app.HandleUpdateKey()
			return nil
		}
		return event
	})
}

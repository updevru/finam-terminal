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
			app.selectedIdx = idx
			updateAccountList(app)

			// Update view immediately with cached data
			updatePositionsTable(app)
			updateInfoPanel(app)
			updateStatusBar(app)

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
		}
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
		next := (int(app.portfolioView.TabbedView.ActiveTab) + 1) % 3
		switchToTab(TabType(next))
	}

	prevTab := func() {
		prev := (int(app.portfolioView.TabbedView.ActiveTab) - 1 + 3) % 3
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
			}
			switch event.Rune() {
			case 'q', 'Q':
				quit()
				return nil
			case 'r', 'R':
				refresh()
				return nil
			case 'a', 'A':
				if table == app.portfolioView.TabbedView.PositionsTable {
					app.OpenOrderModal()
				}
				return nil
			case 'c', 'C':
				if table == app.portfolioView.TabbedView.PositionsTable {
					app.OpenCloseModal()
				}
				return nil
			case 's', 'S':
				app.OpenSearchModal()
				return nil
			}
			return event
		})
	}

	setupTableNavigation(app.portfolioView.TabbedView.PositionsTable)
	setupTableNavigation(app.portfolioView.TabbedView.HistoryTable)
	setupTableNavigation(app.portfolioView.TabbedView.OrdersTable)

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
		case 'q', 'Q':
			quit()
			return nil
		case 'r', 'R':
			refresh()
			return nil
		}
		return event
	})

	// Profile overlay input handler
	app.profilePanel.ChartView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
		case 'a', 'A':
			app.OpenOrderModalWithTicker(app.profileSymbol)
			return nil
		case 'r', 'R':
			if app.selectedIdx >= 0 && app.selectedIdx < len(app.accounts) {
				app.loadProfileAsync(app.accounts[app.selectedIdx].ID, app.profileSymbol, app.profileTimeframe)
			}
			return nil
		case 's', 'S':
			app.OpenSearchModal()
			return nil
		case 'q', 'Q':
			quit()
			return nil
		}
		return event
	})

	app.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If profile overlay is open, let profile handle its own keys
		if app.IsProfileOpen() {
			// Only intercept ESC at global level for modals on top of profile
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
				return event
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
		case 'q', 'Q':
			quit()
			return nil
		case 'r', 'R':
			refresh()
			return nil
		case 's', 'S':
			app.OpenSearchModal()
			return nil
		}
		return event
	})
}

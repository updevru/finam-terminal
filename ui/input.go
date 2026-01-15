package ui

import (
	"github.com/gdamore/tcell/v2"
)

// setupInputHandlers configures keyboard input handling
func setupInputHandlers(app *App) {
	quit := func() {
		app.Stop()
	}

	refresh := func() {
		if app.selectedIdx < len(app.accounts) {
			app.portfolioView.PositionsTable.Clear()
			app.loadDataAsync(app.accounts[app.selectedIdx].ID)
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

			// Trigger fresh data load
			app.loadDataAsync(app.accounts[idx].ID)
		}
	}

	app.portfolioView.AccountTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown:
			switchAccount(app.selectedIdx + 1)
			return nil
		case tcell.KeyUp:
			switchAccount(app.selectedIdx - 1)
			return nil
		case tcell.KeyEnter:
			refresh()
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

	app.portfolioView.PositionsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			app.app.SetFocus(app.portfolioView.AccountTable)
			updateStatusBar(app)
			return nil
		case tcell.KeyDown, tcell.KeyCtrlN:
			row, _ := app.portfolioView.PositionsTable.GetSelection()
			if row < app.portfolioView.PositionsTable.GetRowCount()-1 {
				app.portfolioView.PositionsTable.Select(row+1, 0)
			}
			return nil
		case tcell.KeyUp, tcell.KeyCtrlP:
			row, _ := app.portfolioView.PositionsTable.GetSelection()
			if row > 1 {
				app.portfolioView.PositionsTable.Select(row-1, 0)
			}
			return nil
		}
		switch event.Rune() {
		case 'q', 'Q':
			quit()
			return nil
		case 'r', 'R':
			refresh()
			return nil
		case 'a', 'A':
			app.OpenOrderModal()
			return nil
		case 'c', 'C':
			app.OpenCloseModal()
			return nil
		}
		return event
	})

	app.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If modal is open, intercept specific keys
		if app.IsModalOpen() {
			switch event.Key() {
			case tcell.KeyEscape:
				app.CloseOrderModal()
				return nil
			}
			return event
		}
		if app.IsCloseModalOpen() {
			switch event.Key() {
			case tcell.KeyEscape:
				app.CloseCloseModal()
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
		case tcell.KeyF2, tcell.KeyCtrlR:
			refresh()
			return nil
		case tcell.KeyCtrlC, tcell.KeyEscape:
			quit()
			return nil
		case tcell.KeyTab:
			if app.app.GetFocus() == app.portfolioView.AccountTable {
				app.app.SetFocus(app.portfolioView.PositionsTable)
			} else {
				app.app.SetFocus(app.portfolioView.AccountTable)
			}
			updateStatusBar(app)
			return nil
		case tcell.KeyLeft:
			switchAccount(app.selectedIdx - 1)
			return nil
		case tcell.KeyRight:
			switchAccount(app.selectedIdx + 1)
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
}

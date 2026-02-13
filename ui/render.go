package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// updateAccountList refreshes the account list
func updateAccountList(app *App) {
	app.portfolioView.AccountTable.Clear()
	headers := []string{"ID", "Type", "Equity"}
	for i, h := range headers {
		app.portfolioView.AccountTable.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false))
	}

	for i, acc := range app.accounts {
		equity := acc.Equity
		if val, err := parseFloat(equity); err == nil {
			equity = fmt.Sprintf("%.2f", val)
		}

		app.portfolioView.AccountTable.SetCell(i+1, 0, tview.NewTableCell(acc.ID).SetTextColor(tcell.ColorWhite))
		app.portfolioView.AccountTable.SetCell(i+1, 1, tview.NewTableCell(acc.Type).SetTextColor(tcell.ColorWhite))
		app.portfolioView.AccountTable.SetCell(i+1, 2, tview.NewTableCell(equity).SetTextColor(tcell.ColorWhite))
	}
	// Select the row corresponding to selectedIdx
	// Row 0 is header, so row i+1 matches account i
	app.portfolioView.AccountTable.Select(app.selectedIdx+1, 0)
}

// updatePositionsTable refreshes the positions table
func updatePositionsTable(app *App) {
	app.portfolioView.TabbedView.PositionsTable.Clear()

	headers := []string{"Instrument", "Qty (Lots)", "AvgPrice", "Current", "Daily P&L", "Value", "Unreal P&L"}
	headerStyle := tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite).
		Bold(true)

	for i, h := range headers {
		align := tview.AlignRight
		if i == 0 {
			align = tview.AlignLeft
		}
		cell := tview.NewTableCell(h).
			SetStyle(headerStyle).
			SetAlign(align).
			SetExpansion(1)
		app.portfolioView.TabbedView.PositionsTable.SetCell(0, i, cell)
	}

	app.dataMutex.RLock()
	accountID := app.accounts[app.selectedIdx].ID
	pos := app.positions[accountID]
	q := app.quotes[accountID]
	app.dataMutex.RUnlock()

	for row, p := range pos {
		quote := q[p.Symbol]
		rowNum := row + 1

		qty, _ := parseFloat(p.Quantity)
		displayQty := displayLots(p.Quantity, p.LotSize)

		totalValue := "N/A"
		if quote != nil && quote.Last != "N/A" {
			lastPrice, _ := parseFloat(quote.Last)
			totalValue = fmt.Sprintf("%.2f", qty*lastPrice)
		}

		dailyPnL := p.DailyPnL
		dailyColor := tcell.ColorWhite
		if dailyPnL != "N/A" {
			if val, err := parseFloat(dailyPnL); err == nil {
				if val > 0 {
					dailyPnL = "+" + dailyPnL
					dailyColor = tcell.ColorGreen
				} else if val < 0 {
					dailyColor = tcell.ColorRed
				}
			}
		}

		unrealizedPnL := p.UnrealizedPnL
		unrealColor := tcell.ColorWhite
		if unrealizedPnL != "N/A" {
			if val, err := parseFloat(unrealizedPnL); err == nil {
				if val > 0 {
					unrealizedPnL = "+" + unrealizedPnL
					unrealColor = tcell.ColorGreen
				} else if val < 0 {
					unrealColor = tcell.ColorRed
				}
			}
		}

		displayName := p.Name
		if displayName == "" {
			displayName = p.Ticker
			if p.MIC != "" && p.MIC != "MISX" {
				displayName = fmt.Sprintf("%s@%s", p.Ticker, p.MIC)
			}
		}

		rowBg := tcell.ColorBlack
		if row%2 == 0 {
			rowBg = tcell.ColorDarkGray
		}

		app.portfolioView.TabbedView.PositionsTable.SetCell(rowNum, 0, tview.NewTableCell(displayName).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightYellow)).SetAlign(tview.AlignLeft))
		app.portfolioView.TabbedView.PositionsTable.SetCell(rowNum, 1, tview.NewTableCell(displayQty).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.PositionsTable.SetCell(rowNum, 2, tview.NewTableCell(p.AveragePrice).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.PositionsTable.SetCell(rowNum, 3, tview.NewTableCell(p.CurrentPrice).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightCyan)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.PositionsTable.SetCell(rowNum, 4, tview.NewTableCell(dailyPnL).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(dailyColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.PositionsTable.SetCell(rowNum, 5, tview.NewTableCell(totalValue).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightGreen)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.PositionsTable.SetCell(rowNum, 6, tview.NewTableCell(unrealizedPnL).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(unrealColor)).SetAlign(tview.AlignRight))
	}

	if len(pos) == 0 {
		app.portfolioView.TabbedView.PositionsTable.SetCell(1, 0, tview.NewTableCell("No open positions").
			SetSelectable(false).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorGray))
	}
}

// updateHistoryTable refreshes the trade history table
func updateHistoryTable(app *App) {
	app.portfolioView.TabbedView.HistoryTable.Clear()

	headers := []string{"Instrument", "Side", "Price", "Qty (Lots)", "Total", "Time"}
	headerStyle := tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite).
		Bold(true)

	for i, h := range headers {
		align := tview.AlignRight
		if i == 0 {
			align = tview.AlignLeft
		}
		cell := tview.NewTableCell(h).
			SetStyle(headerStyle).
			SetAlign(align).
			SetExpansion(1)
		app.portfolioView.TabbedView.HistoryTable.SetCell(0, i, cell)
	}

	app.dataMutex.RLock()
	if app.selectedIdx < 0 || app.selectedIdx >= len(app.accounts) {
		app.dataMutex.RUnlock()
		return
	}
	accountID := app.accounts[app.selectedIdx].ID
	history := app.history[accountID]
	app.dataMutex.RUnlock()

	for row, t := range history {
		rowNum := row + 1
		rowBg := tcell.ColorBlack
		if row%2 == 0 {
			rowBg = tcell.ColorDarkGray
		}

		sideColor := tcell.ColorWhite
		if t.Side == "Buy" {
			sideColor = tcell.ColorGreen
		} else if t.Side == "Sell" {
			sideColor = tcell.ColorRed
		}

		timeStr := t.Timestamp.Format("01-02 15:04")

		// Convert quantity to lots
		var lotSize float64
		if app.client != nil {
			lotSize = app.client.GetLotSize(t.Symbol)
		}
		displayQty := displayLots(t.Quantity, lotSize)

		tradeDisplayName := t.Name
		if tradeDisplayName == "" {
			tradeDisplayName = t.Symbol
		}

		app.portfolioView.TabbedView.HistoryTable.SetCell(rowNum, 0, tview.NewTableCell(tradeDisplayName).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightYellow)).SetAlign(tview.AlignLeft))
		app.portfolioView.TabbedView.HistoryTable.SetCell(rowNum, 1, tview.NewTableCell(t.Side).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(sideColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.HistoryTable.SetCell(rowNum, 2, tview.NewTableCell(t.Price).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.HistoryTable.SetCell(rowNum, 3, tview.NewTableCell(displayQty).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.HistoryTable.SetCell(rowNum, 4, tview.NewTableCell(t.Total).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightGreen)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.HistoryTable.SetCell(rowNum, 5, tview.NewTableCell(timeStr).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
	}

	if len(history) == 0 {
		app.portfolioView.TabbedView.HistoryTable.SetCell(1, 0, tview.NewTableCell("No trade history found").
			SetSelectable(false).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorGray))
	}
}

// updateOrdersTable refreshes the active orders table
func updateOrdersTable(app *App) {
	app.portfolioView.TabbedView.OrdersTable.Clear()

	headers := []string{"Instrument", "Side", "Type", "Status", "Qty (Lots)", "Price", "Time"}
	headerStyle := tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite).
		Bold(true)

	for i, h := range headers {
		align := tview.AlignRight
		if i == 0 {
			align = tview.AlignLeft
		}
		cell := tview.NewTableCell(h).
			SetStyle(headerStyle).
			SetAlign(align).
			SetExpansion(1)
		app.portfolioView.TabbedView.OrdersTable.SetCell(0, i, cell)
	}

	app.dataMutex.RLock()
	if app.selectedIdx < 0 || app.selectedIdx >= len(app.accounts) {
		app.dataMutex.RUnlock()
		return
	}
	accountID := app.accounts[app.selectedIdx].ID
	orders := app.activeOrders[accountID]
	app.dataMutex.RUnlock()

	for row, o := range orders {
		rowNum := row + 1
		rowBg := tcell.ColorBlack
		if row%2 == 0 {
			rowBg = tcell.ColorDarkGray
		}

		sideColor := tcell.ColorWhite
		if o.Side == "Buy" {
			sideColor = tcell.ColorGreen
		} else if o.Side == "Sell" {
			sideColor = tcell.ColorRed
		}

		timeStr := o.CreationTime.Format("01-02 15:04")

		orderDisplayName := o.Name
		if orderDisplayName == "" {
			orderDisplayName = o.Symbol
		}

		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 0, tview.NewTableCell(orderDisplayName).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightYellow)).SetAlign(tview.AlignLeft))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 1, tview.NewTableCell(o.Side).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(sideColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 2, tview.NewTableCell(o.Type).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		// Convert quantity to lots
		var lotSize float64
		if app.client != nil {
			lotSize = app.client.GetLotSize(o.Symbol)
		}
		displayQty := displayLots(o.Quantity, lotSize)

		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 3, tview.NewTableCell(o.Status).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightCyan)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 4, tview.NewTableCell(displayQty).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 5, tview.NewTableCell(o.Price).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 6, tview.NewTableCell(timeStr).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
	}

	if len(orders) == 0 {
		app.portfolioView.TabbedView.OrdersTable.SetCell(1, 0, tview.NewTableCell("No active orders").
			SetSelectable(false).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorGray))
	}
}

// updateInfoPanel refreshes the info panel
func updateInfoPanel(app *App) {
	app.dataMutex.RLock()
	accountID := app.accounts[app.selectedIdx].ID
	acc := app.accounts[app.selectedIdx]
	pos := app.positions[accountID]
	app.dataMutex.RUnlock()

	var totalValue float64
	var totalPnL float64

	for _, p := range pos {
		if qty, err := parseFloat(p.Quantity); err == nil {
			if price, err := parseFloat(p.CurrentPrice); err == nil {
				totalValue += qty * price
			}
		}
		if val, err := parseFloat(p.DailyPnL); err == nil {
			totalPnL += val
		}
	}

	app.portfolioView.UpdateSummary(acc)
}

// updateStatusBar refreshes the status bar
func updateStatusBar(app *App) {
	now := time.Now().Format("15:04:05")
	app.dataMutex.RLock()

	var accountID string
	var count int

	if app.selectedIdx >= 0 && app.selectedIdx < len(app.accounts) {
		accountID = app.accounts[app.selectedIdx].ID
		count = len(app.positions[accountID])
	} else {
		accountID = "N/A"
		count = 0
	}

	statusMsg := app.statusMessage
	statusType := app.statusType
	app.dataMutex.RUnlock()

	var statusText string
	switch statusType {
	case StatusLoading:
		if statusMsg == "" {
			statusMsg = "Updating..."
		}
		statusText = fmt.Sprintf("[yellow]%s[white]", statusMsg)
	case StatusSuccess:
		statusText = fmt.Sprintf("[green]%s[white]", statusMsg)
	case StatusError:
		statusText = fmt.Sprintf("[red]%s[white]", statusMsg)
	default:
		statusText = statusMsg
	}

	if statusText != "" {
		statusText = " | " + statusText
	}

	shortcuts := "[yellow]F2[white] Refresh [yellow]Tab[white] Switch Area [yellow]←/→[white] Tabs [yellow]q[white] Quit"
	// Check if TabbedView.PositionsTable is active and focused
	if app.portfolioView.TabbedView.ActiveTab == TabPositions &&
		app.app.GetFocus() == app.portfolioView.TabbedView.PositionsTable {
		shortcuts += " | [yellow]A[white] Buy [yellow]C[white] Close"
	}

	app.statusBar.SetDynamicColors(true)
	// Use colors for keys: Yellow for keys, White for description.
	app.statusBar.SetText(fmt.Sprintf(" %s | %s | Acc: %s | Pos: %d%s ",
		shortcuts, now, maskAccountID(accountID), count, statusText))
}

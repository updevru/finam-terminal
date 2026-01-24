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
	app.portfolioView.PositionsTable.Clear()

	headers := []string{"Symbol", "Qty", "AvgPrice", "Current", "Daily P&L", "", "Unreal P&L"}
	headerStyle := tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite).
		Bold(true)

	for i, h := range headers {
		align := tview.AlignCenter
		if i == 0 {
			align = tview.AlignLeft
		} else if h == "" {
			align = tview.AlignRight
		}
		cell := tview.NewTableCell(h).
			SetStyle(headerStyle).
			SetAlign(align).
			SetExpansion(1)
		app.portfolioView.PositionsTable.SetCell(0, i, cell)
	}

	app.dataMutex.RLock()
	accountID := app.accounts[app.selectedIdx].ID
	pos := app.positions[accountID]
	q := app.quotes[accountID]
	app.dataMutex.RUnlock()

	for row, p := range pos {
		quote := q[p.Symbol]
		rowNum := row + 1

		totalValue := "N/A"
		if quote != nil && quote.Last != "N/A" {
			qty, _ := parseFloat(p.Quantity)
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

		symbol := p.Ticker
		if p.MIC != "" && p.MIC != "MISX" {
			symbol = fmt.Sprintf("%s@%s", p.Ticker, p.MIC)
		}

		rowBg := tcell.ColorBlack
		if row%2 == 0 {
			rowBg = tcell.ColorDarkGray
		}

		app.portfolioView.PositionsTable.SetCell(rowNum, 0, tview.NewTableCell(symbol).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightYellow)).SetAlign(tview.AlignLeft))
		app.portfolioView.PositionsTable.SetCell(rowNum, 1, tview.NewTableCell(p.Quantity).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.PositionsTable.SetCell(rowNum, 2, tview.NewTableCell(p.AveragePrice).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.PositionsTable.SetCell(rowNum, 3, tview.NewTableCell(p.CurrentPrice).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightCyan)).SetAlign(tview.AlignRight))
		app.portfolioView.PositionsTable.SetCell(rowNum, 4, tview.NewTableCell(dailyPnL).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(dailyColor)).SetAlign(tview.AlignRight))
		app.portfolioView.PositionsTable.SetCell(rowNum, 5, tview.NewTableCell(totalValue).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorLightGreen)).SetAlign(tview.AlignRight))
		app.portfolioView.PositionsTable.SetCell(rowNum, 6, tview.NewTableCell(unrealizedPnL).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(unrealColor)).SetAlign(tview.AlignRight))
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

	shortcuts := "[yellow]F2[white] Refresh [yellow]q[white] Quit"
	// Check if PositionsTable is focused
	if app.app.GetFocus() == app.portfolioView.PositionsTable {
		shortcuts += " | [yellow]A[white] New Order [yellow]C[white] Close Pos"
	}

	app.statusBar.SetDynamicColors(true)
	// Use colors for keys: Yellow for keys, White for description.
	app.statusBar.SetText(fmt.Sprintf(" %s | %s | Acc: %s | Pos: %d%s ",
		shortcuts, now, maskAccountID(accountID), count, statusText))
}

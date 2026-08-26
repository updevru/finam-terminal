package ui

import (
	"fmt"
	"strings"
	"time"

	"finam-terminal/models"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// updateAccountList refreshes the account list using two rows per account.
// Row 0: Account ID
// Row 1: Equity (formatted) + PnL (with sign)
// Both rows of the selected account are highlighted with a background color.
func updateAccountList(app *App) {
	app.portfolioView.AccountTable.Clear()

	for i, acc := range app.accounts {
		idRow := accountIdxToRow(i)
		dataRow := idRow + 1
		isSelected := i == app.selectedIdx

		bg := tcell.ColorBlack
		if isSelected {
			bg = tcell.ColorDarkSlateGray
		}

		if acc.LoadError != "" {
			idCell := tview.NewTableCell(acc.ID).
				SetStyle(tcell.StyleDefault.Background(bg).Foreground(tcell.ColorWhite)).
				SetExpansion(1)
			idCell.Transparent = false
			app.portfolioView.AccountTable.SetCell(idRow, 0, idCell)

			errCell := tview.NewTableCell("[error]").
				SetStyle(tcell.StyleDefault.Background(bg).Foreground(tcell.ColorRed)).
				SetExpansion(1)
			errCell.Transparent = false
			app.portfolioView.AccountTable.SetCell(dataRow, 0, errCell)
			continue
		}

		// Row 0: Account ID
		idCell := tview.NewTableCell(acc.ID).
			SetStyle(tcell.StyleDefault.Background(bg).Foreground(tcell.ColorWhite)).
			SetExpansion(1)
		idCell.Transparent = false
		app.portfolioView.AccountTable.SetCell(idRow, 0, idCell)

		// Row 1: Equity + Daily PnL (sum of position DailyPnL)
		equity := "—"
		if val, err := parseFloat(acc.Equity); err == nil {
			equity = formatNumber(val, 2)
		}

		app.dataMutex.RLock()
		var dailyTotal float64
		for _, p := range app.positions[acc.ID] {
			if val, err := parseFloat(p.DailyPnL); err == nil {
				dailyTotal += val
			}
		}
		app.dataMutex.RUnlock()

		pnlText := "0.00"
		pnlTag := "gray"
		if dailyTotal > 0 {
			pnlText = "+" + formatNumber(dailyTotal, 2)
			pnlTag = "green"
		} else if dailyTotal < 0 {
			pnlText = formatNumber(dailyTotal, 2)
			pnlTag = "red"
		}

		dataText := fmt.Sprintf("%s  [%s]%s[-]", equity, pnlTag, pnlText)
		dataCell := tview.NewTableCell(dataText).
			SetStyle(tcell.StyleDefault.Background(bg).Foreground(tcell.ColorWhite)).
			SetExpansion(1)
		dataCell.Transparent = false
		app.portfolioView.AccountTable.SetCell(dataRow, 0, dataCell)
	}

	// Select the ID row of the active account
	if len(app.accounts) > 0 {
		app.portfolioView.AccountTable.Select(accountIdxToRow(app.selectedIdx), 0)
	}
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
		acc := app.accounts[app.selectedIdx]
		if acc.LoadError != "" {
			app.portfolioView.TabbedView.PositionsTable.SetCell(1, 0, tview.NewTableCell("Broker error: "+acc.LoadError).
				SetSelectable(false).
				SetAlign(tview.AlignCenter).
				SetTextColor(tcell.ColorRed))
		} else {
			app.portfolioView.TabbedView.PositionsTable.SetCell(1, 0, tview.NewTableCell("No open positions").
				SetSelectable(false).
				SetAlign(tview.AlignCenter).
				SetTextColor(tcell.ColorGray))
		}
	}
}

// updateHistoryTable refreshes the trade history table
func updateHistoryTable(app *App) {
	app.portfolioView.TabbedView.HistoryTable.Clear()

	headers := []string{"Instrument", "Side", "Price", "Qty (Lots)", "Total", "НКД", "Time"}
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
		switch t.Side {
		case "Buy":
			sideColor = tcell.ColorGreen
		case "Sell":
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
		// Accrued interest (bonds only) combined with its currency; blank for equities.
		app.portfolioView.TabbedView.HistoryTable.SetCell(rowNum, 5, tview.NewTableCell(formatAccruedInterest(t)).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.HistoryTable.SetCell(rowNum, 6, tview.NewTableCell(timeStr).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(tcell.ColorWhite)).SetAlign(tview.AlignRight))
	}

	if len(history) == 0 {
		app.portfolioView.TabbedView.HistoryTable.SetCell(1, 0, tview.NewTableCell("No trade history found").
			SetSelectable(false).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorGray))
	}
}

// orderRow is one display row of the orders table: an order plus whether it is
// the triggered child of a parent rendered immediately above it.
type orderRow struct {
	order   models.Order
	isChild bool
}

// groupTriggeredOrders reorders the active orders so that, when a parent stop
// order and the exchange order it triggered are BOTH present in the set, the
// triggered child is placed directly after its parent and flagged isChild.
// Orders whose counterpart is absent keep their original position and are not
// marked. Original relative order is otherwise preserved.
func groupTriggeredOrders(orders []models.Order) []orderRow {
	// Index present order IDs.
	idToIndex := make(map[string]int, len(orders))
	for i, o := range orders {
		if o.ID != "" {
			idToIndex[o.ID] = i
		}
	}

	// parentToChild[parentIdx] = childIdx when the parent's TriggeredOrderID
	// resolves to another order present in the set; childGrouped marks those
	// children so they are emitted under their parent rather than at top level.
	parentToChild := make(map[int]int, len(orders))
	childGrouped := make(map[int]bool, len(orders))
	for i, o := range orders {
		if o.TriggeredOrderID == "" {
			continue
		}
		if childIdx, ok := idToIndex[o.TriggeredOrderID]; ok && childIdx != i {
			parentToChild[i] = childIdx
			childGrouped[childIdx] = true
		}
	}

	rows := make([]orderRow, 0, len(orders))
	emitted := make([]bool, len(orders))
	for i := range orders {
		if emitted[i] || childGrouped[i] {
			continue // grouped children are emitted under their parent
		}
		// Emit this order, then follow its chain of present triggered children.
		rows = append(rows, orderRow{order: orders[i]})
		emitted[i] = true
		for cur := i; ; {
			childIdx, ok := parentToChild[cur]
			if !ok || emitted[childIdx] {
				break
			}
			rows = append(rows, orderRow{order: orders[childIdx], isChild: true})
			emitted[childIdx] = true
			cur = childIdx
		}
	}
	// Safety net for any leftovers (e.g. cycles): emit in original order.
	for i := range orders {
		if !emitted[i] {
			rows = append(rows, orderRow{order: orders[i], isChild: childGrouped[i]})
			emitted[i] = true
		}
	}
	return rows
}

// updateOrdersTable refreshes the active orders table
func updateOrdersTable(app *App) {
	app.portfolioView.TabbedView.OrdersTable.Clear()

	headers := []string{"Instrument", "Side", "Type", "Status", "Qty", "Executed", "Price/Condition", "Time"}
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

	// Group parent→triggered links: when a stop order and the exchange order it
	// spawned are both present in the current set, render the child directly
	// under its parent (indented, with a ↳ marker) so the link is unambiguous.
	// Orders whose counterpart is absent are shown normally, without a marker.
	orderedRows := groupTriggeredOrders(orders)

	for row, r := range orderedRows {
		o := r.order
		rowNum := row + 1
		rowBg := tcell.ColorBlack
		if row%2 == 0 {
			rowBg = tcell.ColorDarkGray
		}

		// Dim non-cancellable orders
		isCancellable := isOrderCancellable(o.Status)
		fgColor := tcell.ColorWhite
		if !isCancellable {
			fgColor = tcell.ColorDimGray
		}

		sideColor := fgColor
		switch o.Side {
		case "Buy":
			if isCancellable {
				sideColor = tcell.ColorGreen
			}
		case "Sell":
			if isCancellable {
				sideColor = tcell.ColorRed
			}
		}

		timeStr := "—"
		if !o.CreationTime.IsZero() {
			timeStr = o.CreationTime.Format("01-02 15:04")
		}

		orderDisplayName := o.Name
		if orderDisplayName == "" {
			orderDisplayName = o.Symbol
		}

		// A triggered child is rendered indented under its parent with a ↳ marker,
		// making the parent→child link unambiguous by adjacency.
		if r.isChild {
			orderDisplayName = "  ↳ " + orderDisplayName
		}

		// Build Price/Condition display
		priceCondition := formatOrderPriceCondition(o)

		// Convert quantity to lots for display
		var lotSize float64
		if app.client != nil {
			lotSize = app.client.GetLotSize(o.Symbol)
		}
		displayQty := displayLots(o.Quantity, lotSize)

		// Executed quantity in lots
		executedDisplay := displayLots(o.ExecutedQty, lotSize)
		if executedDisplay == "N/A" {
			executedDisplay = ""
		}

		nameColor := tcell.ColorLightYellow
		if !isCancellable {
			nameColor = tcell.ColorDimGray
		}
		statusColor := tcell.ColorLightCyan
		if !isCancellable {
			statusColor = tcell.ColorDimGray
		}

		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 0, tview.NewTableCell(orderDisplayName).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(nameColor)).SetAlign(tview.AlignLeft))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 1, tview.NewTableCell(o.Side).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(sideColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 2, tview.NewTableCell(o.Type).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(fgColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 3, tview.NewTableCell(o.Status).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(statusColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 4, tview.NewTableCell(displayQty).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(fgColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 5, tview.NewTableCell(executedDisplay).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(fgColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 6, tview.NewTableCell(priceCondition).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(fgColor)).SetAlign(tview.AlignRight))
		app.portfolioView.TabbedView.OrdersTable.SetCell(rowNum, 7, tview.NewTableCell(timeStr).
			SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(fgColor)).SetAlign(tview.AlignRight))
	}

	if len(orders) == 0 {
		app.portfolioView.TabbedView.OrdersTable.SetCell(1, 0, tview.NewTableCell("No active orders").
			SetSelectable(false).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorGray))
	}
}

// formatOrderPriceCondition builds a display string for the Price/Condition column.
// Non-GTC validity is appended in parentheses, e.g. "SL: 100.50 ↓ (Day)".
// formatAccruedInterest renders the combined НКД cell as "<amount> <currency>"
// (e.g. "12.34 RUB"). Returns an empty string for trades without accrued interest
// (non-bond instruments), so the column stays blank for them.
func formatAccruedInterest(t models.Trade) string {
	if t.AccruedInterest == "" || t.AccruedInterest == "N/A" {
		return ""
	}
	if t.Currency == "" {
		return t.AccruedInterest
	}
	return t.AccruedInterest + " " + t.Currency
}

func formatOrderPriceCondition(o models.Order) string {
	var result string
	switch o.Type {
	case "SL/TP":
		var parts []string
		if o.SLPrice != "" && o.SLPrice != "N/A" && o.SLPrice != "0" {
			parts = append(parts, "SL:"+o.SLPrice)
		}
		if o.TPPrice != "" && o.TPPrice != "N/A" && o.TPPrice != "0" {
			parts = append(parts, "TP:"+o.TPPrice)
		}
		if len(parts) > 0 {
			result = strings.Join(parts, " / ")
		} else {
			result = o.Price
		}
	case "Stop":
		arrow := ""
		switch o.StopCondition {
		case "Last Down":
			arrow = " ↓"
		case "Last Up":
			arrow = " ↑"
		}
		result = "SL: " + o.StopPrice + arrow
	case "Stop-Limit":
		result = "Stop: " + o.StopPrice + " Lim: " + o.LimitPrice
	case "Limit":
		result = o.LimitPrice
	default:
		result = o.Price
	}

	// Append non-GTC validity
	if o.Validity != "" && o.Validity != "GTC" {
		result += " (" + o.Validity + ")"
	}
	return result
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

	var shortcuts string
	if app.profileOpen {
		shortcuts = "[yellow]1-4[white] Timeframe  [yellow]A[white] Order  [yellow]R[white] Refresh  [yellow]ESC[white] Back"
	} else {
		shortcuts = "[yellow]F2[white] Refresh [yellow]Tab[white] Switch Area [yellow]←/→[white] Tabs [yellow]q[white] Quit"
		// Check if TabbedView.PositionsTable is active and focused
		if app.portfolioView.TabbedView.ActiveTab == TabPositions &&
			app.app.GetFocus() == app.portfolioView.TabbedView.PositionsTable {
			shortcuts += " | [yellow]A[white] Buy [yellow]C[white] Close"
		}
		// Check if TabbedView.OrdersTable is active and focused
		if app.portfolioView.TabbedView.ActiveTab == TabOrders &&
			app.app.GetFocus() == app.portfolioView.TabbedView.OrdersTable {
			shortcuts += " | [yellow]X[white] Cancel [yellow]E[white] Modify [yellow]R[white] Refresh"
		}
		// Check if TabbedView.IndexTable is active and focused
		if app.portfolioView.TabbedView.ActiveTab == TabIndex &&
			app.app.GetFocus() == app.portfolioView.TabbedView.IndexTable {
			shortcuts += " | [yellow]Enter[white] Profile [yellow]A[white] Buy [yellow]R[white] Refresh"
		}
	}

	app.statusBar.SetDynamicColors(true)
	// Use colors for keys: Yellow for keys, White for description.
	app.statusBar.SetText(fmt.Sprintf(" %s | %s | Acc: %s | Pos: %d%s ",
		shortcuts, now, maskAccountID(accountID), count, statusText))
}

// indexRowValues holds the rendered numbers of one Index tab row, already
// formatted and coloured. Keeping the arithmetic in a pure function keeps the
// division guard (a zero close) testable and out of the drawing code.
type indexRowValues struct {
	price     string
	change    string
	changePct string
	volume    string
	colour    tcell.Color
}

// noIndexData is what an empty cell shows: a dash, never "N/A" or a zero that
// could be mistaken for a real flat price.
const noIndexData = "—"

// indexRowFromQuote renders one row's numbers.
//
// The session change comes straight from the broker (Quote.change = last -
// close), so the previous close is derived as last - change rather than read
// from a field that incremental stream updates may omit. A zero close means the
// percentage is undefined — it renders as a dash, never NaN or Inf.
func indexRowFromQuote(q *models.Quote) indexRowValues {
	row := indexRowValues{
		price:     noIndexData,
		change:    noIndexData,
		changePct: noIndexData,
		volume:    noIndexData,
		colour:    tcell.ColorWhite,
	}
	if q == nil {
		return row
	}

	if last, err := parseFloat(q.Last); err == nil {
		row.price = formatNumber(last, 2)

		if change, err := parseFloat(q.Change); err == nil {
			row.change = formatSignedNumber(change, 2)
			switch {
			case change > 0:
				row.colour = tcell.ColorGreen
			case change < 0:
				row.colour = tcell.ColorRed
			}

			if closePrice := last - change; closePrice != 0 {
				row.changePct = formatSignedNumber(change/closePrice*100, 2) + "%"
			}
		}
	}

	if volume, err := parseFloat(q.Volume); err == nil {
		row.volume = formatNumber(volume, 0)
	}

	return row
}

// formatSignedNumber formats a value with an explicit + for a rise. Zero stays
// unsigned, so a flat session does not read as a gain.
func formatSignedNumber(val float64, decimals int) string {
	s := formatNumber(val, decimals)
	if val > 0 {
		return "+" + s
	}
	return s
}

// updateIndexTable draws the Index tab from state already in memory. It never
// calls the API: the composition is loaded by loadIndexAsync and the quotes
// arrive from the stream or the fallback batch.
func updateIndexTable(app *App) {
	table := app.portfolioView.TabbedView.IndexTable
	table.Clear()
	table.SetTitle(fmt.Sprintf(" %s ", activeIndex().Name))

	headers := []string{"Ticker", "Name", "Price", "Chg", "Chg%", "Weight", "Volume"}
	headerStyle := tcell.StyleDefault.
		Background(tcell.ColorDarkBlue).
		Foreground(tcell.ColorWhite).
		Bold(true)

	for i, h := range headers {
		align := tview.AlignRight
		if i <= 1 {
			align = tview.AlignLeft
		}
		table.SetCell(0, i, tview.NewTableCell(h).
			SetStyle(headerStyle).
			SetAlign(align).
			SetExpansion(1))
	}

	app.dataMutex.RLock()
	constituents := sortConstituents(app.indexConstituents)
	loading, loadErr := app.indexLoading, app.indexLoadErr
	quotes := make(map[string]*models.Quote, len(app.indexQuotes))
	for symbol, q := range app.indexQuotes {
		quotes[symbol] = q
	}
	app.dataMutex.RUnlock()

	if len(constituents) == 0 {
		message, colour := "No constituents", tcell.ColorGray
		switch {
		case loading:
			message, colour = "Loading index constituents...", tcell.ColorYellow
		case loadErr != "":
			message, colour = fmt.Sprintf("%s — press R to retry", loadErr), tcell.ColorRed
		}
		table.SetCell(1, 0, tview.NewTableCell(message).
			SetSelectable(false).
			SetAlign(tview.AlignLeft).
			SetTextColor(colour))
		return
	}

	for i, c := range constituents {
		row := i + 1
		rowBg := tcell.ColorBlack
		if i%2 == 0 {
			rowBg = tcell.ColorDarkGray
		}

		values := indexRowFromQuote(quotes[c.Symbol])

		// The weight is shown raw: the API's normalisation is undocumented (the
		// values do not sum to 1), so rendering it as a percentage would state
		// something we cannot verify. It is reliable for ordering, which is what
		// the column is really for.
		weight := noIndexData
		if c.Weight != 0 {
			weight = formatNumber(c.Weight, 4)
		}

		cells := []struct {
			text   string
			colour tcell.Color
			align  int
		}{
			{c.Ticker, tcell.ColorLightYellow, tview.AlignLeft},
			{c.Name, tcell.ColorWhite, tview.AlignLeft},
			{values.price, tcell.ColorWhite, tview.AlignRight},
			{values.change, values.colour, tview.AlignRight},
			{values.changePct, values.colour, tview.AlignRight},
			{weight, tcell.ColorLightGray, tview.AlignRight},
			{values.volume, tcell.ColorWhite, tview.AlignRight},
		}

		for col, cell := range cells {
			table.SetCell(row, col, tview.NewTableCell(cell.text).
				SetStyle(tcell.StyleDefault.Background(rowBg).Foreground(cell.colour)).
				SetAlign(cell.align))
		}
	}
}

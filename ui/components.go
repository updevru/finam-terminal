package ui

import (
	"finam-terminal/models"
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PortfolioView encapsulates the UI components for the portfolio screen
type PortfolioView struct {
	Layout       *tview.Flex
	AccountList  *tview.List // Keeping for now if needed for navigation or legacy
	AccountTable *tview.Table
	TabbedView   *TabbedView
	SummaryArea  *tview.TextView
}

// TabType represents the type of tab in the tabbed view
type TabType int

const (
	TabPositions TabType = iota
	TabHistory
	TabOrders
)

// TabbedView manages a tabbed interface for positions, history, and orders
type TabbedView struct {
	*tview.Flex
	ActiveTab TabType

	PositionsTable *tview.Table
	HistoryTable   *tview.Table
	OrdersTable    *tview.Table
	Content        *tview.Pages // To switch between tables
	Header         *tview.TextView
}

// NewPortfolioView creates a new PortfolioView component
func NewPortfolioView(app *tview.Application) *PortfolioView {
	pv := &PortfolioView{
		AccountList:  createAccountList(),
		AccountTable: createAccountTable(),
		TabbedView:   NewTabbedView(),
		SummaryArea:  createInfoLabel(),
	}

	topFlex := tview.NewFlex().
		AddItem(pv.AccountTable, 30, 1, true).
		AddItem(pv.TabbedView, 0, 1, false)

	pv.Layout = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, true).
		AddItem(pv.SummaryArea, 8, 1, false)

	return pv
}

// NewTabbedView creates a new TabbedView component
func NewTabbedView() *TabbedView {
	tv := &TabbedView{
		Flex:           tview.NewFlex().SetDirection(tview.FlexRow),
		ActiveTab:      TabPositions,
		PositionsTable: createPositionsTable(),
		HistoryTable:   createHistoryTable(),
		OrdersTable:    createOrdersTable(),
		Content:        tview.NewPages(),
		Header:         tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter),
	}

	tv.Header.SetBackgroundColor(tcell.ColorBlack)

	tv.Content.AddPage("positions", tv.PositionsTable, true, true)
	tv.Content.AddPage("history", tv.HistoryTable, true, false)
	tv.Content.AddPage("orders", tv.OrdersTable, true, false)

	tv.AddItem(tv.Header, 1, 0, false)
	tv.AddItem(tv.Content, 0, 1, true)

	tv.UpdateHeader()
	return tv
}

// UpdateHeader updates the visual representation of tabs
func (tv *TabbedView) UpdateHeader() {
	tabs := []string{" Positions ", " History ", " Orders "}
	headerText := ""
	for i, tab := range tabs {
		if TabType(i) == tv.ActiveTab {
			headerText += fmt.Sprintf("[black:yellow]%s[-]", tab)
		} else {
			headerText += fmt.Sprintf("[white:black]%s[-]", tab)
		}
		if i < len(tabs)-1 {
			headerText += " "
		}
	}
	tv.Header.SetText(headerText)
}

// SetTab switches the active tab
func (tv *TabbedView) SetTab(tab TabType) {
	tv.ActiveTab = tab
	switch tab {
	case TabPositions:
		tv.Content.SwitchToPage("positions")
	case TabHistory:
		tv.Content.SwitchToPage("history")
	case TabOrders:
		tv.Content.SwitchToPage("orders")
	}
	tv.UpdateHeader()
}

// UpdateAccounts populates the account table
func (pv *PortfolioView) UpdateAccounts(accounts []models.AccountInfo) {
	pv.AccountTable.Clear()

	headers := []string{"ID", "Type", "Equity"}
	for i, h := range headers {
		pv.AccountTable.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false))
	}

	for i, acc := range accounts {
		pv.AccountTable.SetCell(i+1, 0, tview.NewTableCell(acc.ID).SetTextColor(tcell.ColorWhite))
		pv.AccountTable.SetCell(i+1, 1, tview.NewTableCell(acc.Type).SetTextColor(tcell.ColorWhite))
		pv.AccountTable.SetCell(i+1, 2, tview.NewTableCell(acc.Equity).SetTextColor(tcell.ColorWhite))
	}
}

// UpdateSummary updates the account summary area
func (pv *PortfolioView) UpdateSummary(acc models.AccountInfo) {
	pv.SummaryArea.Clear()

	equity := acc.Equity
	if val, err := strconv.ParseFloat(equity, 64); err == nil {
		equity = fmt.Sprintf("%.2f", val)
	}

	pnl := acc.UnrealizedPnL
	if val, err := strconv.ParseFloat(pnl, 64); err == nil {
		pnl = fmt.Sprintf("%.2f", val)
	}

	_, _ = fmt.Fprintf(pv.SummaryArea, " Account ID: %s\n", acc.ID)
	_, _ = fmt.Fprintf(pv.SummaryArea, " Type:       %s\n", acc.Type)
	_, _ = fmt.Fprintf(pv.SummaryArea, " Status:     %s\n", acc.Status)
	_, _ = fmt.Fprintf(pv.SummaryArea, " Equity:     %s\n", equity)
	_, _ = fmt.Fprintf(pv.SummaryArea, " Total PnL:  %s\n", pnl)
}

// UpdatePositions populates the positions table
func (pv *PortfolioView) UpdatePositions(positions []models.Position) {
	pv.TabbedView.PositionsTable.Clear()

	headers := []string{"Symbol", "Qty", "Avg Price", "Cur Price", "Unreal PnL"}
	for i, h := range headers {
		pv.TabbedView.PositionsTable.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false))
	}

	for i, pos := range positions {
		pv.TabbedView.PositionsTable.SetCell(i+1, 0, tview.NewTableCell(pos.Symbol).SetTextColor(tcell.ColorWhite))
		pv.TabbedView.PositionsTable.SetCell(i+1, 1, tview.NewTableCell(pos.Quantity).SetTextColor(tcell.ColorWhite))
		pv.TabbedView.PositionsTable.SetCell(i+1, 2, tview.NewTableCell(pos.AveragePrice).SetTextColor(tcell.ColorWhite))
		pv.TabbedView.PositionsTable.SetCell(i+1, 3, tview.NewTableCell(pos.CurrentPrice).SetTextColor(tcell.ColorWhite))
		pv.TabbedView.PositionsTable.SetCell(i+1, 4, tview.NewTableCell(pos.UnrealizedPnL).SetTextColor(tcell.ColorWhite))
	}
}

// createAccountTable creates the account table
func createAccountTable() *tview.Table {
	table := tview.NewTable().
		SetFixed(1, 0).
		SetSelectable(true, false)
	table.SetBorder(true).SetTitle(" Accounts ")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack))
	return table
}

// createHeader creates the application header
func createHeader() *tview.TextView {
	h := tview.NewTextView().
		SetText(fmt.Sprintf(" Finam Terminal v%s ", appVersion)).
		SetTextAlign(tview.AlignCenter)
	h.SetBackgroundColor(tcell.ColorDarkCyan)
	h.SetTextColor(tcell.ColorWhite)
	return h
}

// createAccountList creates the account list panel
func createAccountList() *tview.List {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Accounts ")
	list.SetBackgroundColor(tcell.ColorBlack)
	return list
}

// createPositionsTable creates the positions table
func createPositionsTable() *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" Positions ")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetSelectable(true, false)
	return table
}

// createHistoryTable creates the history table
func createHistoryTable() *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" History ")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetSelectable(true, false)
	table.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack))
	return table
}

// createOrdersTable creates the orders table
func createOrdersTable() *tview.Table {
	table := tview.NewTable()
	table.SetBorder(true)
	table.SetTitle(" Orders ")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.SetSelectable(true, false)
	table.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack))
	return table
}

// createInfoLabel creates the info panel
func createInfoLabel() *tview.TextView {
	label := tview.NewTextView()
	label.SetBorder(true)
	label.SetTitle(" Info ")
	label.SetBackgroundColor(tcell.ColorBlack)
	label.SetTextColor(tcell.ColorLightGray)
	return label
}

// createStatusBar creates the status bar
func createStatusBar() *tview.TextView {
	bar := tview.NewTextView()
	bar.SetTextAlign(tview.AlignLeft)
	bar.SetBackgroundColor(tcell.ColorDarkSlateGray)
	bar.SetTextColor(tcell.ColorWhite)
	return bar
}

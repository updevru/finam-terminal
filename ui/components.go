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
	Layout         *tview.Flex
	AccountList    *tview.List // Keeping for now if needed for navigation or legacy
	AccountTable   *tview.Table
	PositionsTable *tview.Table
	SummaryArea    *tview.TextView
}

// NewPortfolioView creates a new PortfolioView component
func NewPortfolioView(app *tview.Application) *PortfolioView {
	pv := &PortfolioView{
		AccountList:    createAccountList(),
		AccountTable:   createAccountTable(),
		PositionsTable: createPositionsTable(),
		SummaryArea:    createInfoLabel(),
	}

	topFlex := tview.NewFlex().
		AddItem(pv.AccountTable, 30, 1, true).
		AddItem(pv.PositionsTable, 0, 1, false)

	pv.Layout = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topFlex, 0, 1, true).
		AddItem(pv.SummaryArea, 8, 1, false)

	return pv
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

	fmt.Fprintf(pv.SummaryArea, " Account ID: %s\n", acc.ID)
	fmt.Fprintf(pv.SummaryArea, " Type:       %s\n", acc.Type)
	fmt.Fprintf(pv.SummaryArea, " Status:     %s\n", acc.Status)
	fmt.Fprintf(pv.SummaryArea, " Equity:     %s\n", equity)
	fmt.Fprintf(pv.SummaryArea, " Total PnL:  %s\n", pnl)
}

// UpdatePositions populates the positions table
func (pv *PortfolioView) UpdatePositions(positions []models.Position) {
	pv.PositionsTable.Clear()

	headers := []string{"Symbol", "Qty", "Avg Price", "Cur Price", "Unreal PnL"}
	for i, h := range headers {
		pv.PositionsTable.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false))
	}

	for i, pos := range positions {
		pv.PositionsTable.SetCell(i+1, 0, tview.NewTableCell(pos.Symbol).SetTextColor(tcell.ColorWhite))
		pv.PositionsTable.SetCell(i+1, 1, tview.NewTableCell(pos.Quantity).SetTextColor(tcell.ColorWhite))
		pv.PositionsTable.SetCell(i+1, 2, tview.NewTableCell(pos.AveragePrice).SetTextColor(tcell.ColorWhite))
		pv.PositionsTable.SetCell(i+1, 3, tview.NewTableCell(pos.CurrentPrice).SetTextColor(tcell.ColorWhite))
		pv.PositionsTable.SetCell(i+1, 4, tview.NewTableCell(pos.UnrealizedPnL).SetTextColor(tcell.ColorWhite))
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
		SetText(fmt.Sprintf(" Finam Trade Terminal v%s ", appVersion)).
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

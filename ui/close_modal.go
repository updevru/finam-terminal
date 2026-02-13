package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ClosePositionModal represents the modal for closing a position
type ClosePositionModal struct {
	Layout   *tview.Flex
	Form     *tview.Form
	Footer   *tview.TextView
	infoArea *tview.TextView
	app      *tview.Application
	callback func(float64)
	onCancel func()
	onError  func(string)

	// Fields
	symbolField   *tview.InputField // Read-only
	quantityField *tview.InputField
	priceField    *tview.InputField // Read-only

	// State
	currentPrice float64
	maxQuantity  float64
	lotSize      float64
}

// NewClosePositionModal creates a new close position modal
func NewClosePositionModal(app *tview.Application, callback func(float64), onCancel func(), onError func(string)) *ClosePositionModal {
	m := &ClosePositionModal{
		Layout:   tview.NewFlex(),
		Form:     tview.NewForm(),
		Footer:   tview.NewTextView(),
		infoArea: tview.NewTextView(),
		app:      app,
		callback: callback,
		onCancel: onCancel,
		onError:  onError,
	}
	m.setupUI()
	return m
}

func (m *ClosePositionModal) setupUI() {
	m.Layout.SetDirection(tview.FlexRow).
		SetBorder(true).
		SetTitle(" Close Position ").
		SetTitleAlign(tview.AlignCenter)

	m.Form.SetBorder(false)
	m.Form.SetBackgroundColor(tcell.ColorBlack)

	// Styling
	m.Form.SetButtonBackgroundColor(tcell.ColorDarkRed).
		SetButtonTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorYellow).
		SetFieldBackgroundColor(tcell.ColorWhite).
		SetFieldTextColor(tcell.ColorBlack)

	// Symbol (Read-only)
	m.symbolField = tview.NewInputField().
		SetLabel("Symbol:       ").
		SetFieldWidth(20).
		SetFieldBackgroundColor(tcell.ColorDarkGray).
		SetFieldTextColor(tcell.ColorWhite)

	// Quantity
	m.quantityField = tview.NewInputField().
		SetLabel("Quantity:     ").
		SetFieldWidth(20)

	// Last Price (Read-only)
	m.priceField = tview.NewInputField().
		SetLabel("Last Price:   ").
		SetFieldWidth(20).
		SetFieldBackgroundColor(tcell.ColorDarkGray).
		SetFieldTextColor(tcell.ColorWhite)

	m.Form.AddFormItem(m.symbolField)
	m.Form.AddFormItem(m.quantityField)
	m.Form.AddFormItem(m.priceField)

	m.Form.AddButton("Close Position", func() {
		if m.Validate() {
			if m.callback != nil {
				m.callback(m.GetQuantity())
			}
		} else {
			if m.onError != nil {
				m.onError(fmt.Sprintf("Invalid quantity. Must be > 0 and <= %v", m.maxQuantity))
			}
		}
	})

	m.Form.AddButton("Cancel", func() {
		if m.onCancel != nil {
			m.onCancel()
		}
	})

	// Info Area (lot info)
	m.infoArea.SetDynamicColors(true)
	m.infoArea.SetBackgroundColor(tcell.ColorBlack)
	m.infoArea.SetTextColor(tcell.ColorLightGray)

	// Footer
	m.Footer.SetBackgroundColor(tcell.ColorDarkSlateGray)
	m.Footer.SetTextColor(tcell.ColorWhite).
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(" [yellow]TAB[white] Move  [yellow]ENTER[white] Execute  [yellow]ESC[white] Cancel")

	m.Layout.AddItem(m.Form, 0, 1, true).
		AddItem(m.infoArea, 2, 0, false).
		AddItem(m.Footer, 1, 0, false)
}

// SetPositionData populates the modal with position details
func (m *ClosePositionModal) SetPositionData(symbol string, quantity float64, price float64, pnl float64) {
	m.symbolField.SetText(symbol)

	// Keep maxQuantity state but leave the field empty for user input as requested
	absQty := quantity
	if absQty < 0 {
		absQty = -absQty
	}

	m.quantityField.SetText("")
	m.priceField.SetText(fmt.Sprintf("%.2f", price))
	m.currentPrice = price
	m.maxQuantity = absQty
}

func (m *ClosePositionModal) GetSymbol() string {
	return m.symbolField.GetText()
}

func (m *ClosePositionModal) GetQuantity() float64 {
	// Allow comma in user input
	val, err := strconv.ParseFloat(strings.ReplaceAll(m.quantityField.GetText(), ",", "."), 64)
	if err != nil {
		return 0
	}
	return val
}

func (m *ClosePositionModal) Validate() bool {
	qty := m.GetQuantity()
	return qty > 0 && qty <= m.maxQuantity
}

// GetLotSize returns the current lot size
func (m *ClosePositionModal) GetLotSize() float64 {
	return m.lotSize
}

// SetPositionDataWithLots populates the modal with position details using lot-based quantities
func (m *ClosePositionModal) SetPositionDataWithLots(symbol string, quantity float64, price float64, pnl float64, lotSize float64) {
	m.symbolField.SetText(symbol)
	m.lotSize = lotSize

	absQty := quantity
	if absQty < 0 {
		absQty = -absQty
	}

	// Convert max quantity to lots
	if lotSize > 0 {
		m.maxQuantity = absQty / lotSize
	} else {
		m.maxQuantity = absQty
	}

	m.quantityField.SetText("")
	m.priceField.SetText(fmt.Sprintf("%.2f", price))
	m.currentPrice = price

	// Update info area
	m.updateInfo()
}

// updateInfo refreshes the info area with lot information
func (m *ClosePositionModal) updateInfo() {
	if m.lotSize <= 0 {
		m.infoArea.SetText("")
		return
	}

	text := fmt.Sprintf(" [yellow]1 lot = %.0f shares[-]  |  Position: %.0f lots", m.lotSize, m.maxQuantity)
	m.infoArea.SetText(text)
}

package ui

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ClosePositionModal represents the modal for closing a position
type ClosePositionModal struct {
	Layout   *tview.Flex
	Form     *tview.Form
	Footer   *tview.TextView
	app      *tview.Application
	callback func(float64)
	onCancel func()

	// Fields
	symbolField    *tview.InputField // Read-only
	quantityField  *tview.InputField
	priceField     *tview.InputField // Read-only

	// State
	currentPrice float64
	maxQuantity  float64
}

// NewClosePositionModal creates a new close position modal
func NewClosePositionModal(app *tview.Application, callback func(float64), onCancel func()) *ClosePositionModal {
	m := &ClosePositionModal{
		Layout:   tview.NewFlex(),
		Form:     tview.NewForm(),
		Footer:   tview.NewTextView(),
		app:      app,
		callback: callback,
		onCancel: onCancel,
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
		SetFieldWidth(20).
		SetAcceptanceFunc(tview.InputFieldInteger)

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
		}
	})

	m.Form.AddButton("Cancel", func() {
		if m.onCancel != nil {
			m.onCancel()
		}
	})

	// Footer
	m.Footer.SetBackgroundColor(tcell.ColorDarkSlateGray)
	m.Footer.SetTextColor(tcell.ColorWhite).
		SetTextAlign(tview.AlignCenter).
		SetText("[TAB] Move  [ENTER] Execute  [ESC] Cancel")

	m.Layout.AddItem(m.Form, 0, 1, true).
		AddItem(m.Footer, 1, 1, false)
}

// SetPositionData populates the modal with position details
func (m *ClosePositionModal) SetPositionData(symbol string, quantity float64, price float64, pnl float64) {
	m.symbolField.SetText(symbol)
	
	// Default to absolute value for display (we always close with a positive qty, direction handled by API)
	absQty := quantity
	if absQty < 0 {
		absQty = -absQty
	}
	
	m.quantityField.SetText(strconv.FormatFloat(absQty, 'f', -1, 64))
	m.priceField.SetText(fmt.Sprintf("%.2f", price))
	m.currentPrice = price
	m.maxQuantity = absQty
}

func (m *ClosePositionModal) GetSymbol() string {
	return m.symbolField.GetText()
}

func (m *ClosePositionModal) GetQuantity() float64 {
	val, err := strconv.ParseFloat(m.quantityField.GetText(), 64)
	if err != nil {
		return 0
	}
	return val
}

func (m *ClosePositionModal) Validate() bool {
	qty := m.GetQuantity()
	return qty > 0 && qty <= m.maxQuantity
}

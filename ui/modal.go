package ui

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// OrderModal represents the order entry modal
type OrderModal struct {
	Layout     *tview.Flex // Main container with border
	Form       *tview.Form
	Footer     *tview.TextView
	app        *tview.Application
	callback   func(string, float64, string)
	onCancel   func()
	
instrument *tview.InputField
	quantity   *tview.InputField
	
	// State
	currentDir string
}

// NewOrderModal creates a new order modal
func NewOrderModal(app *tview.Application, callback func(string, float64, string), onCancel func()) *OrderModal {
	m := &OrderModal{
		Layout:     tview.NewFlex(),
		Form:       tview.NewForm(),
		Footer:     tview.NewTextView(),
		app:        app,
		callback:   callback,
		onCancel:   onCancel,
		currentDir: "Buy",
	}
	m.setupUI()
	return m
}

func (m *OrderModal) setupUI() {
	// Configure Main Layout (The "Window")
	m.Layout.SetDirection(tview.FlexRow).
		SetBorder(true).
		SetTitle(" New Order ").
		SetTitleAlign(tview.AlignCenter)

	// Configure Form (No border, transparent)
	m.Form.SetBorder(false)
	m.Form.SetBackgroundColor(tcell.ColorBlack)
	
	// Form styling
	m.Form.SetButtonBackgroundColor(tcell.ColorDarkGray).
		SetButtonTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorYellow).
		SetFieldBackgroundColor(tcell.ColorWhite).
		SetFieldTextColor(tcell.ColorBlack)

	m.instrument = tview.NewInputField().
		SetLabel("Instrument: ").
		SetFieldWidth(15).
		SetChangedFunc(func(text string) {
			m.updateCreateButton()
		})
	
	m.quantity = tview.NewInputField().
		SetLabel("Quantity:   ").
		SetFieldWidth(15).
		SetText("0").
		SetAcceptanceFunc(tview.InputFieldInteger).
		SetChangedFunc(func(text string) {
			m.updateCreateButton()
		})

	m.Form.AddFormItem(m.instrument)
	m.Form.AddFormItem(m.quantity)

	// Buttons
	m.Form.AddButton("Dir: "+m.currentDir, func() {
		m.ToggleDirection()
	})

	m.Form.AddButton("Create", func() {
		if m.Validate() {
			if m.callback != nil {
				m.callback(m.GetInstrument(), m.GetQuantity(), m.currentDir)
			}
		}
	})

	m.Form.AddButton("Cancel", func() {
		if m.onCancel != nil {
			m.onCancel()
		}
	})
	
m.updateCreateButton()

	// Configure Footer
	m.Footer.SetBackgroundColor(tcell.ColorDarkSlateGray)
	m.Footer.SetTextColor(tcell.ColorWhite).
		SetTextAlign(tview.AlignCenter).
		SetText("[TAB] Move  [ENTER] Select  [ESC] Close")

	// Assemble Layout
	// Form takes available space, Footer takes 1 line at bottom
	m.Layout.AddItem(m.Form, 0, 1, true).
		AddItem(m.Footer, 1, 1, false)
}
func (m *OrderModal) SetInstrument(symbol string) {
	m.instrument.SetText(symbol)
	m.updateCreateButton()
}

func (m *OrderModal) GetInstrument() string {
	return m.instrument.GetText()
}

func (m *OrderModal) SetQuantity(q float64) {
	m.quantity.SetText(strconv.FormatFloat(q, 'f', -1, 64))
	m.updateCreateButton()
}

func (m *OrderModal) GetQuantity() float64 {
	val, err := strconv.ParseFloat(m.quantity.GetText(), 64)
	if err != nil {
		return 0
	}
	return val
}

func (m *OrderModal) ToggleDirection() {
	if m.currentDir == "Buy" {
		m.currentDir = "Sell"
	} else {
		m.currentDir = "Buy"
	}
	// Update button label (Index 0)
	if m.Form.GetButtonCount() > 0 {
		m.Form.GetButton(0).SetLabel("Dir: " + m.currentDir)
	}
}

func (m *OrderModal) GetDirection() string {
	return m.currentDir
}

func (m *OrderModal) Validate() bool {
	if m.GetInstrument() == "" {
		return false
	}
	if m.GetQuantity() <= 0 {
		return false
	}
	return true
}

func (m *OrderModal) updateCreateButton() {
	if m.Form.GetButtonCount() > 1 {
		btn := m.Form.GetButton(1) // Create button
		btn.SetDisabled(!m.Validate())
	}
}


package ui

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// OrderModal represents the order entry modal
type OrderModal struct {
	Form       *tview.Form
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
		Form:       tview.NewForm(),
		app:        app,
		callback:   callback,
		onCancel:   onCancel,
		currentDir: "Buy",
	}
	m.setupUI()
	return m
}

func (m *OrderModal) setupUI() {
	m.Form.SetBorder(true).SetTitle(" New Order ").SetTitleAlign(tview.AlignCenter)
	
	// Form styling
	m.Form.SetButtonBackgroundColor(tcell.ColorDarkGray).
		SetButtonTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorYellow).
		SetFieldBackgroundColor(tcell.ColorWhite). // High contrast for visibility
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
	// Index 0: Direction
	m.Form.AddButton("Dir: "+m.currentDir, func() {
		m.ToggleDirection()
	})

	// Index 1: Create
	m.Form.AddButton("Create", func() {
		if m.Validate() {
			if m.callback != nil {
				m.callback(m.GetInstrument(), m.GetQuantity(), m.currentDir)
			}
		}
	})

	// Index 2: Cancel
	m.Form.AddButton("Cancel", func() {
		if m.onCancel != nil {
			m.onCancel()
		}
	})
	
	// Add Usage Instructions in Footer area by using an empty form item or just adding a text to the end
	// Actually, Form doesn't have a footer, but we can add a text field without label as a separator/info
	m.Form.AddTextView("Instructions:", "[TAB] Move  [ENTER] Select  [ESC] Close", 0, 1, true, false)
	
	m.updateCreateButton()
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


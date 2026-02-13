package ui

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// OrderModal represents the order entry modal
type OrderModal struct {
	Layout   *tview.Flex // Main container with border
	Form     *tview.Form
	Footer   *tview.TextView
	infoArea *tview.TextView
	app      *tview.Application
	callback func(string, float64, string)
	onCancel func()

	instrument *tview.InputField
	quantity   *tview.InputField
	direction  *tview.DropDown

	// State
	currentDir string
	lotSize    float64
	price      float64
}

// NewOrderModal creates a new order modal
func NewOrderModal(app *tview.Application, callback func(string, float64, string), onCancel func()) *OrderModal {
	m := &OrderModal{
		Layout:     tview.NewFlex(),
		Form:       tview.NewForm(),
		Footer:     tview.NewTextView(),
		infoArea:   tview.NewTextView(),
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
	m.Form.SetButtonBackgroundColor(tcell.ColorDarkGreen). // Darker green for idle
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
		SetText("").
		SetAcceptanceFunc(tview.InputFieldInteger).
		SetChangedFunc(func(text string) {
			m.updateCreateButton()
			m.updateInfo()
		})

	m.direction = tview.NewDropDown().
		SetLabel("Direction:  ").
		SetOptions([]string{"Buy", "Sell"}, func(text string, index int) {
			m.currentDir = text
		}).
		SetCurrentOption(0).
		SetFieldWidth(15)

	// Ensure dropdown list is styled consistently
	m.direction.SetListStyles(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack),
		tcell.StyleDefault.Background(tcell.ColorOrange).Foreground(tcell.ColorBlack))

	m.Form.AddFormItem(m.instrument)
	m.Form.AddFormItem(m.quantity)
	m.Form.AddFormItem(m.direction)

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

	// Configure Info Area (lot info, total shares, estimated cost)
	m.infoArea.SetDynamicColors(true)
	m.infoArea.SetBackgroundColor(tcell.ColorBlack)
	m.infoArea.SetTextColor(tcell.ColorLightGray)

	// Configure Footer
	m.Footer.SetBackgroundColor(tcell.ColorDarkSlateGray)
	m.Footer.SetTextColor(tcell.ColorWhite).
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(" [yellow]TAB[white] Move  [yellow]ENTER[white] Select  [yellow]ESC[white] Close")

	// Assemble Layout
	m.Layout.AddItem(m.Form, 0, 1, true).
		AddItem(m.infoArea, 1, 0, false).
		AddItem(m.Footer, 1, 0, false)
}

// SetDisplayName updates the modal title with the instrument's human-readable name.
func (m *OrderModal) SetDisplayName(name string) {
	if name != "" {
		m.Layout.SetTitle(fmt.Sprintf(" New Order — %s ", name))
	} else {
		m.Layout.SetTitle(" New Order ")
	}
}

func (m *OrderModal) SetInstrument(symbol string) {
	m.instrument.SetText(symbol)
	m.updateCreateButton()
}

func (m *OrderModal) GetInstrument() string {
	return m.instrument.GetText()
}

func (m *OrderModal) SetQuantity(q float64) {
	if q == 0 {
		m.quantity.SetText("")
	} else {
		m.quantity.SetText(strconv.FormatFloat(q, 'f', -1, 64))
	}
	m.updateCreateButton()
	m.updateInfo()
}

func (m *OrderModal) GetQuantity() float64 {
	val, err := strconv.ParseFloat(m.quantity.GetText(), 64)
	if err != nil {
		return 0
	}
	return val
}

func (m *OrderModal) GetDirection() string {
	_, text := m.direction.GetCurrentOption()
	return text
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

// SetLotSize sets the lot size for the current instrument and updates the info display
func (m *OrderModal) SetLotSize(lotSize float64) {
	m.lotSize = lotSize
	m.updateInfo()
}

// GetLotSize returns the current lot size
func (m *OrderModal) GetLotSize() float64 {
	return m.lotSize
}

// SetPrice sets the current price for estimated cost calculation
func (m *OrderModal) SetPrice(price float64) {
	m.price = price
	m.updateInfo()
}

// GetTotalShares returns the total shares (quantity in lots * lot size)
func (m *OrderModal) GetTotalShares() float64 {
	qty := m.GetQuantity()
	if m.lotSize > 0 {
		return qty * m.lotSize
	}
	return qty
}

// GetEstimatedCost returns the estimated cost (total shares * price)
func (m *OrderModal) GetEstimatedCost() float64 {
	return m.GetTotalShares() * m.price
}

// updateInfo refreshes the quantity label and info area based on lot size
func (m *OrderModal) updateInfo() {
	// Update quantity label to show lot size
	if m.lotSize > 0 {
		m.quantity.SetLabel(fmt.Sprintf("Lots (size - %.0f): ", m.lotSize))
	} else {
		m.quantity.SetLabel("Quantity:   ")
	}

	// Update info area with estimated cost
	if m.lotSize <= 0 {
		m.infoArea.SetText("")
		return
	}

	qty := m.GetQuantity()
	if qty > 0 && m.price > 0 {
		m.infoArea.SetText(fmt.Sprintf(" Est. Cost: %.2f", m.GetEstimatedCost()))
	} else {
		m.infoArea.SetText("")
	}
}

func (m *OrderModal) updateCreateButton() {
	if m.Form.GetButtonCount() > 1 {
		btn := m.Form.GetButton(1) // Create button
		btn.SetDisabled(!m.Validate())
	}
}

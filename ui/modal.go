package ui

import (
	"finam-terminal/models"
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// OrderSubmission holds all parameters from the order modal
type OrderSubmission struct {
	Instrument string
	Quantity   float64
	Direction  string
	OrderType  string  // models.OrderType* constants
	LimitPrice float64 // For Limit orders
	StopPrice  float64 // For Stop-Loss orders
	SLPrice    float64 // For SL+TP orders
	TPPrice    float64 // For SL+TP and Take-Profit orders
}

// OrderModal represents the order entry modal
type OrderModal struct {
	Layout   *tview.Flex // Main container with border
	Form     *tview.Form
	Footer   *tview.TextView
	infoArea *tview.TextView
	app      *tview.Application
	callback func(OrderSubmission)
	onCancel func()

	instrument *tview.InputField
	quantity   *tview.InputField
	direction  *tview.DropDown
	orderType  *tview.DropDown

	// Dynamic price fields (may be nil when not shown)
	limitPriceField *tview.InputField
	stopPriceField  *tview.InputField
	slPriceField    *tview.InputField
	tpPriceField    *tview.InputField

	// State
	currentDir       string
	currentOrderType string
	lotSize          float64
	price            float64
}

var orderTypeOptions = []string{
	models.OrderTypeMarket,
	models.OrderTypeLimit,
	models.OrderTypeStop,
	models.OrderTypeSLTP,
}

// NewOrderModal creates a new order modal
func NewOrderModal(app *tview.Application, callback func(OrderSubmission), onCancel func()) *OrderModal {
	m := &OrderModal{
		Layout:           tview.NewFlex(),
		Form:             tview.NewForm(),
		Footer:           tview.NewTextView(),
		infoArea:         tview.NewTextView(),
		app:              app,
		callback:         callback,
		onCancel:         onCancel,
		currentDir:       "Buy",
		currentOrderType: models.OrderTypeMarket,
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

	m.orderType = tview.NewDropDown().
		SetLabel("Order Type: ").
		SetOptions(orderTypeOptions, func(text string, index int) {
			if text != m.currentOrderType {
				m.currentOrderType = text
				m.rebuildPriceFields()
				m.updateCreateButton()
				m.updateInfo()
			}
		}).
		SetCurrentOption(0).
		SetFieldWidth(15)

	// Style dropdowns consistently
	dropdownStyle := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	dropdownActiveStyle := tcell.StyleDefault.Background(tcell.ColorOrange).Foreground(tcell.ColorBlack)
	m.direction.SetListStyles(dropdownStyle, dropdownActiveStyle)
	m.orderType.SetListStyles(dropdownStyle, dropdownActiveStyle)

	m.Form.AddFormItem(m.instrument)
	m.Form.AddFormItem(m.quantity)
	m.Form.AddFormItem(m.direction)
	m.Form.AddFormItem(m.orderType)

	m.Form.AddButton("Create", func() {
		if m.Validate() {
			if m.callback != nil {
				m.callback(m.buildSubmission())
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
		AddItem(m.infoArea, 2, 0, false).
		AddItem(m.Footer, 1, 0, false)
}

// rebuildPriceFields removes old dynamic price fields and adds new ones based on order type
func (m *OrderModal) rebuildPriceFields() {
	// Remove existing dynamic price fields (they are after index 3 = orderType dropdown)
	m.removeDynamicFields()

	// Reset field pointers
	m.limitPriceField = nil
	m.stopPriceField = nil
	m.slPriceField = nil
	m.tpPriceField = nil

	priceAcceptFunc := func(text string, lastChar rune) bool {
		// Allow digits and one decimal point
		if lastChar >= '0' && lastChar <= '9' {
			return true
		}
		if lastChar == '.' && !strings.Contains(text[:len(text)-1], ".") {
			return true
		}
		return false
	}

	changedFunc := func(text string) {
		m.updateCreateButton()
	}

	insertIdx := 4 // After orderType dropdown

	switch m.currentOrderType {
	case models.OrderTypeLimit:
		m.limitPriceField = tview.NewInputField().
			SetLabel("Limit Price:").
			SetFieldWidth(15).
			SetAcceptanceFunc(priceAcceptFunc).
			SetChangedFunc(changedFunc)
		m.Form.AddFormItem(m.limitPriceField)
		// Move to correct position (AddFormItem appends, we need it at insertIdx)
		m.moveLastFormItemTo(insertIdx)

	case models.OrderTypeStop:
		m.stopPriceField = tview.NewInputField().
			SetLabel("Stop Price: ").
			SetFieldWidth(15).
			SetAcceptanceFunc(priceAcceptFunc).
			SetChangedFunc(changedFunc)
		m.Form.AddFormItem(m.stopPriceField)
		m.moveLastFormItemTo(insertIdx)

	case models.OrderTypeSLTP:
		m.slPriceField = tview.NewInputField().
			SetLabel("SL Price:   ").
			SetFieldWidth(15).
			SetAcceptanceFunc(priceAcceptFunc).
			SetChangedFunc(changedFunc)
		m.Form.AddFormItem(m.slPriceField)
		m.moveLastFormItemTo(insertIdx)

		m.tpPriceField = tview.NewInputField().
			SetLabel("TP Price:   ").
			SetFieldWidth(15).
			SetAcceptanceFunc(priceAcceptFunc).
			SetChangedFunc(changedFunc)
		m.Form.AddFormItem(m.tpPriceField)
		m.moveLastFormItemTo(insertIdx + 1)
	}
}

// removeDynamicFields removes all form items after the base 4 (instrument, quantity, direction, orderType)
func (m *OrderModal) removeDynamicFields() {
	for m.Form.GetFormItemCount() > 4 {
		m.Form.RemoveFormItem(4)
	}
}

// moveLastFormItemTo moves the last form item to the target index by removing and re-inserting.
// tview doesn't have InsertFormItem, so we remove items after target, then re-add in order.
func (m *OrderModal) moveLastFormItemTo(targetIdx int) {
	count := m.Form.GetFormItemCount()
	if targetIdx >= count-1 {
		return // Already in place
	}
	// Collect items from targetIdx to count-2 (everything except the last one which is the new item)
	lastItem := m.Form.GetFormItem(count - 1)
	var displaced []tview.FormItem
	for i := targetIdx; i < count-1; i++ {
		displaced = append(displaced, m.Form.GetFormItem(i))
	}
	// Remove from targetIdx onwards
	for m.Form.GetFormItemCount() > targetIdx {
		m.Form.RemoveFormItem(targetIdx)
	}
	// Re-add: first the new item, then the displaced ones
	m.Form.AddFormItem(lastItem)
	for _, item := range displaced {
		m.Form.AddFormItem(item)
	}
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

func (m *OrderModal) GetOrderType() string {
	return m.currentOrderType
}

// ResetOrderType resets the order type dropdown to Market
func (m *OrderModal) ResetOrderType() {
	m.currentOrderType = models.OrderTypeMarket
	m.orderType.SetCurrentOption(0)
	m.rebuildPriceFields()
}

func (m *OrderModal) getPriceFieldValue(field *tview.InputField) float64 {
	if field == nil {
		return 0
	}
	val, err := strconv.ParseFloat(field.GetText(), 64)
	if err != nil {
		return 0
	}
	return val
}

func (m *OrderModal) Validate() bool {
	if m.GetInstrument() == "" {
		return false
	}
	if m.GetQuantity() <= 0 {
		return false
	}

	switch m.currentOrderType {
	case models.OrderTypeLimit:
		if m.getPriceFieldValue(m.limitPriceField) <= 0 {
			return false
		}
	case models.OrderTypeStop:
		if m.getPriceFieldValue(m.stopPriceField) <= 0 {
			return false
		}
	case models.OrderTypeSLTP:
		sl := m.getPriceFieldValue(m.slPriceField)
		tp := m.getPriceFieldValue(m.tpPriceField)
		if sl <= 0 && tp <= 0 {
			return false // At least one must be set
		}
	}

	return true
}

func (m *OrderModal) buildSubmission() OrderSubmission {
	sub := OrderSubmission{
		Instrument: m.GetInstrument(),
		Quantity:   m.GetQuantity(),
		Direction:  m.currentDir,
		OrderType:  m.currentOrderType,
	}

	switch m.currentOrderType {
	case models.OrderTypeLimit:
		sub.LimitPrice = m.getPriceFieldValue(m.limitPriceField)
	case models.OrderTypeStop:
		sub.StopPrice = m.getPriceFieldValue(m.stopPriceField)
	case models.OrderTypeSLTP:
		sub.SLPrice = m.getPriceFieldValue(m.slPriceField)
		sub.TPPrice = m.getPriceFieldValue(m.tpPriceField)
	}

	return sub
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

	// Build info text
	var lines []string

	// Current price reference
	if m.price > 0 {
		lines = append(lines, fmt.Sprintf(" Current Price: %.2f", m.price))
	}

	// Estimated cost
	qty := m.GetQuantity()
	if m.lotSize > 0 && qty > 0 && m.price > 0 {
		lines = append(lines, fmt.Sprintf(" Est. Cost: %.2f", m.GetEstimatedCost()))
	}

	if len(lines) > 0 {
		m.infoArea.SetText(strings.Join(lines, "\n"))
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

package ui

import (
	"finam-terminal/models"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// APISearchClient defines the interface for search operations
type APISearchClient interface {
	SearchSecurities(query string) ([]models.SecurityInfo, error)
	GetSnapshots(symbols []string) (map[string]models.Quote, error)
}

// SearchModal represents the security search window
type SearchModal struct {
	Layout   *tview.Flex
	Input    *tview.InputField
	Table    *tview.Table
	Footer   *tview.TextView
	
	app      *tview.Application
	client   APISearchClient
	onSelect func(ticker string)
	onCancel func()
	
	results  []models.SecurityInfo
}

// NewSearchModal creates a new security search modal
func NewSearchModal(app *tview.Application, client APISearchClient, onSelect func(ticker string), onCancel func()) *SearchModal {
	m := &SearchModal{
		Layout:   tview.NewFlex(),
		Input:    tview.NewInputField(),
		Table:    tview.NewTable(),
		Footer:   tview.NewTextView(),
		app:      app,
		client:   client,
		onSelect: onSelect,
		onCancel: onCancel,
	}
	m.setupUI()
	return m
}

func (m *SearchModal) setupUI() {
	m.Layout.SetDirection(tview.FlexRow).
		SetBorder(true).
		SetTitle(" Security Search (S-Key) ").
		SetTitleAlign(tview.AlignCenter)

	// Input Field
	m.Input.SetLabel(" Search: ").
		SetFieldBackgroundColor(tcell.ColorWhite).
		SetFieldTextColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorYellow)

	// Results Table
	m.Table.SetFixed(1, 1).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical).
		SetBorder(true).
		SetTitle(" Results ")

	// Header row for table
	headers := []string{"Ticker", "Name", "Lot", "Currency", "Price", "Change %"}
	for i, h := range headers {
		m.Table.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignCenter))
	}

	// Footer
	m.Footer.SetBackgroundColor(tcell.ColorDarkSlateGray)
	m.Footer.SetTextColor(tcell.ColorWhite).
		SetTextAlign(tview.AlignCenter).
		SetText("[TAB] Switch Focus  [UP/DOWN] Navigate  [A] Buy  [ESC] Close")

	// Assemble
	m.Layout.AddItem(m.Input, 1, 1, true).
		AddItem(m.Table, 0, 1, false).
		AddItem(m.Footer, 1, 1, false)

	// Setup Input Handlers for navigation
	m.setupHandlers()
}

func (m *SearchModal) setupHandlers() {
	m.Input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyDown:
			m.app.SetFocus(m.Table)
			return nil
		case tcell.KeyEscape:
			if m.onCancel != nil {
				m.onCancel()
			}
			return nil
		}
		return event
	})

	m.Table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			m.app.SetFocus(m.Input)
			return nil
		case tcell.KeyEscape:
			if m.onCancel != nil {
				m.onCancel()
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'a' || event.Rune() == 'A' {
				row, _ := m.Table.GetSelection()
				if row > 0 && row <= len(m.results) {
					ticker := m.results[row-1].Ticker
					if m.onSelect != nil {
						m.onSelect(ticker)
					}
				}
				return nil
			}
		}
		return event
	})
}

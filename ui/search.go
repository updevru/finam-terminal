package ui

import (
	"finam-terminal/models"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sync"
	"time"
)

// APISearchClient defines the interface for search operations
type APISearchClient interface {
	SearchSecurities(query string) ([]models.SecurityInfo, error)
	GetSnapshots(symbols []string) (map[string]models.Quote, error)
}

// SearchModal represents the security search window
type SearchModal struct {
	Layout *tview.Flex
	Input  *tview.InputField
	Table  *tview.Table
	Footer *tview.TextView

	app      *tview.Application
	client   APISearchClient
	onSelect func(ticker string)
	onCancel func()

	results     []models.SecurityInfo
	searchTimer *time.Timer
	timerMutex  sync.Mutex
	refreshStop chan struct{}
}

// NewSearchModal creates a new security search modal
func NewSearchModal(app *tview.Application, client APISearchClient, onSelect func(ticker string), onCancel func()) *SearchModal {
	m := &SearchModal{
		Layout:      tview.NewFlex(),
		Input:       tview.NewInputField(),
		Table:       tview.NewTable(),
		Footer:      tview.NewTextView(),
		app:         app,
		client:      client,
		onSelect:    onSelect,
		onCancel:    onCancel,
		refreshStop: make(chan struct{}),
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

	// SetChangedFunc for debounced search
	m.Input.SetChangedFunc(func(text string) {
		m.timerMutex.Lock()
		if m.searchTimer != nil {
			m.searchTimer.Stop()
		}
		m.searchTimer = time.AfterFunc(300*time.Millisecond, func() {
			m.PerformSearch(text)
		})
		m.timerMutex.Unlock()
	})

	// Setup Input Handlers for navigation
	m.setupHandlers()
}

// PerformSearch executes the search and updates the UI
func (m *SearchModal) PerformSearch(query string) {
	m.stopRefresh()

	if query == "" {
		m.app.QueueUpdateDraw(func() {
			m.results = nil
			m.updateTable(nil)
		})
		return
	}

	if m.client == nil {
		return
	}

	results, err := m.client.SearchSecurities(query)
	if err != nil {
		// Handle error (maybe show in status or table)
		return
	}

	// Fetch snapshots for results
	var tickers []string
	for _, res := range results {
		tickers = append(tickers, res.Ticker)
	}

	quotes, _ := m.client.GetSnapshots(tickers)

	m.app.QueueUpdateDraw(func() {
		m.results = results
		m.updateTable(quotes)
	})

	if len(results) > 0 {
		m.startRefresh()
	}
}

func (m *SearchModal) stopRefresh() {
	m.timerMutex.Lock()
	defer m.timerMutex.Unlock()
	close(m.refreshStop)
	m.refreshStop = make(chan struct{})
}

func (m *SearchModal) startRefresh() {
	m.timerMutex.Lock()
	stopChan := m.refreshStop
	m.timerMutex.Unlock()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				m.timerMutex.Lock()
				if len(m.results) == 0 {
					m.timerMutex.Unlock()
					continue
				}
				var tickers []string
				for _, res := range m.results {
					tickers = append(tickers, res.Ticker)
				}
				m.timerMutex.Unlock()

				quotes, err := m.client.GetSnapshots(tickers)
				if err == nil {
					m.app.QueueUpdateDraw(func() {
						m.updatePrices(quotes)
					})
				}
			}
		}
	}()
}

func (m *SearchModal) updatePrices(quotes map[string]models.Quote) {
	for i, res := range m.results {
		if q, ok := quotes[res.Ticker]; ok {
			row := i + 1
			m.Table.SetCell(row, 4, tview.NewTableCell(q.Last).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignRight))
		}
	}
}

func (m *SearchModal) updateTable(quotes map[string]models.Quote) {
	// Clear existing rows (except header)
	m.Table.Clear()

	// Restore headers
	headers := []string{"Ticker", "Name", "Lot", "Currency", "Price", "Change %"}
	for i, h := range headers {
		m.Table.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignCenter))
	}

	if len(m.results) == 0 {
		return
	}

	for i, res := range m.results {
		row := i + 1
		m.Table.SetCell(row, 0, tview.NewTableCell(res.Ticker).SetTextColor(tcell.ColorWhite))
		m.Table.SetCell(row, 1, tview.NewTableCell(res.Name).SetTextColor(tcell.ColorWhite))
		m.Table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", res.Lot)).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignCenter))
		m.Table.SetCell(row, 3, tview.NewTableCell(res.Currency).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignCenter))

		if q, ok := quotes[res.Ticker]; ok {
			m.Table.SetCell(row, 4, tview.NewTableCell(q.Last).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignRight))
			// Change % would need more data (Close price), but for now we put N/A or empty
			m.Table.SetCell(row, 5, tview.NewTableCell("N/A").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		} else {
			m.Table.SetCell(row, 4, tview.NewTableCell("...").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
			m.Table.SetCell(row, 5, tview.NewTableCell("...").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
		}
	}
	m.Table.ScrollToBeginning()
}

func (m *SearchModal) setupHandlers() {
	m.Input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyDown:
			m.app.SetFocus(m.Table)
			return nil
		case tcell.KeyEscape:
			m.stopRefresh()
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
			m.stopRefresh()
			if m.onCancel != nil {
				m.onCancel()
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'a' || event.Rune() == 'A' {
				row, _ := m.Table.GetSelection()
				if row > 0 && row <= len(m.results) {
					ticker := m.results[row-1].Ticker
					m.stopRefresh()
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
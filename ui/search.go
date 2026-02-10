package ui

import (
	"context"
	"finam-terminal/models"
	"fmt"
	"sync"
	"time"

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
	Layout *tview.Flex
	Input  *tview.InputField
	Table  *tview.Table
	Footer *tview.TextView

	app      *tview.Application
	client   APISearchClient
	onSelect func(ticker string)
	onCancel func()

	results      []models.SecurityInfo
	searchTimer  *time.Timer
	searchCancel context.CancelFunc
	timerMutex   sync.Mutex
	refreshStop  chan struct{}

	searching bool
	lastError string
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
		SetTitle(" Security Search ").
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

	// Set header cells with expansion where needed
	m.updateTableHeader()

	// Footer
	m.Footer.SetBackgroundColor(tcell.ColorDarkSlateGray)
	m.Footer.SetTextColor(tcell.ColorWhite).
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	// Assemble
	m.Layout.AddItem(m.Input, 1, 1, true).
		AddItem(m.Table, 0, 1, false).
		AddItem(m.Footer, 1, 1, false)

	m.updateFooter()

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

	m.timerMutex.Lock()
	if m.searchCancel != nil {
		m.searchCancel()
	}

	if len(query) < 3 {
		m.app.QueueUpdateDraw(func() {
			m.results = nil
			m.searching = false
			m.lastError = ""
			m.updateTable(nil)
			m.updateFooter()
		})
		m.searchCancel = nil
		m.timerMutex.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.searchCancel = cancel
	m.timerMutex.Unlock()

	if m.client == nil {
		return
	}

	m.app.QueueUpdateDraw(func() {
		m.searching = true
		m.lastError = ""
		m.updateTable(nil)
		m.updateFooter()
	})

	results, err := m.client.SearchSecurities(query)

	// Check if this search was cancelled
	select {
	case <-ctx.Done():
		return
	default:
	}

	if err != nil {
		m.app.QueueUpdateDraw(func() {
			m.searching = false
			m.lastError = extractUserMessage(err)
			m.updateTable(nil)
			m.updateFooter()
		})
		return
	}

	if len(results) == 0 {
		m.app.QueueUpdateDraw(func() {
			m.searching = false
			m.results = nil
			m.updateTable(nil)
			m.updateFooter()
		})
		return
	}

	// Fetch snapshots for results
	var tickers []string
	for _, res := range results {
		tickers = append(tickers, res.Ticker)
	}

	quotes, _ := m.client.GetSnapshots(tickers)

	// Check again if cancelled before updating UI
	select {
	case <-ctx.Done():
		return
	default:
	}

	m.app.QueueUpdateDraw(func() {
		m.searching = false
		m.results = results
		m.updateTable(quotes)
		m.updateFooter()
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

func (m *SearchModal) updateTableHeader() {
	headers := []string{"Ticker", "Name", "Lot", "Currency", "Price", "Change %"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignCenter)

		// Expand Name column
		if i == 1 {
			cell.SetExpansion(1)
		}

		m.Table.SetCell(0, i, cell)
	}
}

func (m *SearchModal) updateTable(quotes map[string]models.Quote) {
	// Clear existing rows (except header)
	m.Table.Clear()

	// Restore headers
	m.updateTableHeader()

	if m.searching {
		m.Table.SetCell(1, 1, tview.NewTableCell("Searching...").
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
		return
	}

	if m.lastError != "" {
		m.Table.SetCell(1, 1, tview.NewTableCell("Error: "+m.lastError).
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
		return
	}

	if len(m.results) == 0 {
		m.Table.SetCell(1, 1, tview.NewTableCell("No results found").
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
		return
	}

	for i, res := range m.results {
		row := i + 1
		// Ticker
		m.Table.SetCell(row, 0, tview.NewTableCell(res.Ticker).
			SetTextColor(tcell.ColorWhite).
			SetMaxWidth(20))

		// Name (Expandable)
		m.Table.SetCell(row, 1, tview.NewTableCell(res.Name).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(1))

		// Lot
		m.Table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", res.Lot)).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignCenter).
			SetMaxWidth(8))

		// Currency
		m.Table.SetCell(row, 3, tview.NewTableCell(res.Currency).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignCenter).
			SetMaxWidth(8))

		// Price & Change %
		priceCell := tview.NewTableCell("...").
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignRight).
			SetMaxWidth(10)
		
		changeCell := tview.NewTableCell("").
			SetAlign(tview.AlignRight).
			SetMaxWidth(10)

		if q, ok := quotes[res.Ticker]; ok {
			priceCell.SetText(q.Last).SetTextColor(tcell.ColorGreen)

			// Calculate change
			if last, err := parseFloat(q.Last); err == nil {
				if prevClose, err := parseFloat(q.Close); err == nil && prevClose > 0 {
					change := ((last - prevClose) / prevClose) * 100
					changeStr := fmt.Sprintf("%.2f%%", change)
					if change > 0 {
						changeCell.SetText("+" + changeStr).SetTextColor(tcell.ColorGreen)
					} else if change < 0 {
						changeCell.SetText(changeStr).SetTextColor(tcell.ColorRed)
					} else {
						changeCell.SetText(changeStr).SetTextColor(tcell.ColorGray)
					}
				} else {
					changeCell.SetText("N/A").SetTextColor(tcell.ColorGray)
				}
			}
		} else {
			changeCell.SetText("...").SetTextColor(tcell.ColorGray)
		}
		
		m.Table.SetCell(row, 4, priceCell)
		m.Table.SetCell(row, 5, changeCell)
	}
	m.Table.ScrollToBeginning()
}

func (m *SearchModal) updateFooter() {
	shortcuts := " [yellow]TAB[white] Switch Focus [yellow]UP/DOWN[white] Navigate [yellow]ENTER/A[white] Buy [yellow]ESC[white] Close"
	status := ""
	if m.searching {
		status = " | [yellow]Searching...[white]"
	} else if m.lastError != "" {
		status = fmt.Sprintf(" | [red]Error: %s[white]", m.lastError)
	} else if len(m.results) > 0 {
		status = fmt.Sprintf(" | [green]%d results found[white]", len(m.results))
	}

	m.Footer.SetText(fmt.Sprintf("%s%s", shortcuts, status))
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
		case tcell.KeyEnter:
			row, _ := m.Table.GetSelection()
			if row > 0 && row <= len(m.results) {
				ticker := m.results[row-1].Ticker
				m.stopRefresh()
				if m.onSelect != nil {
					m.onSelect(ticker)
				}
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
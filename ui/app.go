package ui

import (
	"sync"
	"time"

	"finam-terminal/api"
	"finam-terminal/models"

	"github.com/rivo/tview"
)

const (
	appVersion    = "1.0.0"
	refreshPeriod = 5 * time.Second
)

// App represents the TUI application
type App struct {
	app           *tview.Application
	client        *api.Client
	accounts      []models.AccountInfo
	positions     map[string][]models.Position
	quotes        map[string]map[string]*models.Quote
	selectedIdx   int
	dataMutex     DataMutex
	stopChan      chan struct{}
	stopOnce      sync.Once
	portfolioView *PortfolioView
	
	// Layout
	pages         *tview.Pages
	orderModal    *OrderModal

	statusMessage string
	statusType    StatusType

	// UI Components
	header    *tview.TextView
	statusBar *tview.TextView
}

type StatusType int

const (
	StatusInfo StatusType = iota
	StatusLoading
	StatusSuccess
	StatusError
)

// DataMutex wraps mutex for thread-safe data access
type DataMutex struct {
	sync.RWMutex
}

// NewApp creates a new TUI application
func NewApp(client *api.Client, accounts []models.AccountInfo) *App {
	a := &App{
		app:         tview.NewApplication(),
		client:      client,
		accounts:    accounts,
		positions:   make(map[string][]models.Position),
		quotes:      make(map[string]map[string]*models.Quote),
		selectedIdx: 0,
		stopChan:    make(chan struct{}),
		pages:       tview.NewPages(),
	}
	a.portfolioView = NewPortfolioView(a.app)
	a.header = createHeader()
	a.statusBar = createStatusBar()
	
	// Initialize OrderModal
	a.orderModal = NewOrderModal(a.app, func(instrument string, quantity float64, buySell string) {
		// Placeholder for Phase 3 submission task
		a.pages.HidePage("modal")
		a.app.SetFocus(a.portfolioView.PositionsTable)
	}, func() {
		// Cancel callback
		a.pages.HidePage("modal")
		a.app.SetFocus(a.portfolioView.PositionsTable)
	})

	return a
}

// Run starts the TUI application
func (a *App) Run() error {
	// Build layout
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexRow)

	flex.AddItem(a.header, 1, 1, false)
	flex.AddItem(a.portfolioView.Layout, 0, 1, true)
	flex.AddItem(a.statusBar, 1, 1, false)
	
	// Setup Pages
	a.pages.AddPage("main", flex, true, true)
	
	// Add Modal (centered)
	modalFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(a.orderModal.Form, 14, 1, true). // Height 14 for form
			AddItem(nil, 0, 1, false), 40, 1, true). // Width 40
		AddItem(nil, 0, 1, false)
	
	a.pages.AddPage("modal", modalFlex, true, false)

	// Setup input handlers
	setupInputHandlers(a)

	// Initialize UI with initial state (empty)
	updateAccountList(a)
	updatePositionsTable(a)
	updateInfoPanel(a)
	updateStatusBar(a)

	// Start background refresh
	go a.backgroundRefresh(a)

	return a.app.SetRoot(a.pages, true).EnableMouse(false).Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.stopOnce.Do(func() {
		close(a.stopChan)
		a.app.Stop()
	})
}

// OpenOrderModal opens the order entry modal
func (a *App) OpenOrderModal() {
	// Get selected row
	row, _ := a.portfolioView.PositionsTable.GetSelection()
	
	// Default to empty if header or invalid
	symbol := ""

	if row > 0 {
		// Map row to position index (row 1 -> index 0)
		idx := row - 1
		
		a.dataMutex.RLock()
		if a.selectedIdx < len(a.accounts) {
			accID := a.accounts[a.selectedIdx].ID
			positions := a.positions[accID]
			if idx >= 0 && idx < len(positions) {
				symbol = positions[idx].Ticker
			}
		}
		a.dataMutex.RUnlock()
	}
	
	a.orderModal.SetInstrument(symbol)
	// Reset quantity to 0
	a.orderModal.SetQuantity(0)
	
	a.pages.ShowPage("modal")
	a.app.SetFocus(a.orderModal.Form)
}

// SetStatus updates the status bar message and type
func (a *App) SetStatus(message string, statusType StatusType) {
	a.dataMutex.Lock()
	a.statusMessage = message
	a.statusType = statusType
	a.dataMutex.Unlock()

	a.app.QueueUpdateDraw(func() {
		updateStatusBar(a)
	})
}

package ui

import (
	"fmt"
	"sync"
	"time"

	"finam-terminal/models"

	"github.com/rivo/tview"
)

const (
	appVersion    = "1.0.0"
	refreshPeriod = 5 * time.Second
)

// APIClient defines the interface for the API client
type APIClient interface {
	GetAccounts() ([]models.AccountInfo, error)
	GetAccountDetails(accountID string) (*models.AccountInfo, []models.Position, error)
	GetQuotes(symbols []string) (map[string]*models.Quote, error)
	PlaceOrder(accountID string, symbol string, buySell string, quantity float64) (string, error)
}

// App represents the TUI application
type App struct {
	app           *tview.Application
	client        APIClient
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
func NewApp(client APIClient, accounts []models.AccountInfo) *App {
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
		if err := a.SubmitOrder(instrument, quantity, buySell); err != nil {
			a.ShowError(err.Error())
		}
	}, func() {
		a.CloseOrderModal()
	})

	return a
}

// CloseOrderModal closes the order modal
func (a *App) CloseOrderModal() {
	a.pages.HidePage("modal")
	a.app.SetFocus(a.portfolioView.PositionsTable)
}

// IsModalOpen returns true if the order modal is currently open
func (a *App) IsModalOpen() bool {
	name, _ := a.pages.GetFrontPage()
	return name == "modal"
}

// SubmitOrder submits a new order
func (a *App) SubmitOrder(symbol string, quantity float64, buySell string) error {
	a.dataMutex.RLock()
	if a.selectedIdx >= len(a.accounts) {
		a.dataMutex.RUnlock()
		return fmt.Errorf("no account selected")
	}
	accountID := a.accounts[a.selectedIdx].ID
	a.dataMutex.RUnlock()

	// Show loading status
	a.SetStatus("Placing order...", StatusLoading)
	
	id, err := a.client.PlaceOrder(accountID, symbol, buySell, quantity)
	if err != nil {
		a.SetStatus(fmt.Sprintf("Order failed: %v", err), StatusError)
		return err
	}

	a.SetStatus(fmt.Sprintf("Order placed: %s", id), StatusSuccess)
	
	// Refresh data
	a.loadDataAsync(accountID)
	
	// Close modal
	a.pages.HidePage("modal")
	a.app.SetFocus(a.portfolioView.PositionsTable)
	
	return nil
}

// ShowError displays an error modal
func (a *App) ShowError(msg string) {
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("alert")
			a.app.SetFocus(a.orderModal.Form)
		})
	
	a.pages.AddPage("alert", modal, false, true)
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
			AddItem(a.orderModal.Layout, 15, 1, true). // Height 15 (14 form + 1 footer)
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

	// Use QueueUpdateDraw only if the app might be running. 
	// In tests, we don't start the main loop, so this would block.
	// tview doesn't provide a direct way to check if it's running, 
	// but we can use a non-blocking check if we had a flag.
	// For now, let's use a goroutine to avoid blocking the caller if the queue is full/unattended.
	go a.app.QueueUpdateDraw(func() {
		updateStatusBar(a)
	})
}

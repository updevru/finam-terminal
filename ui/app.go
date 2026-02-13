package ui

import (
	"fmt"
	"sync"
	"time"

	"finam-terminal/models"

	_ "github.com/gdamore/tcell/v2/encoding" // Register encodings for Windows support
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
	GetQuotes(accountID string, symbols []string) (map[string]*models.Quote, error)
	PlaceOrder(accountID string, symbol string, buySell string, quantity float64) (string, error)
	ClosePosition(accountID string, symbol string, currentQuantity string, closeQuantity float64) (string, error)

	// Search operations
	SearchSecurities(query string) ([]models.SecurityInfo, error)
	GetSnapshots(accountID string, symbols []string) (map[string]models.Quote, error)
	GetLotSize(ticker string) float64
	GetInstrumentName(key string) string

	// History and Orders
	GetTradeHistory(accountID string) ([]models.Trade, error)
	GetActiveOrders(accountID string) ([]models.Order, error)
}

// App represents the TUI application
type App struct {
	app           *tview.Application
	client        APIClient
	accounts      []models.AccountInfo
	positions     map[string][]models.Position
	history       map[string][]models.Trade
	activeOrders  map[string][]models.Order
	quotes        map[string]map[string]*models.Quote
	selectedIdx   int
	dataMutex     DataMutex
	stopChan      chan struct{}
	stopOnce      sync.Once
	portfolioView *PortfolioView

	// Layout
	pages       *tview.Pages
	orderModal  *OrderModal
	closeModal  *ClosePositionModal
	searchModal *SearchModal

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
		history:     make(map[string][]models.Trade),
		activeOrders: make(map[string][]models.Order),
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
			a.ShowError(extractUserMessage(err))
		}
	}, func() {
		a.CloseOrderModal()
	})

	// Initialize ClosePositionModal
	a.closeModal = NewClosePositionModal(a.app, func(quantity float64) {
		if err := a.SubmitClosePosition(quantity); err != nil {
			a.ShowError(extractUserMessage(err))
		}
	}, func() {
		a.CloseCloseModal()
	}, func(msg string) {
		a.ShowError(msg)
	})

	// Initialize SearchModal
	a.searchModal = NewSearchModal(a.app, a.client, func(ticker string) {
		a.CloseSearchModal()
		a.OpenOrderModalWithTicker(ticker)
	}, func() {
		a.CloseSearchModal()
	})

	return a
}

// CloseCloseModal closes the close position modal
func (a *App) CloseCloseModal() {
	a.pages.HidePage("close_modal")
	a.app.SetFocus(a.portfolioView.TabbedView.PositionsTable)
}
func (a *App) CloseOrderModal() {
	a.pages.HidePage("modal")
	a.app.SetFocus(a.portfolioView.TabbedView.PositionsTable)
}

// OpenSearchModal opens the security search modal
func (a *App) OpenSearchModal() {
	a.dataMutex.RLock()
	if a.selectedIdx >= 0 && a.selectedIdx < len(a.accounts) {
		a.searchModal.SetAccountID(a.accounts[a.selectedIdx].ID)
	}
	a.dataMutex.RUnlock()

	a.pages.ShowPage("search_modal")
	a.app.SetFocus(a.searchModal.Input)
}

// CloseSearchModal closes the security search modal
func (a *App) CloseSearchModal() {
	a.pages.HidePage("search_modal")
	a.app.SetFocus(a.portfolioView.TabbedView.PositionsTable)
}

// IsSearchModalOpen returns true if the search modal is currently open
func (a *App) IsSearchModalOpen() bool {
	name, _ := a.pages.GetFrontPage()
	return name == "search_modal"
}

// OpenOrderModalWithTicker opens the order entry modal with a pre-populated ticker
func (a *App) OpenOrderModalWithTicker(ticker string) {
	a.orderModal.SetInstrument(ticker)
	a.orderModal.SetQuantity(0)
	lotSize := a.client.GetLotSize(ticker)
	a.orderModal.SetLotSize(lotSize)
	a.orderModal.SetPrice(0) // Price unknown from search context
	a.orderModal.SetDisplayName(a.client.GetInstrumentName(ticker))
	a.pages.ShowPage("modal")
	a.app.SetFocus(a.orderModal.Form)
}

// IsModalOpen returns true if the order modal is currently open
func (a *App) IsModalOpen() bool {
	name, _ := a.pages.GetFrontPage()
	return name == "modal"
}

// IsCloseModalOpen returns true if the close position modal is currently open
func (a *App) IsCloseModalOpen() bool {
	name, _ := a.pages.GetFrontPage()
	return name == "close_modal"
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
		msg := extractUserMessage(err)
		a.SetStatus(fmt.Sprintf("Order failed: %v", msg), StatusError)
		return err
	}

	a.SetStatus(fmt.Sprintf("Order placed: %s", id), StatusSuccess)

	// Refresh data
	a.loadDataAsync(accountID)

	// Close modal
	a.CloseOrderModal()

	return nil
}

// SubmitClosePosition submits an order to close an existing position
func (a *App) SubmitClosePosition(closeQuantity float64) error {
	// Get selected row to identify the position again
	row, _ := a.portfolioView.TabbedView.PositionsTable.GetSelection()
	if row <= 0 {
		return fmt.Errorf("no position selected")
	}
	idx := row - 1

	a.dataMutex.RLock()
	if a.selectedIdx >= len(a.accounts) {
		a.dataMutex.RUnlock()
		return fmt.Errorf("no account selected")
	}
	accountID := a.accounts[a.selectedIdx].ID
	positions := a.positions[accountID]

	if idx < 0 || idx >= len(positions) {
		a.dataMutex.RUnlock()
		return fmt.Errorf("invalid position selection")
	}

	pos := positions[idx]
	ticker := pos.Symbol // Use Symbol (e.g. Ticker@MIC) instead of just Ticker
	currentQty := pos.Quantity
	a.dataMutex.RUnlock()

	a.SetStatus("Closing position...", StatusLoading)

	id, err := a.client.ClosePosition(accountID, ticker, currentQty, closeQuantity)
	if err != nil {
		msg := extractUserMessage(err)
		a.SetStatus(fmt.Sprintf("Close failed: %v", msg), StatusError)
		return err
	}

	a.SetStatus(fmt.Sprintf("Position closed: %s", id), StatusSuccess)

	// Refresh data
	a.loadDataAsync(accountID)

	// Close modal
	a.CloseCloseModal()

	return nil
}

// ShowError displays an error modal
func (a *App) ShowError(msg string) {
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("alert")
			if a.IsModalOpen() {
				a.app.SetFocus(a.orderModal.Form)
			} else if a.IsCloseModalOpen() {
				a.app.SetFocus(a.closeModal.Form)
			}
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
			AddItem(a.orderModal.Layout, 16, 1, true). // Height 16 (form + info + footer)
			AddItem(nil, 0, 1, false), 50, 1, true).   // Width 50
		AddItem(nil, 0, 1, false)

	a.pages.AddPage("modal", modalFlex, true, false)

	// Add Close Modal (centered)
	closeModalFlex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(a.closeModal.Layout, 18, 1, true). // Height 18 (form + info + footer)
			AddItem(nil, 0, 1, false), 55, 1, true).   // Width 55
		AddItem(nil, 0, 1, false)

	a.pages.AddPage("close_modal", closeModalFlex, true, false)

	// Add Search Modal (full screen)
	a.pages.AddPage("search_modal", a.searchModal.Layout, true, false)

	// Setup input handlers
	setupInputHandlers(a)

	// Initialize UI with initial state (empty)
	updateAccountList(a)
	updatePositionsTable(a)
	updateInfoPanel(a)
	updateStatusBar(a)

	// Start background refresh
	go a.backgroundRefresh()

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
	row, _ := a.portfolioView.TabbedView.PositionsTable.GetSelection()

	// Default to empty if header or invalid
	symbol := ""
	displayName := ""

	if row > 0 {
		// Map row to position index (row 1 -> index 0)
		idx := row - 1

		a.dataMutex.RLock()
		if a.selectedIdx < len(a.accounts) {
			accID := a.accounts[a.selectedIdx].ID
			positions := a.positions[accID]
			if idx >= 0 && idx < len(positions) {
				symbol = positions[idx].Ticker
				displayName = positions[idx].Name
			}
		}
		a.dataMutex.RUnlock()
	}

	a.orderModal.SetInstrument(symbol)
	a.orderModal.SetQuantity(0)
	a.orderModal.SetDisplayName(displayName)

	// Set lot size and price for the selected instrument
	if symbol != "" && a.client != nil {
		lotSize := a.client.GetLotSize(symbol)
		a.orderModal.SetLotSize(lotSize)

		// Try to get current price from positions
		a.dataMutex.RLock()
		if a.selectedIdx < len(a.accounts) {
			accID := a.accounts[a.selectedIdx].ID
			for _, pos := range a.positions[accID] {
				if pos.Ticker == symbol {
					if p, err := parseFloat(pos.CurrentPrice); err == nil {
						a.orderModal.SetPrice(p)
					}
					break
				}
			}
		}
		a.dataMutex.RUnlock()
	} else {
		a.orderModal.SetLotSize(0)
		a.orderModal.SetPrice(0)
	}

	a.pages.ShowPage("modal")
	a.app.SetFocus(a.orderModal.Form)
}

// OpenCloseModal opens the close position modal
func (a *App) OpenCloseModal() {
	// Get selected row
	row, _ := a.portfolioView.TabbedView.PositionsTable.GetSelection()

	if row > 0 {
		idx := row - 1
		a.dataMutex.RLock()
		if a.selectedIdx < len(a.accounts) {
			accID := a.accounts[a.selectedIdx].ID
			positions := a.positions[accID]
			if idx >= 0 && idx < len(positions) {
				pos := positions[idx]
				// Parse values for display
				qty, err := parseFloat(pos.Quantity)
				if err != nil {
					a.ShowError(fmt.Sprintf("Invalid quantity format '%s' for %s", pos.Quantity, pos.Ticker))
					return
				}
				if qty <= 0 {
					a.ShowError(fmt.Sprintf("Position %s has non-positive quantity: %s", pos.Ticker, pos.Quantity))
					return
				}

				price, _ := parseFloat(pos.CurrentPrice)
				pnl, _ := parseFloat(pos.UnrealizedPnL)

				if pos.LotSize > 0 {
					a.closeModal.SetPositionDataWithLots(pos.Ticker, qty, price, pnl, pos.LotSize)
				} else {
					a.closeModal.SetPositionData(pos.Ticker, qty, price, pnl)
				}
				a.closeModal.SetDisplayName(pos.Name)
				a.pages.ShowPage("close_modal")
				a.app.SetFocus(a.closeModal.Form)
			}
		}
		a.dataMutex.RUnlock()
	}
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

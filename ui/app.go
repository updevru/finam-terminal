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
	}
	a.portfolioView = NewPortfolioView(a.app)
	a.header = createHeader()
	a.statusBar = createStatusBar()
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

	// Setup input handlers
	setupInputHandlers(a)

	// Initialize UI with initial state (empty)
	updateAccountList(a)
	updatePositionsTable(a)
	updateInfoPanel(a)
	updateStatusBar(a)

	// Start background refresh
	go a.backgroundRefresh(a)

	return a.app.SetRoot(flex, true).EnableMouse(false).Run()
}

// Stop stops the application
func (a *App) Stop() {
	a.stopOnce.Do(func() {
		close(a.stopChan)
		a.app.Stop()
	})
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

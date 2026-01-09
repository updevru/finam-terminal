package ui

import (
	"fmt"
	"strings"

	"finam-terminal/api"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// SetupApp represents the setup screen application
type SetupApp struct {
	app        *tview.Application
	inputField *tview.InputField
	statusText *tview.TextView
	onSave     func(token string) error
	grpcAddr   string
}

// NewSetupApp creates a new SetupApp
func NewSetupApp(grpcAddr string) *SetupApp {
	s := &SetupApp{
		app:      tview.NewApplication(),
		grpcAddr: grpcAddr,
	}
	s.setupUI()
	return s
}

func (s *SetupApp) setupUI() {
	// Logo and Instructions
	logoText := ApplyOrangeRedGradient(FinamLogo)
	welcomeText := fmt.Sprintf(`%s

[yellow]Welcome to Finam Terminal![white]

Для начала пользования Вам понадобится брокерский счет и токен,
также вы можете открыть демо-счет.

[blue]1. Открыть брокерский счет:[white] https://finam.ru/landings/otkrytie-scheta/
[blue]2. Открыть демо-счет:[white] https://www.finam.ru/landings/demoschet-bonus/
[blue]3. Создать токен (после авторизации):[white] https://tradeapi.finam.ru/docs/tokens/

Вставьте полученный токен ниже.
(Нажмите [::b]Ctrl+V[::-] или просто введите его)`, logoText)

	instructions := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(welcomeText)

	// Input Field
	s.inputField = tview.NewInputField().
		SetLabel("API Token: ").
		SetFieldWidth(60).
		SetMaskCharacter('*')

	// Handle Enter key in InputField
	s.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			s.handleSave()
		}
	})

	// Status Text (for errors)
	s.statusText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Save Button
	btnSave := tview.NewButton("Save & Continue").
		SetSelectedFunc(func() {
			s.handleSave()
		})

	// Layout
	form := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(instructions, 20, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(s.inputField, 60, 1, true).
			AddItem(nil, 0, 1, false), 3, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(btnSave, 20, 1, false).
			AddItem(nil, 0, 1, false), 3, 1, false).
		AddItem(s.statusText, 3, 1, false).
		AddItem(nil, 0, 1, false)

	s.app.SetRoot(form, true)
}

func (s *SetupApp) handleSave() {
	token := strings.TrimSpace(s.inputField.GetText())
	if token == "" {
		s.statusText.SetText("[red]Please enter a token")
		return
	}

	s.statusText.SetText("[yellow]Validating token...")

	go func() {
		client, err := api.NewClient(s.grpcAddr, token)
		if err != nil {
			s.app.QueueUpdateDraw(func() {
				s.statusText.SetText(fmt.Sprintf("[red]Client init error: %v", err))
			})
			return
		}

		// Validate by fetching accounts (lightweight check)
		_, err = client.GetAccounts()
		if err != nil {
			s.app.QueueUpdateDraw(func() {
				s.statusText.SetText(fmt.Sprintf("[red]Validation failed: %v", err))
			})
			return
		}

		// Validation successful
		s.app.QueueUpdateDraw(func() {
			if s.onSave != nil {
				if err := s.onSave(token); err != nil {
					s.statusText.SetText(fmt.Sprintf("[red]Save error: %v", err))
					return
				}
			}
			s.app.Stop()
		})
	}()
}

// SetOnSave sets the callback for when the save button is clicked
func (s *SetupApp) SetOnSave(callback func(token string) error) {
	s.onSave = callback
}

// Run starts the setup application
func (s *SetupApp) Run() error {
	return s.app.Run()
}

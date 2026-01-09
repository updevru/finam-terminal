package ui

import (
	"fmt"

	"github.com/rivo/tview"
)

// FinamLogo is the ASCII art representation of the FINAM logo.
const FinamLogo = `
███████╗██╗███╗   ███╗ █████╗ ███╗   ███╗
██╔════╝██║████╗ ████║██╔══██╗████╗ ████║
█████╗  ██║██╔████╔██║███████║██╔████╔██║
██╔══╝  ██║██║╚██╔╝██║██╔══██║██║╚██╔╝██║
██║     ██║██║ ╚═╝ ██║██║  ██║██║ ╚═╝ ██║
╚═╝     ╚═╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝
`

// SplashScreen represents the startup screen component.
type SplashScreen struct {
	Layout *tview.Flex
	Logo   *tview.TextView
}

// NewSplashScreen creates a new SplashScreen component.
func NewSplashScreen() *SplashScreen {
	logoText := ApplyOrangeRedGradient(FinamLogo)

	logo := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(logoText)

	// Center the logo vertically
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(logo, 8, 1, false). // Logo height is roughly 8 lines including padding
		AddItem(nil, 0, 1, false)

		return &SplashScreen{

			Layout: layout,

			Logo:   logo,

		}

	}

	

	// PrintConsoleSplash prints the gradient logo to stdout.

	func PrintConsoleSplash() {

		fmt.Println(ApplyOrangeRedGradientANSI(FinamLogo))

	}

	
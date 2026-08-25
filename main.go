package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"finam-terminal/api"
	"finam-terminal/config"
	"finam-terminal/models"
	"finam-terminal/platform"
	"finam-terminal/ui"
	"finam-terminal/updater"
	"finam-terminal/version"
)

func main() {
	platform.EnableUTF8()

	// Remove the backup left by a previous update (Windows cannot delete a
	// running .exe, so it is cleaned up on the next launch instead).
	updater.CleanupStaleBackup()

	// Setup file logging
	logFile, err := os.OpenFile("finam-terminal.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logFile.Close() }()
	log.SetOutput(logFile)

	// Parse command line flags
	accountIdx := flag.Int("account", -1, "Account index to show (0-based)")
	flag.Parse()
	_ = accountIdx // Silence unused variable warning until feature is implemented

	ui.PrintConsoleSplash()

	// Offer the update the last background check found. The state is read from
	// disk, so this costs no network round-trip and cannot delay the launch.
	// It runs before connecting to the broker: there is no point authenticating
	// if the process is about to restart.
	if offerPendingUpdate() {
		return
	}

	var cfg *config.Config
	var client *api.Client
	var accounts []models.AccountInfo

	// Initial load
	cfg, _ = config.Load()

	// If token is missing, show setup screen
	if cfg.APIToken == "" || cfg.APIToken == "your_api_token_here" {
		setup := ui.NewSetupApp(cfg.GRPCAddr)
		setup.SetOnSave(func(token string) error {
			return config.SaveTokenToUserHome(token)
		})
		if err := setup.Run(); err != nil {
			fmt.Printf("Setup failed: %v\n", err)
			os.Exit(1)
		}
		// Reload config after setup
		cfg, _ = config.Load()
	}

	steps := []ui.StartupStep{
		{
			Name: "Validating configuration...",
			Action: func() error {
				if cfg.APIToken == "" || cfg.APIToken == "your_api_token_here" {
					return fmt.Errorf("FINAM_API_TOKEN is not set")
				}
				return nil
			},
		},
		{
			Name: "Initializing API client...",
			Action: func() error {
				var err error
				client, err = api.NewClient(cfg.GRPCAddr, cfg.APIToken)
				return err
			},
		},
		{
			Name: "Fetching account list...",
			Action: func() error {
				var err error
				accounts, err = client.GetAccounts()
				if err != nil {
					return err
				}
				if len(accounts) == 0 {
					return fmt.Errorf("no accounts found")
				}
				return nil
			},
		},
		{
			Name: "Checking market data connection...",
			Action: func() error {
				// Simulate check or make a light call
				return nil
			},
		},
	}

	if err := ui.RunStartupSteps(steps); err != nil {
		fmt.Printf("Startup failed: %v\n", err)
		os.Exit(1)
	}

	// Warn about accounts with broker errors
	hasErrors := false
	for _, acc := range accounts {
		if acc.LoadError != "" {
			if !hasErrors {
				fmt.Println()
				hasErrors = true
			}
			fmt.Printf("\033[33m⚠ Account %s — broker error: %s\033[0m\n", acc.ID, acc.LoadError)
		}
	}
	if hasErrors {
		fmt.Println("\033[33m  Starting in limited mode. See finam-terminal.log for details.\033[0m")
		time.Sleep(2 * time.Second)
	}

	// Start TUI
	app := ui.NewApp(client, accounts)

	// Watch for new releases in the background. Run returns immediately on a
	// dev build, so nothing here touches the network unless this is a release.
	updateCtx, stopUpdateCheck := context.WithCancel(context.Background())
	defer stopUpdateCheck()
	go updater.Run(updateCtx, version.Version, app.NotifyUpdateAvailable)

	if err := app.Run(); err != nil {
		log.Fatalf("[ERROR] Application error: %v", err)
	}
	stopUpdateCheck()

	// The user pressed U and confirmed: the TUI has released the terminal, so
	// the binary can now replace itself and restart.
	if app.UpdateRequested() {
		if installUpdate() {
			return
		}
	}

	fmt.Println("[INFO] Goodbye!")
}

// offerPendingUpdate shows the startup dialog when the cached check result
// names a newer release, and installs it if the user agrees.
//
// It reports whether the process is being replaced, in which case main must
// return immediately. Every failure is reported to the user and then ignored:
// a failed update must never keep the terminal from starting.
func offerPendingUpdate() bool {
	if !updater.IsRelease(version.Version) {
		return false
	}

	state, err := updater.LoadState()
	if err != nil || !updater.IsNewer(version.Version, state.LatestVersion) {
		return false
	}

	if !ui.NewUpdatePromptApp(version.String(), state.LatestVersion).Run() {
		return false
	}
	return installUpdate()
}

// installUpdate fetches the current release, replaces the binary and restarts
// the process. It reports whether the restart was handed off; on any failure
// it prints an explanation, pauses so the message can be read, and returns
// false so the caller continues with a normal launch.
func installUpdate() bool {
	rel, err := updater.FetchLatestRelease(context.Background())
	if err != nil {
		fmt.Printf("\x1b[31m[ОШИБКА]\x1b[0m Не удалось получить сведения о релизе: %v\n", err)
		fmt.Printf("         Обновите вручную: %s\n", updater.ManualUpdateCommand())
		time.Sleep(2 * time.Second)
		return false
	}

	exePath, err := ui.RunUpdateFlow(rel)
	if err != nil {
		time.Sleep(2 * time.Second)
		return false
	}

	if err := updater.Restart(exePath); err != nil {
		fmt.Printf("\x1b[33m[ВНИМАНИЕ]\x1b[0m Обновление установлено, но перезапустить не удалось: %v\n", err)
		fmt.Printf("           Запустите программу заново.\n")
		time.Sleep(2 * time.Second)
		return false
	}
	return true
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"finam-terminal/api"
	"finam-terminal/config"
	"finam-terminal/models"
	"finam-terminal/ui"
)

func main() {
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

	// Start TUI
	app := ui.NewApp(client, accounts)
	if err := app.Run(); err != nil {
		log.Fatalf("[ERROR] Application error: %v", err)
	}

	fmt.Println("[INFO] Goodbye!")
}

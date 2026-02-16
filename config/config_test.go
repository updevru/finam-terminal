package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func TestFindToken(t *testing.T) {
	// Create a temporary directory to act as home
	tempHome := t.TempDir()

	// Backup original env/home if needed or just use a helper
	// For testing, we might want to pass the home dir to FindToken or mock it.
	// Let's assume FindToken uses os.UserHomeDir().

	// Create local .env
	localEnv := filepath.Join(t.TempDir(), ".env") // Different temp dir for local
	err := os.WriteFile(localEnv, []byte("FINAM_API_TOKEN=local-token"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Test case 1: No token anywhere
	t.Run("NoToken", func(t *testing.T) {
		// We'll need a way to override home dir in the implementation for testing
		token, path := findTokenInternal(tempHome, "non-existent-local-env")
		if token != "" {
			t.Errorf("Expected empty token, got %s", token)
		}
		if path != "" {
			t.Errorf("Expected empty path, got %s", path)
		}
	})

	// Test case 2: Token in local .env
	t.Run("LocalToken", func(t *testing.T) {
		token, path := findTokenInternal(tempHome, localEnv)
		if token != "local-token" {
			t.Errorf("Expected local-token, got %s", token)
		}
		if path != localEnv {
			t.Errorf("Expected path %s, got %s", localEnv, path)
		}
	})

	// Test case 3: Token in home .env
	t.Run("HomeToken", func(t *testing.T) {
		homeDir := filepath.Join(tempHome, ".finam-cli")
		err := os.MkdirAll(homeDir, 0755)
		if err != nil {
			t.Fatal(err)
		}
		homeEnv := filepath.Join(homeDir, ".env")
		err = os.WriteFile(homeEnv, []byte("FINAM_API_TOKEN=home-token"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		token, path := findTokenInternal(tempHome, "non-existent-local-env")
		if token != "home-token" {
			t.Errorf("Expected home-token, got %s", token)
		}
		if path != homeEnv {
			t.Errorf("Expected path %s, got %s", homeEnv, path)
		}
	})

	// Test case 4: Priority (Home over Local)
	t.Run("Priority", func(t *testing.T) {
		homeDir := filepath.Join(tempHome, ".finam-cli")
		homeEnv := filepath.Join(homeDir, ".env")
		err = os.WriteFile(homeEnv, []byte("FINAM_API_TOKEN=home-token"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		token, path := findTokenInternal(tempHome, localEnv)
		if token != "home-token" {
			t.Errorf("Expected home-token (priority), got %s", token)
		}
		if path != homeEnv {
			t.Errorf("Expected home path %s, got %s", homeEnv, path)
		}
	})
}

func TestSaveTokenToUserHome(t *testing.T) {
	tempHome := t.TempDir()
	token := "newly-saved-token"

	err := saveTokenInternal(tempHome, token)
	if err != nil {
		t.Fatalf("Failed to save token: %v", err)
	}

	homeEnv := filepath.Join(tempHome, ".finam-cli", ".env")
	if _, err := os.Stat(homeEnv); os.IsNotExist(err) {
		t.Fatalf("Expected .env file to exist at %s", homeEnv)
	}

	env, err := godotenv.Read(homeEnv)
	if err != nil {
		t.Fatalf("Failed to read saved .env: %v", err)
	}

	if env["FINAM_API_TOKEN"] != token {
		t.Errorf("Expected token %s, got %s", token, env["FINAM_API_TOKEN"])
	}
}

func TestLoad(t *testing.T) {
	// Clear env vars that might interfere
	origToken := os.Getenv("FINAM_API_TOKEN")
	_ = os.Unsetenv("FINAM_API_TOKEN")
	defer func() { _ = os.Setenv("FINAM_API_TOKEN", origToken) }()

	origAddr := os.Getenv("FINAM_GRPC_ADDR")
	_ = os.Unsetenv("FINAM_GRPC_ADDR")
	defer func() { _ = os.Setenv("FINAM_GRPC_ADDR", origAddr) }()

	origInterval := os.Getenv("REFRESH_INTERVAL")
	_ = os.Unsetenv("REFRESH_INTERVAL")
	defer func() { _ = os.Setenv("REFRESH_INTERVAL", origInterval) }()

	t.Run("DefaultValues", func(t *testing.T) {
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.GRPCAddr != "api.finam.ru:443" {
			t.Errorf("Expected default addr, got %s", cfg.GRPCAddr)
		}
		if cfg.RefreshInterval != 5 {
			t.Errorf("Expected default interval 5, got %d", cfg.RefreshInterval)
		}
	})

	t.Run("EnvValues", func(t *testing.T) {
		_ = os.Setenv("FINAM_API_TOKEN", "env-token")
		_ = os.Setenv("FINAM_GRPC_ADDR", "custom-addr")
		_ = os.Setenv("REFRESH_INTERVAL", "10")
		defer func() { _ = os.Unsetenv("FINAM_API_TOKEN") }()
		defer func() { _ = os.Unsetenv("FINAM_GRPC_ADDR") }()
		defer func() { _ = os.Unsetenv("REFRESH_INTERVAL") }()

		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.APIToken != "env-token" {
			t.Errorf("Expected env-token, got %s", cfg.APIToken)
		}
		if cfg.GRPCAddr != "custom-addr" {
			t.Errorf("Expected custom-addr, got %s", cfg.GRPCAddr)
		}
		if cfg.RefreshInterval != 10 {
			t.Errorf("Expected interval 10, got %d", cfg.RefreshInterval)
		}
	})

	t.Run("InvalidInterval", func(t *testing.T) {
		_ = os.Setenv("REFRESH_INTERVAL", "not-a-number")
		defer func() { _ = os.Unsetenv("REFRESH_INTERVAL") }()
		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.RefreshInterval != 5 {
			t.Errorf("Expected default interval 5 for invalid input, got %d", cfg.RefreshInterval)
		}
	})
}

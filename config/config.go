package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// gRPC connection
	GRPCAddr string

	// API credentials
	APIToken string

	// Application settings
	RefreshInterval int // in seconds
}

// Load reads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Try to load .env file from current directory
	_ = godotenv.Load()

	token := os.Getenv("FINAM_API_TOKEN")
	if token == "" {
		token, _ = FindToken()
	}

	cfg := &Config{
		GRPCAddr:        getEnv("FINAM_GRPC_ADDR", "api.finam.ru:443"),
		APIToken:        token,
		RefreshInterval: getEnvInt("REFRESH_INTERVAL", 5),
	}

	return cfg, nil
}

// userConfigDirName is the single place the ~/.finam-cli directory name is
// spelled. Everything that persists user state (the token .env, the update
// check cache) must resolve its path through the helpers below.
const userConfigDirName = ".finam-cli"

// UserConfigDir returns the per-user configuration directory (~/.finam-cli),
// where the API token .env and the update check cache live. The directory is
// not created — callers that write into it are expected to MkdirAll first.
//
// The error from os.UserHomeDir is propagated: a caller without a home
// directory (a locked-down service account, for example) must decide for
// itself whether that is fatal.
func UserConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return userConfigDirIn(home), nil
}

// userConfigDirIn builds the config directory path under an explicit home
// directory. Tests use it to work against a temporary home.
func userConfigDirIn(homeDir string) string {
	return filepath.Join(homeDir, userConfigDirName)
}

// FindToken searches for the API token in ~/.finam-cli/.env and ./.env
func FindToken() (string, string) {
	home, _ := os.UserHomeDir()
	return findTokenInternal(home, ".env")
}

func findTokenInternal(homeDir, localPath string) (string, string) {
	// 1. Check home directory
	if homeDir != "" {
		homeEnv := filepath.Join(userConfigDirIn(homeDir), ".env")
		if env, err := godotenv.Read(homeEnv); err == nil {
			if token, ok := env["FINAM_API_TOKEN"]; ok && token != "" {
				return token, homeEnv
			}
		}
	}

	// 2. Check local directory
	if env, err := godotenv.Read(localPath); err == nil {
		if token, ok := env["FINAM_API_TOKEN"]; ok && token != "" {
			return token, localPath
		}
	}

	return "", ""
}

// SaveTokenToUserHome saves the API token to ~/.finam-cli/.env
func SaveTokenToUserHome(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return saveTokenInternal(home, token)
}

func saveTokenInternal(homeDir, token string) error {
	dir := userConfigDirIn(homeDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	envPath := filepath.Join(dir, ".env")
	content := "FINAM_API_TOKEN=" + token + "\n"
	return os.WriteFile(envPath, []byte(content), 0644)
}

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns the integer value of an environment variable or a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

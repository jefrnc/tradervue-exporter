package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	Username  string
	Password  string
	DataDir   string
	UserAgent string
}

// Load reads configuration from environment variables (and optional .env file).
// CLI flag values can be passed in to override env vars.
func Load(flagUsername, flagPassword, flagDataDir string) (*Config, error) {
	// Load .env file if it exists (ignoring errors if missing)
	_ = godotenv.Load()

	cfg := &Config{
		Username:  envOrDefault("TRADERVUE_USERNAME", ""),
		Password:  envOrDefault("TRADERVUE_PASSWORD", ""),
		DataDir:   envOrDefault("TVUE_DATA_DIR", "./data"),
		UserAgent: "tvue-cli (https://github.com/jefrnc/tradervue-utils)",
	}

	// CLI flags override env vars
	if flagUsername != "" {
		cfg.Username = flagUsername
	}
	if flagPassword != "" {
		cfg.Password = flagPassword
	}
	if flagDataDir != "" {
		cfg.DataDir = flagDataDir
	}

	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("credentials required: set --username/--password flags or TRADERVUE_USERNAME/TRADERVUE_PASSWORD in .env")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

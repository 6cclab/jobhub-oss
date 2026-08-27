// Package config loads runtime configuration for the job search dashboard
// server from environment variables, with sensible local-dev defaults.
package config

import "os"

// Config holds all runtime configuration for the server.
type Config struct {
	Port        string
	DatabaseURL string
	APIToken    string
}

const (
	defaultPort        = "8080"
	defaultDatabaseURL = "postgres://jobhub:jobhub@localhost:5432/jobhub?sslmode=disable"
	defaultAPIToken    = ""
)

// Load reads configuration from environment variables, falling back to
// defaults for anything unset:
//   - PORT          (default "8080")
//   - DATABASE_URL  (default "postgres://jobhub:jobhub@localhost:5432/jobhub?sslmode=disable")
func Load() Config {
	return Config{
		Port:        getEnvOrDefault("PORT", defaultPort),
		DatabaseURL: getEnvOrDefault("DATABASE_URL", defaultDatabaseURL),
		APIToken:    getEnvOrDefault("API_TOKEN", defaultAPIToken),
	}
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

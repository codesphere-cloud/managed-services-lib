package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration.
type Config struct {
	// Port is the HTTP server port.
	Port int

	// APIKey is the optional API key for authentication.
	APIKey string

	// Kubeconfig is the path to the kubeconfig file.
	// If empty, in-cluster config is used.
	Kubeconfig string

	// Environment is the runtime environment (development, production, test).
	Environment string
}

// Load loads the configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnvInt("PORT", 8080),
		APIKey:      getEnv("API_KEY", ""),
		Kubeconfig:  getEnv("KUBECONFIG", ""),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	return cfg, nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

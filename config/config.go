package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go-api-proxy/logger"
)

// Config holds all application configuration settings
type Config struct {
	BackendHost   string
	Port          string
	WhitelistFile string
	Timeout       time.Duration
}

// Load creates a new Config instance with values from environment variables and defaults
func Load() (*Config, error) {
	logger.ConfigLogger.Info("Loading configuration from environment variables")
	
	config := &Config{
		BackendHost:   getEnvWithDefault("BACKEND_HOST", "https://exp.co2e.cc"),
		Port:          getEnvWithDefault("PORT", "80"),
		WhitelistFile: getEnvWithDefault("WHITELIST_FILE", "whitelist.json"),
		Timeout:       getTimeoutFromEnv("HTTP_TIMEOUT", 30*time.Second),
	}

	logger.ConfigLogger.Debug("Configuration loaded", map[string]interface{}{
		"backend_host":   config.BackendHost,
		"port":           config.Port,
		"whitelist_file": config.WhitelistFile,
		"timeout":        config.Timeout.String(),
	})

	if err := config.Validate(); err != nil {
		logger.ConfigLogger.Error("Configuration validation failed", err)
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.ConfigLogger.Info("Configuration validation successful")
	return config, nil
}

// Validate checks if the configuration values are valid
func (c *Config) Validate() error {
	if c.BackendHost == "" {
		return fmt.Errorf("backend host cannot be empty")
	}

	if !strings.HasPrefix(c.BackendHost, "http://") && !strings.HasPrefix(c.BackendHost, "https://") {
		return fmt.Errorf("backend host must start with http:// or https://")
	}

	if c.Port == "" {
		return fmt.Errorf("port cannot be empty")
	}

	// Validate port is a valid number
	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("port must be a valid number between 1 and 65535")
	}

	if c.WhitelistFile == "" {
		return fmt.Errorf("whitelist file path cannot be empty")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	return nil
}

// GetBackendAPIURL returns the full backend API URL
func (c *Config) GetBackendAPIURL() string {
	return strings.TrimSuffix(c.BackendHost, "/") + "/api/v2"
}

// getEnvWithDefault returns the environment variable value or the default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getTimeoutFromEnv parses timeout from environment variable or returns default
func getTimeoutFromEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}
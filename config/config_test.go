package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Clear environment variables before each test
	clearEnvVars()

	t.Run("loads default configuration", func(t *testing.T) {
		config, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if config.BackendHost != "https://exp.co2e.cc" {
			t.Errorf("expected default backend host 'https://exp.co2e.cc', got '%s'", config.BackendHost)
		}

		if config.Port != "80" {
			t.Errorf("expected default port '80', got '%s'", config.Port)
		}

		if config.WhitelistFile != "whitelist.json" {
			t.Errorf("expected default whitelist file 'whitelist.json', got '%s'", config.WhitelistFile)
		}

		if config.Timeout != 30*time.Second {
			t.Errorf("expected default timeout 30s, got %v", config.Timeout)
		}
	})

	t.Run("loads configuration from environment variables", func(t *testing.T) {
		os.Setenv("BACKEND_HOST", "https://api.example.com")
		os.Setenv("PORT", "8080")
		os.Setenv("WHITELIST_FILE", "custom-whitelist.json")
		os.Setenv("HTTP_TIMEOUT", "60")
		defer clearEnvVars()

		config, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if config.BackendHost != "https://api.example.com" {
			t.Errorf("expected backend host 'https://api.example.com', got '%s'", config.BackendHost)
		}

		if config.Port != "8080" {
			t.Errorf("expected port '8080', got '%s'", config.Port)
		}

		if config.WhitelistFile != "custom-whitelist.json" {
			t.Errorf("expected whitelist file 'custom-whitelist.json', got '%s'", config.WhitelistFile)
		}

		if config.Timeout != 60*time.Second {
			t.Errorf("expected timeout 60s, got %v", config.Timeout)
		}
	})

	t.Run("returns error for invalid configuration", func(t *testing.T) {
		os.Setenv("BACKEND_HOST", "invalid-url")
		defer clearEnvVars()

		_, err := Load()
		if err == nil {
			t.Error("expected error for invalid backend host, got nil")
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid configuration",
			config: Config{
				BackendHost:   "https://api.example.com",
				Port:          "8080",
				WhitelistFile: "whitelist.json",
				Timeout:       30 * time.Second,
			},
			expectError: false,
		},
		{
			name: "empty backend host",
			config: Config{
				BackendHost:   "",
				Port:          "8080",
				WhitelistFile: "whitelist.json",
				Timeout:       30 * time.Second,
			},
			expectError: true,
			errorMsg:    "backend host cannot be empty",
		},
		{
			name: "invalid backend host protocol",
			config: Config{
				BackendHost:   "ftp://api.example.com",
				Port:          "8080",
				WhitelistFile: "whitelist.json",
				Timeout:       30 * time.Second,
			},
			expectError: true,
			errorMsg:    "backend host must start with http:// or https://",
		},
		{
			name: "empty port",
			config: Config{
				BackendHost:   "https://api.example.com",
				Port:          "",
				WhitelistFile: "whitelist.json",
				Timeout:       30 * time.Second,
			},
			expectError: true,
			errorMsg:    "port cannot be empty",
		},
		{
			name: "invalid port number",
			config: Config{
				BackendHost:   "https://api.example.com",
				Port:          "invalid",
				WhitelistFile: "whitelist.json",
				Timeout:       30 * time.Second,
			},
			expectError: true,
			errorMsg:    "port must be a valid number between 1 and 65535",
		},
		{
			name: "port out of range",
			config: Config{
				BackendHost:   "https://api.example.com",
				Port:          "70000",
				WhitelistFile: "whitelist.json",
				Timeout:       30 * time.Second,
			},
			expectError: true,
			errorMsg:    "port must be a valid number between 1 and 65535",
		},
		{
			name: "empty whitelist file",
			config: Config{
				BackendHost:   "https://api.example.com",
				Port:          "8080",
				WhitelistFile: "",
				Timeout:       30 * time.Second,
			},
			expectError: true,
			errorMsg:    "whitelist file path cannot be empty",
		},
		{
			name: "zero timeout",
			config: Config{
				BackendHost:   "https://api.example.com",
				Port:          "8080",
				WhitelistFile: "whitelist.json",
				Timeout:       0,
			},
			expectError: true,
			errorMsg:    "timeout must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestGetBackendAPIURL(t *testing.T) {
	tests := []struct {
		name        string
		backendHost string
		expected    string
	}{
		{
			name:        "host without trailing slash",
			backendHost: "https://api.example.com",
			expected:    "https://api.example.com/api/v2",
		},
		{
			name:        "host with trailing slash",
			backendHost: "https://api.example.com/",
			expected:    "https://api.example.com/api/v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{BackendHost: tt.backendHost}
			result := config.GetBackendAPIURL()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	t.Run("returns environment variable when set", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := getEnvWithDefault("TEST_VAR", "default_value")
		if result != "test_value" {
			t.Errorf("expected 'test_value', got '%s'", result)
		}
	})

	t.Run("returns default when environment variable not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR")

		result := getEnvWithDefault("TEST_VAR", "default_value")
		if result != "default_value" {
			t.Errorf("expected 'default_value', got '%s'", result)
		}
	})
}

func TestGetTimeoutFromEnv(t *testing.T) {
	t.Run("returns parsed timeout from environment", func(t *testing.T) {
		os.Setenv("TEST_TIMEOUT", "45")
		defer os.Unsetenv("TEST_TIMEOUT")

		result := getTimeoutFromEnv("TEST_TIMEOUT", 30*time.Second)
		expected := 45 * time.Second
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("returns default for invalid timeout", func(t *testing.T) {
		os.Setenv("TEST_TIMEOUT", "invalid")
		defer os.Unsetenv("TEST_TIMEOUT")

		result := getTimeoutFromEnv("TEST_TIMEOUT", 30*time.Second)
		expected := 30 * time.Second
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("returns default when environment variable not set", func(t *testing.T) {
		os.Unsetenv("TEST_TIMEOUT")

		result := getTimeoutFromEnv("TEST_TIMEOUT", 30*time.Second)
		expected := 30 * time.Second
		if result != expected {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})
}

// clearEnvVars clears all environment variables used in tests
func clearEnvVars() {
	os.Unsetenv("BACKEND_HOST")
	os.Unsetenv("PORT")
	os.Unsetenv("WHITELIST_FILE")
	os.Unsetenv("HTTP_TIMEOUT")
}
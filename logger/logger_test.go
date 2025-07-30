package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test-component")
	
	if logger.component != "test-component" {
		t.Errorf("Expected component 'test-component', got '%s'", logger.component)
	}
	
	if logger.level != INFO {
		t.Errorf("Expected default level INFO, got %v", logger.level)
	}
}

func TestWithRequestID(t *testing.T) {
	logger := NewLogger("test")
	loggerWithID := logger.WithRequestID("req-123")
	
	expected := "test[req-123]"
	if loggerWithID.component != expected {
		t.Errorf("Expected component '%s', got '%s'", expected, loggerWithID.component)
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(99), "UNKNOWN"},
	}
	
	for _, test := range tests {
		if test.level.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.level.String())
		}
	}
}

func TestLogOutput(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := &Logger{
		component: "test",
		level:     DEBUG,
		logger:    log.New(&buf, "", 0),
	}
	
	// Test info logging
	logger.Info("test message", map[string]interface{}{
		"key": "value",
	})
	
	output := buf.String()
	
	// Verify JSON structure
	var logEntry LogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}
	
	if logEntry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", logEntry.Level)
	}
	
	if logEntry.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", logEntry.Message)
	}
	
	if logEntry.Component != "test" {
		t.Errorf("Expected component 'test', got '%s'", logEntry.Component)
	}
	
	if logEntry.Fields["key"] != "value" {
		t.Errorf("Expected field key=value, got %v", logEntry.Fields["key"])
	}
}

func TestErrorLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		component: "test",
		level:     DEBUG,
		logger:    log.New(&buf, "", 0),
	}
	
	testErr := errors.New("test error")
	logger.Error("error occurred", testErr, map[string]interface{}{
		"context": "testing",
	})
	
	output := buf.String()
	
	var logEntry LogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}
	
	if logEntry.Level != "ERROR" {
		t.Errorf("Expected level ERROR, got %s", logEntry.Level)
	}
	
	if logEntry.Error != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", logEntry.Error)
	}
	
	if logEntry.Fields["context"] != "testing" {
		t.Errorf("Expected context=testing, got %v", logEntry.Fields["context"])
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		component: "test",
		level:     ERROR, // Only ERROR and FATAL should be logged
		logger:    log.New(&buf, "", 0),
	}
	
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message", nil)
	
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// Should only have one line (the error message)
	if len(lines) != 1 {
		t.Errorf("Expected 1 log line, got %d", len(lines))
	}
	
	if !strings.Contains(output, "error message") {
		t.Error("Expected error message to be logged")
	}
}

func TestGetLogLevelFromEnv(t *testing.T) {
	tests := []struct {
		envValue string
		expected LogLevel
	}{
		{"DEBUG", DEBUG},
		{"INFO", INFO},
		{"WARN", WARN},
		{"ERROR", ERROR},
		{"FATAL", FATAL},
		{"debug", DEBUG}, // Should handle lowercase
		{"invalid", INFO}, // Should default to INFO
		{"", INFO},        // Should default to INFO
	}
	
	for _, test := range tests {
		os.Setenv("LOG_LEVEL", test.envValue)
		level := getLogLevelFromEnv()
		if level != test.expected {
			t.Errorf("For env value '%s', expected %v, got %v", test.envValue, test.expected, level)
		}
	}
	
	// Clean up
	os.Unsetenv("LOG_LEVEL")
}
package main

import (
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TestApplicationLifecycle tests the complete application lifecycle including startup and graceful shutdown
func TestApplicationLifecycle(t *testing.T) {
	// Skip this test in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Build the application
	buildCmd := exec.Command("go", "build", "-o", "test-proxy", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build application: %v", err)
	}
	defer os.Remove("test-proxy")
	
	// Set environment variables for testing
	os.Setenv("BACKEND_HOST", "https://httpbin.org")
	os.Setenv("PORT", "8081")
	os.Setenv("WHITELIST_FILE", "whitelist.json")
	defer func() {
		os.Unsetenv("BACKEND_HOST")
		os.Unsetenv("PORT")
		os.Unsetenv("WHITELIST_FILE")
	}()
	
	// Start the application
	cmd := exec.Command("./test-proxy")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	
	// Ensure we clean up the process
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	
	// Wait for the server to start
	time.Sleep(2 * time.Second)
	
	// Test health check endpoint
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8081/health")
	if err != nil {
		t.Fatalf("Failed to make health check request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check failed with status: %d", resp.StatusCode)
	}
	
	// Test graceful shutdown by sending SIGTERM
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Failed to send SIGTERM: %v", err)
	}
	
	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			t.Logf("Process exited with error: %v", err)
		} else {
			t.Log("Process exited cleanly")
		}
	case <-time.After(10 * time.Second):
		t.Error("Process did not exit within timeout")
		cmd.Process.Kill()
	}
}

// TestApplicationStartupWithInvalidConfig tests application behavior with invalid configuration
func TestApplicationStartupWithInvalidConfig(t *testing.T) {
	// Skip this test in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Build the application
	buildCmd := exec.Command("go", "build", "-o", "test-proxy-invalid", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build application: %v", err)
	}
	defer os.Remove("test-proxy-invalid")
	
	// Set invalid configuration
	os.Setenv("BACKEND_HOST", "invalid-protocol://example.com")
	os.Setenv("PORT", "8082")
	defer func() {
		os.Unsetenv("BACKEND_HOST")
		os.Unsetenv("PORT")
	}()
	
	// Start the application
	cmd := exec.Command("./test-proxy-invalid")
	err := cmd.Run()
	
	// Application should exit with error due to invalid configuration
	if err == nil {
		t.Error("Expected application to exit with error due to invalid configuration")
	}
}

// TestApplicationHealthEndpoint tests the health endpoint functionality
func TestApplicationHealthEndpoint(t *testing.T) {
	// Skip this test in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Build the application
	buildCmd := exec.Command("go", "build", "-o", "test-proxy-health", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build application: %v", err)
	}
	defer os.Remove("test-proxy-health")
	
	// Set environment variables for testing
	os.Setenv("BACKEND_HOST", "https://httpbin.org")
	os.Setenv("PORT", "8083")
	os.Setenv("WHITELIST_FILE", "whitelist.json")
	defer func() {
		os.Unsetenv("BACKEND_HOST")
		os.Unsetenv("PORT")
		os.Unsetenv("WHITELIST_FILE")
	}()
	
	// Start the application
	cmd := exec.Command("./test-proxy-health")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start application: %v", err)
	}
	
	// Ensure we clean up the process
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
			cmd.Wait()
		}
	}()
	
	// Wait for the server to start
	time.Sleep(2 * time.Second)
	
	// Test multiple health check requests
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < 3; i++ {
		resp, err := client.Get("http://localhost:8083/health")
		if err != nil {
			t.Fatalf("Failed to make health check request %d: %v", i+1, err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Health check %d failed with status: %d", i+1, resp.StatusCode)
		}
		
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
		
		resp.Body.Close()
	}
}
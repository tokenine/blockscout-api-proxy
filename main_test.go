package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go-api-proxy/config"
	"go-api-proxy/models"
)

// mockBackendServer creates a mock backend server for testing
func mockBackendServer() *httptest.Server {
	mux := http.NewServeMux()
	
	// Mock tokens endpoint
	mux.HandleFunc("/api/v2/tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := models.TokenResponse{
			Items: []models.Token{
				{
					Address:     "0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
					AddressHash: "hash1",
					Name:        "Token1",
					Symbol:      "TK1",
					Type:        "ERC-20",
					Decimals:    "18",
					TotalSupply: "1000000",
					Holders:     "100",
					HoldersCount: "100",
				},
				{
					Address:     "0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
					AddressHash: "hash2",
					Name:        "Token2",
					Symbol:      "TK2",
					Type:        "ERC-20",
					Decimals:    "18",
					TotalSupply: "2000000",
					Holders:     "200",
					HoldersCount: "200",
				},
				{
					Address:     "0x1234567890123456789012345678901234567890",
					AddressHash: "hash3",
					Name:        "Token3",
					Symbol:      "TK3",
					Type:        "ERC-20",
					Decimals:    "18",
					TotalSupply: "3000000",
					Holders:     "300",
					HoldersCount: "300",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	
	// Mock other endpoints
	mux.HandleFunc("/api/v2/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"mock response","path":"` + r.URL.Path + `"}`))
	})
	
	return httptest.NewServer(mux)
}

// createTestWhitelistFile creates a temporary whitelist file for testing
func createTestWhitelistFile(t *testing.T, addresses []string) string {
	t.Helper()
	
	file, err := os.CreateTemp("", "whitelist_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()
	
	whitelist := struct {
		Addresses []string `json:"addresses"`
	}{
		Addresses: addresses,
	}
	
	if err := json.NewEncoder(file).Encode(whitelist); err != nil {
		t.Fatalf("Failed to write whitelist file: %v", err)
	}
	
	return file.Name()
}

func TestProxyServer_NewProxyServer(t *testing.T) {
	// Create test whitelist file
	whitelistFile := createTestWhitelistFile(t, []string{
		"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
	})
	defer os.Remove(whitelistFile)
	
	cfg := &config.Config{
		BackendHost:   "https://example.com",
		Port:          "8080",
		WhitelistFile: whitelistFile,
		Timeout:       30 * time.Second,
	}
	
	server, err := NewProxyServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	
	if server.config != cfg {
		t.Error("Config not set correctly")
	}
	
	if server.whitelist.Size() != 2 {
		t.Errorf("Expected 2 whitelist addresses, got %d", server.whitelist.Size())
	}
}

func TestProxyServer_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		BackendHost:   "https://example.com",
		Port:          "8080",
		WhitelistFile: "nonexistent.json",
		Timeout:       30 * time.Second,
	}
	
	server, err := NewProxyServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	server.healthCheckHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
	
	body := w.Body.String()
	if !strings.Contains(body, "healthy") {
		t.Errorf("Expected health response to contain 'healthy', got: %s", body)
	}
}

func TestProxyServer_TokensEndpointFiltering(t *testing.T) {
	// Start mock backend server
	mockServer := mockBackendServer()
	defer mockServer.Close()
	
	// Create test whitelist file with only first two addresses
	whitelistFile := createTestWhitelistFile(t, []string{
		"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
	})
	defer os.Remove(whitelistFile)
	
	cfg := &config.Config{
		BackendHost:   mockServer.URL,
		Port:          "8080",
		WhitelistFile: whitelistFile,
		Timeout:       30 * time.Second,
	}
	
	server, err := NewProxyServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	
	// Test tokens endpoint
	req := httptest.NewRequest("GET", "/api/v2/tokens", nil)
	w := httptest.NewRecorder()
	
	server.routeHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response models.TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Should only return 2 tokens (filtered by whitelist)
	if len(response.Items) != 2 {
		t.Errorf("Expected 2 filtered tokens, got %d", len(response.Items))
	}
	
	// Verify the correct tokens are returned
	expectedAddresses := map[string]bool{
		"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614": false,
		"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7": false,
	}
	
	for _, token := range response.Items {
		if _, exists := expectedAddresses[token.Address]; exists {
			expectedAddresses[token.Address] = true
		} else {
			t.Errorf("Unexpected token address in response: %s", token.Address)
		}
	}
	
	for addr, found := range expectedAddresses {
		if !found {
			t.Errorf("Expected token address not found in response: %s", addr)
		}
	}
}

func TestProxyServer_TokensEndpointWithQuery(t *testing.T) {
	// Start mock backend server
	mockServer := mockBackendServer()
	defer mockServer.Close()
	
	// Create test whitelist file
	whitelistFile := createTestWhitelistFile(t, []string{
		"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
	})
	defer os.Remove(whitelistFile)
	
	cfg := &config.Config{
		BackendHost:   mockServer.URL,
		Port:          "8080",
		WhitelistFile: whitelistFile,
		Timeout:       30 * time.Second,
	}
	
	server, err := NewProxyServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	
	// Test tokens endpoint with query parameters
	req := httptest.NewRequest("GET", "/api/v2/tokens?limit=10", nil)
	w := httptest.NewRecorder()
	
	server.routeHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response models.TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Should only return 1 token (filtered by whitelist)
	if len(response.Items) != 1 {
		t.Errorf("Expected 1 filtered token, got %d", len(response.Items))
	}
}

func TestProxyServer_StandardProxyEndpoint(t *testing.T) {
	// Start mock backend server
	mockServer := mockBackendServer()
	defer mockServer.Close()
	
	cfg := &config.Config{
		BackendHost:   mockServer.URL,
		Port:          "8080",
		WhitelistFile: "nonexistent.json",
		Timeout:       30 * time.Second,
	}
	
	server, err := NewProxyServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	
	// Test non-tokens endpoint
	req := httptest.NewRequest("GET", "/api/v2/other", nil)
	w := httptest.NewRecorder()
	
	server.routeHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	body := w.Body.String()
	if !strings.Contains(body, "mock response") {
		t.Errorf("Expected mock response, got: %s", body)
	}
	
	if !strings.Contains(body, "/api/v2/other") {
		t.Errorf("Expected path in response, got: %s", body)
	}
}

func TestProxyServer_IsTokensEndpoint(t *testing.T) {
	cfg := &config.Config{
		BackendHost:   "https://example.com",
		Port:          "8080",
		WhitelistFile: "nonexistent.json",
		Timeout:       30 * time.Second,
	}
	
	server, err := NewProxyServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/v2/tokens", true},
		{"/api/v2/tokens/", true},
		{"/api/v2/tokens?limit=10", true},
		{"/api/v2/tokens?limit=10&offset=0", true},
		{"/api/v2/other", false},
		{"/api/v2/tokens/123", false},
		{"/api/v1/tokens", false},
		{"/tokens", false},
		{"", false},
	}
	
	for _, test := range tests {
		result := server.isTokensEndpoint(test.path)
		if result != test.expected {
			t.Errorf("isTokensEndpoint(%q) = %v, expected %v", test.path, result, test.expected)
		}
	}
}

func TestProxyServer_Integration(t *testing.T) {
	// Start mock backend server
	mockServer := mockBackendServer()
	defer mockServer.Close()
	
	// Create test whitelist file
	whitelistFile := createTestWhitelistFile(t, []string{
		"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
	})
	defer os.Remove(whitelistFile)
	
	cfg := &config.Config{
		BackendHost:   mockServer.URL,
		Port:          "0", // Use random port for testing
		WhitelistFile: whitelistFile,
		Timeout:       30 * time.Second,
	}
	
	server, err := NewProxyServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	
	// Start test server
	testServer := httptest.NewServer(server.server.Handler)
	defer testServer.Close()
	
	// Test health endpoint
	resp, err := http.Get(testServer.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make health request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check failed with status: %d", resp.StatusCode)
	}
	
	// Test tokens endpoint
	resp, err = http.Get(testServer.URL + "/api/v2/tokens")
	if err != nil {
		t.Fatalf("Failed to make tokens request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Tokens request failed with status: %d, body: %s", resp.StatusCode, string(body))
	}
	
	var tokenResponse models.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		t.Fatalf("Failed to decode tokens response: %v", err)
	}
	
	if len(tokenResponse.Items) != 1 {
		t.Errorf("Expected 1 filtered token, got %d", len(tokenResponse.Items))
	}
	
	// Test standard proxy endpoint
	resp, err = http.Get(testServer.URL + "/api/v2/other")
	if err != nil {
		t.Fatalf("Failed to make proxy request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Proxy request failed with status: %d", resp.StatusCode)
	}
}

func TestProxyServer_GracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		BackendHost:   "https://example.com",
		Port:          "0", // Use random port for testing
		WhitelistFile: "nonexistent.json",
		Timeout:       30 * time.Second,
	}
	
	server, err := NewProxyServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}
	
	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Graceful shutdown failed: %v", err)
	}
}
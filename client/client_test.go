package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-api-proxy/config"
	"go-api-proxy/models"
)

func TestNewHTTPClient(t *testing.T) {
	cfg := &config.Config{
		BackendHost: "https://example.com",
		Timeout:     30 * time.Second,
	}

	client := NewHTTPClient(cfg)

	if client == nil {
		t.Fatal("NewHTTPClient returned nil")
	}

	if client.client.Timeout != cfg.Timeout {
		t.Errorf("Expected timeout %v, got %v", cfg.Timeout, client.client.Timeout)
	}

	expectedURL := "https://example.com/api/v2"
	if client.backendURL != expectedURL {
		t.Errorf("Expected backend URL %s, got %s", expectedURL, client.backendURL)
	}
}

func TestProxyRequest_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.URL.Path != "/api/v2/test" {
			t.Errorf("Expected path /api/v2/test, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		
		// Check forwarded headers
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept header to be forwarded")
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		BackendHost: server.URL,
		Timeout:     5 * time.Second,
	}
	client := NewHTTPClient(cfg)

	// Create original request
	originalReq := httptest.NewRequest("GET", "/test", nil)
	originalReq.Header.Set("Accept", "application/json")

	ctx := context.Background()
	resp, err := client.ProxyRequest(ctx, originalReq, "/test")

	if err != nil {
		t.Fatalf("ProxyRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expected := `{"message": "success"}`
	if string(body) != expected {
		t.Errorf("Expected body %s, got %s", expected, string(body))
	}
}

func TestProxyRequest_WithBody(t *testing.T) {
	requestBody := `{"test": "data"}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}
		
		if string(body) != requestBody {
			t.Errorf("Expected request body %s, got %s", requestBody, string(body))
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"received": true}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		BackendHost: server.URL,
		Timeout:     5 * time.Second,
	}
	client := NewHTTPClient(cfg)

	originalReq := httptest.NewRequest("POST", "/test", strings.NewReader(requestBody))
	originalReq.Header.Set("Content-Type", "application/json")

	ctx := context.Background()
	resp, err := client.ProxyRequest(ctx, originalReq, "/test")

	if err != nil {
		t.Fatalf("ProxyRequest failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestProxyRequest_NetworkError(t *testing.T) {
	cfg := &config.Config{
		BackendHost: "http://nonexistent.example.com",
		Timeout:     1 * time.Second,
	}
	client := NewHTTPClient(cfg)

	originalReq := httptest.NewRequest("GET", "/test", nil)
	ctx := context.Background()

	_, err := client.ProxyRequest(ctx, originalReq, "/test")

	if err == nil {
		t.Fatal("Expected network error, got nil")
	}

	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Errorf("Expected NetworkError, got %T: %v", err, err)
	}

	if !IsNetworkError(err) {
		t.Error("IsNetworkError should return true for network errors")
	}
}

func TestGetTokens_Success(t *testing.T) {
	expectedTokens := models.TokenResponse{
		Items: []models.Token{
			{
				Address:     "0x1234567890abcdef",
				AddressHash: "hash1",
				Name:        "Test Token",
				Symbol:      "TEST",
				Decimals:    "18",
				Type:        "ERC-20",
			},
			{
				Address:     "0xabcdef1234567890",
				AddressHash: "hash2",
				Name:        "Another Token",
				Symbol:      "ANOTHER",
				Decimals:    "8",
				Type:        "ERC-20",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/tokens" {
			t.Errorf("Expected path /api/v2/tokens, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		
		// Check headers
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Expected Accept header to be application/json")
		}
		if r.Header.Get("User-Agent") != "go-api-proxy/1.0" {
			t.Error("Expected User-Agent header to be go-api-proxy/1.0")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedTokens)
	}))
	defer server.Close()

	cfg := &config.Config{
		BackendHost: server.URL,
		Timeout:     5 * time.Second,
	}
	client := NewHTTPClient(cfg)

	ctx := context.Background()
	tokens, err := client.GetTokens(ctx)

	if err != nil {
		t.Fatalf("GetTokens failed: %v", err)
	}

	if len(tokens.Items) != len(expectedTokens.Items) {
		t.Errorf("Expected %d tokens, got %d", len(expectedTokens.Items), len(tokens.Items))
	}

	for i, token := range tokens.Items {
		expected := expectedTokens.Items[i]
		if token.Address != expected.Address {
			t.Errorf("Token %d: expected address %s, got %s", i, expected.Address, token.Address)
		}
		if token.Name != expected.Name {
			t.Errorf("Token %d: expected name %s, got %s", i, expected.Name, token.Name)
		}
	}
}

func TestGetTokens_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		BackendHost: server.URL,
		Timeout:     5 * time.Second,
	}
	client := NewHTTPClient(cfg)

	ctx := context.Background()
	_, err := client.GetTokens(ctx)

	if err == nil {
		t.Fatal("Expected API error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("Expected APIError, got %T: %v", err, err)
	}

	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code 500, got %d", apiErr.StatusCode)
	}

	if !IsAPIError(err) {
		t.Error("IsAPIError should return true for API errors")
	}
}

func TestGetTokens_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	cfg := &config.Config{
		BackendHost: server.URL,
		Timeout:     5 * time.Second,
	}
	client := NewHTTPClient(cfg)

	ctx := context.Background()
	_, err := client.GetTokens(ctx)

	if err == nil {
		t.Fatal("Expected JSON parsing error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse tokens response") {
		t.Errorf("Expected JSON parsing error, got: %v", err)
	}
}

func TestForwardHeaders(t *testing.T) {
	cfg := &config.Config{
		BackendHost: "https://example.com",
		Timeout:     5 * time.Second,
	}
	client := NewHTTPClient(cfg)

	originalReq := httptest.NewRequest("GET", "/test", nil)
	originalReq.Header.Set("Accept", "application/json")
	originalReq.Header.Set("User-Agent", "test-agent")
	originalReq.Header.Set("X-Custom-Header", "should-not-forward")
	originalReq.RemoteAddr = "192.168.1.1:12345"

	backendReq := httptest.NewRequest("GET", "https://example.com/test", nil)
	client.forwardHeaders(originalReq, backendReq)

	// Check forwarded headers
	if backendReq.Header.Get("Accept") != "application/json" {
		t.Error("Accept header not forwarded")
	}
	if backendReq.Header.Get("User-Agent") != "test-agent" {
		t.Error("User-Agent header not forwarded")
	}
	if backendReq.Header.Get("X-Custom-Header") != "" {
		t.Error("Custom header should not be forwarded")
	}
	if backendReq.Header.Get("X-Forwarded-For") != "192.168.1.1" {
		t.Errorf("Expected X-Forwarded-For to be 192.168.1.1, got %s", backendReq.Header.Get("X-Forwarded-For"))
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedIP     string
	}{
		{
			name: "X-Forwarded-For header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
				return req
			},
			expectedIP: "192.168.1.1",
		},
		{
			name: "X-Real-IP header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "192.168.1.2")
				return req
			},
			expectedIP: "192.168.1.2",
		},
		{
			name: "RemoteAddr",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.168.1.3:12345"
				return req
			},
			expectedIP: "192.168.1.3",
		},
		{
			name: "No IP available",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "" // Clear the default RemoteAddr
				return req
			},
			expectedIP: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		target      interface{}
		expectError bool
	}{
		{
			name:        "Valid JSON",
			data:        []byte(`{"name": "test", "value": 123}`),
			target:      &map[string]interface{}{},
			expectError: false,
		},
		{
			name:        "Empty data",
			data:        []byte{},
			target:      &map[string]interface{}{},
			expectError: true,
		},
		{
			name:        "Invalid JSON",
			data:        []byte(`{invalid json`),
			target:      &map[string]interface{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseJSON(tt.data, tt.target)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestNetworkError(t *testing.T) {
	originalErr := fmt.Errorf("connection refused")
	netErr := &NetworkError{
		Operation: "test_operation",
		URL:       "https://example.com",
		Err:       originalErr,
	}

	expectedMsg := "network error during test_operation to https://example.com: connection refused"
	if netErr.Error() != expectedMsg {
		t.Errorf("Expected error message %s, got %s", expectedMsg, netErr.Error())
	}

	if netErr.Unwrap() != originalErr {
		t.Error("Unwrap should return the original error")
	}
}

func TestAPIError(t *testing.T) {
	apiErr := &APIError{
		StatusCode: 404,
		Status:     "404 Not Found",
		URL:        "https://example.com/api/test",
	}

	expectedMsg := "API error: 404 Not Found (404) from https://example.com/api/test"
	if apiErr.Error() != expectedMsg {
		t.Errorf("Expected error message %s, got %s", expectedMsg, apiErr.Error())
	}
}
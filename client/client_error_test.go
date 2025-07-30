package client

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go-api-proxy/config"
)

func TestNetworkErrorDetection(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "NetworkError type",
			err:      &NetworkError{Operation: "test", URL: "http://test.com", Err: errors.New("connection failed")},
			expected: true,
		},
		{
			name:     "net.OpError",
			err:      &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			expected: true,
		},
		{
			name:     "url.Error",
			err:      &url.Error{Op: "Get", URL: "http://test.com", Err: errors.New("timeout")},
			expected: true,
		},
		{
			name:     "DNS error",
			err:      &net.DNSError{Err: "no such host", Name: "invalid.domain"},
			expected: true,
		},
		{
			name:     "timeout error message",
			err:      errors.New("context deadline exceeded (Client.Timeout exceeded while awaiting headers)"),
			expected: true,
		},
		{
			name:     "connection refused message",
			err:      errors.New("dial tcp 127.0.0.1:8080: connect: connection refused"),
			expected: true,
		},
		{
			name:     "no such host message",
			err:      errors.New("dial tcp: lookup invalid.domain: no such host"),
			expected: true,
		},
		{
			name:     "network unreachable message",
			err:      errors.New("dial tcp 192.168.1.1:80: connect: network is unreachable"),
			expected: true,
		},
		{
			name:     "broken pipe message",
			err:      errors.New("write tcp 127.0.0.1:8080: write: broken pipe"),
			expected: true,
		},
		{
			name:     "i/o timeout message",
			err:      errors.New("read tcp 127.0.0.1:8080: i/o timeout"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsNetworkError(test.err)
			if result != test.expected {
				t.Errorf("Expected %v, got %v for error: %v", test.expected, result, test.err)
			}
		})
	}
}

func TestAPIErrorDetection(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "APIError type",
			err:      &APIError{StatusCode: 500, Status: "Internal Server Error", URL: "http://test.com"},
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "NetworkError type",
			err:      &NetworkError{Operation: "test", URL: "http://test.com", Err: errors.New("connection failed")},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsAPIError(test.err)
			if result != test.expected {
				t.Errorf("Expected %v, got %v for error: %v", test.expected, result, test.err)
			}
		})
	}
}

func TestHTTPClientErrorHandling(t *testing.T) {
	// Test server that returns different error scenarios
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scenario := r.URL.Query().Get("scenario")
		
		// Handle tokens endpoint specifically
		if r.URL.Path == "/tokens" {
			switch scenario {
			case "500":
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			case "404":
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "not found"}`))
			case "invalid_json":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`invalid json response`))
			case "empty_response":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(``))
			case "success":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"items": []}`))
			default:
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"items": []}`))
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		BackendHost:   server.URL,
		Port:          "8080",
		WhitelistFile: "test.json",
		Timeout:       5 * time.Second,
	}

	client := NewHTTPClient(cfg)
	ctx := context.Background()

	t.Run("API Error 500", func(t *testing.T) {
		// Create a separate test server for this specific scenario
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/tokens" {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer testServer.Close()

		originalURL := client.backendURL
		client.backendURL = testServer.URL
		defer func() { client.backendURL = originalURL }()

		_, err := client.GetTokens(ctx)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !IsAPIError(err) {
			t.Errorf("Expected API error, got: %v", err)
		}

		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("Expected *APIError, got %T", err)
		}

		if apiErr.StatusCode != 500 {
			t.Errorf("Expected status code 500, got %d", apiErr.StatusCode)
		}
	})

	t.Run("API Error 404", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/tokens" {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "not found"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer testServer.Close()

		originalURL := client.backendURL
		client.backendURL = testServer.URL
		defer func() { client.backendURL = originalURL }()

		_, err := client.GetTokens(ctx)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !IsAPIError(err) {
			t.Errorf("Expected API error, got: %v", err)
		}
	})

	t.Run("Invalid JSON Response", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/tokens" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`invalid json response`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer testServer.Close()

		originalURL := client.backendURL
		client.backendURL = testServer.URL
		defer func() { client.backendURL = originalURL }()

		_, err := client.GetTokens(ctx)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "invalid JSON") {
			t.Errorf("Expected JSON parsing error, got: %v", err)
		}
	})

	t.Run("Empty Response", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/tokens" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(``))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer testServer.Close()

		originalURL := client.backendURL
		client.backendURL = testServer.URL
		defer func() { client.backendURL = originalURL }()

		_, err := client.GetTokens(ctx)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !strings.Contains(err.Error(), "empty response body") {
			t.Errorf("Expected empty response error, got: %v", err)
		}
	})

	t.Run("Network Error - Invalid Host", func(t *testing.T) {
		originalURL := client.backendURL
		client.backendURL = "http://invalid.domain.that.does.not.exist.12345"
		defer func() { client.backendURL = originalURL }()

		_, err := client.GetTokens(ctx)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !IsNetworkError(err) {
			t.Errorf("Expected network error, got: %v", err)
		}
	})

	t.Run("Context Timeout", func(t *testing.T) {
		// Create a context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for context to timeout
		time.Sleep(1 * time.Millisecond)

		_, err := client.GetTokens(ctx)
		if err == nil {
			t.Fatal("Expected timeout error, got nil")
		}

		if !IsNetworkError(err) {
			t.Errorf("Expected network error for timeout, got: %v", err)
		}
	})
}

func TestProxyRequestErrorHandling(t *testing.T) {
	// Test server for proxy requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("scenario") {
		case "slow":
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result": "success"}`))
		case "error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"result": "success"}`))
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		BackendHost:   server.URL,
		Port:          "8080",
		WhitelistFile: "test.json",
		Timeout:       5 * time.Second,
	}

	client := NewHTTPClient(cfg)

	t.Run("Successful Proxy Request", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		ctx := context.Background()

		resp, err := client.ProxyRequest(ctx, req, "/test")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Proxy Request with Context Timeout", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Wait for context to timeout
		time.Sleep(1 * time.Millisecond)

		_, err := client.ProxyRequest(ctx, req, "/test?scenario=slow")
		if err == nil {
			t.Fatal("Expected timeout error, got nil")
		}

		if !IsNetworkError(err) {
			t.Errorf("Expected network error for timeout, got: %v", err)
		}
	})

	t.Run("Proxy Request to Invalid Host", func(t *testing.T) {
		originalURL := client.backendURL
		client.backendURL = "http://invalid.domain.12345"
		defer func() { client.backendURL = originalURL }()

		req, _ := http.NewRequest("GET", "/test", nil)
		ctx := context.Background()

		_, err := client.ProxyRequest(ctx, req, "/test")
		if err == nil {
			t.Fatal("Expected network error, got nil")
		}

		if !IsNetworkError(err) {
			t.Errorf("Expected network error, got: %v", err)
		}
	})
}

func TestErrorTypes(t *testing.T) {
	t.Run("NetworkError", func(t *testing.T) {
		originalErr := errors.New("connection failed")
		netErr := &NetworkError{
			Operation: "test_operation",
			URL:       "http://test.com",
			Err:       originalErr,
		}

		expectedMsg := "network error during test_operation to http://test.com: connection failed"
		if netErr.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, netErr.Error())
		}

		if netErr.Unwrap() != originalErr {
			t.Errorf("Expected unwrapped error to be original error")
		}
	})

	t.Run("APIError", func(t *testing.T) {
		apiErr := &APIError{
			StatusCode: 500,
			Status:     "Internal Server Error",
			URL:        "http://test.com/api",
		}

		expectedMsg := "API error: Internal Server Error (500) from http://test.com/api"
		if apiErr.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, apiErr.Error())
		}
	})
}
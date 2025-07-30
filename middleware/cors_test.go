package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSHandler_ServeHTTP(t *testing.T) {
	// Create a simple next handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create CORS handler
	corsHandler := NewCORSHandler(nextHandler)

	tests := []struct {
		name           string
		method         string
		origin         string
		requestHeaders string
		expectedStatus int
		checkHeaders   map[string]string
	}{
		{
			name:           "GET request with origin",
			method:         "GET",
			origin:         "https://example.com",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "https://example.com",
				"Access-Control-Allow-Credentials": "true",
			},
		},
		{
			name:           "GET request without origin",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		},
		{
			name:           "OPTIONS preflight request",
			method:         "OPTIONS",
			origin:         "https://example.com",
			requestHeaders: "Content-Type, Authorization",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "https://example.com",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS, HEAD, PATCH",
				"Access-Control-Max-Age":       "86400",
			},
		},
		{
			name:           "POST request with custom headers",
			method:         "POST",
			origin:         "https://app.example.com",
			requestHeaders: "X-Custom-Header",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://app.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(tt.method, "/test", nil)
			
			// Add request ID to context
			ctx := context.WithValue(req.Context(), "request_id", "test-123")
			req = req.WithContext(ctx)
			
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.requestHeaders != "" {
				req.Header.Set("Access-Control-Request-Headers", tt.requestHeaders)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			corsHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check headers
			for header, expectedValue := range tt.checkHeaders {
				actualValue := rr.Header().Get(header)
				if actualValue != expectedValue {
					t.Errorf("Expected header %s: %s, got: %s", header, expectedValue, actualValue)
				}
			}

			// Check that CORS headers are always present
			requiredHeaders := []string{
				"Access-Control-Allow-Origin",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
				"Access-Control-Max-Age",
			}

			for _, header := range requiredHeaders {
				if rr.Header().Get(header) == "" {
					t.Errorf("Required CORS header %s is missing", header)
				}
			}
		})
	}
}

func TestCORSHandler_NewCORSHandler(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	corsHandler := NewCORSHandler(nextHandler)

	if corsHandler == nil {
		t.Error("NewCORSHandler should not return nil")
	}

	if corsHandler.next != nextHandler {
		t.Error("NewCORSHandler should set the next handler correctly")
	}
}

func TestContains(t *testing.T) {
	slice := []string{"Content-Type", "Authorization", "Accept"}

	tests := []struct {
		item     string
		expected bool
	}{
		{"Content-Type", true},
		{"content-type", true},
		{"CONTENT-TYPE", true},
		{"Authorization", true},
		{"X-Custom", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			result := contains(slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %s) = %v, expected %v", slice, tt.item, result, tt.expected)
			}
		})
	}
}
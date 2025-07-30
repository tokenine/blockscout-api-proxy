package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-api-proxy/client"
)

// MockProxyClient implements ProxyClientInterface for testing
type MockProxyClient struct {
	response *http.Response
	err      error
}

func (m *MockProxyClient) ProxyRequest(ctx context.Context, originalReq *http.Request, endpoint string) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestNewStandardProxyHandler(t *testing.T) {
	mockClient := &MockProxyClient{}
	handler := NewStandardProxyHandler(mockClient)
	
	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}
	
	if handler.httpClient != mockClient {
		t.Error("Expected handler to use provided client")
	}
}

func TestStandardProxyHandler_ServeHTTP_Success(t *testing.T) {
	// Create a mock response
	responseBody := `{"message": "success"}`
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Cache-Control":  []string{"no-cache"},
			"Custom-Header":  []string{"custom-value"},
		},
		Body: io.NopCloser(strings.NewReader(responseBody)),
	}
	
	mockClient := &MockProxyClient{response: mockResp}
	handler := NewStandardProxyHandler(mockClient)
	
	// Create test request
	req := httptest.NewRequest("GET", "/api/v2/users", nil)
	req.Header.Set("Authorization", "Bearer token")
	
	// Create response recorder
	w := httptest.NewRecorder()
	
	// Execute handler
	handler.ServeHTTP(w, req)
	
	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	if w.Body.String() != responseBody {
		t.Errorf("Expected body %q, got %q", responseBody, w.Body.String())
	}
	
	// Verify headers are copied
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type header to be copied")
	}
	
	if w.Header().Get("Custom-Header") != "custom-value" {
		t.Error("Expected custom header to be copied")
	}
}

func TestStandardProxyHandler_ServeHTTP_WithQueryParams(t *testing.T) {
	responseBody := `{"data": []}`
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}
	
	mockClient := &MockProxyClient{response: mockResp}
	handler := NewStandardProxyHandler(mockClient)
	
	// Create test request with query parameters
	req := httptest.NewRequest("GET", "/api/v2/users?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	if w.Body.String() != responseBody {
		t.Errorf("Expected body %q, got %q", responseBody, w.Body.String())
	}
}

func TestStandardProxyHandler_ServeHTTP_NetworkError(t *testing.T) {
	networkErr := &client.NetworkError{
		Operation: "backend_request",
		URL:       "https://example.com/api/v2/users",
		Err:       errors.New("connection refused"),
	}
	
	mockClient := &MockProxyClient{err: networkErr}
	handler := NewStandardProxyHandler(mockClient)
	
	req := httptest.NewRequest("GET", "/api/v2/users", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, w.Code)
	}
	
	if !strings.Contains(w.Body.String(), "Backend API unreachable") {
		t.Error("Expected error message about backend being unreachable")
	}
}

func TestStandardProxyHandler_ServeHTTP_APIError(t *testing.T) {
	apiErr := &client.APIError{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		URL:        "https://example.com/api/v2/users",
	}
	
	mockClient := &MockProxyClient{err: apiErr}
	handler := NewStandardProxyHandler(mockClient)
	
	req := httptest.NewRequest("GET", "/api/v2/users", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, w.Code)
	}
	
	if !strings.Contains(w.Body.String(), "Backend API error") {
		t.Error("Expected error message about backend API error")
	}
}

func TestStandardProxyHandler_ServeHTTP_GenericError(t *testing.T) {
	genericErr := errors.New("some generic error")
	
	mockClient := &MockProxyClient{err: genericErr}
	handler := NewStandardProxyHandler(mockClient)
	
	req := httptest.NewRequest("GET", "/api/v2/users", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
	
	if !strings.Contains(w.Body.String(), "Internal Server Error") {
		t.Error("Expected generic internal server error message")
	}
}

func TestStandardProxyHandler_CopyHeaders(t *testing.T) {
	handler := &StandardProxyHandler{}
	
	// Create source headers
	src := http.Header{
		"Content-Type":       []string{"application/json"},
		"Cache-Control":      []string{"no-cache"},
		"Connection":         []string{"keep-alive"}, // hop-by-hop header
		"Transfer-Encoding":  []string{"chunked"},    // hop-by-hop header
		"Custom-Header":      []string{"value1", "value2"},
	}
	
	// Create destination headers
	dst := http.Header{}
	
	// Copy headers
	handler.copyHeaders(src, dst)
	
	// Verify regular headers are copied
	if dst.Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type to be copied")
	}
	
	if dst.Get("Cache-Control") != "no-cache" {
		t.Error("Expected Cache-Control to be copied")
	}
	
	// Verify multiple values are copied
	customValues := dst["Custom-Header"]
	if len(customValues) != 2 || customValues[0] != "value1" || customValues[1] != "value2" {
		t.Error("Expected multiple custom header values to be copied")
	}
	
	// Verify hop-by-hop headers are NOT copied
	if dst.Get("Connection") != "" {
		t.Error("Expected Connection header to be filtered out")
	}
	
	if dst.Get("Transfer-Encoding") != "" {
		t.Error("Expected Transfer-Encoding header to be filtered out")
	}
}

func TestStandardProxyHandler_IsHopByHopHeader(t *testing.T) {
	handler := &StandardProxyHandler{}
	
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
	
	for _, header := range hopByHopHeaders {
		if !handler.isHopByHopHeader(header) {
			t.Errorf("Expected %s to be identified as hop-by-hop header", header)
		}
	}
	
	regularHeaders := []string{
		"Content-Type",
		"Cache-Control",
		"Authorization",
		"Accept",
		"User-Agent",
	}
	
	for _, header := range regularHeaders {
		if handler.isHopByHopHeader(header) {
			t.Errorf("Expected %s to NOT be identified as hop-by-hop header", header)
		}
	}
}

func TestStandardProxyHandler_ServeHTTP_DifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			responseBody := `{"method": "` + method + `"}`
			mockResp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(responseBody)),
			}
			
			mockClient := &MockProxyClient{response: mockResp}
			handler := NewStandardProxyHandler(mockClient)
			
			req := httptest.NewRequest(method, "/api/v2/test", nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d for %s, got %d", http.StatusOK, method, w.Code)
			}
			
			if w.Body.String() != responseBody {
				t.Errorf("Expected body %q for %s, got %q", responseBody, method, w.Body.String())
			}
		})
	}
}

func TestStandardProxyHandler_ServeHTTP_DifferentStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}
	
	for _, statusCode := range statusCodes {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			mockResp := &http.Response{
				StatusCode: statusCode,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}
			
			mockClient := &MockProxyClient{response: mockResp}
			handler := NewStandardProxyHandler(mockClient)
			
			req := httptest.NewRequest("GET", "/api/v2/test", nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			if w.Code != statusCode {
				t.Errorf("Expected status %d, got %d", statusCode, w.Code)
			}
		})
	}
}
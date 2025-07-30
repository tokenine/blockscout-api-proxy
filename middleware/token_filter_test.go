package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-api-proxy/client"
	"go-api-proxy/config"
	"go-api-proxy/logger"
	"go-api-proxy/models"
)

// mockHTTPClient implements a mock HTTP client for testing
type mockHTTPClient struct {
	tokenResponse *models.TokenResponse
	err           error
}

func (m *mockHTTPClient) GetTokens(ctx context.Context) (*models.TokenResponse, error) {
	return m.tokenResponse, m.err
}

func (m *mockHTTPClient) ProxyRequest(ctx context.Context, originalReq *http.Request, endpoint string) (*http.Response, error) {
	return nil, errors.New("not implemented in mock")
}

func TestTokenFilterHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		tokenResponse  *models.TokenResponse
		whitelistAddrs []string
		clientError    error
		expectedStatus int
		expectedCount  int
		expectError    bool
	}{
		{
			name: "successful filtering with matches",
			tokenResponse: &models.TokenResponse{
				Items: []models.Token{
					{Address: "0x1234", Name: "Token1", Symbol: "TK1"},
					{Address: "0x5678", Name: "Token2", Symbol: "TK2"},
					{Address: "0x9abc", Name: "Token3", Symbol: "TK3"},
				},
			},
			whitelistAddrs: []string{"0x1234", "0x9abc"},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectError:    false,
		},
		{
			name: "successful filtering with no matches",
			tokenResponse: &models.TokenResponse{
				Items: []models.Token{
					{Address: "0x1234", Name: "Token1", Symbol: "TK1"},
					{Address: "0x5678", Name: "Token2", Symbol: "TK2"},
				},
			},
			whitelistAddrs: []string{"0xaaaa", "0xbbbb"},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
			expectError:    false,
		},
		{
			name: "empty whitelist returns all tokens",
			tokenResponse: &models.TokenResponse{
				Items: []models.Token{
					{Address: "0x1234", Name: "Token1", Symbol: "TK1"},
					{Address: "0x5678", Name: "Token2", Symbol: "TK2"},
				},
			},
			whitelistAddrs: []string{},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectError:    false,
		},
		{
			name:           "empty token response",
			tokenResponse:  &models.TokenResponse{Items: []models.Token{}},
			whitelistAddrs: []string{"0x1234"},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
			expectError:    false,
		},
		{
			name:           "network error from backend",
			tokenResponse:  nil,
			whitelistAddrs: []string{"0x1234"},
			clientError:    &client.NetworkError{Operation: "get_tokens", URL: "test", Err: errors.New("connection failed")},
			expectedStatus: http.StatusBadGateway,
			expectedCount:  0,
			expectError:    true,
		},
		{
			name:           "API error from backend",
			tokenResponse:  nil,
			whitelistAddrs: []string{"0x1234"},
			clientError:    &client.APIError{StatusCode: 500, Status: "Internal Server Error", URL: "test"},
			expectedStatus: http.StatusBadGateway,
			expectedCount:  0,
			expectError:    true,
		},
		{
			name:           "generic error from backend",
			tokenResponse:  nil,
			whitelistAddrs: []string{"0x1234"},
			clientError:    errors.New("generic error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client
			mockClient := &mockHTTPClient{
				tokenResponse: tt.tokenResponse,
				err:           tt.clientError,
			}

			// Setup whitelist
			whitelist := models.NewTokenWhitelist()
			for _, addr := range tt.whitelistAddrs {
				whitelist.AddAddress(addr)
			}

			// Create handler
			handler := NewTokenFilterHandler(mockClient, whitelist)

			// Create test request
			req := httptest.NewRequest("GET", "/api/v2/tokens", nil)
			w := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}

			// Parse response
			if tt.expectError {
				var errorResp models.ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
					t.Errorf("Failed to decode error response: %v", err)
				}
				if errorResp.Error == "" {
					t.Error("Expected error message in response")
				}
			} else {
				var tokenResp models.TokenResponse
				if err := json.NewDecoder(w.Body).Decode(&tokenResp); err != nil {
					t.Errorf("Failed to decode token response: %v", err)
				}
				if len(tokenResp.Items) != tt.expectedCount {
					t.Errorf("Expected %d tokens, got %d", tt.expectedCount, len(tokenResp.Items))
				}
			}
		})
	}
}

func TestTokenFilterHandler_filterTokens(t *testing.T) {
	tests := []struct {
		name           string
		input          *models.TokenResponse
		whitelistAddrs []string
		expectedCount  int
		expectedAddrs  []string
	}{
		{
			name: "filter with multiple matches",
			input: &models.TokenResponse{
				Items: []models.Token{
					{Address: "0x1111", Name: "Token1"},
					{Address: "0x2222", Name: "Token2"},
					{Address: "0x3333", Name: "Token3"},
					{Address: "0x4444", Name: "Token4"},
				},
			},
			whitelistAddrs: []string{"0x1111", "0x3333"},
			expectedCount:  2,
			expectedAddrs:  []string{"0x1111", "0x3333"},
		},
		{
			name: "filter with no matches",
			input: &models.TokenResponse{
				Items: []models.Token{
					{Address: "0x1111", Name: "Token1"},
					{Address: "0x2222", Name: "Token2"},
				},
			},
			whitelistAddrs: []string{"0x9999", "0x8888"},
			expectedCount:  0,
			expectedAddrs:  []string{},
		},
		{
			name: "filter with all matches",
			input: &models.TokenResponse{
				Items: []models.Token{
					{Address: "0x1111", Name: "Token1"},
					{Address: "0x2222", Name: "Token2"},
				},
			},
			whitelistAddrs: []string{"0x1111", "0x2222"},
			expectedCount:  2,
			expectedAddrs:  []string{"0x1111", "0x2222"},
		},
		{
			name:           "nil input response",
			input:          nil,
			whitelistAddrs: []string{"0x1111"},
			expectedCount:  0,
			expectedAddrs:  []string{},
		},
		{
			name: "empty input response",
			input: &models.TokenResponse{
				Items: []models.Token{},
			},
			whitelistAddrs: []string{"0x1111"},
			expectedCount:  0,
			expectedAddrs:  []string{},
		},
		{
			name: "empty whitelist returns all tokens",
			input: &models.TokenResponse{
				Items: []models.Token{
					{Address: "0x1111", Name: "Token1"},
					{Address: "0x2222", Name: "Token2"},
				},
			},
			whitelistAddrs: []string{},
			expectedCount:  2,
			expectedAddrs:  []string{"0x1111", "0x2222"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup whitelist
			whitelist := models.NewTokenWhitelist()
			for _, addr := range tt.whitelistAddrs {
				whitelist.AddAddress(addr)
			}

			// Create handler
			handler := NewTokenFilterHandler(nil, whitelist)

			// Execute filtering
			mockLogger := logger.NewLogger("test")
			result := handler.filterTokens(tt.input, mockLogger)

			// Check result count
			if len(result.Items) != tt.expectedCount {
				t.Errorf("Expected %d tokens, got %d", tt.expectedCount, len(result.Items))
			}

			// Check specific addresses if expected
			if len(tt.expectedAddrs) > 0 {
				resultAddrs := make([]string, len(result.Items))
				for i, token := range result.Items {
					resultAddrs[i] = token.Address
				}

				for _, expectedAddr := range tt.expectedAddrs {
					found := false
					for _, resultAddr := range resultAddrs {
						if resultAddr == expectedAddr {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected address %s not found in results", expectedAddr)
					}
				}
			}
		})
	}
}

func TestTokenFilterHandler_handleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "network error",
			err:            &client.NetworkError{Operation: "test", URL: "test", Err: errors.New("connection failed")},
			expectedStatus: http.StatusBadGateway,
		},
		{
			name:           "API error",
			err:            &client.APIError{StatusCode: 500, Status: "Internal Server Error", URL: "test"},
			expectedStatus: http.StatusBadGateway,
		},
		{
			name:           "generic error",
			err:            errors.New("generic error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler
			handler := NewTokenFilterHandler(nil, models.NewTokenWhitelist())

			// Create test response recorder
			w := httptest.NewRecorder()

			// Execute error handling
			req := httptest.NewRequest("GET", "/api/v2/tokens", nil)
			handler.handleError(w, req, tt.err)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}

			// Check error response format
			var errorResp models.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
				t.Errorf("Failed to decode error response: %v", err)
			}

			if errorResp.Error == "" {
				t.Error("Expected error message in response")
			}
		})
	}
}

func TestNewTokenFilterHandler(t *testing.T) {
	// Create test dependencies
	cfg, _ := config.Load()
	httpClient := client.NewHTTPClient(cfg)
	whitelist := models.NewTokenWhitelist()

	// Create handler
	handler := NewTokenFilterHandler(httpClient, whitelist)

	// Verify handler is created correctly
	if handler == nil {
		t.Error("Expected handler to be created, got nil")
	}

	if handler.httpClient != httpClient {
		t.Error("Expected httpClient to be set correctly")
	}

	if handler.whitelist != whitelist {
		t.Error("Expected whitelist to be set correctly")
	}
}

// Integration test with real HTTP client (using mock server)
func TestTokenFilterHandler_Integration(t *testing.T) {
	// Create mock backend server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tokens" {
			response := models.TokenResponse{
				Items: []models.Token{
					{Address: "0x1234", Name: "Token1", Symbol: "TK1"},
					{Address: "0x5678", Name: "Token2", Symbol: "TK2"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create config pointing to mock server
	cfg := &config.Config{
		BackendHost:   mockServer.URL,
		Port:          "8080",
		WhitelistFile: "test_whitelist.json",
		Timeout:       30 * time.Second,
	}

	// Create HTTP client
	httpClient := client.NewHTTPClient(cfg)

	// Create whitelist with one matching address
	whitelist := models.NewTokenWhitelist()
	whitelist.AddAddress("0x1234")

	// Create handler
	handler := NewTokenFilterHandler(httpClient, whitelist)

	// Create test request
	req := httptest.NewRequest("GET", "/api/v2/tokens", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response models.TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	// Should only return the whitelisted token
	if len(response.Items) != 1 {
		t.Errorf("Expected 1 token, got %d", len(response.Items))
		return
	}

	if response.Items[0].Address != "0x1234" {
		t.Errorf("Expected address 0x1234, got %s", response.Items[0].Address)
	}
}
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-api-proxy/config"
	"go-api-proxy/logger"
	"go-api-proxy/models"
)

// HTTPClient wraps the standard HTTP client with proxy-specific functionality
type HTTPClient struct {
	client      *http.Client
	backendURL  string
	config      *config.Config
}

// NewHTTPClient creates a new HTTP client configured for backend communication
func NewHTTPClient(cfg *config.Config) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		backendURL: cfg.GetBackendAPIURL(),
		config:     cfg,
	}
}

// ProxyRequest forwards an HTTP request to the backend API
func (c *HTTPClient) ProxyRequest(ctx context.Context, originalReq *http.Request, endpoint string) (*http.Response, error) {
	// Construct the full backend URL
	targetURL := c.backendURL + endpoint
	
	requestID := getRequestIDFromContext(ctx)
	clientLogger := logger.ClientLogger.WithRequestID(requestID)
	
	clientLogger.Debug("Proxying request to backend", map[string]interface{}{
		"method":     originalReq.Method,
		"endpoint":   endpoint,
		"target_url": targetURL,
	})
	
	// Create new request with the same method and body
	var body io.Reader
	if originalReq.Body != nil {
		bodyBytes, err := io.ReadAll(originalReq.Body)
		if err != nil {
			clientLogger.Error("Failed to read request body", err)
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		body = bytes.NewReader(bodyBytes)
		// Reset original request body for potential reuse
		originalReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	
	req, err := http.NewRequestWithContext(ctx, originalReq.Method, targetURL, body)
	if err != nil {
		clientLogger.Error("Failed to create backend request", err)
		return nil, fmt.Errorf("failed to create backend request: %w", err)
	}
	
	// Forward headers from original request
	c.forwardHeaders(originalReq, req)
	
	// Make the request to backend
	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)
	
	if err != nil {
		clientLogger.Error("Backend request failed", err, map[string]interface{}{
			"target_url": targetURL,
			"duration":   duration.String(),
		})
		return nil, &NetworkError{
			Operation: "backend_request",
			URL:       targetURL,
			Err:       err,
		}
	}
	
	clientLogger.Info("Backend request completed", map[string]interface{}{
		"status_code": resp.StatusCode,
		"duration":    duration.String(),
		"target_url":  targetURL,
	})
	
	return resp, nil
}

// GetTokens fetches tokens from the backend API
func (c *HTTPClient) GetTokens(ctx context.Context) (*models.TokenResponse, error) {
	targetURL := c.backendURL + "/tokens"
	
	requestID := getRequestIDFromContext(ctx)
	clientLogger := logger.ClientLogger.WithRequestID(requestID)
	
	clientLogger.Debug("Fetching tokens from backend", map[string]interface{}{
		"target_url": targetURL,
	})
	
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		clientLogger.Error("Failed to create tokens request", err)
		return nil, fmt.Errorf("failed to create tokens request: %w", err)
	}
	
	// Set standard headers for API requests
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "go-api-proxy/1.0")
	
	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)
	
	if err != nil {
		clientLogger.Error("Failed to fetch tokens from backend", err, map[string]interface{}{
			"target_url": targetURL,
			"duration":   duration.String(),
		})
		return nil, &NetworkError{
			Operation: "get_tokens",
			URL:       targetURL,
			Err:       err,
		}
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		clientLogger.Error("Backend returned non-200 status for tokens", nil, map[string]interface{}{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
			"target_url":  targetURL,
			"duration":    duration.String(),
		})
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			URL:        targetURL,
		}
	}
	
	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		clientLogger.Error("Failed to read tokens response body", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Parse JSON response
	var tokenResponse models.TokenResponse
	if err := parseJSON(bodyBytes, &tokenResponse); err != nil {
		clientLogger.Error("Failed to parse tokens JSON response", err, map[string]interface{}{
			"response_size": len(bodyBytes),
		})
		return nil, fmt.Errorf("failed to parse tokens response: %w", err)
	}
	
	clientLogger.Info("Successfully fetched and parsed tokens", map[string]interface{}{
		"token_count":   len(tokenResponse.Items),
		"response_size": len(bodyBytes),
		"duration":      duration.String(),
	})
	
	return &tokenResponse, nil
}

// forwardHeaders copies relevant headers from the original request to the backend request
func (c *HTTPClient) forwardHeaders(originalReq, backendReq *http.Request) {
	// Headers to forward to backend
	headersToForward := []string{
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Cache-Control",
		"Content-Type",
		"User-Agent",
		"X-Forwarded-For",
		"X-Real-IP",
	}
	
	for _, header := range headersToForward {
		if value := originalReq.Header.Get(header); value != "" {
			backendReq.Header.Set(header, value)
		}
	}
	
	// Add X-Forwarded-For header if not present
	if originalReq.Header.Get("X-Forwarded-For") == "" {
		if remoteAddr := getClientIP(originalReq); remoteAddr != "" {
			backendReq.Header.Set("X-Forwarded-For", remoteAddr)
		}
	}
	
	// Set our own User-Agent if none provided
	if originalReq.Header.Get("User-Agent") == "" {
		backendReq.Header.Set("User-Agent", "go-api-proxy/1.0")
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(req *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	if req.RemoteAddr != "" {
		// Remove port if present
		if idx := strings.LastIndex(req.RemoteAddr, ":"); idx != -1 {
			return req.RemoteAddr[:idx]
		}
		return req.RemoteAddr
	}
	
	return ""
}

// NetworkError represents a network-related error
type NetworkError struct {
	Operation string
	URL       string
	Err       error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error during %s to %s: %v", e.Operation, e.URL, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// APIError represents an HTTP API error response
type APIError struct {
	StatusCode int
	Status     string
	URL        string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: %s (%d) from %s", e.Status, e.StatusCode, e.URL)
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	
	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return true
	}
	
	// Check for various network-related errors
	var netOpErr *net.OpError
	var urlErr *url.Error
	var dnsErr *net.DNSError
	
	if errors.As(err, &netOpErr) || errors.As(err, &urlErr) || errors.As(err, &dnsErr) {
		return true
	}
	
	// Check error message for common network error patterns
	errStr := strings.ToLower(err.Error())
	networkKeywords := []string{
		"network", "timeout", "connection", "refused", "unreachable",
		"no such host", "dns", "dial", "i/o timeout", "broken pipe",
	}
	
	for _, keyword := range networkKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	
	return false
}

// IsAPIError checks if an error is an API error
func IsAPIError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr)
}

// parseJSON parses JSON data into the provided interface
func parseJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("empty response body")
	}
	
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	
	return nil
}

// getRequestIDFromContext extracts request ID from context
func getRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}
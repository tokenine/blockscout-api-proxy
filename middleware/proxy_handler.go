package middleware

import (
	"context"
	"io"
	"net/http"
	"time"

	"go-api-proxy/client"
	"go-api-proxy/logger"
)

// ProxyClientInterface defines the interface for proxy client operations
type ProxyClientInterface interface {
	ProxyRequest(ctx context.Context, originalReq *http.Request, endpoint string) (*http.Response, error)
}

// StandardProxyHandler handles requests for non-token endpoints by forwarding them to the backend
type StandardProxyHandler struct {
	httpClient ProxyClientInterface
}

// NewStandardProxyHandler creates a new standard proxy handler
func NewStandardProxyHandler(httpClient ProxyClientInterface) *StandardProxyHandler {
	return &StandardProxyHandler{
		httpClient: httpClient,
	}
}

// ServeHTTP implements the http.Handler interface for standard proxy functionality
func (h *StandardProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	
	requestID := getRequestIDFromContext(ctx)
	middlewareLogger := logger.MiddlewareLogger.WithRequestID(requestID)
	
	// Extract the endpoint path from the request URL
	endpoint := r.URL.Path
	if r.URL.RawQuery != "" {
		endpoint += "?" + r.URL.RawQuery
	}
	
	middlewareLogger.Debug("Processing standard proxy request", map[string]interface{}{
		"method":   r.Method,
		"endpoint": endpoint,
	})
	
	// Forward the request to the backend
	resp, err := h.httpClient.ProxyRequest(ctx, r, endpoint)
	if err != nil {
		middlewareLogger.Error("Failed to proxy request", err, map[string]interface{}{
			"endpoint": endpoint,
		})
		h.handleError(w, r, err)
		return
	}
	defer resp.Body.Close()
	
	// Copy response headers from backend to client
	h.copyHeaders(resp.Header, w.Header())
	
	// Set the status code
	w.WriteHeader(resp.StatusCode)
	
	// Copy response body from backend to client
	if _, err := io.Copy(w, resp.Body); err != nil {
		middlewareLogger.Error("Error copying response body", err, map[string]interface{}{
			"endpoint":    endpoint,
			"status_code": resp.StatusCode,
		})
		return
	}
	
	middlewareLogger.Info("Successfully proxied request", map[string]interface{}{
		"method":      r.Method,
		"endpoint":    endpoint,
		"status_code": resp.StatusCode,
	})
}

// copyHeaders copies headers from source to destination
func (h *StandardProxyHandler) copyHeaders(src, dst http.Header) {
	for key, values := range src {
		// Skip hop-by-hop headers that shouldn't be forwarded
		if h.isHopByHopHeader(key) {
			continue
		}
		
		// Skip CORS headers from backend - let our CORS middleware handle them
		if h.isCORSHeader(key) {
			continue
		}
		
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// isHopByHopHeader checks if a header is a hop-by-hop header that shouldn't be forwarded
func (h *StandardProxyHandler) isHopByHopHeader(header string) bool {
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
	
	for _, hopHeader := range hopByHopHeaders {
		if header == hopHeader {
			return true
		}
	}
	return false
}

// isCORSHeader checks if a header is a CORS header that should be handled by our CORS middleware
func (h *StandardProxyHandler) isCORSHeader(header string) bool {
	corsHeaders := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Credentials",
		"Access-Control-Expose-Headers",
		"Access-Control-Max-Age",
		"Access-Control-Request-Method",
		"Access-Control-Request-Headers",
	}
	
	for _, corsHeader := range corsHeaders {
		if header == corsHeader {
			return true
		}
	}
	return false
}

// handleError handles different types of errors and returns appropriate HTTP responses
func (h *StandardProxyHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := getRequestIDFromContext(r.Context())
	middlewareLogger := logger.MiddlewareLogger.WithRequestID(requestID)
	
	middlewareLogger.Error("Proxy handler error", err, map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	
	// Check if it's a network error (backend unreachable)
	if client.IsNetworkError(err) {
		middlewareLogger.Warn("Backend API unreachable", map[string]interface{}{
			"error_type": "network_error",
		})
		http.Error(w, "Bad Gateway: Backend API unreachable", http.StatusBadGateway)
		return
	}
	
	// Check if it's an API error from backend
	if client.IsAPIError(err) {
		middlewareLogger.Warn("Backend API error", map[string]interface{}{
			"error_type": "api_error",
		})
		http.Error(w, "Bad Gateway: Backend API error", http.StatusBadGateway)
		return
	}
	
	// Generic internal server error
	middlewareLogger.Error("Internal server error in proxy handler", err, map[string]interface{}{
		"error_type": "internal_error",
	})
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// getRequestIDFromContext extracts request ID from context
func getRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}
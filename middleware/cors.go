package middleware

import (
	"net/http"
	"strings"

	"go-api-proxy/logger"
)

// CORSHandler handles CORS (Cross-Origin Resource Sharing) for all requests
type CORSHandler struct {
	next http.Handler
}

// NewCORSHandler creates a new CORS middleware handler
func NewCORSHandler(next http.Handler) *CORSHandler {
	return &CORSHandler{
		next: next,
	}
}

// ServeHTTP implements the http.Handler interface for CORS middleware
func (c *CORSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get request ID from context for logging
	requestID, _ := r.Context().Value("request_id").(string)
	corsLogger := logger.MainLogger.WithRequestID(requestID)

	// Set CORS headers
	c.setCORSHeaders(w, r)
	
	// Add request ID to response headers for tracing
	if requestID != "" {
		w.Header().Set("X-Request-ID", requestID)
	}

	// Handle preflight OPTIONS request
	if r.Method == http.MethodOptions {
		corsLogger.Debug("Handling CORS preflight request", map[string]interface{}{
			"origin":  r.Header.Get("Origin"),
			"method":  r.Header.Get("Access-Control-Request-Method"),
			"headers": r.Header.Get("Access-Control-Request-Headers"),
		})
		
		w.WriteHeader(http.StatusOK)
		return
	}

	// Continue with the next handler
	c.next.ServeHTTP(w, r)
}

// setCORSHeaders sets the appropriate CORS headers
func (c *CORSHandler) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	
	// Allow all origins for now (you can restrict this in production)
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	
	// Allow credentials
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	
	// Allow common HTTP methods
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD, PATCH")
	
	// Allow common headers
	allowedHeaders := []string{
		"Accept",
		"Accept-Language",
		"Content-Type",
		"Content-Language",
		"Origin",
		"Authorization",
		"X-Requested-With",
		"X-HTTP-Method-Override",
		"Cache-Control",
		"Pragma",
	}
	
	// Add any additional headers from the request
	if requestHeaders := r.Header.Get("Access-Control-Request-Headers"); requestHeaders != "" {
		additionalHeaders := strings.Split(requestHeaders, ",")
		for _, header := range additionalHeaders {
			header = strings.TrimSpace(header)
			if header != "" && !contains(allowedHeaders, header) {
				allowedHeaders = append(allowedHeaders, header)
			}
		}
	}
	
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
	
	// Expose common headers to the client
	exposedHeaders := []string{
		"Content-Length",
		"Content-Type",
		"Date",
		"Server",
		"X-Request-ID",
		"Cache-Control",
		"Etag",
		"Last-Modified",
	}
	w.Header().Set("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
	
	// Set max age for preflight cache (24 hours)
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// contains checks if a slice contains a specific string (case-insensitive)
func contains(slice []string, item string) bool {
	item = strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == item {
			return true
		}
	}
	return false
}
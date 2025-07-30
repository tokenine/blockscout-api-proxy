package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go-api-proxy/client"
	"go-api-proxy/logger"
	"go-api-proxy/models"
)

// HTTPClientInterface defines the interface for HTTP client operations
type HTTPClientInterface interface {
	GetTokens(ctx context.Context) (*models.TokenResponse, error)
}

// TokenFilterHandler handles requests to /api/v2/tokens with whitelist filtering
type TokenFilterHandler struct {
	httpClient HTTPClientInterface
	whitelist  *models.TokenWhitelist
}

// NewTokenFilterHandler creates a new token filter handler
func NewTokenFilterHandler(httpClient HTTPClientInterface, whitelist *models.TokenWhitelist) *TokenFilterHandler {
	return &TokenFilterHandler{
		httpClient: httpClient,
		whitelist:  whitelist,
	}
}

// ServeHTTP implements the http.Handler interface for token filtering
func (h *TokenFilterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	
	requestID := getRequestIDFromContext(ctx)
	middlewareLogger := logger.MiddlewareLogger.WithRequestID(requestID)
	
	middlewareLogger.Debug("Processing token filter request")
	
	// Fetch tokens from backend API
	tokenResponse, err := h.httpClient.GetTokens(ctx)
	if err != nil {
		middlewareLogger.Error("Failed to fetch tokens from backend", err)
		h.handleError(w, r, err)
		return
	}
	
	middlewareLogger.Debug("Fetched tokens from backend", map[string]interface{}{
		"token_count": len(tokenResponse.Items),
	})
	
	// Filter tokens against whitelist
	filteredResponse := h.filterTokens(tokenResponse, middlewareLogger)
	
	// Return filtered response
	if err := json.NewEncoder(w).Encode(filteredResponse); err != nil {
		middlewareLogger.Error("Error encoding filtered token response", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	middlewareLogger.Info("Successfully filtered and returned tokens", map[string]interface{}{
		"original_count": len(tokenResponse.Items),
		"filtered_count": len(filteredResponse.Items),
		"whitelist_size": h.whitelist.Size(),
	})
}

// filterTokens filters the token response against the whitelist
func (h *TokenFilterHandler) filterTokens(response *models.TokenResponse, logger *logger.Logger) *models.TokenResponse {
	if response == nil || len(response.Items) == 0 {
		logger.Debug("Empty or nil token response, returning empty result")
		return &models.TokenResponse{Items: []models.Token{}}
	}
	
	// If whitelist is empty, log warning and return all tokens
	if h.whitelist.Size() == 0 {
		logger.Warn("Whitelist is empty, returning all tokens", map[string]interface{}{
			"token_count": len(response.Items),
		})
		return response
	}
	
	logger.Debug("Filtering tokens against whitelist", map[string]interface{}{
		"input_tokens":   len(response.Items),
		"whitelist_size": h.whitelist.Size(),
	})
	
	// Filter tokens based on whitelist
	var filteredTokens []models.Token
	matchedAddresses := make([]string, 0)
	
	for _, token := range response.Items {
		if h.whitelist.Contains(token.Address) {
			filteredTokens = append(filteredTokens, token)
			matchedAddresses = append(matchedAddresses, token.Address)
		}
	}
	
	logger.Debug("Token filtering completed", map[string]interface{}{
		"matched_tokens":    len(filteredTokens),
		"matched_addresses": matchedAddresses,
	})
	
	// Return filtered response (empty array if no matches)
	return &models.TokenResponse{Items: filteredTokens}
}

// handleError handles different types of errors and returns appropriate HTTP responses
func (h *TokenFilterHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := getRequestIDFromContext(r.Context())
	middlewareLogger := logger.MiddlewareLogger.WithRequestID(requestID)
	
	middlewareLogger.Error("Token filter error", err, map[string]interface{}{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	
	// Check if it's a network error (backend unreachable)
	if client.IsNetworkError(err) {
		middlewareLogger.Warn("Backend API unreachable for token request", map[string]interface{}{
			"error_type": "network_error",
		})
		h.writeErrorResponse(w, http.StatusBadGateway, "Backend API unreachable", err.Error())
		return
	}
	
	// Check if it's an API error from backend
	if client.IsAPIError(err) {
		middlewareLogger.Warn("Backend API error for token request", map[string]interface{}{
			"error_type": "api_error",
		})
		h.writeErrorResponse(w, http.StatusBadGateway, "Backend API error", err.Error())
		return
	}
	
	// Generic internal server error
	middlewareLogger.Error("Internal server error in token filter", err, map[string]interface{}{
		"error_type": "internal_error",
	})
	h.writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", err.Error())
}

// writeErrorResponse writes a JSON error response
func (h *TokenFilterHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errorResponse := models.ErrorResponse{
		Error:   message,
		Message: detail,
	}
	
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		logger.MiddlewareLogger.Error("Error encoding error response", err, map[string]interface{}{
			"status_code": statusCode,
			"message":     message,
		})
	}
}


package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-api-proxy/client"
	"go-api-proxy/config"
	"go-api-proxy/logger"
	"go-api-proxy/middleware"
	"go-api-proxy/models"
)

// ProxyServer holds the main server components
type ProxyServer struct {
	config            *config.Config
	httpClient        *client.HTTPClient
	whitelist         *models.TokenWhitelist
	tokenHandler      *middleware.TokenFilterHandler
	standardHandler   *middleware.StandardProxyHandler
	server            *http.Server
}

// NewProxyServer creates a new proxy server instance
func NewProxyServer(cfg *config.Config) (*ProxyServer, error) {
	// Initialize HTTP client
	httpClient := client.NewHTTPClient(cfg)
	
	// Initialize whitelist
	whitelist := models.NewTokenWhitelist()
	if err := whitelist.LoadFromFile(cfg.WhitelistFile); err != nil {
		logger.MainLogger.Error("Failed to load whitelist, continuing with empty whitelist", err, map[string]interface{}{
			"whitelist_file": cfg.WhitelistFile,
		})
		// Continue with empty whitelist
	}
	
	// Create handlers
	tokenHandler := middleware.NewTokenFilterHandler(httpClient, whitelist)
	standardHandler := middleware.NewStandardProxyHandler(httpClient)
	
	// Create HTTP server
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	proxyServer := &ProxyServer{
		config:          cfg,
		httpClient:      httpClient,
		whitelist:       whitelist,
		tokenHandler:    tokenHandler,
		standardHandler: standardHandler,
		server:          server,
	}
	
	// Setup routes
	proxyServer.setupRoutes(mux)
	
	return proxyServer, nil
}

// setupRoutes configures the HTTP routes and handlers
func (ps *ProxyServer) setupRoutes(mux *http.ServeMux) {
	// Health check endpoint (with CORS)
	healthHandler := middleware.NewCORSHandler(http.HandlerFunc(ps.healthCheckHandler))
	mux.Handle("/health", healthHandler)
	
	// Main routing handler (with CORS)
	routeHandler := middleware.NewCORSHandler(http.HandlerFunc(ps.routeHandler))
	mux.Handle("/", routeHandler)
}

// routeHandler implements the main request routing logic
func (ps *ProxyServer) routeHandler(w http.ResponseWriter, r *http.Request) {
	// Generate request ID for tracing
	requestID := generateRequestID()
	requestLogger := logger.MainLogger.WithRequestID(requestID)
	
	// Log incoming request
	requestLogger.Info("Incoming request", map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"query":       r.URL.RawQuery,
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.Header.Get("User-Agent"),
	})
	
	// Add request ID to context
	ctx := context.WithValue(r.Context(), "request_id", requestID)
	r = r.WithContext(ctx)
	
	// Check if this is a tokens endpoint request
	if ps.isTokensEndpoint(r.URL.Path) {
		requestLogger.Debug("Routing to token filter handler")
		ps.tokenHandler.ServeHTTP(w, r)
		return
	}
	
	// Handle all other requests with standard proxy
	requestLogger.Debug("Routing to standard proxy handler")
	ps.standardHandler.ServeHTTP(w, r)
}

// isTokensEndpoint checks if the request path is for the tokens endpoint
func (ps *ProxyServer) isTokensEndpoint(path string) bool {
	// Normalize path by removing trailing slashes
	normalizedPath := strings.TrimSuffix(path, "/")
	
	// Check for exact match or with query parameters
	return normalizedPath == "/api/v2/tokens" || strings.HasPrefix(path, "/api/v2/tokens?")
}

// healthCheckHandler provides a health check endpoint
func (ps *ProxyServer) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	logger.MainLogger.Debug("Health check request received")
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"go-api-proxy"}`))
}

// Start starts the HTTP server
func (ps *ProxyServer) Start() error {
	logger.MainLogger.Info("Starting Go API Proxy server", map[string]interface{}{
		"port":             ps.config.Port,
		"backend_api":      ps.config.GetBackendAPIURL(),
		"whitelist_file":   ps.config.WhitelistFile,
		"whitelist_count":  ps.whitelist.Size(),
		"timeout":          ps.config.Timeout.String(),
	})
	
	return ps.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (ps *ProxyServer) Shutdown(ctx context.Context) error {
	logger.MainLogger.Info("Shutting down server...")
	return ps.server.Shutdown(ctx)
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.MainLogger.Fatal("Failed to load configuration", err)
	}
	
	// Create proxy server
	proxyServer, err := NewProxyServer(cfg)
	if err != nil {
		logger.MainLogger.Fatal("Failed to create proxy server", err)
	}
	
	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Start server in a goroutine
	go func() {
		if err := proxyServer.Start(); err != nil && err != http.ErrServerClosed {
			logger.MainLogger.Fatal("Server failed to start", err)
		}
	}()
	
	// Wait for shutdown signal
	logger.MainLogger.Info("Server started successfully, waiting for shutdown signal")
	<-sigChan
	
	// Graceful shutdown with timeout
	logger.MainLogger.Info("Shutdown signal received, starting graceful shutdown")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := proxyServer.Shutdown(ctx); err != nil {
		logger.MainLogger.Error("Server shutdown error", err)
	} else {
		logger.MainLogger.Info("Server shutdown complete")
	}
}

// generateRequestID generates a unique request ID for tracing
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}
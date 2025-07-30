package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"go-api-proxy/logger"
)

// Token represents a token from the API response
type Token struct {
	Address              string  `json:"address"`
	AddressHash          string  `json:"address_hash"`
	CirculatingMarketCap *string `json:"circulating_market_cap"`
	Decimals             string  `json:"decimals"`
	ExchangeRate         *string `json:"exchange_rate"`
	Holders              string  `json:"holders"`
	HoldersCount         string  `json:"holders_count"`
	IconURL              *string `json:"icon_url"`
	Name                 string  `json:"name"`
	Symbol               string  `json:"symbol"`
	TotalSupply          string  `json:"total_supply"`
	Type                 string  `json:"type"`
	Volume24h            *string `json:"volume_24h"`
}

// TokenResponse represents the API response containing tokens
type TokenResponse struct {
	Items []Token `json:"items"`
}

// TokenWhitelist manages a thread-safe list of whitelisted token addresses
type TokenWhitelist struct {
	Addresses []string `json:"addresses"`
	mu        sync.RWMutex
}

// NewTokenWhitelist creates a new TokenWhitelist instance
func NewTokenWhitelist() *TokenWhitelist {
	return &TokenWhitelist{
		Addresses: make([]string, 0),
	}
}

// LoadFromJSON loads whitelist addresses from JSON data
func (tw *TokenWhitelist) LoadFromJSON(data []byte) error {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	
	logger.ModelsLogger.Debug("Parsing whitelist JSON data", map[string]interface{}{
		"data_size": len(data),
	})
	
	var temp struct {
		Addresses []string `json:"addresses"`
	}
	
	if err := json.Unmarshal(data, &temp); err != nil {
		logger.ModelsLogger.Error("Failed to unmarshal whitelist JSON", err)
		return err
	}
	
	// Ensure addresses is never nil
	if temp.Addresses == nil {
		tw.Addresses = make([]string, 0)
	} else {
		tw.Addresses = temp.Addresses
	}
	
	logger.ModelsLogger.Debug("Successfully parsed whitelist JSON", map[string]interface{}{
		"address_count": len(tw.Addresses),
	})
	
	return nil
}

// Contains checks if an address exists in the whitelist (thread-safe)
func (tw *TokenWhitelist) Contains(address string) bool {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	
	for _, addr := range tw.Addresses {
		if addr == address {
			return true
		}
	}
	return false
}

// GetAddresses returns a copy of the addresses slice (thread-safe)
func (tw *TokenWhitelist) GetAddresses() []string {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	
	addresses := make([]string, len(tw.Addresses))
	copy(addresses, tw.Addresses)
	return addresses
}

// Size returns the number of addresses in the whitelist (thread-safe)
func (tw *TokenWhitelist) Size() int {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	
	return len(tw.Addresses)
}

// LoadFromFile loads whitelist addresses from a JSON file
func (tw *TokenWhitelist) LoadFromFile(filename string) error {
	logger.ModelsLogger.Debug("Loading whitelist from file", map[string]interface{}{
		"filename": filename,
	})
	
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		logger.ModelsLogger.Warn("Whitelist file does not exist, continuing with empty whitelist", map[string]interface{}{
			"filename": filename,
		})
		return nil
	}
	
	// Read file contents
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.ModelsLogger.Error("Failed to read whitelist file", err, map[string]interface{}{
			"filename": filename,
		})
		return fmt.Errorf("failed to read whitelist file %s: %w", filename, err)
	}
	
	logger.ModelsLogger.Debug("Read whitelist file", map[string]interface{}{
		"filename": filename,
		"size":     len(data),
	})
	
	// Load from JSON data
	if err := tw.LoadFromJSON(data); err != nil {
		logger.ModelsLogger.Error("Failed to parse whitelist JSON", err, map[string]interface{}{
			"filename": filename,
		})
		return fmt.Errorf("failed to parse whitelist file %s: %w", filename, err)
	}
	
	// Validate the loaded whitelist
	if err := tw.Validate(); err != nil {
		logger.ModelsLogger.Error("Whitelist validation failed", err, map[string]interface{}{
			"filename": filename,
		})
		return fmt.Errorf("whitelist validation failed for file %s: %w", filename, err)
	}
	
	logger.ModelsLogger.Info("Successfully loaded whitelist", map[string]interface{}{
		"filename":      filename,
		"address_count": tw.Size(),
		"addresses":     tw.GetAddresses(),
	})
	
	return nil
}

// Validate checks if the whitelist data is valid
func (tw *TokenWhitelist) Validate() error {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	
	if tw.Addresses == nil {
		return fmt.Errorf("addresses slice is nil")
	}
	
	// Check for duplicate addresses
	seen := make(map[string]bool)
	for i, addr := range tw.Addresses {
		if addr == "" {
			return fmt.Errorf("empty address at index %d", i)
		}
		if seen[addr] {
			return fmt.Errorf("duplicate address found: %s", addr)
		}
		seen[addr] = true
	}
	
	return nil
}

// AddAddress adds a new address to the whitelist (thread-safe)
func (tw *TokenWhitelist) AddAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	
	tw.mu.Lock()
	defer tw.mu.Unlock()
	
	// Check if address already exists
	for _, addr := range tw.Addresses {
		if addr == address {
			return fmt.Errorf("address %s already exists in whitelist", address)
		}
	}
	
	tw.Addresses = append(tw.Addresses, address)
	return nil
}

// RemoveAddress removes an address from the whitelist (thread-safe)
func (tw *TokenWhitelist) RemoveAddress(address string) bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	
	for i, addr := range tw.Addresses {
		if addr == address {
			tw.Addresses = append(tw.Addresses[:i], tw.Addresses[i+1:]...)
			return true
		}
	}
	return false
}

// Clear removes all addresses from the whitelist (thread-safe)
func (tw *TokenWhitelist) Clear() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	
	tw.Addresses = make([]string, 0)
}
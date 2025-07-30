package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
)

func TestToken_JSONMarshaling(t *testing.T) {
	// Test Token struct JSON marshaling/unmarshaling
	token := Token{
		Address:     "0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		AddressHash: "0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		Decimals:    "18",
		Name:        "Test Token",
		Symbol:      "TEST",
		TotalSupply: "1000000",
		Type:        "ERC-20",
	}

	// Marshal to JSON
	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("Failed to marshal token: %v", err)
	}

	// Unmarshal back to Token
	var unmarshaled Token
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal token: %v", err)
	}

	// Verify fields
	if unmarshaled.Address != token.Address {
		t.Errorf("Expected address %s, got %s", token.Address, unmarshaled.Address)
	}
	if unmarshaled.Name != token.Name {
		t.Errorf("Expected name %s, got %s", token.Name, unmarshaled.Name)
	}
	if unmarshaled.Symbol != token.Symbol {
		t.Errorf("Expected symbol %s, got %s", token.Symbol, unmarshaled.Symbol)
	}
}

func TestWhitelistToken_JSONMarshaling(t *testing.T) {
	// Test WhitelistToken struct JSON marshaling/unmarshaling
	iconURL := "https://example.com/icon.png"
	whitelistToken := WhitelistToken{
		Address: "0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		IconURL: &iconURL,
	}

	// Marshal to JSON
	data, err := json.Marshal(whitelistToken)
	if err != nil {
		t.Fatalf("Failed to marshal whitelist token: %v", err)
	}

	// Unmarshal back to WhitelistToken
	var unmarshaled WhitelistToken
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal whitelist token: %v", err)
	}

	// Verify fields
	if unmarshaled.Address != whitelistToken.Address {
		t.Errorf("Expected address %s, got %s", whitelistToken.Address, unmarshaled.Address)
	}
	if unmarshaled.IconURL == nil || *unmarshaled.IconURL != *whitelistToken.IconURL {
		t.Errorf("Expected icon_url %s, got %v", *whitelistToken.IconURL, unmarshaled.IconURL)
	}
}

func TestTokenResponse_JSONMarshaling(t *testing.T) {
	// Test TokenResponse struct JSON marshaling/unmarshaling
	tokens := []Token{
		{
			Address: "0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			Name:    "Token 1",
			Symbol:  "TK1",
		},
		{
			Address: "0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
			Name:    "Token 2",
			Symbol:  "TK2",
		},
	}

	response := TokenResponse{Items: tokens}

	// Marshal to JSON
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal token response: %v", err)
	}

	// Unmarshal back to TokenResponse
	var unmarshaled TokenResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal token response: %v", err)
	}

	// Verify items count
	if len(unmarshaled.Items) != len(tokens) {
		t.Errorf("Expected %d items, got %d", len(tokens), len(unmarshaled.Items))
	}

	// Verify first token
	if len(unmarshaled.Items) > 0 {
		if unmarshaled.Items[0].Address != tokens[0].Address {
			t.Errorf("Expected first token address %s, got %s", tokens[0].Address, unmarshaled.Items[0].Address)
		}
	}
}

func TestNewTokenWhitelist(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	if whitelist == nil {
		t.Fatal("NewTokenWhitelist returned nil")
	}
	
	if whitelist.Addresses == nil {
		t.Fatal("Addresses slice is nil")
	}
	
	if len(whitelist.Addresses) != 0 {
		t.Errorf("Expected empty addresses slice, got length %d", len(whitelist.Addresses))
	}
}

func TestTokenWhitelist_LoadFromJSON(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Test valid JSON
	validJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
		]
	}`
	
	err := whitelist.LoadFromJSON([]byte(validJSON))
	if err != nil {
		t.Fatalf("Failed to load valid JSON: %v", err)
	}
	
	if len(whitelist.Addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(whitelist.Addresses))
	}
	
	expected := []string{
		"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
	}
	
	for i, addr := range expected {
		if whitelist.Addresses[i] != addr {
			t.Errorf("Expected address %s at index %d, got %s", addr, i, whitelist.Addresses[i])
		}
	}
}

func TestTokenWhitelist_LoadFromJSON_InvalidJSON(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Test invalid JSON
	invalidJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
		// Missing closing bracket
	}`
	
	err := whitelist.LoadFromJSON([]byte(invalidJSON))
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestTokenWhitelist_Contains(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Load test data
	validJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
		]
	}`
	
	err := whitelist.LoadFromJSON([]byte(validJSON))
	if err != nil {
		t.Fatalf("Failed to load JSON: %v", err)
	}
	
	// Test existing address
	if !whitelist.Contains("0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614") {
		t.Error("Expected address to be found in whitelist")
	}
	
	// Test non-existing address
	if whitelist.Contains("0x1234567890123456789012345678901234567890") {
		t.Error("Expected address to not be found in whitelist")
	}
	
	// Test empty string
	if whitelist.Contains("") {
		t.Error("Expected empty string to not be found in whitelist")
	}
}

func TestTokenWhitelist_GetAddresses(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Load test data
	validJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
		]
	}`
	
	err := whitelist.LoadFromJSON([]byte(validJSON))
	if err != nil {
		t.Fatalf("Failed to load JSON: %v", err)
	}
	
	addresses := whitelist.GetAddresses()
	
	if len(addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(addresses))
	}
	
	// Verify it's a copy (modifying returned slice shouldn't affect original)
	addresses[0] = "modified"
	if whitelist.Addresses[0] == "modified" {
		t.Error("GetAddresses should return a copy, not the original slice")
	}
}

func TestTokenWhitelist_Size(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Test empty whitelist
	if whitelist.Size() != 0 {
		t.Errorf("Expected size 0 for empty whitelist, got %d", whitelist.Size())
	}
	
	// Load test data
	validJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
		]
	}`
	
	err := whitelist.LoadFromJSON([]byte(validJSON))
	if err != nil {
		t.Fatalf("Failed to load JSON: %v", err)
	}
	
	if whitelist.Size() != 2 {
		t.Errorf("Expected size 2, got %d", whitelist.Size())
	}
}

func TestTokenWhitelist_ThreadSafety(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Load initial data
	validJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614"
		]
	}`
	
	err := whitelist.LoadFromJSON([]byte(validJSON))
	if err != nil {
		t.Fatalf("Failed to load JSON: %v", err)
	}
	
	// Test concurrent access
	var wg sync.WaitGroup
	numGoroutines := 10
	
	// Start multiple goroutines that read from the whitelist
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = whitelist.Contains("0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614")
				_ = whitelist.Size()
				_ = whitelist.GetAddresses()
			}
		}()
	}
	
	// Start a goroutine that writes to the whitelist
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			newJSON := `{
				"addresses": [
					"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
					"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
				]
			}`
			_ = whitelist.LoadFromJSON([]byte(newJSON))
		}
	}()
	
	wg.Wait()
	
	// If we reach here without deadlock or race conditions, the test passes
}

func TestTokenWhitelist_LoadFromFile(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Test with non-existent file (should not error, just log)
	err := whitelist.LoadFromFile("non_existent_file.json")
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got: %v", err)
	}
	
	// Create a temporary file with valid JSON
	validJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
		]
	}`
	
	tmpFile := "test_whitelist.json"
	err = ioutil.WriteFile(tmpFile, []byte(validJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tmpFile) // Clean up
	
	// Test loading from valid file
	err = whitelist.LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load from valid file: %v", err)
	}
	
	if whitelist.Size() != 2 {
		t.Errorf("Expected 2 addresses after loading, got %d", whitelist.Size())
	}
	
	// Test with invalid JSON file
	invalidJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		// Invalid JSON
	}`
	
	invalidFile := "test_invalid.json"
	err = ioutil.WriteFile(invalidFile, []byte(invalidJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid test file: %v", err)
	}
	defer os.Remove(invalidFile) // Clean up
	
	err = whitelist.LoadFromFile(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid JSON file, got nil")
	}
}

func TestTokenWhitelist_Validate(t *testing.T) {
	// Test valid whitelist
	whitelist := NewTokenWhitelist()
	validJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
		]
	}`
	
	err := whitelist.LoadFromJSON([]byte(validJSON))
	if err != nil {
		t.Fatalf("Failed to load JSON: %v", err)
	}
	
	err = whitelist.Validate()
	if err != nil {
		t.Errorf("Expected valid whitelist to pass validation, got: %v", err)
	}
	
	// Test whitelist with empty address
	whitelist2 := NewTokenWhitelist()
	invalidJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7"
		]
	}`
	
	err = whitelist2.LoadFromJSON([]byte(invalidJSON))
	if err != nil {
		t.Fatalf("Failed to load JSON: %v", err)
	}
	
	err = whitelist2.Validate()
	if err == nil {
		t.Error("Expected validation error for empty address, got nil")
	}
	
	// Test whitelist with duplicate addresses
	whitelist3 := NewTokenWhitelist()
	duplicateJSON := `{
		"addresses": [
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
			"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
			"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614"
		]
	}`
	
	err = whitelist3.LoadFromJSON([]byte(duplicateJSON))
	if err != nil {
		t.Fatalf("Failed to load JSON: %v", err)
	}
	
	err = whitelist3.Validate()
	if err == nil {
		t.Error("Expected validation error for duplicate address, got nil")
	}
}

func TestTokenWhitelist_AddAddress(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Test adding valid address
	err := whitelist.AddAddress("0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614")
	if err != nil {
		t.Errorf("Failed to add valid address: %v", err)
	}
	
	if whitelist.Size() != 1 {
		t.Errorf("Expected size 1 after adding address, got %d", whitelist.Size())
	}
	
	if !whitelist.Contains("0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614") {
		t.Error("Address should be found after adding")
	}
	
	// Test adding empty address
	err = whitelist.AddAddress("")
	if err == nil {
		t.Error("Expected error when adding empty address, got nil")
	}
	
	// Test adding duplicate address
	err = whitelist.AddAddress("0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614")
	if err == nil {
		t.Error("Expected error when adding duplicate address, got nil")
	}
	
	if whitelist.Size() != 1 {
		t.Errorf("Expected size to remain 1 after duplicate add, got %d", whitelist.Size())
	}
}

func TestTokenWhitelist_RemoveAddress(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Add some addresses
	addresses := []string{
		"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
	}
	
	for _, addr := range addresses {
		err := whitelist.AddAddress(addr)
		if err != nil {
			t.Fatalf("Failed to add address: %v", err)
		}
	}
	
	// Test removing existing address
	removed := whitelist.RemoveAddress("0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614")
	if !removed {
		t.Error("Expected RemoveAddress to return true for existing address")
	}
	
	if whitelist.Size() != 1 {
		t.Errorf("Expected size 1 after removing one address, got %d", whitelist.Size())
	}
	
	if whitelist.Contains("0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614") {
		t.Error("Address should not be found after removal")
	}
	
	// Test removing non-existent address
	removed = whitelist.RemoveAddress("0x1234567890123456789012345678901234567890")
	if removed {
		t.Error("Expected RemoveAddress to return false for non-existent address")
	}
	
	if whitelist.Size() != 1 {
		t.Errorf("Expected size to remain 1 after removing non-existent address, got %d", whitelist.Size())
	}
}

func TestTokenWhitelist_Clear(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	// Add some addresses
	addresses := []string{
		"0x5db2B3f16E1a28ad4fe1229a2dc01f264a3f0614",
		"0x7254B7303A9d5d0A2F232eB62B0B27a06E068Ac7",
	}
	
	for _, addr := range addresses {
		err := whitelist.AddAddress(addr)
		if err != nil {
			t.Fatalf("Failed to add address: %v", err)
		}
	}
	
	if whitelist.Size() != 2 {
		t.Errorf("Expected size 2 before clear, got %d", whitelist.Size())
	}
	
	// Clear the whitelist
	whitelist.Clear()
	
	if whitelist.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", whitelist.Size())
	}
	
	for _, addr := range addresses {
		if whitelist.Contains(addr) {
			t.Errorf("Address %s should not be found after clear", addr)
		}
	}
}

func TestTokenWhitelist_ConcurrentOperations(t *testing.T) {
	whitelist := NewTokenWhitelist()
	
	var wg sync.WaitGroup
	numGoroutines := 10
	
	// Test concurrent add operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := fmt.Sprintf("0x%040d", id)
			_ = whitelist.AddAddress(addr)
		}(i)
	}
	
	// Test concurrent read operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := fmt.Sprintf("0x%040d", id)
			for j := 0; j < 10; j++ {
				_ = whitelist.Contains(addr)
				_ = whitelist.Size()
			}
		}(i)
	}
	
	// Test concurrent remove operations
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			addr := fmt.Sprintf("0x%040d", id)
			_ = whitelist.RemoveAddress(addr)
		}(i)
	}
	
	wg.Wait()
	
	// If we reach here without deadlock or race conditions, the test passes
}
package models

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestWhitelistErrorHandling(t *testing.T) {
	t.Run("Load from non-existent file", func(t *testing.T) {
		whitelist := NewTokenWhitelist()
		
		// Try to load from a file that doesn't exist
		err := whitelist.LoadFromFile("non_existent_file.json")
		
		// Should not return an error, but should log a warning
		if err != nil {
			t.Errorf("Expected no error for non-existent file, got: %v", err)
		}
		
		// Whitelist should be empty
		if whitelist.Size() != 0 {
			t.Errorf("Expected empty whitelist, got size %d", whitelist.Size())
		}
	})
	
	t.Run("Load from file with invalid JSON", func(t *testing.T) {
		// Create a temporary file with invalid JSON
		tmpFile, err := ioutil.TempFile("", "invalid_whitelist_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		// Write invalid JSON
		invalidJSON := `{"addresses": ["addr1", "addr2",}`
		if _, err := tmpFile.WriteString(invalidJSON); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()
		
		whitelist := NewTokenWhitelist()
		err = whitelist.LoadFromFile(tmpFile.Name())
		
		// Should return an error
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
		
		// Error should mention parsing failure
		if err != nil && !contains(err.Error(), "failed to parse") {
			t.Errorf("Expected parsing error, got: %v", err)
		}
	})
	
	t.Run("Load from file with invalid structure", func(t *testing.T) {
		// Create a temporary file with valid JSON but wrong structure
		tmpFile, err := ioutil.TempFile("", "invalid_structure_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		// Write JSON with wrong structure
		wrongStructure := `{"tokens": ["addr1", "addr2"]}`
		if _, err := tmpFile.WriteString(wrongStructure); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()
		
		whitelist := NewTokenWhitelist()
		err = whitelist.LoadFromFile(tmpFile.Name())
		
		// Should not return an error (missing addresses field results in empty array)
		if err != nil {
			t.Errorf("Expected no error for missing addresses field, got: %v", err)
		}
		
		// Whitelist should be empty
		if whitelist.Size() != 0 {
			t.Errorf("Expected empty whitelist, got size %d", whitelist.Size())
		}
	})
	
	t.Run("Load from file with empty addresses", func(t *testing.T) {
		// Create a temporary file with empty addresses array
		tmpFile, err := ioutil.TempFile("", "empty_addresses_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		// Write JSON with empty addresses
		emptyAddresses := `{"addresses": []}`
		if _, err := tmpFile.WriteString(emptyAddresses); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()
		
		whitelist := NewTokenWhitelist()
		err = whitelist.LoadFromFile(tmpFile.Name())
		
		// Should not return an error
		if err != nil {
			t.Errorf("Expected no error for empty addresses, got: %v", err)
		}
		
		// Whitelist should be empty
		if whitelist.Size() != 0 {
			t.Errorf("Expected empty whitelist, got size %d", whitelist.Size())
		}
	})
	
	t.Run("Validation with duplicate addresses", func(t *testing.T) {
		// Create a temporary file with duplicate addresses
		tmpFile, err := ioutil.TempFile("", "duplicate_addresses_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		// Write JSON with duplicate addresses
		duplicateAddresses := `{"addresses": ["0x123", "0x456", "0x123"]}`
		if _, err := tmpFile.WriteString(duplicateAddresses); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()
		
		whitelist := NewTokenWhitelist()
		err = whitelist.LoadFromFile(tmpFile.Name())
		
		// Should return a validation error
		if err == nil {
			t.Error("Expected validation error for duplicate addresses, got nil")
		}
		
		if err != nil && !contains(err.Error(), "duplicate address") {
			t.Errorf("Expected duplicate address error, got: %v", err)
		}
	})
	
	t.Run("Validation with empty address", func(t *testing.T) {
		// Create a temporary file with empty address
		tmpFile, err := ioutil.TempFile("", "empty_address_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		// Write JSON with empty address
		emptyAddress := `{"addresses": ["0x123", "", "0x456"]}`
		if _, err := tmpFile.WriteString(emptyAddress); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()
		
		whitelist := NewTokenWhitelist()
		err = whitelist.LoadFromFile(tmpFile.Name())
		
		// Should return a validation error
		if err == nil {
			t.Error("Expected validation error for empty address, got nil")
		}
		
		if err != nil && !contains(err.Error(), "empty address") {
			t.Errorf("Expected empty address error, got: %v", err)
		}
	})
	
	t.Run("File permission error", func(t *testing.T) {
		// Create a temporary file and remove read permissions
		tmpFile, err := ioutil.TempFile("", "no_permission_*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		// Write valid JSON
		validJSON := `{"addresses": ["0x123", "0x456"]}`
		if _, err := tmpFile.WriteString(validJSON); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
		tmpFile.Close()
		
		// Remove read permissions (this might not work on all systems)
		if err := os.Chmod(tmpFile.Name(), 0000); err != nil {
			t.Skip("Cannot change file permissions on this system")
		}
		defer os.Chmod(tmpFile.Name(), 0644) // Restore permissions for cleanup
		
		whitelist := NewTokenWhitelist()
		err = whitelist.LoadFromFile(tmpFile.Name())
		
		// Should return a read error
		if err == nil {
			t.Error("Expected permission error, got nil")
		}
		
		if err != nil && !contains(err.Error(), "failed to read") {
			t.Errorf("Expected read error, got: %v", err)
		}
	})
	
	t.Run("AddAddress error handling", func(t *testing.T) {
		whitelist := NewTokenWhitelist()
		
		// Add a valid address first
		err := whitelist.AddAddress("0x123")
		if err != nil {
			t.Fatalf("Failed to add valid address: %v", err)
		}
		
		// Try to add empty address
		err = whitelist.AddAddress("")
		if err == nil {
			t.Error("Expected error for empty address, got nil")
		}
		
		// Try to add duplicate address
		err = whitelist.AddAddress("0x123")
		if err == nil {
			t.Error("Expected error for duplicate address, got nil")
		}
		
		if err != nil && !contains(err.Error(), "already exists") {
			t.Errorf("Expected duplicate error, got: %v", err)
		}
	})
	
	t.Run("Concurrent access safety", func(t *testing.T) {
		whitelist := NewTokenWhitelist()
		whitelist.AddAddress("0x123")
		
		// Test concurrent read/write operations
		done := make(chan bool, 2)
		
		// Goroutine 1: Read operations
		go func() {
			for i := 0; i < 100; i++ {
				whitelist.Contains("0x123")
				whitelist.Size()
				whitelist.GetAddresses()
			}
			done <- true
		}()
		
		// Goroutine 2: Write operations
		go func() {
			for i := 0; i < 100; i++ {
				whitelist.AddAddress("0x" + string(rune(i)))
				whitelist.RemoveAddress("0x" + string(rune(i)))
			}
			done <- true
		}()
		
		// Wait for both goroutines to complete
		<-done
		<-done
		
		// Should still contain the original address
		if !whitelist.Contains("0x123") {
			t.Error("Original address should still be present after concurrent operations")
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
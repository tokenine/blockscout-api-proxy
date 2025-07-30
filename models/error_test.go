package models

import (
	"encoding/json"
	"testing"
)

func TestErrorResponse(t *testing.T) {
	t.Run("NewErrorResponse", func(t *testing.T) {
		errorResp := NewErrorResponse("test error", "detailed message")
		
		if errorResp.Error != "test error" {
			t.Errorf("Expected error 'test error', got '%s'", errorResp.Error)
		}
		
		if errorResp.Message != "detailed message" {
			t.Errorf("Expected message 'detailed message', got '%s'", errorResp.Message)
		}
	})
	
	t.Run("JSON Marshaling", func(t *testing.T) {
		errorResp := &ErrorResponse{
			Error:   "validation failed",
			Message: "field 'name' is required",
		}
		
		jsonData, err := json.Marshal(errorResp)
		if err != nil {
			t.Fatalf("Failed to marshal ErrorResponse: %v", err)
		}
		
		expected := `{"error":"validation failed","message":"field 'name' is required"}`
		if string(jsonData) != expected {
			t.Errorf("Expected JSON '%s', got '%s'", expected, string(jsonData))
		}
	})
	
	t.Run("JSON Unmarshaling", func(t *testing.T) {
		jsonData := `{"error":"parse error","message":"invalid syntax at line 5"}`
		
		var errorResp ErrorResponse
		if err := json.Unmarshal([]byte(jsonData), &errorResp); err != nil {
			t.Fatalf("Failed to unmarshal ErrorResponse: %v", err)
		}
		
		if errorResp.Error != "parse error" {
			t.Errorf("Expected error 'parse error', got '%s'", errorResp.Error)
		}
		
		if errorResp.Message != "invalid syntax at line 5" {
			t.Errorf("Expected message 'invalid syntax at line 5', got '%s'", errorResp.Message)
		}
	})
	
	t.Run("Empty Fields", func(t *testing.T) {
		errorResp := &ErrorResponse{}
		
		jsonData, err := json.Marshal(errorResp)
		if err != nil {
			t.Fatalf("Failed to marshal empty ErrorResponse: %v", err)
		}
		
		expected := `{"error":"","message":""}`
		if string(jsonData) != expected {
			t.Errorf("Expected JSON '%s', got '%s'", expected, string(jsonData))
		}
	})
}
package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func setupSetVMMetadataMockClient(cpi *CPI, mockHandler http.HandlerFunc) *httptest.Server {
	mockServer := httptest.NewServer(mockHandler)

	return mockServer
}

func TestSetVMMetadata_ValidateArguments(t *testing.T) {
	tests := []struct {
		name      string
		arguments json.RawMessage
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Missing arguments",
			arguments: json.RawMessage(`[]`),
			wantErr:   true,
			errMsg:    "expected 2 arguments for set_vm_metadata, got 0",
		},
		{
			name:      "Empty server ID",
			arguments: json.RawMessage(`["", {"key": "value"}]`),
			wantErr:   true,
			errMsg:    "vm_cid must be a valid uuid, got: ''",
		},
		{
			name:      "Missing metadata",
			arguments: json.RawMessage(`["` + TestServerID + `"]`),
			wantErr:   true,
			errMsg:    "expected 2 arguments for set_vm_metadata, got 1",
		},
		{
			name:      "Null metadata",
			arguments: json.RawMessage(`["` + TestServerID + `", null]`),
			wantErr:   true,
			errMsg:    "metadata must be provided",
		},
		{
			name:      "Empty metadata object",
			arguments: json.RawMessage(`["` + TestServerID + `", {}]`),
			wantErr:   false,
			errMsg:    "",
		},
		{
			name:      "Valid arguments",
			arguments: json.RawMessage(`["` + TestServerID + `", {"key": "value"}]`),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := lib.SetVMMetadataArgs{}
			err := json.Unmarshal(tt.arguments, &args)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("SetVMMetadata() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("SetVMMetadata() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("SetVMMetadata() unexpected error = %v", err)
			}
		})
	}
}

func TestSetVMMetadata_SuccessfulUpdate(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SetVMMetadata, t)

	cpi.Config.HumanReadableVMNames = true
	// Set up the mock response for a successful metadata update
	serverID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
	metadata := map[string]any{
		"director":      "test-director",
		"deployment":    "test-deployment",
		"name":          "test/instance-id",
		"job":           "test-job",
		"id":            "instance-id",
		"index":         "1",
		"instance_name": "test-instance-name",
		"created_at":    time.Now().UTC().Format(time.RFC3339),
	}

	// Override the arguments to match our test data
	args := []any{serverID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	var updateRequested bool
	var requestBody map[string]any

	// Mock server that simulates successful metadata update
	mockServer := setupSetVMMetadataMockClient(cpi, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Exact path match is crucial and must accept PATCH method (used by SDK by default)
		expectedPath := fmt.Sprintf("/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID)

		// Accept PATCH method since the SDK's UpdateServer uses PATCH by default
		if r.Method == http.MethodPatch && r.URL.Path == expectedPath {
			updateRequested = true

			// Verify the request body contains the metadata
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&requestBody); err != nil {
				t.Errorf("Failed to decode request body: %v", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			// Return a successful response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(iaas.Server{
				Id:          utils.Ptr(uuid.New().String()),
				MachineType: "test",
			})
			return
		}

		// Log the mismatch for debugging
		t.Logf("Path mismatch. Got: %s, Expected: %s, Method: %s",
			r.URL.Path, expectedPath, r.Method)

		// Default: Teapot
		http.Error(w, fmt.Sprintf("Teapot: %s", r.URL.Path), http.StatusTeapot)
	}))
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = sClient

	// Call the SetVMMetadata method
	resp, err := cpi.SetVMMetadata(req)
	if err != nil {
		t.Errorf("SetVMMetadata() unexpected error = %v", err)
		return
	}

	// Check that the update request was made
	if !updateRequested {
		t.Errorf("Expected update request to be made, but it wasn't")
	}

	// Check if name was removed from the request as expected
	if requestBody != nil {
		labels, ok := requestBody["labels"].(map[string]any)
		if !ok {
			t.Errorf("Expected labels to be a map, got %T", requestBody["labels"])
		} else {
			// Verify name was fixed to not include a `/`
			if name, exists := labels["name"]; exists && name != "test-instance-id" {
				t.Errorf("Expected 'name' to be 'test-instance-id' in request, got %v", name)
			}

			// Verify created_at was set
			if createdAt, exists := labels["created_at"]; !exists {
				t.Errorf("Expected 'created_at' to be present in request")
			} else {
				// Check if created_at is in the expected format
				_, err := time.Parse("20060102_150405", createdAt.(string))
				if err != nil {
					t.Errorf("Expected 'created_at' to be in format '20060102_150405', got %v", createdAt)
				}
			}
		}
	}

	// Check the result - should be nil for successful update
	result := resp.Result
	if result != nil {
		t.Errorf("Expected nil result, got %T", resp.Result)
		return
	}
}

func TestSetVMMetadata_ServerNotFound(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SetVMMetadata, t)

	// Set up the mock response for a non-existent server
	// Use a valid UUID format to ensure the test path isn't short-circuited
	serverID := "11111111-2222-3333-4444-555555555555"
	metadata := map[string]any{
		"director":   "test-director",
		"deployment": "test-deployment",
	}

	// Override the arguments to match our test data
	args := []any{serverID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	// Mock server that simulates server not found
	mockServer := setupSetVMMetadataMockClient(cpi, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a 404 error for non-existent server
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Server not found"}`))
	}))
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = sClient

	// Call the SetVMMetadata method, expect an error
	resp, err := cpi.SetVMMetadata(req)

	// This should result in an error
	if err == nil {
		t.Errorf("SetVMMetadata() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

func TestSetVMMetadata_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SetVMMetadata, t)

	// Set up the mock response for an API error
	// Use a valid UUID format to ensure the test path isn't short-circuited
	serverID := "22222222-3333-4444-5555-666666666666"
	metadata := map[string]any{
		"director":   "test-director",
		"deployment": "test-deployment",
	}

	// Override the arguments to match our test data
	args := []any{serverID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	// Mock server that simulates an API error
	mockServer := setupSetVMMetadataMockClient(cpi, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return 500 error for any request to this server
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = sClient

	// Call the SetVMMetadata method, expect an error
	resp, err := cpi.SetVMMetadata(req)

	// This should result in an error
	if err == nil {
		t.Errorf("SetVMMetadata() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

func TestSetVMMetadata_InvalidMetadata(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SetVMMetadata, t)

	// Set up test for invalid metadata format (like invalid characters in keys)
	// Use a valid UUID format to ensure the test path isn't short-circuited
	serverID := "33333333-4444-5555-6666-777777777777"
	metadata := map[string]any{
		"invalid!key": "value-with-invalid-key",
		"director":    "test-director",
	}

	// Override the arguments to match our test data
	args := []any{serverID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	// Mock server that simulates validation error for metadata
	mockServer := setupSetVMMetadataMockClient(cpi, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, serverID) {
			// Return a 400 Bad Request for invalid metadata
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "Invalid metadata format"}`))
			return
		}
		// Default: Teapot
		http.Error(w, fmt.Sprintf("Teapot: %s", r.URL.Path), http.StatusTeapot)
	}))
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = sClient

	// Call the SetVMMetadata method, expect an error
	resp, err := cpi.SetVMMetadata(req)

	// This should result in an error
	if err == nil {
		t.Errorf("SetVMMetadata() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

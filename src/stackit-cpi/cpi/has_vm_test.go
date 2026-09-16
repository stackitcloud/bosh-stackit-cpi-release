package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func TestHasVM_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 1 argument for has_vm, got 0",
		},
		{
			name:      "Empty server ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "invalid arguments: vm_cid must be provided",
		},
		{
			name:      "Valid server ID",
			arguments: json.RawMessage(fmt.Sprintf(`["%s"]`, TestServerID)),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.HasVM, t)

			// Set up mock responses for GetServer call to avoid nil pointer dereference
			mockResponses := map[string]any{
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID): TestServerPayload,
			}
			mockServer := SetupMockClient(cpi, mockResponses)
			defer mockServer.Close()

			// Override arguments for this test
			req.Arguments = tt.arguments

			resp, err := cpi.HasVM(req)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("HasVM() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("HasVM() error = %v, want %v", err.Error(), tt.errMsg)
				}
				// Verify error in response
				if !strings.Contains(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = '%v', want '%v'", resp.Error.Message, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("HasVM() unexpected error = '%v'", err)
			}
		})
	}
}

func TestHasVM_VMExists(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.HasVM, t)

	// Set up the mock response for an existing server - use UUID format
	serverID := "11111111-1111-1111-1111-111111111111"

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Mock response with an existing server
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): iaas.Server{
			Id:   utils.Ptr(serverID),
			Name: "test-server",
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the HasVM method
	resp, err := cpi.HasVM(req)
	if err != nil {
		t.Errorf("HasVM() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for an existing VM
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for existing VM, got %v", result)
	}
}

func TestHasVM_VMDoesNotExist(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.HasVM, t)

	// Set up the mock response for a non-existing server - use UUID format
	serverID := "22222222-2222-2222-2222-222222222222"

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Mock response with a null server to simulate the server not existing
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusNotFound,
			Body:         []byte("Not Found"),
			ErrorMessage: "404",
			Model:        nil,
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the HasVM method
	resp, err := cpi.HasVM(req)
	if err != nil {
		t.Errorf("HasVM() unexpected error = %v", err)
		return
	}

	// Check the result - should be false for a non-existing VM
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if result {
		t.Errorf("Expected result to be false for non-existing VM, got %v", result)
	}
}

func TestHasVM_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.HasVM, t)

	// Set up a scenario where the API returns an error - use UUID format
	serverID := "33333333-3333-3333-3333-333333333333"

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Create a mock server that returns an error for this specific server ID
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, serverID) {
			// Return a 500 error
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Default: Teapot
		http.Error(w, "Teapot", http.StatusTeapot)
	}))
	defer mockServer.Close()

	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the HasVM method, expecting an error
	resp, err := cpi.HasVM(req)

	// This should result in an error
	if err == nil {
		t.Errorf("HasVM() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

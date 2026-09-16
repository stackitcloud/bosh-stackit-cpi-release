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

func TestRebootVM_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 1 argument for reboot_vm, got 0",
		},
		{
			name:      "Empty server ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "vm_cid must be provided",
		},
		{
			name:      "Valid server ID",
			arguments: json.RawMessage(`["12345678-1234-1234-1234-123456789012"]`),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.RebootVM, t)

			// Set up mock responses for RebootServer call
			mockResponses := map[string]interface{}{
				fmt.Sprintf("POST_/v2/projects/%s/regions/%s/servers/12345678-1234-1234-1234-123456789012/reboot", cpi.Config.ProjectID, cpi.Config.RegionID): FakeResponse{
					StatusCode: 202,
					Body:       nil,
				},
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/12345678-1234-1234-1234-123456789012", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.Server{
					Id:   utils.Ptr("12345678-1234-1234-1234-123456789012"),
					Name: "test-server",
				},
			}
			mockServer := SetupMockClient(cpi, mockResponses)
			defer mockServer.Close()
			sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

			cpi.SClient = sClient

			// Override arguments for this test
			req.Arguments = tt.arguments

			resp, err := cpi.RebootVM(req)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("RebootVM() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("RebootVM() error = %v, want %v", err.Error(), tt.errMsg)
				}
				// Verify error in response
				if !strings.Contains(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = %v, want %v", resp.Error.Message, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("RebootVM() unexpected error = %v", err)
			}
		})
	}
}

func TestRebootVM_Success(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.RebootVM, t)

	// Set up the mock response for an existing server - use UUID format
	serverID := "11111111-1111-1111-1111-111111111111"

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	mockResponses := map[string]interface{}{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/servers/%s/reboot", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): FakeResponse{
			StatusCode: 202,
			Body:       nil,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): iaas.Server{
			Id:   utils.Ptr(serverID),
			Name: "test-server",
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the RebootVM method
	resp, err := cpi.RebootVM(req)
	if err != nil {
		t.Errorf("RebootVM() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for successful reboot
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for successful reboot, got %v", result)
	}
}

func TestRebootVM_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.RebootVM, t)

	// Set up a scenario where the API returns an error - use UUID format
	serverID := "33333333-3333-3333-3333-333333333333"

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	mockResponses := map[string]interface{}{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/servers/%s/reboot", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): oapierror.GenericOpenAPIError{
			StatusCode:   500,
			Body:         []byte(`Internal Server Error`),
			ErrorMessage: "Internal Server Error",
			Model:        nil,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): iaas.Server{
			Id:   utils.Ptr(serverID),
			Name: "test-server",
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the RebootVM method, expecting an error
	resp, err := cpi.RebootVM(req)

	// This should result in an error
	if err == nil {
		t.Errorf("RebootVM() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}

	// Verify result is false for failed reboot
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if result {
		t.Errorf("Expected result to be false for failed reboot, got %v", result)
	}
}

func TestRebootVM_ServerNotFound(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.RebootVM, t)

	// Set up a scenario for a non-existent server
	serverID := "44444444-4444-4444-4444-444444444444"

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Create a mock server that returns 404 for non-existent server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, serverID) {
			// Return 404 for non-existent server
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		// Default: Teapot
		http.Error(w, "Teapot", http.StatusTeapot)
	}))
	defer mockServer.Close()
	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the RebootVM method, expecting an error
	resp, err := cpi.RebootVM(req)

	// This should result in an error
	if err == nil {
		t.Errorf("RebootVM() expected error for non-existent server, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response for non-existent server, got none")
	}

	// Check that the error message mentions the server wasn't found
	if !strings.Contains(strings.ToLower(resp.Error.Message), "not found") &&
		!strings.Contains(strings.ToLower(resp.Error.Message), "404") {
		t.Errorf("Expected error message to indicate server not found, got: %s", resp.Error.Message)
	}
}

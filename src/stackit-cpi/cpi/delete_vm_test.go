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
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func TestDeleteVM_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 1 argument for delete_vm, got 0",
		},
		{
			name:      "Empty server ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "vm_cid (server id) must be provided",
		},
		{
			name:      "Valid server ID",
			arguments: json.RawMessage(`["12345678-1234-1234-1234-123456789012"]`), // Valid UUID format
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.DeleteVM, t)
			// Override arguments for this test
			req.Arguments = tt.arguments

			// For the valid case, we need a proper mock that handles NIC listing and deletion
			var mockServer *httptest.Server

			// For valid server ID, we need to properly handle NIC listing, NIC deletion, server deletion
			serverID := "12345678-1234-1234-1234-123456789012"
			mockResponses := map[string]any{
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): iaas.NICListResponse{Items: []iaas.NIC{}},
				fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): FakeResponse{
					StatusCode: 204,
					Body:       nil,
				},
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): oapierror.GenericOpenAPIError{
					StatusCode:   http.StatusNotFound,
					Body:         []byte(`Not Found`),
					ErrorMessage: `Not Found`,
					Model:        iaas.Server{},
				},
			}
			mockServer = SetupMockClient(cpi, mockResponses)
			defer mockServer.Close()

			resp, err := cpi.DeleteVM(req)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteVM() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DeleteVM() error = %v, want %v", err.Error(), tt.errMsg)
				}
				// Verify error in response
				if !strings.Contains(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = %v, want %v", resp.Error.Message, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("DeleteVM() unexpected error = %v", err)
			}
		})
	}
}

func TestDeleteVM_SuccessfulDeletion(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteVM, t)

	// Set up the mock response for a successful VM deletion
	serverID := "12345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Mock server that simulates successful deletion
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): iaas.NICListResponse{Items: []iaas.NIC{}},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): FakeResponse{
			StatusCode: 204,
			Body:       nil,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusNotFound,
			Body:         []byte(`Not Found`),
			ErrorMessage: `Not Found`,
			Model:        iaas.Server{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteVM method
	resp, err := cpi.DeleteVM(req)
	if err != nil {
		t.Errorf("DeleteVM() unexpected error = %v", err)
		return
	}

	// Check the result - should be null for successful deletion per CPI specs
	if resp.Result != true {
		t.Errorf("Expected nil result for successful deletion, got %v", resp.Result)
	}
}

func TestDeleteVM_WithNICs(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteVM, t)

	// Set up the mock response for a VM with NICs
	serverID := "12345678-1234-1234-1234-123456789012"
	nicID1 := "12345678-1234-1234-1234-123456789000"
	nicID2 := "12345678-1234-1234-1234-123456789001"
	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Mock server that simulates VM with NICs
	// Note: The actual implementation only deletes the first NIC and continues even if it fails
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): iaas.NICListResponse{Items: []iaas.NIC{
			{Id: utils.Ptr(nicID2), NetworkId: utils.Ptr(nicID1)},
			{Id: utils.Ptr(nicID1), NetworkId: utils.Ptr(nicID2)},
		}},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s/nics/%s", cpi.Config.ProjectID, cpi.Config.RegionID, nicID2, nicID1): FakeResponse{
			StatusCode: 204,
			Body:       nil,
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s/nics/%s", cpi.Config.ProjectID, cpi.Config.RegionID, nicID1, nicID2): FakeResponse{
			StatusCode: 204,
			Body:       nil,
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): FakeResponse{
			StatusCode: 204,
			Body:       nil,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusNotFound,
			Body:         []byte(`Not Found`),
			ErrorMessage: `Not Found`,
			Model:        iaas.Server{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteVM method
	resp, err := cpi.DeleteVM(req)
	if err != nil {
		t.Errorf("DeleteVM() unexpected error = %v", err)
		return
	}

	// Check the result - should be null for successful deletion per CPI specs
	if resp.Result != true {
		t.Errorf("Expected true result for successful deletion, got %v", resp.Result)
	}
}

func TestDeleteVM_AlreadyDeleted(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteVM, t)

	// Set up a very short timeout and reduce retries for fast test execution
	cpi.Config.Timeout = 1
	cpi.Config.RetryCount = 1

	// Test server ID
	serverID := "12345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Mock server that simulates server not found
	// serverPayload := TestServerPayload
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusNotFound,
			Body:         []byte(`Not Found`),
			ErrorMessage: `Not Found`,
			Model:        iaas.Server{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteVM method
	resp, err := cpi.DeleteVM(req)
	if err != nil {
		t.Errorf("DeleteVM() unexpected error = %v", err)
		return
	}

	// Check the result - should be nil for successful deletion (even if already deleted)
	if resp.Result != true {
		t.Errorf("Expected nil result when VM already deleted, got %v", resp.Result)
	}
}

func TestDeleteVM_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteVM, t)

	// Set up a scenario where the API returns an error
	serverID := "12345678-1234-1234-1234-123456789012"

	// Set retry count to a small value for faster tests
	cpi.Config.RetryCount = 1

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Create a mock server that returns an error for this specific server ID
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): TestServerPayload,
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				Body:         []byte(`Internal Server Error`),
				ErrorMessage: `Internal Server Error`,
				Model:        iaas.Server{},
			},
			FinallyRespondWith: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				Body:         []byte(`Internal Server Error`),
				ErrorMessage: `Internal Server Error`,
				Model:        iaas.Server{},
			},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteVM method, expecting an error
	resp, err := cpi.DeleteVM(req)

	// This should result in an error
	if err == nil {
		t.Errorf("DeleteVM() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || !resp.Error.OkToRetry {
		t.Errorf("Expected retryable error in response, got OkToRetry=%v",
			resp.Error != nil && resp.Error.OkToRetry)
	}
}

func TestDeleteVM_RetrySuccess(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteVM, t)

	// Set up a scenario where the API fails initially but succeeds on retry
	serverID := TestServerPayload.GetId()

	// Override the arguments to match our test server ID
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Ensure minimal retries
	cpi.Config.RetryCount = 1
	cpi.Config.Timeout = 3

	// Create a mock server that fails once then succeeds
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): AfterRetryResponse{
			FirstRespond: TestServerPayload,
			FinallyRespondWith: oapierror.GenericOpenAPIError{
				StatusCode: http.StatusNotFound,
				Body:       []byte(`not found`),
			},
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				Body:         []byte(`Internal Server Error`),
				ErrorMessage: `Internal Server Error`,
				Model:        iaas.Server{},
			},
			FinallyRespondWith: FakeResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(`200`),
			},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteVM method
	resp, err := cpi.DeleteVM(req)
	if err != nil {
		t.Errorf("DeleteVM() unexpected error = %v", err)
		return
	}

	// Check the result - should be nil for successful deletion after retry
	if resp.Result != true {
		t.Errorf("Expected true result for successful deletion after retry, got %v", resp.Result)
	}
}

// Test timeouts during VM deletion
func TestDeleteVM_Timeout(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteVM, t)

	// Setup for timeout testing - extremely short timeout
	cpi.Config.Timeout = 1 // Use 1 second timeout which is small enough for testing

	serverID := "12345678-1234-1234-1234-123456789012"
	req.Arguments = json.RawMessage(`["` + serverID + `"]`)

	// Mock server that simulates a slow response
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): TestServerPayload,
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): FakeTimeout{
			After: 2,
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)

	defer mockServer.Close()

	// Call DeleteVM, expecting a timeout error
	resp, err := cpi.DeleteVM(req)

	// Should have a timeout error
	if err == nil {
		t.Errorf("DeleteVM() expected timeout error, got nil")
	}

	// Error should be retryable
	if resp.Error == nil || !resp.Error.OkToRetry {
		t.Errorf("Expected retryable error for timeout, got OkToRetry=%v",
			resp.Error != nil && resp.Error.OkToRetry)
	}
}

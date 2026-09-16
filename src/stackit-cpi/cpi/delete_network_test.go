package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"
)

func TestDeleteNetwork_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 1 argument for delete_network, got 0",
		},
		{
			name:      "Empty network ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "network_id must be provided",
		},
		{
			name:      "Valid network ID",
			arguments: json.RawMessage(`["` + TestNetworkID + `"]`),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.DeleteNetwork, t)

			// Override arguments for this test
			req.Arguments = tt.arguments

			mockResponses := map[string]any{
				fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
					StatusCode:   http.StatusNotFound,
					Body:         []byte(`Not Found`),
					ErrorMessage: `Not Found`,
					Model:        iaas.Network{},
				},
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
					StatusCode:   http.StatusNotFound,
					Body:         []byte(`Not Found`),
					ErrorMessage: `Not Found`,
					Model:        iaas.Network{},
				},
			}
			mockServer := SetupMockClient(cpi, mockResponses)
			defer mockServer.Close()

			resp, err := cpi.DeleteNetwork(req)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteNetwork() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DeleteNetwork() error = %v, want %v", err.Error(), tt.errMsg)
				}
				// Verify error in response
				if !strings.Contains(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = %v, want %v", resp.Error.Message, tt.errMsg)
				}
			} else {

				if err != nil {
					t.Errorf("DeleteNetwork() unexpected error = %v", err)
				}
				// Check the result for valid arguments case
				result, ok := resp.Result.(bool)
				if !ok {
					t.Errorf("Expected boolean result, got %T", resp.Result)
				}
				if !result {
					t.Errorf("Expected result to be true, got %v", result)
				}
			}
		})
	}
}

func TestDeleteNetwork_SuccessfulDeletion(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteNetwork, t)

	// Set up the mock response for a successful network deletion
	networkID := TestNetworkID

	// Override the arguments to match our test network ID
	req.Arguments = json.RawMessage(`["` + networkID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): FakeResponse{
			StatusCode: http.StatusOK,
			Body:       []byte(`OK`),
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusNotFound,
			Body:         []byte(`Not Found`),
			ErrorMessage: `Not Found`,
			Model:        iaas.Network{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteNetwork method
	resp, err := cpi.DeleteNetwork(req)
	if err != nil {
		t.Errorf("DeleteNetwork() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for successful deletion
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for successful deletion, got %v", result)
	}
}

func TestDeleteNetwork_AlreadyDeleted(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteNetwork, t)

	// Set up the mock response for a network that is already deleted
	networkID := TestNetworkID

	// Override the arguments to match our test network ID
	req.Arguments = json.RawMessage(`["` + networkID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusNotFound,
			Body:         []byte(`Not Found`),
			ErrorMessage: `Not Found`,
			Model:        iaas.Network{},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusNotFound,
			Body:         []byte(`Not Found`),
			ErrorMessage: `Not Found`,
			Model:        iaas.Network{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteNetwork method
	resp, err := cpi.DeleteNetwork(req)
	if err != nil {
		t.Errorf("DeleteNetwork() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for successful deletion (even if already deleted)
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true when network already deleted, got %v", result)
	}
}

func TestDeleteNetwork_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteNetwork, t)

	// Set up a scenario where the API returns an error
	networkID := TestNetworkID

	// Reduce retry count for faster tests
	cpi.Config.RetryCount = 1

	// Override the arguments to match our test network ID
	req.Arguments = json.RawMessage(`["` + networkID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   500,
			Body:         []byte(`Internal Server Error`),
			ErrorMessage: `Internal Server Error`,
			Model:        iaas.Network{},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   500,
			Body:         []byte(`Internal Server Error`),
			ErrorMessage: `Internal Server Error`,
			Model:        iaas.Network{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteNetwork method, expecting an error
	resp, err := cpi.DeleteNetwork(req)

	// This should result in an error
	if err == nil {
		t.Errorf("DeleteNetwork() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}

	// Make sure the response contains the expected error message
	if !strings.Contains(resp.Error.Message, "Internal Server Error") {
		t.Errorf("Error message does not contain expected text, got: %s", resp.Error.Message)
	}

	// Result should be false on error
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if result {
		t.Errorf("Expected result to be false on error, got %v", result)
	}
}

func TestDeleteNetwork_DependencyError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteNetwork, t)

	// Set up a scenario where the API returns a dependency error (e.g., network still has active ports)
	networkID := TestNetworkID

	// Override the arguments to match our test network ID
	req.Arguments = json.RawMessage(`["` + networkID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): iaas.Network{
			Name:   `test-network`,
			Id:     TestNetworkID,
			Status: wait.CreateSuccess,
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   409,
			ErrorMessage: `A conflict has occured`,
			Body:         []byte(`A conflict has occured`),
			Model:        iaas.Network{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteNetwork method, expecting an error
	resp, err := cpi.DeleteNetwork(req)

	// This should result in an error
	if err == nil {
		t.Errorf("DeleteNetwork() expected error, got nil")
	}

	// Verify error in response mentions dependency issue
	if resp.Error == nil || !strings.Contains(resp.Error.Message, "Error when calling `DeleteNetwork`") {
		t.Errorf("Expected dependency error in response, got: %v", resp.Error)
	}

	// Result should be false on error
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if result {
		t.Errorf("Expected result to be false on error, got %v", result)
	}
}

func TestDeleteNetwork_WaitError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteNetwork, t)
	cpi.Config.Timeout = 1
	// Set up a scenario where the API succeeds but waiting for deletion fails
	networkID := TestNetworkID

	// Override the arguments to match our test network ID
	req.Arguments = json.RawMessage(`["` + networkID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): iaas.Network{
			Name:   `test-network`,
			Id:     TestNetworkID,
			Status: wait.CreateSuccess,
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): FakeTimeout{
			After: 2,
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteNetwork method, expecting an error
	resp, err := cpi.DeleteNetwork(req)

	// This should result in an error
	if err == nil {
		t.Errorf("DeleteNetwork() expected error, got nil")
	}

	// Verify error in response mentions wait error
	if resp.Error == nil || !strings.Contains(resp.Error.Message, "context deadline exceeded") {
		t.Errorf("Expected wait error in response, got: %v", resp.Error)
	}

	// Result should be false on error
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if result {
		t.Errorf("Expected result to be false on error, got %v", result)
	}
}

func TestDeleteNetwork_RetrySuccess(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteNetwork, t)

	// Set up a scenario where the API fails initially but succeeds on retry
	networkID := TestNetworkID

	// Set a retry count for testing
	cpi.Config.RetryCount = 2

	// Override the arguments to match our test network ID
	req.Arguments = json.RawMessage(`["` + networkID + `"]`)

	// Mock server that simulates API failure then success
	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusInternalServerError,
				Body:         []byte(`Not Found`),
				ErrorMessage: `Internal Server Error`,
				Model:        iaas.Network{},
			},
			FinallyRespondWith: FakeResponse{
				StatusCode: http.StatusOK,
				Body:       []byte("OK"),
			},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode: http.StatusNotFound,
			Body:       []byte("Not Found"),
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	originalRegion := cpi.Config.RegionID
	defer func() { cpi.Config.RegionID = originalRegion }()

	// Call the DeleteNetwork method
	resp, err := cpi.DeleteNetwork(req)
	if err != nil {
		t.Errorf("DeleteNetwork() failed after retries: %v", err)
		return
	}

	// Check the result
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for successful deletion after retry, got %v", result)
	}
}

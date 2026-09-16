package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

var TestMaxRetryAttempts = 2

func TestDeleteDisk_SuccessfulDeletion(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteDisk, t)

	// Set up the mock response for a successful disk deletion
	diskID := "12345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test disk ID
	req.Arguments = json.RawMessage(`["` + diskID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID): FakeResponse{
			StatusCode: http.StatusOK,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID): oapierror.GenericOpenAPIError{
			StatusCode: http.StatusNotFound,
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{
		config.WithEndpoint(mockServer.URL),
		config.WithoutAuthentication(),
	})
	cpi.SClient = sClient

	// Call the DeleteDisk method
	resp, err := cpi.DeleteDisk(req)
	if err != nil {
		t.Errorf("DeleteDisk() unexpected error = %v", err)
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

func TestDeleteDisk_AlreadyDeleted(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteDisk, t)

	// Set up the mock response for a disk that is already deleted
	diskID := "12345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test disk ID
	req.Arguments = json.RawMessage(`["` + diskID + `"]`)

	// Mock server that simulates disk not found
	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID): oapierror.GenericOpenAPIError{
			StatusCode: http.StatusNotFound,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID): oapierror.GenericOpenAPIError{
			StatusCode: http.StatusNotFound,
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteDisk method
	resp, err := cpi.DeleteDisk(req)
	if err != nil {
		t.Errorf("DeleteDisk() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for successful deletion (even if already deleted)
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true when disk already deleted, got %v", result)
	}
}

func TestDeleteDisk_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteDisk, t)

	// Set up a scenario where the API returns an error
	diskID := "12345678-1234-1234-1234-123456789012"

	// Reduce retry count for faster tests
	cpi.Config.RetryCount = 0

	// Override the arguments to match our test disk ID
	req.Arguments = json.RawMessage(`["` + diskID + `"]`)

	// Create a mock server that returns an error for this specific disk ID
	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/test-region/volumes/%s", cpi.Config.ProjectID, diskID): oapierror.GenericOpenAPIError{
			StatusCode: http.StatusBadGateway,
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteDisk method, expecting an error
	resp, err := cpi.DeleteDisk(req)

	// This should result in an error
	if err == nil {
		t.Errorf("DeleteDisk() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || !resp.Error.OkToRetry {
		t.Errorf("Expected retryable error in response, got OkToRetry=%v",
			resp.Error != nil && resp.Error.OkToRetry)
	}
}

func TestDeleteDisk_RetrySuccess(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteDisk, t)
	cpi.Config.RetryCount = 3

	// Set up a scenario where the API fails initially but succeeds on retry
	diskID := "12345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test disk ID
	req.Arguments = json.RawMessage(`["` + diskID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/test-region/volumes/%s", cpi.Config.ProjectID, diskID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode: http.StatusBadGateway,
			},
			FinallyRespondWith: FakeResponse{
				StatusCode: http.StatusOK,
			},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/test-region/volumes/%s", cpi.Config.ProjectID, diskID): oapierror.GenericOpenAPIError{
			StatusCode: http.StatusNotFound,
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	cpi.Config.RetryCount = 2

	// Call the DeleteDisk method
	resp, err := cpi.DeleteDisk(req)
	if err != nil {
		t.Errorf("DeleteDisk() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for successful deletion after retry
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for successful deletion after retry, got %v", result)
	}
}

package cpi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func TestDeleteSnapshot_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 1 argument for delete_snapshot, got 0",
		},
		{
			name:      "Empty snapshot ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "snapshot_cid must be provided",
		},
		{
			name:      "Valid snapshot ID",
			arguments: json.RawMessage(fmt.Sprintf(`["%s"]`, testSnapshotID)),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.DeleteSnapshot, t)

			// Override arguments for this test
			req.Arguments = tt.arguments
			mockResponses := map[string]any{
				fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/snapshots/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testSnapshotID): FakeResponse{
					StatusCode: 204,
					Body:       []byte(``),
				},
			}
			// Create a mock server that does nothing for argument validation tests
			mockServer := SetupMockClient(cpi, mockResponses)
			defer mockServer.Close()

			resp, err := cpi.DeleteSnapshot(req)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteSnapshot() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DeleteSnapshot() error = %v, want %v", err.Error(), tt.errMsg)
				}
				// Verify error in response
				if !strings.Contains(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = %v, want %v", resp.Error.Message, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("DeleteSnapshot() unexpected error = %v", err)
			}
		})
	}
}

func TestDeleteSnapshot_SuccessfulDeletion(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteSnapshot, t)

	// Override the arguments to match our test snapshot ID
	req.Arguments = json.RawMessage(`["` + testSnapshotID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/snapshots/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testSnapshotID): FakeResponse{
			StatusCode: 204,
			Body:       []byte(``),
		},
	}
	// Create a mock server that does nothing for argument validation tests
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteSnapshot method
	resp, err := cpi.DeleteSnapshot(req)
	if err != nil {
		t.Errorf("DeleteSnapshot() unexpected error = %v", err)
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

func TestDeleteSnapshot_AlreadyDeleted(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteSnapshot, t)

	// Set up the mock response for a snapshot that is already deleted
	testSnapshotID := "22345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test snapshot ID
	req.Arguments = json.RawMessage(`["` + testSnapshotID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/snapshots/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testSnapshotID): FakeResponse{
			StatusCode: 404,
			Body:       []byte(`Not Found`),
		},
	}
	// Create a mock server that does nothing for argument validation tests
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteSnapshot method
	resp, err := cpi.DeleteSnapshot(req)
	if err != nil {
		// Check if the error message contains "not found" which is acceptable
		if !strings.Contains(strings.ToLower(err.Error()), "not found") {
			t.Errorf("DeleteSnapshot() unexpected error = %v", err)
			return
		}
	}

	// Check the result - should be true for successful deletion (even if already deleted)
	// or we have an error with "not found" message
	if err == nil {
		result, ok := resp.Result.(bool)
		if !ok {
			t.Errorf("Expected boolean result, got %T", resp.Result)
			return
		}

		if !result {
			t.Errorf("Expected result to be true when snapshot already deleted, got %v", result)
		}
	}
}

func TestDeleteSnapshot_RetryFailTerminally(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteSnapshot, t)
	cpi.Config.RetryCount = 3

	// Set up a scenario where the API returns an error
	testSnapshotID := "32345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test snapshot ID
	req.Arguments = json.RawMessage(`["` + testSnapshotID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/snapshots/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testSnapshotID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				ErrorMessage: "Internal Server Error",
				Body:         []byte(`Internal Server Error`),
				Model:        iaas.Snapshot{},
			},
			FinallyRespondWith: FakeResponse{
				StatusCode: 401,
				Body:       []byte(`OK`),
			},
		},
	}
	// Create a mock server that does nothing for argument validation tests
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteSnapshot method, expecting an error
	resp, err := cpi.DeleteSnapshot(req)
	// This should not result in an error
	if err == nil {
		t.Errorf("DeleteSnapshot() expected no error, got %s", err)
	}
	if resp.Error.OkToRetry == true {
		t.Errorf("Expected OkToRetry to be false")
	}
}

func TestDeleteSnapshot_RetryFailRetryably(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteSnapshot, t)
	cpi.Config.RetryCount = 3

	// Set up a scenario where the API returns an error
	testSnapshotID := "32345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test snapshot ID
	req.Arguments = json.RawMessage(`["` + testSnapshotID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/snapshots/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testSnapshotID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				ErrorMessage: "Internal Server Error",
				Body:         []byte(`Internal Server Error`),
				Model:        iaas.Snapshot{},
			},
			FinallyRespondWith: FakeResponse{
				StatusCode: 504,
				Body:       []byte(`OK`),
			},
		},
	}
	// Create a mock server that does nothing for argument validation tests
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteSnapshot method, expecting an error
	resp, err := cpi.DeleteSnapshot(req)
	// This should not result in an error
	if err == nil {
		t.Errorf("DeleteSnapshot() expected no error, got %s", err)
	}
	if resp.Error.OkToRetry != true {
		t.Errorf("Expected OkToRetry to be false")
	}
}

func TestDeleteSnapshot_RetrySuccess(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteSnapshot, t)
	cpi.Config.RetryCount = 3

	// Set up a scenario where the API returns an error
	testSnapshotID := "32345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test snapshot ID
	req.Arguments = json.RawMessage(`["` + testSnapshotID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/snapshots/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testSnapshotID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				ErrorMessage: "Internal Server Error",
				Body:         []byte(`Internal Server Error`),
				Model:        iaas.Snapshot{},
			},
			FinallyRespondWith: FakeResponse{
				StatusCode: 200,
				Body:       []byte(`OK`),
			},
		},
	}
	// Create a mock server that does nothing for argument validation tests
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteSnapshot method, expecting an error
	_, err := cpi.DeleteSnapshot(req)
	// This should not result in an error
	if err != nil {
		t.Errorf("DeleteSnapshot() expected no error, got %s", err)
	}
}

func TestDeleteSnapshot_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteSnapshot, t)
	cpi.Config.RetryCount = 3

	// Set up a scenario where the API returns an error
	testSnapshotID := "32345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test snapshot ID
	req.Arguments = json.RawMessage(`["` + testSnapshotID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/snapshots/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testSnapshotID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				ErrorMessage: "Internal Server Error",
				Body:         []byte(`Internal Server Error`),
				Model:        iaas.Snapshot{},
			},
			FinallyRespondWith: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				ErrorMessage: "Internal Server Error",
				Body:         []byte(`Internal Server Error`),
				Model:        iaas.Snapshot{},
			},
		},
	}
	// Create a mock server that does nothing for argument validation tests
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteSnapshot method, expecting an error
	resp, err := cpi.DeleteSnapshot(req)

	// This should result in an error
	if err == nil {
		t.Errorf("DeleteSnapshot() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}

	// Verify the error is retryable
	if resp.Error == nil || !resp.Error.OkToRetry {
		t.Errorf("Expected retryable error in response, got OkToRetry=%v",
			resp.Error != nil && resp.Error.OkToRetry)
	}
}

func TestDeleteSnapshot_LockedError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteSnapshot, t)

	cpi.Config.RetryCount = 1

	// Set up a scenario where the API returns a locked error (snapshot in use)
	testSnapshotID := "42345678-1234-1234-1234-123456789012"

	// Override the arguments to match our test snapshot ID
	req.Arguments = json.RawMessage(`["` + testSnapshotID + `"]`)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/snapshots/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testSnapshotID): oapierror.GenericOpenAPIError{
			StatusCode:   409, // set up actual retryable error code
			ErrorMessage: "in use",
			Body:         []byte(`in use`),
			Model:        iaas.Snapshot{},
		},
	}
	// Create a mock server that does nothing for argument validation tests
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DeleteSnapshot method, expecting an error
	resp, err := cpi.DeleteSnapshot(req)

	// This should result in an error
	if err == nil {
		t.Errorf("DeleteSnapshot() expected error, got nil")
	}

	// Verify error in response indicates snapshot is in use and is retryable
	if resp.Error == nil {
		t.Errorf("Expected error in response, got none")
		return
	}

	if !strings.Contains(resp.Error.Message, "is in use and cannot be deleted") {
		t.Errorf("Expected error message to contain 'is in use and cannot be deleted', got: %s",
			resp.Error.Message)
	}
}

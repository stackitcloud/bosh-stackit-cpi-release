// snapshot_disk_test.go
package cpi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

// Valid UUID-formatted IDs for testing
const (
	testSnapshotDiskID      = "87654321-4321-4321-4321-210987654321"
	testSnapshotID          = "11112222-3333-4444-5555-666677778888"
	testRetrySnapshotDiskID = "87654321-4321-4321-4321-210987654322"
)

func TestSnapshotDisk_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 1 argument for snapshot_disk, got 0",
		},
		{
			name:      "Empty disk ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "disk_cid must be provided",
		},
		{
			name:      "Missing metadata",
			arguments: json.RawMessage(`["` + testSnapshotDiskID + `"]`),
			wantErr:   true,
			errMsg:    "metadata must be provided",
		},
		{
			name:      "Valid arguments",
			arguments: json.RawMessage(`["` + testSnapshotDiskID + `", {"deployment": "test-deployment"}]`),
			wantErr:   false,
			errMsg:    "",
		},
		{
			name:      "Retries",
			arguments: json.RawMessage(`["` + testRetrySnapshotDiskID + `", {"deployment": "test-deployment"}]`),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.SnapshotDisk, t)

			req.Arguments = tt.arguments

			// Set up a mock handler that returns success without making actual API calls
			mockResponses := map[string]any{
				fmt.Sprintf("POST_/v2/projects/%s/regions/%s/snapshots", cpi.Config.ProjectID, cpi.Config.RegionID): &iaas.Snapshot{
					Id:       utils.Ptr(testSnapshotID),
					VolumeId: testSnapshotDiskID,
				},
			}
			mockServer := SetupMockClient(cpi, mockResponses)
			defer mockServer.Close()

			// Run the test
			resp, err := cpi.SnapshotDisk(req)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("SnapshotDisk() error = nil, expected error with message: %v", tt.errMsg)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("SnapshotDisk() error = '%v', want '%v'", err.Error(), tt.errMsg)
				}
				if !strings.Contains(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = %v, want %v", resp.Error.Message, tt.errMsg)
				}
				return
			}

			// Check the result for valid arguments case
			result, ok := resp.Result.(string)
			if !ok {
				t.Errorf("Expected string result (snapshot ID), got %T", resp.Result)
			} else if result != testSnapshotID {
				t.Errorf("Expected result to be %s, got %s", testSnapshotID, result)
			}
		})
	}
}

func TestSnapshotDisk_SuccessfulAfterRetries(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SnapshotDisk, t)
	req.Arguments = json.RawMessage(fmt.Sprintf(`["%s",{}]`, testSnapshotDiskID))
	cpi.Config.RetryCount = 3
	// Set up a mock handler that returns success without making actual API calls
	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/snapshots", cpi.Config.ProjectID, cpi.Config.RegionID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   503,
				ErrorMessage: "Service Unavailable",
				Body:         []byte(``),
				Model:        iaas.Snapshot{},
			},
			FinallyRespondWith: &iaas.Snapshot{
				Id:       utils.Ptr(testSnapshotID),
				VolumeId: testSnapshotDiskID,
			},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = lib.NewRetryableStackitClient(sClient, cpi.Config.RetryCount, cpi.Log, lib.WithDelayBaseInSeconds(0))

	// Run the test
	resp, err := cpi.SnapshotDisk(req)
	if err != nil {
		t.Errorf("expected no error but got '%v'", err)
	}

	if resp.Result.(string) != testSnapshotID {
		t.Errorf("expected snapshot id to be '%s' but got '%s'", testSnapshotID, resp.Result)
	}
}

func TestSnapshotDisk_DiskNotFound(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SnapshotDisk, t)

	// Set up the mock response for a disk that doesn't exist
	diskID := "non-existing-disk-id"

	// Create metadata for snapshot
	metadata := map[string]any{
		"deployment": "test-deployment",
	}

	// Override the arguments to match our test ID and metadata
	args := []any{diskID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	cpi.Config.RetryCount = 1
	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/snapshots", cpi.Config.ProjectID, cpi.Config.RegionID): oapierror.GenericOpenAPIError{
			StatusCode:   404,
			Body:         []byte(`Not Found`),
			ErrorMessage: `Not Found`,
			Model:        iaas.Snapshot{},
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the SnapshotDisk method, expect an error
	resp, err := cpi.SnapshotDisk(req)

	// This should result in an error
	if err == nil {
		t.Errorf("SnapshotDisk() expected error when disk not found, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

func TestSnapshotDisk_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SnapshotDisk, t)

	// Set up a scenario where the API returns an error
	diskID := testSnapshotDiskID

	// Create metadata for snapshot
	metadata := map[string]any{
		"deployment": "test-deployment",
	}

	// Override the arguments to match our test ID and metadata
	args := []any{diskID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON
	cpi.Config.RetryCount = 1
	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/snapshots", cpi.Config.ProjectID, cpi.Config.RegionID): oapierror.GenericOpenAPIError{
			StatusCode:   503,
			ErrorMessage: "Service Unavailable",
			Body:         []byte{},
			Model:        iaas.Snapshot{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the SnapshotDisk method, expecting an error
	resp, err := cpi.SnapshotDisk(req)

	// This should result in an error
	if err == nil {
		t.Errorf("SnapshotDisk() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}

	// This should be marked as retryable for bosh

	if resp.Error.OkToRetry != true {
		t.Error("expected error to be retryable")
	}
}

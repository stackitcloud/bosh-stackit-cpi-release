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

func TestHasDisk_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 1 argument for has_disk, got 0",
		},
		{
			name:      "Empty disk ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "disk_cid must be provided",
		},
		{
			name:      "Valid disk ID",
			arguments: json.RawMessage(`["12345678-1234-1234-1234-123456789012"]`),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := lib.HasDiskArgs{}

			err := json.Unmarshal(tt.arguments, &args)

			if err == nil {
				err = args.Validate()
			}

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateDiskArgs Validate() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("CreateDiskArgs Validate() error = '%v', want '%v'", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("CreateDiskArgs Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestHasDisk_DiskExists(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.HasDisk, t)

	// Set up the mock response for an existing disk - use UUID format
	diskID := "11111111-1111-1111-1111-111111111111"

	// Override the arguments to match our test disk ID
	req.Arguments = json.RawMessage(`["` + diskID + `"]`)

	// Mock response with an existing disk
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID): iaas.Volume{
			AvailabilityZone: "some-az",
			Id:               utils.Ptr(diskID),
			Name:             utils.Ptr("test-disk"),
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the HasDisk method
	resp, err := cpi.HasDisk(req)
	if err != nil {
		t.Errorf("HasDisk() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for an existing disk
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for existing disk, got %v", result)
	}
}

func TestHasDisk_DiskDoesNotExist(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.HasDisk, t)

	// Set up the mock response for a non-existing disk - use UUID format
	diskID := "22222222-2222-2222-2222-222222222222"

	// Override the arguments to match our test disk ID
	req.Arguments = json.RawMessage(`["` + diskID + `"]`)

	// Mock response with a null disk to simulate the disk not existing
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID): oapierror.GenericOpenAPIError{
			StatusCode: 404,
			Body:       []byte(`Not Found`),
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the HasDisk method
	resp, err := cpi.HasDisk(req)
	if err != nil {
		t.Errorf("HasDisk() unexpected error = %v", err)
		return
	}

	// Check the result - should be false for a non-existing disk
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if result {
		t.Errorf("Expected result to be false for non-existing disk, got %v", result)
	}
}

func TestHasDisk_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.HasDisk, t)

	// Set up a scenario where the API returns an error
	diskID := "error-disk-id"

	// Override the arguments to match our test disk ID
	req.Arguments = json.RawMessage(`["` + diskID + `"]`)

	// Create a mock server that returns an error for this specific disk ID
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, diskID) {
			// Return a 500 error
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Teapot: %s"}`, r.URL.Path), http.StatusTeapot)
	}))
	defer mockServer.Close()

	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the HasDisk method, expecting an error
	resp, err := cpi.HasDisk(req)

	// This should result in an error
	if err == nil {
		t.Errorf("HasDisk() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

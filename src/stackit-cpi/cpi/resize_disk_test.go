package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

const (
	testResizeDiskID      = "12345678-1234-1234-1234-123456789012"
	testResizeDiskNewSize = int64(20480)
)

func TestResizeDisk_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 2 arguments for resize_disk, got 0",
		},
		{
			name:      "Empty disk ID",
			arguments: json.RawMessage(`["", 20480]`),
			wantErr:   true,
			errMsg:    "disk_cid must be provided",
		},
		{
			name:      "Invalid new size (zero)",
			arguments: json.RawMessage(`["disk-id", 0]`),
			wantErr:   true,
			errMsg:    "new_size must be a positive integer representing MiB",
		},
		{
			name:      "Invalid new size (negative)",
			arguments: json.RawMessage(`["disk-id", -1024]`),
			wantErr:   true,
			errMsg:    "new_size must be a positive integer representing MiB",
		},
		{
			name:      "Valid arguments",
			arguments: json.RawMessage(`["` + testResizeDiskID + `", 20480]`),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.ResizeDisk, t)

			// Override arguments for this test
			req.Arguments = tt.arguments

			mockResponses := map[string]any{
				fmt.Sprintf("POST_/v2/projects/%s/regions/%s/volumes/%s/resize", cpi.Config.ProjectID, cpi.Config.RegionID, testResizeDiskID): FakeResponse{
					StatusCode: 202,
					Body:       []byte{},
				},
			}

			mockServer := SetupMockClient(cpi, mockResponses)
			defer mockServer.Close()

			resp, err := cpi.ResizeDisk(req)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("ResizeDisk() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ResizeDisk() error = %v, want %v", err.Error(), tt.errMsg)
				}
				// Verify error in response
				if !strings.Contains(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = %v, want %v", resp.Error.Message, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("ResizeDisk() unexpected error = %v", err)
			}
		})
	}
}

func TestResizeDisk_SuccessfulResize(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.ResizeDisk, t)

	// Override the arguments to match our test data
	args := []any{testResizeDiskID, testResizeDiskNewSize}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/volumes/%s/resize", cpi.Config.ProjectID, cpi.Config.RegionID, testResizeDiskID): FakeResponse{
			StatusCode: 202,
			Body:       []byte{},
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithEndpoint(mockServer.URL), config.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the ResizeDisk method
	resp, err := cpi.ResizeDisk(req)
	if err != nil {
		t.Errorf("ResizeDisk() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for successful operation
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true, got %v", result)
	}
}

func TestResizeDisk_DiskNotFound(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.ResizeDisk, t)

	newSize := int64(20480)

	// Override the arguments to match our test data
	args := []interface{}{testResizeDiskID, newSize}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/volumes/%s/resize", cpi.Config.ProjectID, cpi.Config.RegionID, testResizeDiskID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusNotFound,
			Body:         []byte(`Not Found`),
			ErrorMessage: `Not Found`,
			Model:        nil,
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	resp, err := cpi.ResizeDisk(req)

	if err == nil {
		t.Errorf("ResizeDisk() expected error for non-existent disk, got nil")
		return
	}

	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
		return
	}

	if !strings.Contains(strings.ToLower(resp.Error.Message), "not found") {
		t.Errorf("Expected 'not found' in error message, got: %s", resp.Error.Message)
	}
}

func TestResizeDisk_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.ResizeDisk, t)

	newSize := int64(20480)

	args := []interface{}{testResizeDiskID, newSize}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/volumes/%s/resize", cpi.Config.ProjectID, cpi.Config.RegionID, testResizeDiskID): oapierror.GenericOpenAPIError{
			StatusCode:   500,
			Body:         []byte(`Internal Server Error`),
			ErrorMessage: `Internal Server Error`,
			Model:        nil,
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the ResizeDisk method
	resp, err := cpi.ResizeDisk(req)

	// This should result in an error
	if err == nil {
		t.Errorf("ResizeDisk() expected error, got nil")
		return
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
		return
	}

	// Check if error message contains expected error message
	if !strings.Contains(resp.Error.Message, "Error when calling `ResizeVolume`") {
		t.Errorf("Expected 'Error when calling `ResizeVolume`' in error message, got: %s", resp.Error.Message)
	}
}

func TestResizeDisk_QuotaExceeded(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.ResizeDisk, t)

	newSize := int64(102400) // 100 GiB

	// Override the arguments to match our test data
	args := []interface{}{testResizeDiskID, newSize}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	// Mock a 403 Forbidden for quota exceeded
	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/volumes/%s/resize", cpi.Config.ProjectID, cpi.Config.RegionID, testResizeDiskID): oapierror.GenericOpenAPIError{
			StatusCode:   403,
			Body:         []byte(`quota exceeded`),
			ErrorMessage: `quota exceeded`,
			Model:        nil,
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the ResizeDisk method
	resp, err := cpi.ResizeDisk(req)

	// This should result in an error
	if err == nil {
		t.Errorf("ResizeDisk() expected error for quota exceeded, got nil")
		return
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
		return
	}

	// Check if the error message contains "quota exceeded"
	if !strings.Contains(strings.ToLower(resp.Error.Message), "quota exceeded") {
		t.Errorf("Expected 'quota exceeded' in error message, got: %s", resp.Error.Message)
	}
}

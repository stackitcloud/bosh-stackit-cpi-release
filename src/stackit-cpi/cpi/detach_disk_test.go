package cpi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func TestDetachDisk_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 2 arguments for detach_disk, got 0",
		},
		{
			name:      "Empty server ID",
			arguments: json.RawMessage(`["", "` + TestDetachDiskID + `"]`),
			wantErr:   true,
			errMsg:    "vm_cid must be provided",
		},
		{
			name:      "Empty disk ID",
			arguments: json.RawMessage(`["` + TestDetachServerID + `", ""]`),
			wantErr:   true,
			errMsg:    "disk_cid must be provided",
		},
		{
			name:      "Only server ID provided",
			arguments: json.RawMessage(`["` + TestDetachServerID + `"]`),
			wantErr:   true,
			errMsg:    "expected 2 arguments for detach_disk, got 1",
		},
		{
			name:      "Valid arguments",
			arguments: json.RawMessage(`["` + TestDetachServerID + `", "` + TestDetachDiskID + `"]`),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.DetachDisk, t)

			// Override arguments for this test
			req.Arguments = tt.arguments

			listResp := map[string]any{
				"items": []iaas.VolumeAttachment{
					{
						DeleteOnTermination: utils.Ptr(true),
						ServerId:            utils.Ptr(TestDetachServerID),
						VolumeId:            utils.Ptr(TestDetachDiskID),
					},
				},
			}
			listRespBytes, err := json.Marshal(listResp)
			if err != nil {
				t.Errorf("failed setting up fake response: %s", err)
			}
			if tt.name == "Valid arguments" {
				mockResponses := map[string]any{
					fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments", cpi.Config.ProjectID, cpi.Config.RegionID, TestDetachServerID): FakeResponse{
						StatusCode: 200,
						Body:       listRespBytes,
					},
					fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestDetachServerID, TestDetachDiskID): FakeResponse{
						StatusCode: 204,
						Body:       []byte(``),
					},
					fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestDetachServerID, TestDetachDiskID): oapierror.GenericOpenAPIError{
						StatusCode:   404,
						Body:         []byte(``),
						ErrorMessage: "Not Found",
						Model:        iaas.VolumeAttachment{},
					},
				}
				mockServer := SetupMockClient(cpi, mockResponses)
				defer mockServer.Close()

				// Run the test
				resp, err := cpi.DetachDisk(req)
				if err != nil {
					t.Errorf("DetachDisk() unexpected error = %v", err)
				}

				// Check the result for valid arguments case
				result, ok := resp.Result.(bool)
				if !ok {
					t.Errorf("Expected boolean result, got %T", resp.Result)
				}
				if !result {
					t.Errorf("Expected result to be true, got %v", result)
				}
				return
			}

			resp, err := cpi.DetachDisk(req)

			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("DetachDisk() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DetachDisk() error = %v, want %v", err.Error(), tt.errMsg)
				}
				// Verify error in response
				if !strings.Contains(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = %v, want %v", resp.Error.Message, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("DetachDisk() unexpected error = %v", err)
			}
		})
	}
}

func TestDetachDisk_SuccessfulDetachment(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DetachDisk, t)

	// Override the arguments to match our test IDs
	req.Arguments = json.RawMessage(`["` + TestServerID + `", "` + TestDiskID + `"]`)
	listResp := map[string]any{
		"items": []iaas.VolumeAttachment{
			{
				DeleteOnTermination: utils.Ptr(true),
				ServerId:            utils.Ptr(TestServerID),
				VolumeId:            utils.Ptr(TestDiskID),
			},
		},
	}
	listRespBytes, _ := json.Marshal(listResp)

	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID): FakeResponse{
			StatusCode: 200,
			Body:       listRespBytes,
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): FakeResponse{
			StatusCode: 204,
			Body:       []byte(``),
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): oapierror.GenericOpenAPIError{
			StatusCode:   404,
			ErrorMessage: "Not Found",
			Body:         []byte(`Not Found`),
			Model:        iaas.VolumeAttachment{},
		},
	}
	// Mock server that simulates successful detachment
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DetachDisk method
	resp, err := cpi.DetachDisk(req)
	if err != nil {
		t.Errorf("DetachDisk() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for successful detachment
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for successful detachment, got %v", result)
	}
}

func TestDetachDisk_AlreadyDetached(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DetachDisk, t)

	// Override the arguments to match our test IDs
	req.Arguments = json.RawMessage(`["` + TestServerID + `", "` + TestDiskID + `"]`)

	listResp := map[string]any{
		"items": []iaas.VolumeAttachment{},
	}
	listRespBytes, _ := json.Marshal(listResp)
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID): FakeResponse{
			StatusCode: 200,
			Body:       listRespBytes,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): oapierror.GenericOpenAPIError{
			StatusCode:   404,
			ErrorMessage: "Not Found",
			Body:         []byte(`Not Found`),
			Model:        iaas.VolumeAttachment{},
		},
	}
	// Mock server that simulates successful detachment
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DetachDisk method
	resp, err := cpi.DetachDisk(req)
	if err != nil {
		t.Errorf("DetachDisk() unexpected error = %v", err)
		return
	}

	// Check the result - should be true for successful detachment (even if already detached)
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true when disk already detached, got %v", result)
	}
}

func TestDetachDisk_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DetachDisk, t)

	cpi.Config.RetryCount = 2

	req.Arguments = json.RawMessage(`["` + TestServerID + `", "` + TestDiskID + `"]`)

	// Create a mock server that returns an error
	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestDetachServerID, TestDetachDiskID): &oapierror.GenericOpenAPIError{
			StatusCode:   500,
			Body:         []byte(``),
			ErrorMessage: "Internal Server Error",
			Model:        iaas.VolumeAttachment{},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestDetachServerID, TestDetachDiskID): oapierror.GenericOpenAPIError{
			StatusCode:   500,
			Body:         []byte(``),
			ErrorMessage: "Internal Server Error",
			Model:        iaas.VolumeAttachment{},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the DetachDisk method, expecting an error
	resp, err := cpi.DetachDisk(req)

	// This should result in an error
	if err == nil {
		t.Errorf("DetachDisk() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

func TestDetachDisk_RetrySuccess(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DetachDisk, t)

	// Set a minimal retry count for faster test execution
	cpi.Config.RetryCount = 3

	// Override the arguments to match our test IDs
	req.Arguments = json.RawMessage(`["` + TestServerID + `", "` + TestDiskID + `"]`)
	listResp := map[string]any{
		"items": []iaas.VolumeAttachment{
			{
				DeleteOnTermination: utils.Ptr(true),
				ServerId:            utils.Ptr(TestServerID),
				VolumeId:            utils.Ptr(TestDiskID),
			},
		},
	}
	listRespBytes, _ := json.Marshal(listResp)

	mockResponses := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): FakeResponse{
			StatusCode: 204,
			Body:       []byte(``),
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID): FakeResponse{
			StatusCode: 200,
			Body:       listRespBytes,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				ErrorMessage: "Internal Server Error",
				Body:         []byte(`Internal Server Error`),
				Model:        iaas.VolumeAttachment{},
			},
			FinallyRespondWith: oapierror.GenericOpenAPIError{
				StatusCode:   404,
				ErrorMessage: "Not Found",
				Body:         []byte(`Not Found`),
				Model:        iaas.VolumeAttachment{},
			},
		},
	}
	// Create a mock server that fails twice then succeeds
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Execute the DetachDisk method with retries built-in to the CPI implementation
	resp, err := cpi.DetachDisk(req)
	if err != nil {
		t.Errorf("DetachDisk() failed after retries: %v", err)
		return
	}

	// The result should be true for successful detachment after retry
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for successful detachment after retry, got %v", result)
	}
}

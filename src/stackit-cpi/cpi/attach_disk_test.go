package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

func TestAttachDisk_Success(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.AttachDisk, t)
	req.Arguments = json.RawMessage(fmt.Sprintf(`["%s","%s"]`, TestServerID, TestDiskID))

	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID):                                   TestServerPayload,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestDiskID):                                     TestDiskPayload,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): TestDiskAttachmentPayload,
		fmt.Sprintf("PUT_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): TestDiskAttachmentPayload,
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	resp, err := cpi.AttachDisk(req)
	if err != nil {
		t.Fatalf("AttachDisk() unexpected error = %v", err)
	}

	expectedHint := map[string]any{
		"path":      fmt.Sprintf("/dev/disk/by-id/virtio-%s", TestDiskID[:20]),
		"volume_id": fmt.Sprintf("/dev/disk/by-id/virtio-%s", TestDiskID[:20]),
	}
	resMap := resp.Result.(map[string]any)

	// deep comparing maps is hard and fiddly, serializing them and comparing the bytes isn't

	expBytes, _ := json.Marshal(expectedHint)
	resBytes, _ := json.Marshal(resMap)

	if !slices.Equal(expBytes, resBytes) {
		// this should contain the disk_hint
		t.Errorf("Expected result to contain disk_hint ID %s, got %v", expectedHint, resMap)
	}
}

func TestAttachDisk_APIError(t *testing.T) {
	tests := []struct {
		name           string
		errorResponse  string
		statusCode     int
		wantRetryable  bool
		errMsgContains string
	}{
		{
			name:           "DiskNotFound",
			errorResponse:  `{"code": 404, "message": "Volume not found"}`,
			statusCode:     http.StatusNotFound,
			wantRetryable:  false,
			errMsgContains: "Error when calling `AddVolumeToServer`",
		},
		{
			name:           "ServerNotFound",
			errorResponse:  `{"code": 404, "message": "Server not found"}`,
			statusCode:     http.StatusNotFound,
			wantRetryable:  false,
			errMsgContains: "Error when calling `AddVolumeToServer`",
		},
		{
			name:           "DiskAlreadyAttached",
			errorResponse:  `{"code": 409, "message": "Volume already attached"}`,
			statusCode:     http.StatusConflict,
			wantRetryable:  false,
			errMsgContains: "Error when calling `AddVolumeToServer`",
		},
		{
			name:           "Timeout",
			errorResponse:  `{"code": 408, "message": "Request timeout"}`,
			statusCode:     http.StatusRequestTimeout,
			wantRetryable:  true,
			errMsgContains: "Error when calling `AddVolumeToServer`",
		},
		{
			name:           "ServerError",
			errorResponse:  `{"code": 500, "message": "Internal server error"}`,
			statusCode:     http.StatusInternalServerError,
			wantRetryable:  true,
			errMsgContains: "Error when calling `AddVolumeToServer`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.AttachDisk, t)
			req.Arguments = json.RawMessage(fmt.Sprintf(`["%s","%s"]`, TestServerID, TestDiskID))

			apiErr := oapierror.GenericOpenAPIError{
				StatusCode: tt.statusCode,
				Body:       []byte(tt.errorResponse),
			}

			mockResponses := map[string]any{
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID):                                   TestServerPayload,
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestDiskID):                                     TestDiskPayload,
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): TestDiskAttachmentPayload,
				fmt.Sprintf("PUT_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): apiErr,
			}
			mockServer := SetupMockClient(cpi, mockResponses)
			defer mockServer.Close()

			resp, err := cpi.AttachDisk(req)

			if err == nil {
				t.Fatalf("expected error, got nil (resp=%+v)", resp)
			}
			if resp == nil || resp.Error == nil || resp.Error.Message == "" {
				t.Fatalf("expected response error, got: %+v", resp)
			}
			if !strings.Contains(resp.Error.Message, tt.errMsgContains) {
				t.Errorf("error message %q does not contain %q", resp.Error.Message, tt.errMsgContains)
			}
			if resp.Error.OkToRetry != tt.wantRetryable {
				t.Errorf("test % sexpected retry '%v' got '%v'", tt.name, tt.wantRetryable, resp.Error.OkToRetry)
			}
		})
	}
}

// TestAttachDisk_AZMismatch Tests that attaching disks across availability zones fails gracefully
func TestAttachDisk_AZMismatch(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.AttachDisk, t)
	req.Arguments = json.RawMessage(fmt.Sprintf(`["%s","%s"]`, TestServerID, TestDiskID))

	TestServerDifferentAZ := map[string]any{
		"id":               TestServerID,
		"availabilityZone": "zone-a",
		"machineType":      "provided",
		"name":             "fake",
	}

	TestVolumeDifferentAZ := map[string]any{
		"id":               TestDiskID,
		"availabilityZone": "zone-b", // different from server -> trigger mismatch
	}
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID):                                   TestServerDifferentAZ,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestDiskID):                                     TestVolumeDifferentAZ,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): TestDiskAttachmentPayload,
		fmt.Sprintf("PUT_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): TestDiskAttachmentPayload,
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the AttachDisk method, expecting an error
	resp, err := cpi.AttachDisk(req)
	// This should result in an error
	if err == nil {
		t.Errorf("AttachDisk() expected error for AZ mismatch, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}

	msg := resp.Error.Message

	if !strings.Contains(msg, "cannot attach disk") {
		t.Errorf("error message missing 'cannot attach disk': %q", msg)
	}

	// Check if the error message contains AZ mismatch text
	// The error detection in attach_disk.go identifies AZ mismatch from the API error but does not get to there
	// as there is a check before that point that returns similar error
	if !strings.Contains(msg, "different availability zones") {
		t.Errorf("Error message should mention different availability zones, got: %s",
			resp.Error.Message)
	}

	// Also check for the informative message
	if !strings.Contains(msg, "same AZ") {
		t.Errorf("error message missing 'same AZ': %q", msg)
	}
}

func TestAttachDisk_WaitError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.AttachDisk, t)
	req.Arguments = json.RawMessage(fmt.Sprintf(`["%s","%s"]`, TestServerID, TestDiskID))
	cpi.Config.Timeout = 1
	cpi.Config.RetryCount = 2

	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID):                                   TestServerPayload,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestDiskID):                                     TestDiskPayload,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): FakeTimeout{After: 2},
		fmt.Sprintf("PUT_/v2/projects/%s/regions/%s/servers/%s/volume-attachments/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID, TestDiskID): TestDiskAttachmentPayload,
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the AttachDisk method, expecting an error during the wait operation
	resp, err := cpi.AttachDisk(req)

	if err == nil {
		t.Fatalf("expected wait timeout, got nil (resp=%+v)", resp)
	}
	if resp == nil || resp.Error == nil {
		t.Fatalf("expected RPC error, got: %+v", resp)
	}
	if !strings.Contains(resp.Error.Message, "timed out waiting for attachment") &&
		!strings.Contains(resp.Error.Message, "context deadline exceeded") {
		t.Errorf("timeout text missing in: %q", resp.Error.Message)
	}
}

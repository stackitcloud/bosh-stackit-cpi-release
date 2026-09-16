package cpi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

// Valid UUID-formatted IDs for testing
const (
	ExistingServerID            = "12345678-1234-1234-1234-123456789012"
	ServerIDWithMissingVolumes  = "12345678-1234-1234-1234-000000000000"
	ServerIDWithAPIFailureError = "12345678-1234-1234-1234-000000000001"
	testGetDisksDisk1ID         = "87654321-4321-4321-4321-210987654321"
	testGetDisksDisk2ID         = "76543210-3210-3210-3210-109876543210"
)

func TestGetDisks(t *testing.T) {
	tests := []struct {
		name                 string
		arguments            json.RawMessage
		expectedResponse     []string
		wantErr              bool
		errMsg               string
		expectRetryableError bool
	}{
		{
			name:      "Missing arguments",
			arguments: json.RawMessage(`[]`),
			wantErr:   true,
			errMsg:    "expected 1 argument for get_disks, got 0",
		},
		{
			name:      "Empty server ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "vm_cid must be provided",
		},
		{
			name:             "Valid server ID",
			arguments:        json.RawMessage(fmt.Sprintf(`["%s"]`, ExistingServerID)),
			expectedResponse: []string{"abc-def", "def-hij"},
			wantErr:          false,
		},
		{
			name:             "server ID with no disks attached",
			arguments:        json.RawMessage(fmt.Sprintf(`["%s"]`, ServerIDWithMissingVolumes)),
			wantErr:          false,
			expectedResponse: []string{},
			errMsg:           "server not found",
		},
		{
			name:                 "50x error from api",
			arguments:            json.RawMessage(fmt.Sprintf(`["%s"]`, ServerIDWithAPIFailureError)),
			wantErr:              true,
			errMsg:               "Service Unavailable",
			expectRetryableError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.HasDisk, t)

			someFakeID := uuid.NewString()

			mockResponse := map[string]any{
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, ExistingServerID): iaas.Server{
					Id: utils.Ptr(ExistingServerID),
					BootVolume: &iaas.BootVolume{
						Id: utils.Ptr(someFakeID),
					},
					Volumes: []string{someFakeID, "abc-def", "def-hij"},
				},
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, ServerIDWithMissingVolumes): iaas.Server{
					Id: utils.Ptr(ExistingServerID),
					BootVolume: &iaas.BootVolume{
						Id: utils.Ptr(someFakeID),
					},
					Volumes: []string{someFakeID},
				},
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, ServerIDWithAPIFailureError): oapierror.GenericOpenAPIError{
					StatusCode:   503,
					Body:         []byte("Service Unavailable"),
					ErrorMessage: "Service Unavailable",
					Model:        iaas.VolumeAttachmentListResponse{},
				},
			}
			testServer := SetupMockClient(cpi, mockResponse)
			defer testServer.Close()
			req.Arguments = tt.arguments
			resp, err := cpi.GetDisks(req)

			// Check validation results
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got none")
					return
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to match: '%s', got: '%s'", tt.errMsg, err.Error())
					return
				}
				if resp.Error == nil || resp.Error.OkToRetry != tt.expectRetryableError {
					t.Errorf("expecrted retryable to be '%v'", tt.expectRetryableError)
					return
				}
			}
		})
	}
}

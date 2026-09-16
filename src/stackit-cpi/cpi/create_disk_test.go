package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	config "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"
)

func TestCreateDisk_Success(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateDisk, t)

	// Set a shorter timeout for testing
	cpi.Config.Timeout = 2
	cpi.Config.RetryCount = 1

	diskID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
	diskSizeMB := int64(10240)                       // 10 GB in MB
	serverId := "12345678-1234-1234-1234-123456789012"

	// Set up arguments - BOSH passes size in MB
	req.Arguments = json.RawMessage(fmt.Sprintf(`[%d, {}, "%s"]`, diskSizeMB, serverId))

	// Mock responses for successful disk creation - important to use proper UUID
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverId): iaas.Server{
			AvailabilityZone: utils.Ptr("eu01-1"),
			Id:               utils.Ptr(serverId),
			Name:             "requires-persistent-disk",
			Status:           utils.Ptr(wait.ServerActiveStatus), // Add status immediately as available
		},
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/volumes", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.Volume{
			AvailabilityZone: "eu01-1",
			Id:               utils.Ptr(diskID),
			Name:             utils.Ptr("bosh-disk-12345678-1234-1234-1234-123456789012-test-request-id"),
			Status:           utils.Ptr(wait.VolumeAvailableStatus), // Add status immediately as available
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID): iaas.Volume{
			AvailabilityZone: "eu01-1",
			Id:               utils.Ptr(diskID),
			Name:             utils.Ptr("bosh-disk-12345678-1234-1234-1234-123456789012-test-request-id"),
			Status:           utils.Ptr(wait.VolumeAvailableStatus), // Add status immediately as available
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the CreateDisk method
	resp, err := cpi.CreateDisk(req)
	if err != nil {
		t.Errorf("CreateDisk() unexpected error = %v", err)
		return
	}

	// Check result
	result, ok := resp.Result.(string)
	if !ok {
		t.Errorf("Expected string result, got %T", resp.Result)
		return
	}

	if result != diskID {
		t.Errorf("Expected disk ID %s, got %s", diskID, result)
	}
}

func TestCreateDisk_WithCustomProperties(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateDisk, t)

	// Set a shorter timeout for testing
	cpi.Config.Timeout = 1
	cpi.Config.RetryCount = 1

	diskID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
	diskSizeMB := int64(10240)                       // 10 GB in MB
	serverID := "12345678-1234-1234-1234-123456789012"
	performanceClass := "storage_premium_perf2"

	// Set up arguments with custom properties - BOSH passes size in MB
	req.Arguments = json.RawMessage(fmt.Sprintf(`[%d, {"type": "%s", "encrypted": true}, "%s"]`,
		diskSizeMB, performanceClass, serverID))

	expectedResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, serverID): TestServerPayload,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/volumes", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.Volume{
			AvailabilityZone: "eu01-1",
			Id:               utils.Ptr(diskID),
			Name:             utils.Ptr("bosh-disk-12345678-1234-1234-1234-123456789012-test-request-id"),
			Size:             utils.Ptr(int64(10)), // 10 GB (API expects GB)
			Status:           utils.Ptr(wait.VolumeAvailableStatus),
			PerformanceClass: utils.Ptr(performanceClass),
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID): iaas.Volume{
			AvailabilityZone: "eu01-1",
			Id:               utils.Ptr(diskID),
			Name:             utils.Ptr("bosh-disk-12345678-1234-1234-1234-123456789012-test-request-id"),
			Size:             utils.Ptr(int64(10)), // 10 GB (API expects GB)
			Status:           utils.Ptr(wait.VolumeAvailableStatus),
			PerformanceClass: utils.Ptr(performanceClass),
		},
	}
	mockServer := SetupMockClient(cpi, expectedResponses)
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = sClient

	// Call the CreateDisk method
	resp, err := cpi.CreateDisk(req)
	if err != nil {
		t.Errorf("CreateDisk() unexpected error = %v", err)
		return
	}

	// Check result
	result, ok := resp.Result.(string)
	if !ok {
		t.Errorf("Expected string result, got %T", resp.Result)
		return
	}

	if result != diskID {
		t.Errorf("Expected disk ID %s, got %s", diskID, result)
	}
}

func TestCreateDisk_APIError(t *testing.T) {
	tests := []struct {
		name          string
		upstreamError oapierror.GenericOpenAPIError
		statusCode    int
		wantRetryable bool
	}{
		{
			name: "Forbidden",
			upstreamError: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusForbidden,
				ErrorMessage: `Forbidden`,
			},
			wantRetryable: false,
		},
		{
			name: "Internal Server Error",
			upstreamError: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusInternalServerError,
				ErrorMessage: `Internal Server Error`,
			},
			wantRetryable: true,
		},
		{
			name: "Unauthorized",
			upstreamError: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusUnauthorized,
				ErrorMessage: `Unathorized`,
			},
			wantRetryable: false,
		},
		{
			name: "Gateway Timeout",
			upstreamError: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusGatewayTimeout,
				ErrorMessage: `Unathorized`,
			},
			wantRetryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.CreateDisk, t)

			cpi.Config.Timeout = 1
			cpi.Config.RetryCount = 0

			diskSizeMB := int64(10240) // 10 GB in MB

			// Set up arguments - BOSH passes size in MB
			req.Arguments = json.RawMessage(fmt.Sprintf(`[%d, {}, "%s"]`, diskSizeMB, TestServerID))
			mockResponses := map[string]any{
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerID): TestServerPayload,
				fmt.Sprintf("POST_/v2/projects/%s/regions/%s/volumes", cpi.Config.ProjectID, cpi.Config.RegionID):                 tt.upstreamError,
			}

			mockServer := SetupMockClient(cpi, mockResponses)
			resp, err := cpi.CreateDisk(req)
			defer mockServer.Close()
			if err == nil {
				t.Errorf("%s expected error to not be nil", tt.name)
			}
			if tt.wantRetryable != resp.Error.OkToRetry {
				t.Errorf("%s expected result to be retryable %v but was %v", tt.name, tt.wantRetryable, resp.Error.OkToRetry)
			}
		})
	}
}

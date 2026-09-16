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
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"
)

func TestCreateNetwork_Success(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateNetwork, t)

	// Set a shorter timeout for testing
	cpi.Config.Timeout = 3
	cpi.Config.RetryCount = 0

	networkID := "12345678-1234-1234-1234-123456789012" // Valid UUID format

	// Set up arguments for a valid network creation
	req.Arguments = json.RawMessage(`[{
		"type": "manual",
		"cloud_properties": {
			"net_id": "test-net-id",
			"security_groups": ["default"],
			"nameservers": ["8.8.8.8", "1.1.1.1"]
		},
		"range": "192.168.1.0/24",
		"gateway": "192.168.1.1",
		"netmask_bits": 24
	}]`)
	expectedNameServers := []string{"1.2.3.4", "5.6.7.8"}

	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks", cpi.Config.ProjectID, cpi.Config.RegionID): &iaas.Network{
			Name:   "test-network",
			Id:     networkID,
			Status: wait.CreateSuccess,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, networkID): &iaas.Network{
			Name:   "test-network",
			Id:     networkID,
			Status: wait.CreateSuccess,
			Ipv4: &iaas.NetworkIPv4{
				Nameservers: expectedNameServers,
				Prefixes:    []string{"192.168.1.0/24"},
				Gateway:     *iaas.NewNullableString(utils.Ptr("192.168.1.1")),
			},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = sClient

	// Call the CreateNetwork method
	resp, err := cpi.CreateNetwork(req)
	if err != nil {
		t.Errorf("CreateNetwork() unexpected error = %v", err)
		return
	}

	// Check result
	result, ok := resp.Result.([]any)
	if !ok {
		t.Errorf("Expected result to be a slice of any, got %T", resp.Result)
		return
	}
	if result[0].(string) != networkID {
		t.Errorf("Expected result to be true for successful network creation, got %v", result)
	}
}

func TestCreateNetwork_WithDefaultNameservers(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateNetwork, t)

	// Set a shorter timeout for testing
	cpi.Config.Timeout = 3
	cpi.Config.RetryCount = 0

	networkID := "12345678-1234-1234-1234-123456789012" // Valid UUID format

	// Set up arguments without nameservers
	req.Arguments = json.RawMessage(`[{
		"type": "manual",
		"cloud_properties": {
			"net_id": "test-net-id",
			"security_groups": ["default"]
		},
		"dns": ["1.2.3.4", "5.6.7.8"],
		"range": "192.168.1.0/24",
		"gateway": "192.168.1.1",
		"netmask_bits": 24
	}]`)

	expectedNameServers := []string{"1.2.3.4", "5.6.7.8"}
	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks", cpi.Config.ProjectID, cpi.Config.RegionID): &iaas.Network{
			Name:   "test-network",
			Id:     networkID,
			Status: wait.CreateSuccess,
			Ipv4: &iaas.NetworkIPv4{
				Nameservers: expectedNameServers,
				Prefixes:    []string{"192.168.1.0/24"},
			},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, networkID): &iaas.Network{
			Name:   "test-network",
			Id:     networkID,
			Status: wait.CreateSuccess,

			Ipv4: utils.Ptr(iaas.NetworkIPv4{
				Nameservers: expectedNameServers,
				Prefixes:    []string{"192.168.1.0/24"},
				Gateway:     *iaas.NewNullableString(utils.Ptr("192.168.1.1")),
			}),
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = sClient

	// Call the CreateNetwork method
	resp, err := cpi.CreateNetwork(req)
	if err != nil {
		t.Errorf("CreateNetwork() unexpected error = %v", err)
		return
	}

	// Check result
	result, ok := resp.Result.([]any)
	if !ok {
		t.Errorf("Expected result to be slice of any, got %T", resp.Result)
		return
	}

	if result[0].(string) != networkID {
		t.Errorf("Expected the first element of result to be a network id, got %v", result[0])
	}
	responseNameServers := result[2].(map[string]any)["dns"].([]string)
	for index, nameserver := range responseNameServers {
		if nameserver != expectedNameServers[index] {
			t.Errorf("expected nameservers in the response to match nameservers in the request")
		}
	}
}

func TestCreateNetwork_APIError(t *testing.T) {
	tests := []struct {
		name           string
		errorResponse  string
		statusCode     int
		wantRetryable  bool
		errMsgContains string
	}{
		{
			name:           "Forbidden",
			errorResponse:  "Forbidden",
			statusCode:     http.StatusForbidden,
			wantRetryable:  false,
			errMsgContains: "Forbidden",
		},
		{
			name:           "InsufficientResources",
			errorResponse:  "insufficient resources",
			statusCode:     http.StatusInsufficientStorage,
			wantRetryable:  false,
			errMsgContains: "insufficient resources",
		},
		{
			name:           "DuplicateNetworkName",
			errorResponse:  "already exists",
			statusCode:     http.StatusConflict,
			wantRetryable:  false,
			errMsgContains: "already exists",
		},
		{
			name:           "Timeout",
			errorResponse:  "timeout waiting for operation",
			statusCode:     http.StatusRequestTimeout,
			wantRetryable:  true,
			errMsgContains: "timeout",
		},
		{
			name:           "GenericError",
			errorResponse:  "internal server error",
			statusCode:     http.StatusInternalServerError,
			wantRetryable:  true,
			errMsgContains: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.CreateNetwork, t)

			networkID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
			args := []lib.CreateNetworkArgs{{
				Type: "dynamic",
				Properties: lib.NetProperties{
					NetID: networkID,
				},
				Range:       "10.0.0.0/24",
				Gateway:     "10.0.0.1",
				NetmaskBits: 24,
			}}

			argBytes, _ := json.Marshal(args)
			req.Arguments = argBytes

			// Set ultra-minimal timeout and no retries for faster tests
			cpi.Config.Timeout = 1 // Small timeout for quick tests
			cpi.Config.RetryCount = 0

			// Create a mock server that returns appropriate error responses based on the test case
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Log the request for debugging
				cpi.Log.Debugf("Mock server received request: %s %s", r.Method, r.URL.Path)

				// For create network requests, simulate the error condition
				if strings.Contains(r.URL.Path, "/networks") && r.Method == http.MethodPost {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					_, _ = fmt.Fprintf(w, `{"error": {"message": "%s"}}`, tt.errorResponse)
					return
				}

				// Default response for unhandled requests
				http.Error(w, "Teapot", http.StatusTeapot)
			}))
			defer mockServer.Close()
			sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
			if err != nil {
				t.Error(err.Error())
			}
			cpi.SClient = sClient

			resp, err := cpi.CreateNetwork(req)

			// This should result in an error
			if err == nil {
				t.Errorf("CreateNetwork() expected error, got nil")
			}

			// Verify error in response
			if resp.Error == nil || resp.Error.Message == "" {
				t.Errorf("Expected error in response, got none")
			}

			// Check if the error message contains expected text
			if !strings.Contains(resp.Error.Message, tt.errMsgContains) {
				t.Errorf("Error message '%s' does not contain expected text '%s'",
					resp.Error.Message, tt.errMsgContains)
			}

			// Verify retryable flag matches what we set
			if resp.Error.OkToRetry != tt.wantRetryable {
				t.Errorf("Error OkToRetry = %v, want %v",
					resp.Error.OkToRetry, tt.wantRetryable)
			}

			// Result should be false on error
			if resp.Result != false {
				t.Errorf("Expected Result to be false on error, got %v", resp.Result)
			}
		})
	}
}

func TestCreateNetwork_WaitError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateNetwork, t)

	networkID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
	args := []lib.CreateNetworkArgs{{
		Type: "dynamic",
		Properties: lib.NetProperties{
			NetID: networkID,
		},
		Range:       "10.0.0.0/24",
		Gateway:     "10.0.0.1",
		NetmaskBits: 24,
	}}

	argBytes, _ := json.Marshal(args)
	req.Arguments = argBytes

	// Set a shorter timeout for testing
	cpi.Config.Timeout = 4
	cpi.Config.RetryCount = 0

	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks", cpi.Config.ProjectID, cpi.Config.RegionID): &iaas.Network{
			Name:   "test-network",
			Id:     networkID,
			Status: wait.CreateSuccess,
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/networks/%s", cpi.Config.ProjectID, cpi.Config.RegionID, networkID): AfterRetryResponse{
			FirstRespond: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				ErrorMessage: `Internal Server Error`,
				Body:         []byte(`Internal Server Error`),
				Model:        iaas.Network{},
			},
			FinallyRespondWith: &iaas.Network{
				Name:   "test-network",
				Id:     networkID,
				Status: wait.CreateSuccess,
				Ipv4: utils.Ptr(iaas.NetworkIPv4{
					Nameservers: []string{},
					Prefixes:    []string{"192.168.1.0/24"},
					Gateway:     *iaas.NewNullableString(utils.Ptr("192.168.1.1")),
				}),
			},
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		t.Error(err.Error())
	}
	cpi.SClient = sClient

	// Call the CreateNetwork method
	resp, err := cpi.CreateNetwork(req)
	if err != nil {
		t.Errorf("Expected error to be nil, got %s", err.Error())
	}

	resultPayload, ok := resp.Result.([]any)
	if !ok {
		t.Errorf("expected result to be a slice of any")
	}
	resultNetworkID, ok := resultPayload[0].(string)
	if !ok {
		t.Errorf("expected the first element of result to be a stringt")
	}

	if resultNetworkID != networkID {
		t.Fatalf("expected result network id to match the testsetup network id")
	}

	// Verify error in response
	if resp.Error != nil {
		t.Errorf("Expected no error in response, got %s", err.Error())
	}
}

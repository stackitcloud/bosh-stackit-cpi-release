package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

// Test for GetUserData function
func TestCreateVMGetUserData(t *testing.T) {
	cpi, _ := SetupTestCPIAndRequest(lib.CreateVM, t)

	testAgentConfig := `{"agent_id":"test-agent-id","networks":{"default":{"ip":"10.0.0.10"}}}`
	userData := cpi.GetUserData(testAgentConfig)

	// Check that it's valid base64
	if userData == "" {
		t.Errorf("GetUserData() returned empty string")
	}

	// Validate that the user data is correctly encoded
	expected := "eyJhZ2VudF9pZCI6InRlc3QtYWdlbnQtaWQiLCJuZXR3b3JrcyI6eyJkZWZhdWx0Ijp7ImlwIjoiMTAuMC4wLjEwIn19fQ=="
	if userData != expected {
		t.Errorf("GetUserData() = %s, want %s", userData, expected)
	}
}

// TODO add test for AttemptToCleanupOrphaned nics

func TestCreateVM_CleaningNICSOnFailedAttempt(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateVM, t)

	argBytes, err := json.Marshal(CreateVMTestArgs)
	if err != nil {
		t.Errorf("it seems the default test args are invalid")
	}
	req.Arguments = argBytes

	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/regions/%s/availability-zones", cpi.Config.RegionID): iaas.AvailabilityZoneListResponse{
			Items: []string{"a", "b", "test-zone"},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/machine-types", cpi.Config.ProjectID, cpi.Config.RegionID):                          TestVMTypePayload,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID):       TestNicPayload1,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestSecondNetworkID): TestNicPayload2,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/public-ips", cpi.Config.ProjectID, cpi.Config.RegionID):                             TestPublicIPListPayload,
		fmt.Sprintf("PATCH_/v2/projects/%s/regions/%s/public-ips/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestPublicIPID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusUnauthorized,
			ErrorMessage: `Unauthorized`,
			Body:         []byte{},
			Model:        nil,
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s/nics/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID, TestNICID1): FakeResponse{
			StatusCode: http.StatusNoContent,
			Body:       []byte{},
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s/nics/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestSecondNetworkID, TestNICID2): FakeResponse{
			StatusCode: http.StatusNoContent,
			Body:       []byte{},
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	resp, err := cpi.CreateVM(req)

	// Should return an error
	if err == nil {
		t.Errorf("CreateVM() error = nil, want error")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "failed to create nics") {
		t.Errorf("Error = %v, want to contain 'failed to create nics'", err)
	}

	// Check that the error was set in the response
	if resp.Error.Message == "" {
		t.Errorf("Response error message is empty")
	}
}

// Test for CreateVM success case
func TestCreateVM_Success(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateVM, t)
	// Mock API client with success responses
	argBytes, err := json.Marshal(CreateVMTestArgs)
	if err != nil {
		t.Errorf("it seems the default test args are invalid")
	}
	req.Arguments = argBytes

	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/regions/%s/availability-zones", cpi.Config.RegionID): iaas.AvailabilityZoneListResponse{
			Items: []string{"a", "b", "test-zone"},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/machine-types", cpi.Config.ProjectID, cpi.Config.RegionID):                          TestVMTypePayload,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID):       TestNicPayload1,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestSecondNetworkID): TestNicPayload2,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/public-ips", cpi.Config.ProjectID, cpi.Config.RegionID):                             TestPublicIPListPayload,

		fmt.Sprintf("PATCH_/v2/projects/%s/regions/%s/public-ips/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestPublicIPID):       TestPublicIPPayload,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/servers/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestServerPayload.GetId()): TestServerPayload,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/servers", cpi.Config.ProjectID, cpi.Config.RegionID):                              TestServerPayload,
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()
	resp, err := cpi.CreateVM(req)
	if err != nil {
		t.Fatalf("CreateVM() error = %v", err)
	}

	// Check that the result is set correctly
	result, ok := resp.Result.([]any)
	if !ok {
		t.Fatalf("Result type is not []any, got %T", resp.Result)
	}

	serverID, ok := result[0].(string)
	if !ok {
		t.Fatalf("First result element is not a string, got %T", result[0])
	}

	if serverID != "12345678-1234-1234-1234-123456789012" {
		t.Errorf("Server ID = %s, want 12345678-1234-1234-1234-123456789012", serverID)
	}

	// Check that networks are in the second position
	if len(result) < 2 {
		t.Fatalf("Result array length is < 2")
	}

	networks, ok := result[1].(lib.Networks)
	if !ok {
		t.Fatalf("Second result element is not a map[string]Network, got %T", result[1])
	}

	// Check that the network info is present
	if len(networks) == 0 {
		t.Errorf("Networks map is empty")
	}
}

// Test for CreateVM with network configuration failures
func TestCreateVM_NetworkConfigurationFailure(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateVM, t)
	argBytes, err := json.Marshal(CreateVMTestArgs)
	if err != nil {
		t.Errorf("it seems the default test args are invalid")
	}
	req.Arguments = argBytes

	vmTypePayload := &iaas.MachineTypeListResponse{
		Items: []iaas.MachineType{
			{
				Disk:  int64(20),
				Name:  "valid",
				Ram:   int64(20),
				Vcpus: int64(1),
			},
		},
	}
	// Mock API client with responses
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/regions/%s/availability-zones", cpi.Config.RegionID): iaas.AvailabilityZoneListResponse{
			Items: []string{"a", "b", "test-zone"},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/machine-types", cpi.Config.ProjectID, cpi.Config.RegionID): vmTypePayload,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestSecondNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   403,
			ErrorMessage: `unauthorized`,
			Body:         []byte(`unauthorized`),
			Model:        iaas.NIC{},
		},
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   403,
			ErrorMessage: `unauthorized`,
			Body:         []byte(`unauthorized`),
			Model:        iaas.NIC{},
		},
	}

	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	resp, err := cpi.CreateVM(req)

	// Should return an error
	if err == nil {
		t.Errorf("CreateVM() error = nil, want error")
	}

	// Check the error message
	if !strings.Contains(err.Error(), "failed to create nics") {
		t.Errorf("Error = %v, want to contain 'failed to create nics'", err)
	}

	// Check that the error was set in the response
	if resp.Error.Message == "" {
		t.Errorf("Response error message is empty")
	}
}

// Test for CreateVM with server creation failure and a bosh retryable error
func TestCreateVM_ServerCreationFailure(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateVM, t)
	argBytes, err := json.Marshal(CreateVMTestArgs)
	if err != nil {
		t.Errorf("it seems the default test args are invalid")
	}
	req.Arguments = argBytes

	// Set up mock responses for the mock server
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/regions/%s/availability-zones", cpi.Config.RegionID): iaas.AvailabilityZoneListResponse{
			Items: []string{"a", "b", "test-zone"},
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/machine-types", cpi.Config.ProjectID, cpi.Config.RegionID):                          TestVMTypePayload,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID):       TestNicPayload1,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestSecondNetworkID): TestNicPayload2,
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/public-ips", cpi.Config.ProjectID, cpi.Config.RegionID):                             TestPublicIPListPayload,
		fmt.Sprintf("PATCH_/v2/projects/%s/regions/%s/public-ips/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestPublicIPID):        TestPublicIPPayload,
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/servers", cpi.Config.ProjectID, cpi.Config.RegionID): oapierror.GenericOpenAPIError{
			StatusCode:   429,
			ErrorMessage: `Too many requests`,
			Body:         []byte(`Too Many Requests`),
			Model:        nil,
		},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s/nics/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNicPayload1.GetNetworkId(), TestNicPayload1.GetId()): FakeResponse{StatusCode: http.StatusAccepted},
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/networks/%s/nics/%s", cpi.Config.ProjectID, cpi.Config.RegionID, TestNicPayload2.GetNetworkId(), TestNicPayload2.GetId()): FakeResponse{StatusCode: http.StatusAccepted},
	}

	// Set up the mock server with our responses
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the method under test
	resp, err := cpi.CreateVM(req)

	// Should return an error
	if err == nil {
		t.Errorf("CreateVM() error = nil, want error")
	}

	// Check that the error was set in the response
	if resp.Error.Type != "Bosh::Clouds::CloudError" {
		t.Errorf("Error type = %s, want Bosh::Clouds::CloudError", resp.Error.Type)
	}

	if !resp.Error.OkToRetry {
		t.Errorf("Error OkToRetry = false, want true")
	}
}

// Test for InvalidArgs for CreateVM
func TestCreateVM_InvalidArguments(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateVM, t)

	// Corrupt the arguments JSON to trigger unmarshalling error
	req.Arguments = json.RawMessage(`{"invalid": "json"]`)

	// Call the method under test
	_, err := cpi.CreateVM(req)

	// Should return an error
	if err == nil {
		t.Errorf("CreateVM() error = nil, want error")
	}

	// Check that the error was related to JSON unmarshalling
	if !strings.Contains(err.Error(), "failed unmarshalling Arguments JSON") {
		t.Errorf("Error = %v, want to contain 'failed unmarshalling Arguments JSON'", err)
	}
}

// TestCreateVM_InvalidAvailabilityZone tests that invalid availability zones are rejected
func TestCreateVM_InvalidAvailabilityZone(t *testing.T) {
	// Setup mock server for availability zones API
	cpi, req := SetupTestCPIAndRequest(lib.CreateVM, t)

	argsCopy := CreateVMTestArgs

	argsCopy[2] = map[string]any{ // Cloud properties
		"availability_zone": "invalid-zone",
		"instance_type":     "valid",
		"root_disk": map[string]any{
			"size": 30720, // 30GB in MB
		},
	}

	// Convert to JSON and set in request
	argsBytes, marshalErr := json.Marshal(argsCopy)
	if marshalErr != nil {
		t.Fatalf("Failed to marshal args: %v", marshalErr)
	}
	req.Arguments = argsBytes

	mockResponses := map[string]any{
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusBadRequest,
			ErrorMessage: "resource not found: availability zone",
		},
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/networks/%s/nics", cpi.Config.ProjectID, cpi.Config.RegionID, TestSecondNetworkID): oapierror.GenericOpenAPIError{
			StatusCode:   http.StatusBadRequest,
			ErrorMessage: "resource not found: availability zone",
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	// Call the method under test
	_, err := cpi.CreateVM(req)

	// Should return an error
	if err == nil {
		t.Errorf("CreateVM() error = nil, want error for invalid availability zone")
	}

	// Check that the error is related to invalid availability zone
	if !strings.Contains(err.Error(), "resource not found: availability zone") {
		t.Errorf("Error = '%v', 'resource not found: availability zone", err)
	}
}

// Test ResolveSecurityGroups function
func TestCreateVMResolveSecurityGroups(t *testing.T) {
	tests := []struct {
		name           string
		inputGroups    []string
		mockResponse   *iaas.SecurityGroupListResponse
		mockError      error
		expectedGroups []string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "Empty security groups list",
			inputGroups:    []string{},
			expectedGroups: []string{},
			expectedError:  false,
		},
		{
			name:        "All UUIDs - no lookup needed",
			inputGroups: []string{"123e4567-e89b-12d3-a456-426614174000", "223e4567-e89b-12d3-a456-426614174000"},
			mockResponse: &iaas.SecurityGroupListResponse{
				Items: []iaas.SecurityGroup{
					{
						Id:   utils.Ptr("123e4567-e89b-12d3-a456-426614174000"),
						Name: "sg-1",
					},
					{
						Id:   utils.Ptr("223e4567-e89b-12d3-a456-426614174000"),
						Name: "sg-2",
					},
				},
			},
			expectedGroups: []string{"123e4567-e89b-12d3-a456-426614174000", "223e4567-e89b-12d3-a456-426614174000"},
			expectedError:  false,
		},
		{
			name:        "All names - need lookup",
			inputGroups: []string{"web-sg", "db-sg"},
			mockResponse: &iaas.SecurityGroupListResponse{
				Items: []iaas.SecurityGroup{
					{
						Id:   utils.Ptr("123e4567-e89b-12d3-a456-426614174000"),
						Name: "web-sg",
					},
					{
						Id:   utils.Ptr("223e4567-e89b-12d3-a456-426614174000"),
						Name: "db-sg",
					},
					{
						Id:   utils.Ptr("333e4567-e89b-12d3-a456-426614174000"),
						Name: "other-sg",
					},
				},
			},
			expectedGroups: []string{"123e4567-e89b-12d3-a456-426614174000", "223e4567-e89b-12d3-a456-426614174000"},
			expectedError:  false,
		},
		{
			name:        "Mixed UUIDs and names",
			inputGroups: []string{"123e4567-e89b-12d3-a456-426614174000", "db-sg"},
			mockResponse: &iaas.SecurityGroupListResponse{
				Items: []iaas.SecurityGroup{
					{
						Id:   utils.Ptr("123e4567-e89b-12d3-a456-426614174000"),
						Name: "web-sg",
					},
					{
						Id:   utils.Ptr("223e4567-e89b-12d3-a456-426614174000"),
						Name: "db-sg",
					},
				},
			},
			expectedGroups: []string{"123e4567-e89b-12d3-a456-426614174000", "223e4567-e89b-12d3-a456-426614174000"},
			expectedError:  false,
		},
		{
			name:        "Name not found",
			inputGroups: []string{"non-existent-sg"},
			mockResponse: &iaas.SecurityGroupListResponse{
				Items: []iaas.SecurityGroup{
					{
						Id:   utils.Ptr("123e4567-e89b-12d3-a456-426614174000"),
						Name: "web-sg",
					},
				},
			},
			expectedError: true,
			errorContains: "requested: [non-existent-sg], missing: [non-existent-sg]",
		},
		{
			name:        "UUID not found",
			inputGroups: []string{"999e4567-e89b-12d3-a456-426614174000"},
			mockResponse: &iaas.SecurityGroupListResponse{
				Items: []iaas.SecurityGroup{
					{
						Id:   utils.Ptr("123e4567-e89b-12d3-a456-426614174000"),
						Name: "web-sg",
					},
				},
			},
			expectedError: true,
			errorContains: "requested: [999e4567-e89b-12d3-a456-426614174000], missing: [999e4567-e89b-12d3-a456-426614174000]",
		},
		{
			name:          "API error",
			inputGroups:   []string{"web-sg"},
			mockError:     fmt.Errorf("API error"),
			expectedError: true,
			errorContains: "failed to list security groups",
		},
		{
			name:        "Empty response from API",
			inputGroups: []string{"web-sg"},
			mockResponse: &iaas.SecurityGroupListResponse{
				Items: []iaas.SecurityGroup{},
			},
			expectedError: true,
			errorContains: "not enough security groups found. 0 exist in project, but 1 were requested",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that responds with our mock data
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tt.mockError != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				_ = json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Create a test CPI instance
			cpi, _ := SetupTestCPIAndRequest(lib.CreateVM, t)

			sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{
				config.WithEndpoint(server.URL),
				config.WithoutAuthentication(),
			})
			if err != nil {
				t.Fatalf("failed setting up test with: %s", err)
			}
			cpi.SClient = sClient
			// Call the function
			result, err := cpi.ResolveSecurityGroups(tt.inputGroups)

			// Check error expectations
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorContains)
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check result if no error expected
			if !tt.expectedError && err == nil {
				if len(result) != len(tt.expectedGroups) {
					t.Errorf("Result length = %d, want %d", len(result), len(tt.expectedGroups))
				}
				slices.Sort(result)
				slices.Sort(tt.expectedGroups)
				for i := range len(result) {
					if result[i] != tt.expectedGroups[i] {
						t.Errorf("the resolved sec groups did not match the expected value. \nresolved: %v\nexpected: %v", result, tt.expectedGroups)
					}
				}
			}
		})
	}
}

//// Test DefaultSecurityGroups functionality
//func TestDefaultSecurityGroups(t *testing.T) {
//	tests := []struct {
//		name                   string
//		defaultSecurityGroups  []string
//		networkSecurityGroups  []string
//		expectedSecurityGroups []string
//	}{
//		{
//			name:                   "No default security groups",
//			defaultSecurityGroups:  []string{},
//			networkSecurityGroups:  []string{"network-sg-1", "network-sg-2"},
//			expectedSecurityGroups: []string{"network-sg-1", "network-sg-2"},
//		},
//		{
//			name:                   "Only default security groups",
//			defaultSecurityGroups:  []string{"default-sg-1", "default-sg-2"},
//			networkSecurityGroups:  []string{},
//			expectedSecurityGroups: []string{"default-sg-1", "default-sg-2"},
//		},
//		{
//			name:                   "Default and network security groups",
//			defaultSecurityGroups:  []string{"default-sg-1", "default-sg-2"},
//			networkSecurityGroups:  []string{"network-sg-1", "network-sg-2"},
//			expectedSecurityGroups: []string{"network-sg-1", "network-sg-2", "default-sg-1", "default-sg-2"},
//		},
//		{
//			name:                   "Duplicate security groups are removed",
//			defaultSecurityGroups:  []string{"shared-sg", "default-sg"},
//			networkSecurityGroups:  []string{"network-sg", "shared-sg"},
//			expectedSecurityGroups: []string{"network-sg", "shared-sg", "default-sg"},
//		},
//		{
//			name:                   "UUIDs and names mixed",
//			defaultSecurityGroups:  []string{"123e4567-e89b-12d3-a456-426614174000", "default-sg-name"},
//			networkSecurityGroups:  []string{"network-sg-name", "223e4567-e89b-12d3-a456-426614174000"},
//			expectedSecurityGroups: []string{"network-sg-name", "223e4567-e89b-12d3-a456-426614174000", "123e4567-e89b-12d3-a456-426614174000", "default-sg-name"},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Create a mock network config
//			netConfig := Network{
//				ManualNetworks: []Network{
//					{
//						Properties: NetProperties{
//							NetID:          "test-network",
//							SecurityGroups: tt.networkSecurityGroups,
//						},
//					},
//				},
//			}
//
//			// Create a test CPI instance with default security groups
//			cpi := &CPI{
//				Config: Config{
//					DefaultSecurityGroups: tt.defaultSecurityGroups,
//				},
//			}
//
//			// Call GetPrimaryNetworkDetails to get security groups from networks
//			_, securityGroups := cpi.GetPrimaryNetworkDetails(netConfig)
//
//			// Add default security groups as done in CreateVM
//			if len(cpi.Config.DefaultSecurityGroups) > 0 {
//				securityGroups = append(securityGroups, cpi.Config.DefaultSecurityGroups...)
//				securityGroups = UniqueArray(securityGroups)
//			}
//
//			// Check that we got the expected security groups
//			if len(securityGroups) != len(tt.expectedSecurityGroups) {
//				t.Errorf("Expected %d security groups, got %d", len(tt.expectedSecurityGroups), len(securityGroups))
//			}
//
//			slices.Sort(securityGroups)
//			slices.Sort(tt.expectedSecurityGroups)
//			for i := range len(securityGroups) {
//				if securityGroups[i] != tt.expectedSecurityGroups[i] {
//					t.Errorf("the resolved sec groups did not match the expected value. \nresolved: %v\nexpected: %v", securityGroups, tt.expectedSecurityGroups)
//				}
//			}
//		})
//	}
//}

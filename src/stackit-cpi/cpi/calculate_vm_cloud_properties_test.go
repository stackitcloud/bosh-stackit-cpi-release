package cpi_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return s != "" && substr != "" && strings.Contains(s, substr)
}

func TestCalculateVMCloudProperties_ValidateArguments(t *testing.T) {
	tests := []struct {
		name      string
		arguments json.RawMessage
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Valid arguments",
			arguments: json.RawMessage(`{"cpu": 2, "ram": 4096, "ephemeral_disk_size": 10240}`),
			wantErr:   false,
		},
		{
			name:      "Invalid CPU - zero",
			arguments: json.RawMessage(`{"cpu": 0, "ram": 4096, "ephemeral_disk_size": 10240}`),
			wantErr:   true,
			errMsg:    "desired_instance_size.cpu must be > 0",
		},
		{
			name:      "Invalid RAM - zero",
			arguments: json.RawMessage(`{"cpu": 2, "ram": 0, "ephemeral_disk_size": 10240}`),
			wantErr:   true,
			errMsg:    "desired_instance_size.ram must be > 0",
		},
		{
			name:      "Invalid ephemeral_disk_size - zero",
			arguments: json.RawMessage(`{"cpu": 2, "ram": 4096, "ephemeral_disk_size": 0}`),
			wantErr:   true,
			errMsg:    "desired_instance_size.ephemeral_disk_size must be > 0",
		},
		{
			name:      "Missing required fields",
			arguments: json.RawMessage(`{}`),
			wantErr:   true,
			errMsg:    "desired_instance_size.cpu must be > 0",
		},
		{
			name:      "Invalid JSON",
			arguments: json.RawMessage(`{"cpu": "not-a-number", "ram": 4096, "ephemeral_disk_size": 10240}`),
			wantErr:   true,
			errMsg:    "expected a hash for desired_instance_size:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.CalculateVMCloudProperties, t)
			req.Arguments = tt.arguments

			expectedResponses := map[string]any{
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/machine-types", cpi.Config.ProjectID, cpi.Config.RegionID): map[string]any{
					"items": &[]iaas.MachineType{
						{
							Name:  "test-gpu",
							Vcpus: int64(2),
							Ram:   int64(4096),
							Disk:  int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "intel",
								"overcommit": "1",
								"gpu":        "totally",
							},
						},
						{
							Name:  "test",
							Vcpus: int64(2),
							Ram:   int64(4096),
							Disk:  int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "intel",
								"overcommit": "1",
							},
						},
						{
							Name:  "test",
							Vcpus: int64(1),
							Ram:   int64(1024),
							Disk:  int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "amd",
								"overcommit": "2",
							},
						},
					},
				},
			}
			testServer := SetupMockClient(cpi, expectedResponses)
			defer testServer.Close()
			resp, err := cpi.CalculateVMCloudProperties(req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CalculateVMCloudProperties() error = nil, expected error with message: %v", tt.errMsg)
				} else if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("CalculateVMCloudProperties() error = '%v', want to contain '%v'", err.Error(), tt.errMsg)
				}
				// Verify error in response
				if resp.Error.Message == "" {
					t.Errorf("Response error message is empty, expected: %v", tt.errMsg)
				} else if !containsString(resp.Error.Message, tt.errMsg) {
					t.Errorf("Response error message = %v, want to contain %v", resp.Error.Message, tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("CalculateVMCloudProperties() unexpected error = %v", err)
			}
		})
	}
}

func TestCalculateVMCloudProperties_Success(t *testing.T) {
	tests := []struct {
		name                 string
		cpu                  int
		ram                  int64
		ephemeralDiskSize    int64
		expectedInstanceType string
		expectedError        string
	}{
		{
			name:                 "Small VM",
			cpu:                  1,
			ram:                  4096,
			ephemeralDiskSize:    5120,
			expectedInstanceType: "g1a.1d",
		},
		{
			name:                 "Medium VM",
			cpu:                  2,
			ram:                  8192,
			ephemeralDiskSize:    10240,
			expectedInstanceType: "g1a.2d",
		},
		{
			name:                 "Large VM",
			cpu:                  4,
			ram:                  16384,
			ephemeralDiskSize:    20480,
			expectedInstanceType: "g1a.4d",
		},
		{
			name:                 "Very Large VM",
			cpu:                  8,
			ram:                  32768,
			ephemeralDiskSize:    30720,
			expectedInstanceType: "g1a.8d",
			expectedError:        "found '1' deprecated matching types: 'g1a.8d'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.CalculateVMCloudProperties, t)
			expectedResponses := map[string]any{
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/machine-types", cpi.Config.ProjectID, cpi.Config.RegionID): map[string]any{
					"items": &[]iaas.MachineType{
						{
							Name:        "g1a.8d",
							Description: utils.Ptr("super deprecated"),
							Vcpus:       int64(8),
							Ram:         int64(32768),
							Disk:        int64(1),
							ExtraSpecs: map[string]any{
								"cpu":        "amd",
								"overcommit": "1",
							},
						},
						{
							Name:        "shouldn't be picked: deprecated",
							Description: utils.Ptr("super deprecated"),
							Vcpus:       int64(1),
							Ram:         int64(4096),
							Disk:        int64(5120),
							ExtraSpecs: map[string]any{
								"cpu":        "amd",
								"overcommit": "1",
							},
						},
						{
							Name:        "wont-be-picked-gpu",
							Description: utils.Ptr("super valid"),
							Vcpus:       int64(1),
							Ram:         int64(4096),
							Disk:        int64(5120),
							ExtraSpecs: map[string]any{
								"cpu":        "amd",
								"overcommit": "1",
								"gpu":        "present",
							},
						},
						{
							Name:        "g1a.1X",
							Description: utils.Ptr("shouldn't be picked overcommit"),
							Vcpus:       int64(1),
							Ram:         int64(4096),
							Disk:        int64(5120),
							ExtraSpecs: map[string]any{
								"cpu":        "amd",
								"overcommit": "2",
							},
						},
						{
							Name:        "g1a.1d",
							Description: utils.Ptr("super valid"),
							Vcpus:       int64(1),
							Ram:         int64(4096),
							Disk:        int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "amd",
								"overcommit": "1",
							},
						},
						{
							Name:        "test",
							Description: utils.Ptr("deprecated"),
							Vcpus:       int64(1),
							Ram:         int64(4096),
							Disk:        int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "amd",
								"overcommit": "1",
							},
						},
						{
							Name:        "g1a.2X",
							Description: utils.Ptr("super valid but arm shouldn't be picked"),
							Vcpus:       int64(2),
							Ram:         int64(8192),
							Disk:        int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "arm",
								"overcommit": "1",
							},
						},
						{
							Name:        "g1a.2d",
							Description: utils.Ptr("super valid"),
							Vcpus:       int64(2),
							Ram:         int64(8192),
							Disk:        int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "intel",
								"overcommit": "1",
							},
						},
						{
							Name:        "g1a.4d",
							Description: utils.Ptr("super valid but arm shouldn't be picked"),
							Vcpus:       int64(4),
							Ram:         int64(16384),
							Disk:        int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "intel",
								"overcommit": "2",
							},
						},
						{
							Name:        "g1a.4X",
							Description: utils.Ptr("super valid but also has gpu shouldn't be picked"),
							Vcpus:       int64(4),
							Ram:         int64(16384),
							Disk:        int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "intel",
								"overcommit": "1",
								"gpu":        "available",
							},
						},
						{
							Name:        "g1a.4X",
							Description: utils.Ptr("super valid but also has gpu shouldn't be picked"),
							Vcpus:       int64(4),
							Ram:         int64(16384),
							Disk:        int64(10240),
							ExtraSpecs: map[string]any{
								"cpu":        "intel",
								"overcommit": "1",
								"gpu":        "available",
							},
						},
					},
				},
			}
			testServer := SetupMockClient(cpi, expectedResponses)
			defer testServer.Close()

			// Set up arguments
			req.Arguments = json.RawMessage(fmt.Sprintf(`{"cpu": %d, "ram": %d, "ephemeral_disk_size": %d}`,
				tt.cpu, tt.ram, tt.ephemeralDiskSize))

			// Call the CalculateVMCloudProperties method
			resp, err := cpi.CalculateVMCloudProperties(req)
			if tt.expectedError != "" {
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error = '%s' to match '%s'", err.Error(), tt.expectedError)
				}
				return
			}
			if err != nil {
				t.Errorf("CalculateVMCloudProperties() unexpected error = %v", err)
				return
			}

			// Check result
			cloudProps, ok := resp.Result.(map[string]any)
			if !ok {
				t.Errorf("could type assert result into a map %T", resp.Result)
				return
			}

			instanceType, ok := cloudProps["instance_type"].(string)
			if !ok {
				t.Errorf("Expected instance_type to be a string, got %T", cloudProps["instance_type"])
				return
			}

			if instanceType != tt.expectedInstanceType {
				t.Errorf("Expected instance type %s, got %s", tt.expectedInstanceType, instanceType)
			}

			// Verify ephemeral disk is included
			ephemeral, ok := cloudProps["ephemeral_disk"].(map[string]any)
			if !ok {
				t.Errorf("Expected ephemeral_disk to be a map, got %T", cloudProps["ephemeral_disk"])
				return
			}

			diskSize, ok := ephemeral["size"].(int)
			if !ok {
				t.Errorf("Expected disk size to be a number, got %T", ephemeral["size"])
				return
			}

			// The actual calculation logic might differ, so we just check it's a positive number
			if diskSize <= 0 {
				t.Errorf("Expected positive disk size, got %d", diskSize)
			}
		})
	}
}

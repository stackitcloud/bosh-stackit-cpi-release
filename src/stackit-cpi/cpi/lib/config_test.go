package lib_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	yaml "gopkg.in/yaml.v3"
)

func TestNewConfig(t *testing.T) {
	// Create a temporary test directory to store test config files
	testDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(testDir) }()

	// Helper function to create a test config file
	createConfigFile := func(filename string, configData map[string]any) string {
		// Convert to YAML structure expected by the code
		yamlData := map[string]any{
			"stackit": configData,
		}
		data, err := yaml.Marshal(yamlData)
		if err != nil {
			t.Fatalf("Failed to marshal test config: %v", err)
		}

		// Create the full path
		fullPath := filepath.Join(testDir, filename)
		err = os.WriteFile(fullPath, data, 0o644)
		if err != nil {
			t.Fatalf("Failed to write test config file: %v", err)
		}
		return fullPath
	}

	// Test cases
	tests := []struct {
		name        string
		configPath  string
		configData  map[string]any
		wantErr     bool
		errContains string
		validateFn  func(t *testing.T, config *lib.Config)
	}{
		{
			name:       "Valid complete config",
			configPath: "valid_config.yml",
			configData: map[string]any{
				"project_id":                     "test-project-id",
				"region":                         "eu02",
				"service_account_json":           "{\"type\":\"service_account\"}",
				"timeout":                        300,
				"lrp_timeout":                    3000,
				"log_file":                       "/var/log/cpi.log",
				"retry_count":                    5,
				"log_level":                      "debug",
				"default_root_volume_type":       "storage_premium_perf3",
				"default_persistent_volume_type": "storage_premium_perf5",
				"default_ssh_key_name":           "test-key",
			},
			wantErr: false,
			validateFn: func(t *testing.T, config *lib.Config) {
				if config.ProjectID != "test-project-id" {
					t.Errorf("Expected ProjectID to be 'test-project-id', got '%s'", config.ProjectID)
				}
				if config.RegionID != "eu02" {
					t.Errorf("Expected RegionID to be 'eu02', got '%s'", config.RegionID)
				}
				if config.ServiceAccountJSON != "{\"type\":\"service_account\"}" {
					t.Errorf("Expected ServiceAccountJSON to be set, got '%s'", config.ServiceAccountJSON)
				}
				if config.LRPTimeout != 3000 {
					t.Errorf("Expected LRPTimeout to be 3000, got %d", config.Timeout)
				}
				if config.Timeout != 300 {
					t.Errorf("Expected Timeout to be 300, got %d", config.Timeout)
				}
				if config.LogFile != "/var/log/cpi.log" {
					t.Errorf("Expected LogFile to be '/var/log/cpi.log', got '%s'", config.LogFile)
				}
				if config.RetryCount != 5 {
					t.Errorf("Expected RetryCount to be 5, got %d", config.RetryCount)
				}
				if config.LogLevel != "debug" {
					t.Errorf("Expected LogLevel to be 'debug', got '%s'", config.LogLevel)
				}
				if config.DefaultRootVolumeType != "storage_premium_perf3" {
					t.Errorf("Expected DefaultRootVolumeType to be 'storage_premium_perf3', got '%s'", config.DefaultRootVolumeType)
				}
				if config.DefaultPersistentVolumeType != "storage_premium_perf5" {
					t.Errorf("Expected DefaultPersistentVolumeType to be 'storage_premium_perf5', got '%s'", config.DefaultPersistentVolumeType)
				}
				if config.DefaultSSHKeyName != "test-key" {
					t.Errorf("Expected DefaultSSHKeyName to be test-key but was '%s'", config.DefaultSSHKeyName)
				}
			},
		},
		{
			name:       "Config with service account key path",
			configPath: "service_account_config.yml",
			configData: map[string]any{
				"project_id":           "test-project-id",
				"service_account_json": "{\"type\":\"service_account\"}",
			},
			wantErr: false,
			validateFn: func(t *testing.T, config *lib.Config) {
				if config.ServiceAccountJSON == "" {
					t.Errorf("Expected ServiceAccountJSON to be set, got '%s'", config.ServiceAccountJSON)
				}
				// Check defaults are set correctly
				if config.RegionID != "eu01" {
					t.Errorf("Expected default RegionID to be 'eu01', got '%s'", config.RegionID)
				}
				if config.LogLevel != "info" {
					t.Errorf("Expected default LogLevel to be 'info', got '%s'", config.LogLevel)
				}
				if config.LRPTimeout != 3600 {
					t.Errorf("Expected default Timeout to be 3600, got %d", config.Timeout)
				}
				if config.Timeout != 600 {
					t.Errorf("Expected default Timeout to be 600, got %d", config.Timeout)
				}
				if config.RetryCount != 3 {
					t.Errorf("Expected default RetryCount to be 3, got %d", config.RetryCount)
				}
				if config.DefaultRootVolumeType != "storage_premium_perf2" {
					t.Errorf("Expected default DefaultRootVolumeType to be 'storage_premium_perf2', got '%s'", config.DefaultRootVolumeType)
				}
				if config.DefaultPersistentVolumeType != "storage_premium_perf2" {
					t.Errorf("Expected default DefaultPersistentVolumeType to be 'storage_premium_perf2', got '%s'", config.DefaultPersistentVolumeType)
				}
			},
		},
		{
			name:       "Config with minimum required fields",
			configPath: "minimal_config.yml",
			configData: map[string]any{
				"project_id":           "test-project-id",
				"service_account_json": "{\"type\":\"service_account\"}",
			},
			wantErr: false,
			validateFn: func(t *testing.T, config *lib.Config) {
				// Just check the required fields, defaults tested in other test cases
				if config.ProjectID != "test-project-id" {
					t.Errorf("Expected ProjectID to be 'test-project-id', got '%s'", config.ProjectID)
				}
				if config.ServiceAccountJSON != "{\"type\":\"service_account\"}" {
					t.Errorf("Expected ServiceAccountJSON to be set, got '%s'", config.ServiceAccountJSON)
				}
			},
		},
		{
			name:       "Missing project ID error",
			configPath: "missing_project_id.yml",
			configData: map[string]any{
				"service_account_json": "{\"type\":\"service_account\"}",
			},
			wantErr:     true,
			errContains: "project_id is required",
		},
		{
			name:       "Missing auth credentials error",
			configPath: "missing_auth.yml",
			configData: map[string]any{
				"project_id": "test-project-id",
			},
			wantErr:     true,
			errContains: "service_account_json is required",
		},
		{
			name:        "File not found error",
			configPath:  "non_existent_file.yml", // Don't create this file
			wantErr:     true,
			errContains: "failed to read config file",
		},
		{
			name:       "Invalid YAML error",
			configPath: "invalid_yaml.yml",
			// Write invalid YAML directly to file in the test
			wantErr:     true,
			errContains: "failed to unmarshal config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string

			if tt.name == "Invalid YAML error" {
				// Special case for invalid YAML
				configPath = filepath.Join(testDir, tt.configPath)
				err = os.WriteFile(configPath, []byte("invalid: yaml: ["), 0o644)
				if err != nil {
					t.Fatalf("Failed to write invalid YAML file: %v", err)
				}
			} else if tt.name != "File not found error" {
				// Create config file for all tests except "File not found error"
				configPath = createConfigFile(tt.configPath, tt.configData)
			} else {
				// For "File not found error" test, use the path without creating the file
				configPath = filepath.Join(testDir, tt.configPath)
			}

			// Call NewConfig
			config, err := lib.NewConfig(configPath)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewConfig() expected error containing '%s', got nil", tt.errContains)
					return
				}
				if strings.Contains(err.Error(), tt.errContains) {
					return
				}
			}

			// If no error expected but got one
			if err != nil {
				t.Errorf("NewConfig() unexpected error = %v", err)
				return
			}

			// If a validation function is provided, run it
			if tt.validateFn != nil {
				tt.validateFn(t, config)
			}
		})
	}

	// Test with empty filename (should use default path)
	t.Run("Empty filename uses default path", func(t *testing.T) {
		// This should fail because the default path doesn't exist in test environment
		_, err := lib.NewConfig("")
		if err == nil {
			t.Errorf("NewConfig() with empty path should fail but didn't")
		}
		if !strings.Contains(err.Error(), "failed to read config file") {
			t.Errorf("NewConfig() with empty path error = %v, expected it to contain 'failed to read config file'", err)
		}
	})
}

func TestConfigDefaultSecurityGroups(t *testing.T) {
	// Create a temporary test directory to store test config files
	testDir, err := os.MkdirTemp("", "config-security-groups-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(testDir) }()

	tests := []struct {
		name           string
		configContent  string
		expectedGroups []string
	}{
		{
			name: "No default security groups",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
`,
			expectedGroups: nil,
		},
		{
			name: "Empty default security groups",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups: []
`,
			expectedGroups: []string{},
		},
		{
			name: "Single default security group",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups:
    - default-sg
`,
			expectedGroups: []string{"default-sg"},
		},
		{
			name: "Multiple default security groups",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups:
    - default-sg-1
    - default-sg-2
    - 123e4567-e89b-12d3-a456-426614174000
`,
			expectedGroups: []string{"default-sg-1", "default-sg-2", "123e4567-e89b-12d3-a456-426614174000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config file
			configPath := filepath.Join(testDir, "test-config.yml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0o644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Load config
			config, err := lib.NewConfig(configPath)
			if err != nil {
				t.Fatalf("NewConfig() error = %v", err)
			}

			// Check DefaultSecurityGroups
			if tt.expectedGroups == nil {
				if config.DefaultSecurityGroups != nil {
					t.Errorf("Expected DefaultSecurityGroups to be nil, got %v", config.DefaultSecurityGroups)
				}
			} else {
				if !reflect.DeepEqual(config.DefaultSecurityGroups, tt.expectedGroups) {
					t.Errorf("DefaultSecurityGroups = %v, want %v", config.DefaultSecurityGroups, tt.expectedGroups)
				}
			}
		})
	}
}

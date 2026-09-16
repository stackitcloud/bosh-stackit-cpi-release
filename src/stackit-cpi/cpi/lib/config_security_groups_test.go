package lib_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

func TestConfigDefaultSecurityGroupsHandling(t *testing.T) {
	// Create a temporary test directory to store test config files
	testDir, err := os.MkdirTemp("", "config-sg-handling-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(testDir) }()

	tests := []struct {
		name           string
		configContent  string
		expectedGroups []string
		wantErr        bool
		errContains    string
	}{
		{
			name: "Valid array of security groups",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups:
    - sg-1
    - sg-2
    - 123e4567-e89b-12d3-a456-426614174000
`,
			expectedGroups: []string{"sg-1", "sg-2", "123e4567-e89b-12d3-a456-426614174000"},
			wantErr:        false,
		},
		{
			name: "Empty array of security groups",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups: []
`,
			expectedGroups: []string{},
			wantErr:        false,
		},
		{
			name: "Null security groups",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups: null
`,
			expectedGroups: nil,
			wantErr:        false,
		},
		{
			name: "Single string security group should error",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups: sg-default
`,
			wantErr:     true,
			errContains: "cannot unmarshal",
		},
		{
			name: "Empty string security group should error",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups: ""
`,
			wantErr:     true,
			errContains: "cannot unmarshal",
		},
		{
			name: "Number as security group (should error)",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups: 123
`,
			wantErr:     true,
			errContains: "cannot unmarshal",
		},
		{
			name: "Boolean as security group (should error)",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups: true
`,
			wantErr:     true,
			errContains: "cannot unmarshal",
		},
		{
			name: "Object as security group (should error)",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups:
    name: sg-1
    id: 123
`,
			wantErr:     true,
			errContains: "cannot unmarshal",
		},
		{
			name: "Mixed array types",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups:
    - sg-1
    - 123
    - true
`,
			expectedGroups: []string{"sg-1", "123", "true"},
			wantErr:        false,
		},
		{
			name: "Array with empty strings (empty strings now kept)",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups:
    - sg-1
    - ""
    - sg-2
    - ""
`,
			wantErr:     true,
			errContains: "security group at index 1 is empty",
		},
		{
			name: "Security group with spaces (no automatic trimming)",
			configContent: `
stackit:
  project_id: test-project
  region: eu01
  service_account_json: '{"type":"service_account"}'
  default_security_groups:
    - "  sg-1  "
    - " sg-2"
    - "sg-3 "
`,
			expectedGroups: []string{"  sg-1  ", " sg-2", "sg-3 "},
			wantErr:        false,
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

			// Attempt to load config
			config, err := lib.NewConfig(configPath)

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errContains)
					if config != nil {
						t.Logf("Got config with DefaultSecurityGroups: %v", config.DefaultSecurityGroups)
					}
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, want it to contain '%s'", err, tt.errContains)
				}
				return
			}

			// If no error expected but got one
			if err != nil {
				t.Errorf("Unexpected error = %v", err)
				return
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

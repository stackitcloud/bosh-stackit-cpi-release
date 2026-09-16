package lib_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cloudfoundry/bosh-agent/v2/settings"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

func TestConfigCredRedaction(t *testing.T) {
	config := lib.Config{
		ServiceAccountJSON: `test-value`,
	}

	s := config.String()

	if strings.Contains(s, "test-value") {
		t.Errorf("the cred was not redacted")
	}
	if !strings.Contains(s, "redacted") {
		t.Errorf("the cred should have been replaced with <redacted>")
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name           string
		input          error // oapierror.GenericOpenAPIError
		expectedReturn bool
	}{
		{
			name:           "generic non openapi error",
			input:          errors.New("some generic non retryable error"),
			expectedReturn: false,
		},
		{
			name: "generic openapi eror with status code 500",
			input: oapierror.GenericOpenAPIError{
				StatusCode:   500,
				ErrorMessage: "Internal server error",
				Body:         []byte(``),
				Model:        nil,
			},
			expectedReturn: true,
		},
		{
			name: "generic openapi erro with status code 404",
			input: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusNotFound,
				ErrorMessage: "area not found",
				Body:         []byte(``),
				Model:        nil,
			},
			expectedReturn: false,
		},
		{
			name: "geeneric openapi error with status code 400",
			input: oapierror.GenericOpenAPIError{
				StatusCode:   400,
				ErrorMessage: "The affinity group policy is not supported",
				Body:         []byte(``),
				Model:        nil,
			},
			expectedReturn: false,
		},
		{
			name: "geeneric openapi error with status code 401",
			input: oapierror.GenericOpenAPIError{
				StatusCode:   401,
				ErrorMessage: "Forbidden",
				Body:         []byte(``),
				Model:        nil,
			},
			expectedReturn: false,
		},
		{
			name: "geeneric openapi error with status code 403",
			input: oapierror.GenericOpenAPIError{
				StatusCode:   403,
				ErrorMessage: "Forbidden",
				Body:         []byte(``),
				Model:        nil,
			},
			expectedReturn: false,
		},
		{
			name: "generic openapi error with status code 429",
			input: oapierror.GenericOpenAPIError{
				StatusCode:   429,
				ErrorMessage: "Too Many Requests",
				Body:         []byte(``),
				Model:        nil,
			},
			expectedReturn: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret := lib.IsRetryableOpenAPIError(tt.input)
			if tt.expectedReturn != ret {
				t.Errorf("expected %v to be retryable==%v but was %v", tt.input, tt.expectedReturn, ret)
			}
		})
	}
}

func TestUtilsUniqueArray(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "Empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "No duplicates",
			input: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "With duplicates",
			input: []string{"a", "b", "a", "c", "b", "d"},
			want:  []string{"a", "b", "c", "d"},
		},
		{
			name:  "All duplicates",
			input: []string{"a", "a", "a"},
			want:  []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lib.UniqueArray(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UniqueArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name string
		uuid string
		want bool
	}{
		{
			name: "Valid UUID",
			uuid: "123e4567-e89b-12d3-a456-426614174000",
			want: true,
		},
		{
			name: "Valid UUID with uppercase",
			uuid: "123E4567-E89B-12D3-A456-426614174000",
			want: true,
		},
		{
			name: "Invalid UUID - too short",
			uuid: "123e4567-e89b-12d3-a456",
			want: false,
		},
		{
			name: "Invalid UUID - too long",
			uuid: "123e4567-e89b-12d3-a456-426614174000-extra",
			want: false,
		},
		{
			name: "Invalid UUID - missing hyphens",
			uuid: "123e4567e89b12d3a456426614174000",
			want: false,
		},
		{
			name: "Invalid UUID - wrong hyphen positions",
			uuid: "123e456-7e89b-12d3-a456-426614174000",
			want: false,
		},
		{
			name: "Invalid UUID - non-hex characters",
			uuid: "123e4567-e89b-12d3-a456-42661417400g",
			want: false,
		},
		{
			name: "Empty string",
			uuid: "",
			want: false,
		},
		{
			name: "Not a UUID at all",
			uuid: "not-a-uuid",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lib.IsValidUUID(tt.uuid)
			if got != tt.want {
				t.Errorf("IsValidUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAttachDisk_ValidateArguments tests the argument validation logic without making any API calls
func TestAttachDisk_ValidateArguments(t *testing.T) {
	tests := []struct {
		name      string
		arguments json.RawMessage
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Valid arguments",
			arguments: json.RawMessage(`["valid-server-id", "valid-disk-id"]`),
			wantErr:   false,
		},
		{
			name:      "Missing server ID",
			arguments: json.RawMessage(`["", "valid-disk-id"]`),
			wantErr:   true,
			errMsg:    "vm_cid must be provided",
		},
		{
			name:      "Missing disk ID",
			arguments: json.RawMessage(`["valid-server-id", ""]`),
			wantErr:   true,
			errMsg:    "disk_cid must be provided",
		},
		{
			name:      "Both IDs missing",
			arguments: json.RawMessage(`["", ""]`),
			wantErr:   true,
			errMsg:    "vm_cid must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args lib.AttachDiskArgs
			err := json.Unmarshal(tt.arguments, &args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateAttachDiskArgs() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCreateVMArgs() error = %v, want error containing %q", err, tt.errMsg)
				}

				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateAttachDiskArgs() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("validateAttachDiskArgs() unexpected error = %v", err)
			}
		})
	}
}

func TestCreateDisk_ValidateArguments(t *testing.T) {
	tests := []struct {
		name      string
		arguments json.RawMessage
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Valid arguments",
			arguments: json.RawMessage(`[1024, {}, "valid-server-id"]`),
			wantErr:   false,
		},
		{
			name:      "Missing server ID",
			arguments: json.RawMessage(`[1024, {}, ""]`),
			wantErr:   true,
			errMsg:    "cpi.CreateDisk(): VM CID cannot be empty",
		},
		{
			name:      "Size too small",
			arguments: json.RawMessage(`[10, {}, "valid-server-id"]`),
			wantErr:   true,
			errMsg:    fmt.Sprintf("cpi.CreateDisk(): Disk size must be at least %d MiB", lib.MinDiskSizeMB),
		},
		{
			name:      "Size too large",
			arguments: json.RawMessage(fmt.Sprintf(`[%v, {}, "valid-server-id"]`, 20*1024*1024)),
			wantErr:   true,
			errMsg:    lib.ErrDiskSizeTooLarge.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only test the ValidateCreateDiskArgs function
			var args lib.CreateDiskArgs
			err := json.Unmarshal(tt.arguments, &args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCreateDiskArgs() error = nil, expected error with message: %v", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("ValidateCreateDiskArgs() error = %v, want %v", err.Error(), tt.errMsg)
				}
				// No need to verify response error as we're only testing validation
			} else if err != nil {
				t.Errorf("ValidateCreateDiskArgs() unexpected error = %v", err)
			}
		})
	}
}

func TestCreateStemcell_ValidateArguments(t *testing.T) {
	tests := []struct {
		name      string
		arguments []any
		errMsg    string
	}{
		{
			name:      "Missing arguments",
			arguments: []any{},
			errMsg:    "missing required arguments",
		},
		{
			name:      "Invalid first argument (not a string)",
			arguments: []any{123, map[string]any{}},
			errMsg:    "first argument must be a string",
		},
		{
			name:      "Invalid second argument (not a map)",
			arguments: []any{"path/to/image", "not-a-map"},
			errMsg:    "second argument must be a map of stemcell properties",
		},
		{
			name: "Missing required name property",
			arguments: []any{"path/to/image", map[string]any{
				"version":     "1.0",
				"disk_format": "raw",
			}},
			errMsg: "stemcell name is required",
		},
		{
			name: "Missing required version property",
			arguments: []any{"path/to/image", map[string]any{
				"name":        "stemcell-name",
				"disk_format": "raw",
			}},
			errMsg: "stemcell version is required",
		},
		{
			name: "Missing required disk_format property",
			arguments: []any{"path/to/image", map[string]any{
				"name":    "stemcell-name",
				"version": "1.0",
			}},
			errMsg: "disk format is required",
		},
		{
			name: "Unsupported disk format",
			arguments: []any{"path/to/image", map[string]any{
				"name":        "stemcell-name",
				"version":     "1.0",
				"disk_format": "vhd", // Unsupported format
			}},
			errMsg: "unsupported disk format: vhd",
		},
		{
			name: "Unsupported OS type",
			arguments: []any{"path/to/image", map[string]any{
				"name":        "stemcell-name",
				"version":     "1.0",
				"disk_format": "raw",
				"os_type":     "solaris", // Unsupported OS
			}},
			errMsg: "unsupported OS type: solaris",
		},
		{
			name:      "Invalid JSON in arguments array",
			arguments: []any{},
			errMsg:    "missing required arguments",
		},
		{
			name:      "Wrong number of arguments",
			arguments: []any{"path/to/image", map[string]any{}, "extra-arg"},
			errMsg:    "expected 2 arguments for create_stemcell, got 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args lib.CreateStemcellArgs

			argBytes, err := json.Marshal(tt.arguments)
			if err != nil {
				t.Errorf("test setup failed: %s", err)
				return
			}
			err = json.Unmarshal(argBytes, &args)

			if err == nil {
				t.Errorf("expected error but got nil: %s", tt.name)
				return
			}
			if tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain '%s' but got '%s'", tt.errMsg, err.Error())
					return
				}
			}
		})
	}
}

// Tests for ValidateCreateVMArgs function
func TestValidateCreateVMArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      *lib.CreateVMArgs
		wantErr   bool
		errSubstr string
	}{
		{
			name: "Valid arguments",
			args: &lib.CreateVMArgs{
				AgentID:    "test-agent-id",
				StemcellID: "test-stemcell-id",
				Properties: lib.VMProperties{
					AvailabilityZone: "test-zone",
					InstanceType:     "valid",
					RootDisk: lib.VMRootDisk{
						Size: 10,
					},
				},
				Networks: map[string]lib.Network{
					"default": {
						Network: settings.Network{
							Type: "manual",
							IP:   "10.0.0.10",
						},
						Properties: lib.NetProperties{
							NetID: "01234567-1234-1234-1234-123456789012",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing agent ID",
			args: &lib.CreateVMArgs{
				StemcellID: "test-stemcell-id",
				Properties: lib.VMProperties{
					AvailabilityZone: "test-zone",
					InstanceType:     "valid",
					RootDisk: lib.VMRootDisk{
						Size: 10,
					},
				},
				Networks: map[string]lib.Network{
					"default": {
						Network: settings.Network{
							Type: "manual",
							IP:   "10.0.0.10",
						},
						Properties: lib.NetProperties{
							NetID: "01234567-1234-1234-1234-123456789012",
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "agent_id is required",
		},
		{
			name: "Missing stemcell ID",
			args: &lib.CreateVMArgs{
				AgentID: "test-agent-id",
				Properties: lib.VMProperties{
					AvailabilityZone: "test-zone",
					InstanceType:     "valid",
					RootDisk: lib.VMRootDisk{
						Size: 10,
					},
				},
				Networks: map[string]lib.Network{
					"default": {
						Network: settings.Network{
							Type: "manual",
							IP:   "10.0.0.10",
						},
						Properties: lib.NetProperties{
							NetID: "01234567-1234-1234-1234-123456789012",
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "stemcell_cid is required",
		},
		{
			name: "Empty networks",
			args: &lib.CreateVMArgs{
				AgentID:    "test-agent-id",
				StemcellID: "test-stemcell-id",
				Properties: lib.VMProperties{
					AvailabilityZone: "test-zone",
					InstanceType:     "valid",
					RootDisk: lib.VMRootDisk{
						Size: 10,
					},
				},
				Networks: map[string]lib.Network{},
			},
			wantErr:   true,
			errSubstr: "at least one network must be specified",
		},
		{
			name: "Missing availability zone",
			args: &lib.CreateVMArgs{
				AgentID:    "test-agent-id",
				StemcellID: "test-stemcell-id",
				Properties: lib.VMProperties{
					InstanceType: "valid",
					RootDisk: lib.VMRootDisk{
						Size: 10,
					},
				},
				Networks: map[string]lib.Network{
					"default": {
						Network: settings.Network{
							Type: "manual",
							IP:   "10.0.0.10",
						},
						Properties: lib.NetProperties{
							NetID: "01234567-1234-1234-1234-123456789012",
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "availability_zone must be specified",
		},
		{
			name: "Missing instance type",
			args: &lib.CreateVMArgs{
				AgentID:    "test-agent-id",
				StemcellID: "test-stemcell-id",
				Properties: lib.VMProperties{
					AvailabilityZone: "test-zone",
					RootDisk: lib.VMRootDisk{
						Size: 10,
					},
				},
				Networks: map[string]lib.Network{
					"default": {
						Network: settings.Network{
							Type: "manual",
							IP:   "10.0.0.10",
						},
						Properties: lib.NetProperties{
							NetID: "01234567-1234-1234-1234-123456789012",
						},
					},
				},
			},
			wantErr:   true,
			errSubstr: "instance_type must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate CreateVMArgs error = nil, want error containing %q", tt.errSubstr)
				} else if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("Validate CreateVMArgs error = %v, want error containing %q", err, tt.errSubstr)
				}
			} else if err != nil {
				t.Errorf("Validate CreateVMArgs error = %v, want nil", err)
			}
		})
	}
}

func TestValidate_DeleteDiskArguments(t *testing.T) {
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
			errMsg:    "expected 1 argument for delete_disk, got 0",
		},
		{
			name:      "Empty disk ID",
			arguments: json.RawMessage(`[""]`),
			wantErr:   true,
			errMsg:    "disk ID cannot be empty",
		},
		{
			name:      "Valid disk ID",
			arguments: json.RawMessage(`["12345678-1234-1234-1234-123456789012"]`), // Valid UUID format
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := lib.DeleteDiskArgs{}
			err := json.Unmarshal(tt.arguments, &args)

			if err == nil {
				err = args.Validate()
			}
			// Check error conditions
			if tt.wantErr {
				if err == nil {
					t.Errorf("validate DeleteDiskArgs expected error with message: %v", tt.errMsg)
					return
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validate DeleteDiskArgs error = %v, want %v", err.Error(), tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validate DeleteDiskArgs error message = %v, want %v", err.Error(), tt.errMsg)
					return
				}
			} else if err != nil {
				t.Errorf("validate DeleteDiskArgs unexpected error = %v", err)
				return
			}
		})
	}
}

func TestMakeIaasCompatibleServerNameString(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Output string
	}{
		{
			Name:   "invalid char `*`",
			Input:  "an*input*string*",
			Output: "aninputstring",
		},
		{
			Name:   "invalid char `/`",
			Input:  "an/input/string*",
			Output: "an-input-string",
		},
		{
			Name:   "invalid char `@`",
			Input:  "an@input@string*",
			Output: "aninputstring",
		},
		{
			Name:   "an FQDN without path",
			Input:  "my.bosh.vm.com",
			Output: "my.bosh.vm.com",
		},
		{
			Name:   "an FQDN with path",
			Input:  "my.bosh.vm.com/docs",
			Output: "my.bosh.vm.com-docs",
		},
		{
			Name:   "a bosh human readable name",
			Input:  "router/9A80393C-1445-475A-9150-044A9B926078",
			Output: "router-9a80393c-1445-475a-9150-044a9b926078",
		},
		{
			Name:   "a human readable compilation vm name",
			Input:  "compilation-c7df0e16-8ab2-44ba-9b5e-98808f2cf012/d41766b0-dc03-4c29-a5f2-d3bd4636ef21",
			Output: "compilation-c7df0e16-8ab2-44ba-9b5e-98808f2cf012-d41766b0-dc03",
		},
		{
			Name:   "Valid DNS name - no changes needed",
			Input:  "example.com",
			Output: "example.com",
		},
		{
			Name:   "Valid single label - no changes needed",
			Input:  "hostname",
			Output: "hostname",
		},
		{
			Name:   "Uppercase to lowercase conversion",
			Input:  "Example.COM",
			Output: "example.com",
		},
		{
			Name:   "Spaces replaced with hyphens",
			Input:  "my server name",
			Output: "my-server-name",
		},
		{
			Name:   "Underscores replaced with hyphens",
			Input:  "my_server_name",
			Output: "my-server-name",
		},
		{
			Name:   "Slashes replaced with hyphens",
			Input:  "my/server/name",
			Output: "my-server-name",
		},
		{
			Name:   "Invalid characters removed",
			Input:  "my@server#name!",
			Output: "myservername",
		},
		{
			Name:   "Invalid characters removed and pretty log message printed",
			Input:  "my@server@name@",
			Output: "myservername",
		},
		{
			Name:   "Multiple consecutive hyphens collapsed",
			Input:  "my---server--name",
			Output: "my-server-name",
		},
		{
			Name:   "Multiple consecutive dots collapsed",
			Input:  "my..server...name",
			Output: "my.server.name",
		},
		{
			Name:   "Leading and trailing hyphens removed",
			Input:  "-hostname-",
			Output: "hostname",
		},
		{
			Name:   "Label exceeding `cpi.ServerNameMaxLength` characters truncated",
			Input:  strings.Repeat("a", 70),
			Output: strings.Repeat("a", lib.ServerNameMaxLength),
		},
		{
			Name:   "Empty input returns error",
			Input:  "",
			Output: "",
		},
		{
			Name:   "Complex BOSH VM name",
			Input:  "traefik/c50f32a0-65f4-40a2-9dde-bff12560c14d",
			Output: "traefik-c50f32a0-65f4-40a2-9dde-bff12560c14d",
		},
		{
			Name:   "Name with dots and hyphens",
			Input:  "web-server.example.com",
			Output: "web-server.example.com",
		},
		{
			Name:   "Name with trailing dots removed",
			Input:  "example.com.",
			Output: "example.com",
		},
		{
			Name:   "All special characters",
			Input:  "!@#$%^&*()",
			Output: "",
		},
		{
			Name:   "Mixed case with spaces and special chars",
			Input:  "My Test_Server/Instance#01",
			Output: "my-test-server-instance01",
		},
		{
			Name:   "Empty labels removed",
			Input:  "example..com",
			Output: "example.com",
		},
		{
			Name:   "Hyphens at label boundaries",
			Input:  "example-.com",
			Output: "example.com",
		},
		{
			Name:   "Numbers and hyphens",
			Input:  "test-123-server",
			Output: "test-123-server",
		},
		{
			Name:   "Starting with number",
			Input:  "123-test",
			Output: "123-test",
		},
		{
			Name:   "Only dots",
			Input:  "...",
			Output: "",
		},
		{
			Name:   "Unicode characters removed",
			Input:  "café-résumé",
			Output: "caf-rsum",
		},
		{
			Name:   "Backslashes replaced",
			Input:  "domain\\user",
			Output: "domain-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			got := lib.MakeIaasCompatibleServerNameString(tt.Input)

			// Check if error status matches expectation
			// Check the result
			if got != tt.Output {
				t.Errorf("MakeDNScompatibleName() = %v, want %v", got, tt.Output)
			}
		})
	}
}

func TestSetDiskMetadata_ValidateArguments(t *testing.T) {
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
			errMsg:    "expected 2 arguments for set_disk_metadata, got 0",
		},
		{
			name:      "Empty disk ID",
			arguments: json.RawMessage(`["", {"key": "value"}]`),
			wantErr:   true,
			errMsg:    "disk_cid must be provided",
		},
		{
			name:      "Missing metadata",
			arguments: json.RawMessage(`["12345678-1234-1234-1234-123456789012"]`),
			wantErr:   true,
			errMsg:    "expected 2 arguments for set_disk_metadata, got 1",
		},
		{
			name:      "Null metadata",
			arguments: json.RawMessage(`["12345678-1234-1234-1234-123456789012", null]`),
			wantErr:   true,
			errMsg:    "metadata must be provided",
		},
		{
			name:      "Empty metadata object",
			arguments: json.RawMessage(`["12345678-1234-1234-1234-123456789012", {}]`),
			wantErr:   false,
			errMsg:    "",
		},
		{
			name:      "Valid arguments",
			arguments: json.RawMessage(`["12345678-1234-1234-1234-123456789012", {"key": "value"}]`),
			wantErr:   false,
			errMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := lib.SetDiskMetadataArgs{}

			err := json.Unmarshal(tt.arguments, &args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SetDiskMetadata() error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("SetDiskMetadata() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("SetDiskMetadata() unexpected error = %v", err)
			}
		})
	}
}

func TestCreateNetwork_ValidateArguments(t *testing.T) {
	tests := []struct {
		name      string
		arguments json.RawMessage
		wantErr   bool
		errMsg    string
	}{
		{
			name: "Valid manual network",
			arguments: json.RawMessage(`[{
				"type": "manual",
				"cloud_properties": {
					"net_id": "test-net-id",
					"security_groups": ["default"]
				},
				"range": "192.168.1.0/24",
				"gateway": "192.168.1.1",
				"netmask_bits": 24
			}]`),
			wantErr: false,
		},
		{
			name: "Valid dhcp network",
			arguments: json.RawMessage(`[{
				"type": "dynamic",
				"cloud_properties": {
					"net_id": "test-net-id",
					"security_groups": ["default"]
				},
				"range": "192.168.1.0/24",
				"gateway": "192.168.1.1",
				"netmask_bits": 24
			}]`),
			wantErr: false,
		},
		{
			name: "Invalid network type",
			arguments: json.RawMessage(`[{
				"type": "invalid",
				"cloud_properties": {
					"net_id": "test-net-id",
					"security_groups": ["default"]
				},
				"range": "192.168.1.0/24",
				"gateway": "192.168.1.1",
				"netmask_bits": 24
			}]`),
			wantErr: true,
			errMsg:  `invalid network type 'invalid', expected 'manual' or 'dynamic'`,
		},
		{
			name: "Invalid netmask bits",
			arguments: json.RawMessage(`[{
				"type": "manual",
				"cloud_properties": {
					"net_id": "test-net-id",
					"security_groups": ["default"]
				},
				"range": "192.168.1.0/24",
				"gateway": "192.168.1.1",
				"netmask_bits": 33
			}]`),
			wantErr: true,
			errMsg:  "expected network `netmask_bits` key to be between 0 and 32, got 33",
		},
		{
			name: "Invalid netmask bits negative",
			arguments: json.RawMessage(`[{
				"type": "manual",
				"cloud_properties": {
					"net_id": "test-net-id",
					"security_groups": ["default"]
				},
				"range": "192.168.1.0/24",
				"gateway": "192.168.1.1",
				"netmask_bits": -1
			}]`),
			wantErr: true,
			errMsg:  "expected network `netmask_bits` key to be between 0 and 32, got -1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := lib.CreateNetworkArgs{}

			err := json.Unmarshal(tt.arguments, &args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateCreateNetworkArgs error = nil, expected error with message: %v", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validating CreateNetworkArgs error = '%v', want '%v'", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("validate CreateNetworkArgs unexpected error = %v", err)
			}
		})
	}
}

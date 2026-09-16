package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
)

// setupDeleteStemcellMockClient creates a mock HTTP server for delete_stemcell tests
func setupDeleteStemcellMockClient(cpi *CPI, mockPayloads map[string]any) *httptest.Server {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if payload, ok := mockPayloads[fmt.Sprintf("%s_%s", r.Method, r.URL.Path)]; ok {
			if data, ok := payload.(map[string]any); ok {
				w.WriteHeader(data["status"].(int))
				_ = json.NewEncoder(w).Encode(data["payload"])
				return
			}
			_ = json.NewEncoder(w).Encode(payload)
			return
		}
		// Default response for unhandled routes
		http.Error(w, "Not found", http.StatusNotFound)
	}))

	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		panic(err)
	}
	cpi.SClient = sClient
	return mockServer
}

// TestDeleteStemcellArgs_UnmarshalJSON tests the unmarshaling of DeleteStemcellArgs
func TestDeleteStemcellArgs_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid arguments",
			json:    `["12345678-1234-1234-1234-123456789012"]`,
			wantErr: false,
			errMsg:  "",
		},
		{
			name:    "Malformed JSON",
			json:    `{this is not valid json}`,
			wantErr: true,
			errMsg:  "invalid character 't' looking for beginning of object key string",
		},
		{
			name:    "Empty array",
			json:    `[]`,
			wantErr: true,
			errMsg:  "expected 1 argument for delete_stemcell, got 0",
		},
		{
			name:    "Too many arguments",
			json:    `["stemcell-id", "extra-arg"]`,
			wantErr: true,
			errMsg:  "expected 1 argument for delete_stemcell, got 2",
		},
		{
			name:    "Invalid stemcell_cid type",
			json:    `[123]`,
			wantErr: true,
			errMsg:  "failed to unmarshal stemcell_cid:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args lib.DeleteStemcellArgs
			err := json.Unmarshal([]byte(tt.json), &args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
		})
	}
}

// TestDeleteStemcell_ArgumentsErrorHandling tests error handling in DeleteStemcell method
// specifically for argument unmarshaling errors
func TestDeleteStemcell_ArgumentsErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		args           string
		expectedErrMsg string
	}{
		{
			name:           "Malformed JSON arguments",
			args:           `{this is not json}`,
			expectedErrMsg: "failed Unmarshalling Arguments JSON Array",
		},
		{
			name:           "Wrong number of arguments",
			args:           `[]`,
			expectedErrMsg: "expected 1 argument for delete_stemcell, got 0",
		},
		{
			name:           "Invalid stemcell_cid type",
			args:           `[123]`,
			expectedErrMsg: "failed to unmarshal stemcell_cid:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the standard test setup but immediately override the arguments
			cpi, req := SetupTestCPIAndRequest(lib.DeleteStemcell, t)

			// Override arguments with our test case
			req.Arguments = json.RawMessage(tt.args)

			resp, err := cpi.DeleteStemcell(req)
			if err == nil {
				t.Errorf("Expected error but got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErrMsg, err.Error())
			}

			// Also check that the error was properly set in the CPI response
			if !strings.Contains(resp.Error.Message, tt.expectedErrMsg) {
				t.Errorf("Expected response error message '%s', got '%s'", tt.expectedErrMsg, resp.Error.Message)
			}
		})
	}
}

// TestDeleteStemcell_Success tests the successful execution of DeleteStemcell
func TestDeleteStemcell_Success(t *testing.T) {
	imageID := "12345678-1234-1234-1234-123456789012"

	cpi, req := SetupTestCPIAndRequest(lib.DeleteStemcell, t)
	req.Arguments = []byte(fmt.Sprintf(`["%s"]`, imageID))

	mockPayload := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/images/%s", cpi.Config.ProjectID, cpi.Config.RegionID, imageID): map[string]any{"payload": "{}", "status": 204},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/images/%s", cpi.Config.ProjectID, cpi.Config.RegionID, imageID):    map[string]any{"payload": `{"error": "not found"}`, "status": 404},
	}
	testServer := setupDeleteStemcellMockClient(cpi, mockPayload)
	defer testServer.Close()

	resp, err := cpi.DeleteStemcell(req)
	if err != nil {
		t.Errorf("DeleteStemcell() unexpected error = %v", err)
	}

	// Check result is true
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected bool result, got %T", resp.Result)
	}
	if !result {
		t.Errorf("Expected result to be true, got %v", result)
	}
}

// TestDeleteStemcell_Error tests error handling during stemcell deletion
func TestDeleteStemcell_Error(t *testing.T) {
	imageID := "12345678-1234-1234-1234-123456789012"
	cpi, req := SetupTestCPIAndRequest(lib.DeleteStemcell, t)
	req.Arguments = []byte(fmt.Sprintf(`["%s"]`, imageID))

	mockPayloads := map[string]any{
		fmt.Sprintf("DELETE_/v2/projects/%s/regions/%s/images/%s", cpi.Config.ProjectID, cpi.Config.RegionID, imageID): map[string]any{"payload": "{}", "status": 204},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/images/%s", cpi.Config.ProjectID, cpi.Config.RegionID, imageID):    map[string]any{"payload": `{"error": "service unavailable"}`, "status": 503},
	}
	mockServer := setupDeleteStemcellMockClient(cpi, mockPayloads)
	defer mockServer.Close()

	resp, err := cpi.DeleteStemcell(req)
	if err == nil {
		t.Errorf("DeleteStemcell() expected error, got nil")
	}

	// Check error details in response
	if resp.Error.Type != "Bosh::Clouds::CloudError" {
		t.Errorf("Expected error type 'Bosh::Clouds::CloudError', got '%s'", resp.Error.Type)
	}

	if !resp.Error.OkToRetry {
		t.Errorf("Expected OkToRetry to be true, got false")
	}

	if !strings.Contains(resp.Error.Message, "failed to delete stemcell") {
		t.Errorf("Expected error message to contain 'failed to delete stemcell', got '%s'", resp.Error.Message)
	}
}

// TestDeleteStemcell_InvalidArgs tests error handling for invalid arguments
func TestDeleteStemcell_InvalidArgs(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.DeleteStemcell, t)
	req.Arguments = []byte(`[]`)

	resp, err := cpi.DeleteStemcell(req)
	if err == nil {
		t.Errorf("DeleteStemcell() expected error for empty stemcell ID, got nil")
	}

	// Check error details in response
	if !strings.Contains(resp.Error.Message, "expected 1 argument for delete_stemcell, got 0") {
		t.Errorf("Expected error message to contain 'expected 1 argument for delete_stemcell, got 0', got '%s'", resp.Error.Message)
	}
}

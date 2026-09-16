package cpi

import (
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/* Test the "info" method of the RPC
 *
 * ref: https://bosh.io/docs/cpi-api-v2-method/info/
 */
func TestInfo(t *testing.T) {
	c := &CPI{
		Config: lib.Config{
			ProjectID:  "test-project",
			RegionID:   "test-region",
			Timeout:    60,
			RetryCount: 3,
			LogLevel:   "INFO",
		},
	}
	req := lib.RPCRequest{
		Context: lib.RPCContext{
			DirectorUUID: "test-director-uuid",
			RequestID:    "test-request-id",
		},
	}

	// Directly call the Info method
	resp, err := c.Info(req)
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	// Verify the response contains expected values
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("Result is not a map: %v", resp.Result)
	}

	// Check API version
	apiVersion, ok := result["api_version"].(int)
	if !ok || apiVersion != 2 {
		t.Errorf("Expected api_version to be 2, got %v", result["api_version"])
	}

	// Check stemcell formats
	stemcellFormats, ok := result["stemcell_formats"].([]string)
	if !ok || len(stemcellFormats) == 0 {
		t.Errorf("Expected stemcell_formats to be a non-empty array, got %v", result["stemcell_formats"])
	}

	// Check config values
	config, ok := result["config"].(map[string]any)
	if !ok {
		t.Errorf("Expected config to be a map, got %v", result["config"])
	}

	// Verify a few config values
	if config["project_id"] != "test-project" {
		t.Errorf("Expected project_id to be 'test-project', got %v", config["project_id"])
	}
}

package cpi_test

import (
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
)

func TestUpdateDisk_MethodNotSupportedError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.UpdateDisk, t)

	// Call the UpdateDisk method
	resp, err := cpi.UpdateDisk(req)
	if err == nil {
		t.Errorf("UpdateDisk() is expected to return an error, but instead got %v", err)
		return
	}
	if !strings.Contains(err.Error(), "method not supported") {
		t.Errorf("Expected error that contains 'method not supported', but instead got %v", err)
		return
	}
	if resp == nil || resp.Error == nil {
		t.Fatalf("Expected error in response, got none")
	}

	if !strings.Contains(resp.Error.Message, "method not supported") {
		t.Fatalf("Expected response error message containing 'method not supported', got: %s", resp.Error.Message)
	}
}

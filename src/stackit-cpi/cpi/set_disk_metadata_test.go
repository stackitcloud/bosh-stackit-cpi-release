package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	. "github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	iaasConfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func setupSetDiskMetadataMockClient(cpi *CPI, mockHandler http.HandlerFunc) *httptest.Server {
	mockServer := httptest.NewServer(mockHandler)

	return mockServer
}

func TestSetDiskMetadata_SuccessfulUpdate(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SetDiskMetadata, t)

	// Set up the mock response for a successful metadata update
	diskID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
	metadata := map[string]any{
		"director":      "test-director",
		"deployment":    "test-deployment",
		"instance_id":   "test-instance-id",
		"job":           "test-job",
		"instance_name": "test-instance-name",
		"attached_at":   "2022-01-01T00:00:00Z",
	}

	// Override the arguments to match our test data
	args := []any{diskID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	var updateRequested bool

	// Mock server that simulates successful metadata update
	mockServer := setupSetDiskMetadataMockClient(cpi, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Exact path match is crucial and must use PATCH method
		expectedPath := fmt.Sprintf("/v2/projects/%s/regions/%s/volumes/%s", cpi.Config.ProjectID, cpi.Config.RegionID, diskID)

		if r.Method == http.MethodPatch && r.URL.Path == expectedPath {
			updateRequested = true

			// Verify the request body contains the metadata
			var requestBody map[string]any
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&requestBody); err != nil {
				t.Errorf("Failed to decode request body: %v", err)
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			// Check if labels are in the request
			labels, ok := requestBody["labels"]
			if !ok {
				t.Errorf("Expected labels in request, got none")
			} else {
				// Verify labels match our metadata
				labelsMap, ok := labels.(map[string]any)
				if !ok {
					t.Errorf("Expected labels to be map, got %T", labels)
				} else {
					for k, v := range metadata {
						// Special case for attached_at which gets processed to remove non-numeric chars
						if k == "attached_at" && v == "2022-01-01T00:00:00Z" {
							expectedValue := "20220101000000"
							if lv, ok := labelsMap[k]; !ok || lv != expectedValue {
								t.Errorf("Expected label %s=%v, got %v", k, expectedValue, lv)
							}
						} else {
							if lv, ok := labelsMap[k]; !ok || lv != v {
								t.Errorf("Expected label %s=%v, got %v", k, v, lv)
							}
						}
					}
				}
			}

			// Return a successful response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(iaas.Volume{
				Id:               utils.Ptr(uuid.New().String()),
				AvailabilityZone: "test",
			})
			return
		}

		// Log the mismatch for debugging
		t.Logf("Path mismatch. Got: %s, Expected: %s, Method: %s",
			r.URL.Path, expectedPath, r.Method)

		// Default: Teapot
		http.Error(w, "Teapot", http.StatusTeapot)
	}))
	defer mockServer.Close()
	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []iaasConfig.ConfigurationOption{iaasConfig.WithEndpoint(mockServer.URL), iaasConfig.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the SetDiskMetadata method
	resp, err := cpi.SetDiskMetadata(req)
	if err != nil {
		t.Errorf("SetDiskMetadata() unexpected error = %v", err)
		return
	}

	// Check that the update request was made
	if !updateRequested {
		t.Errorf("Expected update request to be made, but it wasn't")
	}

	// Check the result - should be true for successful update
	result, ok := resp.Result.(bool)
	if !ok {
		t.Errorf("Expected boolean result, got %T", resp.Result)
		return
	}

	if !result {
		t.Errorf("Expected result to be true for successful metadata update, got %v", result)
	}
}

func TestSetDiskMetadata_DiskNotFound(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SetDiskMetadata, t)

	// Set up the mock response for a non-existent disk
	diskID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
	metadata := map[string]any{
		"director":   "test-director",
		"deployment": "test-deployment",
	}

	// Override the arguments to match our test data
	args := []any{diskID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	// Mock server that simulates disk not found
	mockServer := setupSetDiskMetadataMockClient(cpi, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, diskID) {
			// Return a 404 error for non-existent disk
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "Disk not found"}`))
			return
		}
		// Default: Teapot
		http.Error(w, "Teapot", http.StatusTeapot)
	}))
	defer mockServer.Close()
	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []iaasConfig.ConfigurationOption{iaasConfig.WithEndpoint(mockServer.URL), iaasConfig.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the SetDiskMetadata method, expect an error
	resp, err := cpi.SetDiskMetadata(req)

	// This should result in an error
	if err == nil {
		t.Errorf("SetDiskMetadata() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

func TestSetDiskMetadata_APIError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SetDiskMetadata, t)

	// Set up the mock response for an API error
	diskID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
	metadata := map[string]any{
		"director":   "test-director",
		"deployment": "test-deployment",
	}

	// Override the arguments to match our test data
	args := []any{diskID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	// Mock server that simulates an API error
	mockServer := setupSetDiskMetadataMockClient(cpi, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, diskID) {
			// Return a 500 error
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "Internal Server Error"}`))
			return
		}
		// Default: Teapot
		http.Error(w, "Teapot", http.StatusTeapot)
	}))
	defer mockServer.Close()
	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []iaasConfig.ConfigurationOption{iaasConfig.WithEndpoint(mockServer.URL), iaasConfig.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the SetDiskMetadata method, expect an error
	resp, err := cpi.SetDiskMetadata(req)

	// This should result in an error
	if err == nil {
		t.Errorf("SetDiskMetadata() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

func TestSetDiskMetadata_InvalidMetadata(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.SetDiskMetadata, t)

	// Set up test for invalid metadata format (like invalid characters in keys)
	diskID := "12345678-1234-1234-1234-123456789012" // Valid UUID format
	metadata := map[string]any{
		"invalid!key": "value-with-invalid-key",
		"director":    "test-director",
	}

	// Override the arguments to match our test data
	args := []any{diskID, metadata}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}
	req.Arguments = argsJSON

	// Mock server that simulates validation error for metadata
	mockServer := setupSetDiskMetadataMockClient(cpi, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, diskID) {
			// Return a 400 Bad Request for invalid metadata
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "Invalid metadata format"}`))
			return
		}
		// Default: Teapot
		http.Error(w, "Teapot", http.StatusTeapot)
	}))
	defer mockServer.Close()
	sClient, _ := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []iaasConfig.ConfigurationOption{iaasConfig.WithEndpoint(mockServer.URL), iaasConfig.WithoutAuthentication()})

	cpi.SClient = sClient

	// Call the SetDiskMetadata method, expect an error
	resp, err := cpi.SetDiskMetadata(req)

	// This should result in an error
	if err == nil {
		t.Errorf("SetDiskMetadata() expected error, got nil")
	}

	// Verify error in response
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("Expected error in response, got none")
	}
}

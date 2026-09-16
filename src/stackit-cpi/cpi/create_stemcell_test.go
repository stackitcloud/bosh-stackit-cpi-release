package cpi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"
)

func TestCreateStemcell_Success(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateStemcell, t)

	currentDir, err := os.Getwd()
	if err != nil {
		t.Errorf("failed getting current dir")
	}
	fakeStemcellPath := fmt.Sprintf("%s/%s", currentDir, "assets/fake_stemcell.tgz")
	args := []any{
		fakeStemcellPath,
		map[string]any{
			"name":             "test-stemcell",
			"version":          "1.0",
			"disk_format":      "raw",
			"os_type":          "linux",
			"os_distro":        "ubuntu",
			"architecture":     "x86_64",
			"infrastructure":   "openstack",
			"hypervisor":       "kvm",
			"disk":             5120,
			"container_format": "bare",
			"auto_disk_config": true,
		},
	}
	bytes, err := json.Marshal(args)
	if err != nil {
		t.Errorf("failed to marshal stemcell args")
	}
	uploadTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer uploadTarget.Close()
	testImageID := "00000000-0000-0000-0000-000000000001"
	req.Arguments = bytes
	mockServer := SetupMockClient(cpi, map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/images", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.ImageListResponse{
			Items: []iaas.Image{},
		},
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/images", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.ImageCreateResponse{
			Id:        testImageID,
			UploadUrl: fmt.Sprintf("%s/upload-test", uploadTarget.URL),
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/images/%s", cpi.Config.ProjectID, cpi.Config.RegionID, testImageID): iaas.Image{
			Id:         utils.Ptr(testImageID),
			Name:       "test-stemcell",
			DiskFormat: "qcow",
			Status:     utils.Ptr(wait.ImageAvailableStatus),
		},
	})

	defer mockServer.Close()

	resp, err := cpi.CreateStemcell(req)
	if err != nil {
		t.Errorf("didn't expect an error but received %v", err)
	}
	// Check that result contains the stemcell ID
	result, ok := resp.Result.(string)
	if !ok {
		t.Errorf("Expected string result, got %T", resp.Result)
		return
	}
	if result != testImageID {
		t.Errorf("Expected result to be 'new-image-id-need-to-be-36characters', got %s", result)
	}
}

func TestCreateStemcell_UploadError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateStemcell, t)
	currentDir, _ := os.Getwd()

	args := []any{
		fmt.Sprintf("%s/assets/fake_stemcell.tgz", currentDir),
		map[string]any{
			"name":             "test-stemcell",
			"version":          "1.0",
			"disk_format":      "raw",
			"os_type":          "linux",
			"os_distro":        "ubuntu",
			"architecture":     "x86_64",
			"infrastructure":   "openstack",
			"hypervisor":       "kvm",
			"disk":             5120,
			"container_format": "bare",
			"auto_disk_config": true,
		},
	}

	// Marshal the arguments to json.RawMessage
	argsBytes, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}

	req.Arguments = argsBytes

	cpi.Config.ProjectID = "duplicate-0000-0000-0000-00000-00004"
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/images", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.ImageListResponse{
			Items: []iaas.Image{},
		},
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/images", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.ImageCreateResponse{
			Id:        "",
			UploadUrl: "",
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	resp, err := cpi.CreateStemcell(req)

	expectedErrMsg := "received nil UploadUrl or Id from CreateImage response"
	if err == nil {
		t.Errorf("CreateStemcell() expected error, got nil")
	} else if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}

	if !strings.Contains(resp.Error.Message, expectedErrMsg) {
		t.Errorf("Expected response error message '%s', got '%s'", expectedErrMsg, resp.Error.Message)
	}
}

func TestCreateStemcell_ProcessingError(t *testing.T) {
	cpi, req := SetupTestCPIAndRequest(lib.CreateStemcell, t)

	imageID := "new-image-id-need-to-be-36-char-long"

	cpi.Config.ProjectID = "duplicate-0000-0000-0000-00000-00004"
	mockResponses := map[string]any{
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/images", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.ImageListResponse{
			Items: []iaas.Image{},
		},
		fmt.Sprintf("POST_/v2/projects/%s/regions/%s/images", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.ImageCreateResponse{
			Id:        imageID,
			UploadUrl: "upload",
		},
		"PUT_/upload": FakeResponse{
			StatusCode: http.StatusOK,
			Body:       []byte(``),
		},
		fmt.Sprintf("GET_/v2/projects/%s/regions/%s/images/%s", cpi.Config.ProjectID, cpi.Config.RegionID, imageID): iaas.Image{
			Id:     utils.Ptr(imageID),
			Status: utils.Ptr(wait.ErrorStatus),
		},
	}
	mockServer := SetupMockClient(cpi, mockResponses)
	defer mockServer.Close()

	_, err := cpi.CreateStemcell(req)
	if err == nil {
		t.Error("CreateStemcell(): expected error for failed processing, got nil")
	}
}

func TestHandleImageCreationError(t *testing.T) {
	currentDir, err := os.Getwd()
	args := []any{
		fmt.Sprintf("%s/assets/fake_stemcell.tgz", currentDir),
		map[string]any{
			"name":             "test-stemcell",
			"version":          "1.0",
			"disk_format":      "raw",
			"os_type":          "linux",
			"os_distro":        "ubuntu",
			"architecture":     "x86_64",
			"infrastructure":   "openstack",
			"hypervisor":       "kvm",
			"disk":             5120,
			"container_format": "bare",
			"auto_disk_config": true,
		},
	}

	// Marshal the arguments to json.RawMessage
	argsBytes, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal arguments: %v", err)
	}

	tests := []struct {
		name           string
		expectedOutput string // What we expect to find in the output
		postResponse   oapierror.GenericOpenAPIError
		wantRetry      bool
	}{
		{
			name:           "too many requests",
			expectedOutput: "too many requests",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusTooManyRequests,
				ErrorMessage: "too many requests",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: true,
		},
		{
			name:           "bad request",
			expectedOutput: "bad request",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusBadRequest,
				ErrorMessage: "bad requests",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: false,
		},
		{
			name:           "unauthorized",
			expectedOutput: "unauthorized",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusUnauthorized,
				ErrorMessage: "untauthorized",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: false,
		},
		{
			name:           "forbidden",
			expectedOutput: "forbidden",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusForbidden,
				ErrorMessage: "forbidden",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: false,
		},
		{
			name:           "not found",
			expectedOutput: "not found",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusNotFound,
				ErrorMessage: "not found",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: false,
		},
		{
			name:           "internal server error",
			expectedOutput: "internal server error",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusInternalServerError,
				ErrorMessage: "internal server error",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: true,
		},
		{
			name:           "not implemented",
			expectedOutput: "not implemented",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusNotImplemented,
				ErrorMessage: "not implemented",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: false,
		},
		{
			name:           "bad gateway",
			expectedOutput: "bad gateway",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusBadGateway,
				ErrorMessage: "bad gateway",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: true,
		},
		{
			name:           "service unavailable",
			expectedOutput: "service unavailable",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusServiceUnavailable,
				ErrorMessage: "service unavailable",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: true,
		},
		{
			name:           "gateway timeout",
			expectedOutput: "gateway timeout",
			postResponse: oapierror.GenericOpenAPIError{
				StatusCode:   http.StatusGatewayTimeout,
				ErrorMessage: "gateway timeout",
				Body:         []byte{},
				Model:        nil,
			},
			wantRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpi, req := SetupTestCPIAndRequest(lib.CreateStemcell, t)

			req.Arguments = argsBytes
			// Mock response with no matching images
			mockResponses := map[string]any{
				fmt.Sprintf("GET_/v2/projects/%s/regions/%s/images", cpi.Config.ProjectID, cpi.Config.RegionID): iaas.ImageListResponse{
					Items: []iaas.Image{
						{},
					},
				},
				fmt.Sprintf("POST_/v2/projects/%s/regions/%s/images", cpi.Config.ProjectID, cpi.Config.RegionID): tt.postResponse,
			}
			testServer := SetupMockClient(cpi, mockResponses)
			defer testServer.Close()
			resp, err := cpi.CreateStemcell(req)
			if err == nil {
				t.Errorf("expected error but got nil")
			}
			if resp.Error.OkToRetry != tt.wantRetry {
				t.Errorf("retry flag = %v, want %v",
					resp.Error.OkToRetry, tt.wantRetry)
			}
		})
	}
}

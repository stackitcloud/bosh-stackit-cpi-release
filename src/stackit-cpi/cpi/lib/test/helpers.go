// Package lib_test contains test setup helpers
// so they can be shared across suites
package lib_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/cloudfoundry/bosh-agent/v2/settings"
	"github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"
)

// Valid UUID-formatted IDs for testing
const (
	TestNetworkID       = "12345678-1234-1234-1234-123456789012"
	TestSecondNetworkID = "12345678-1234-1234-1234-123456789034"
	TestPublicIPAddr    = "1.2.3.4"
	TestPublicIPID      = "12345678-1234-1234-1234-123456789056"
	TestNICID1          = "11111111-1234-1234-1234-123456789056"
	TestNICID2          = "22222222-1234-1234-1234-123456789056"
	TestServerID        = "12345678-1234-1234-1234-123456789012"
	TestDiskID          = "87654321-4321-4321-4321-210987654321"
	TestDetachServerID  = "12345678-1234-1234-1234-123456789012"
	TestDetachDiskID    = "87654321-4321-4321-4321-210987654321"
)

var (
	TestPublicIPPayload = iaas.PublicIp{
		Id: utils.Ptr(TestPublicIPID),
		Ip: utils.Ptr(TestPublicIPAddr),
	}
	TestNicPayload1 = &iaas.NIC{
		Id:        utils.Ptr(TestNICID1),
		Name:      utils.Ptr("some-test-nic"),
		NetworkId: utils.Ptr(TestNetworkID),
		Status:    utils.Ptr(wait.CreateSuccess),
		Mac:       utils.Ptr("11:1A:2B:3C:4D:5E"),
	}
	TestNicPayload2 = &iaas.NIC{
		Id:        utils.Ptr(TestNICID2),
		Name:      utils.Ptr("some-test-nic-2"),
		NetworkId: utils.Ptr(TestSecondNetworkID),
		Status:    utils.Ptr(wait.CreateSuccess),
		Mac:       utils.Ptr("00:1A:2B:3C:4D:5E"),
	}

	TestPublicIPListPayload = iaas.PublicIpListResponse{
		Items: []iaas.PublicIp{
			TestPublicIPPayload,
		},
	}
	TestVMTypePayload = &iaas.MachineTypeListResponse{
		Items: []iaas.MachineType{
			{
				Disk:  int64(20),
				Name:  "valid",
				Ram:   int64(20),
				Vcpus: int64(1),
			},
		},
	}
	TestServerPayload = iaas.Server{
		Id:               utils.Ptr(TestServerID),
		Name:             "test-server-1234",
		MachineType:      "some-type",
		Status:           utils.Ptr(wait.ServerActiveStatus),
		AvailabilityZone: utils.Ptr("az-x"),
	}
	TestDiskAttachmentPayload = iaas.VolumeAttachment{
		DeleteOnTermination: utils.Ptr(true),
		ServerId:            utils.Ptr(TestServerID),
		VolumeId:            utils.Ptr(TestDiskID),
	}
	TestDiskPayload = iaas.Volume{
		AvailabilityZone: "az-x",
		Id:               utils.Ptr(TestDiskID),
	}
)

func NewTestLogger(t testing.TB, req lib.RPCRequest) *lib.CPILogger {
	logFile, err := os.CreateTemp("/tmp", "test-")
	if err != nil {
		t.Errorf("failed setting up logfile")
	}
	logger, err := lib.NewCPILogger(logFile.Name(), slog.LevelDebug, 1, 0, false)
	if err != nil {
		t.Errorf("failed setting up logger")
	}
	logger.WithContext(req.GetLoggingContext())
	if os.Getenv("TEST_DEBUG") == "true" {
		fmt.Println("log location:", logFile.Name())
	}

	return logger
}

type FakeStemcellPostResponse struct {
	ID        string
	UploadURL string
}
type FakeResponse struct {
	StatusCode int
	Body       []byte
}
type FakeTimeout struct {
	After int
}

type AfterRetryResponse struct {
	FirstRespond       any
	FinallyRespondWith any
}

func SetupTestCPIAndRequest(cpiMethod string, t *testing.T) (*cpi.CPI, lib.RPCRequest) {
	// Set up a minimal CPI instance for testing

	req := lib.RPCRequest{
		Method: cpiMethod,
		Context: lib.RPCContext{
			DirectorUUID: "test-director-uuid",
			RequestID:    "test-request-id",
		},
	}
	logger := NewTestLogger(t, req)
	cpi := &cpi.CPI{
		Log: logger, // Initialize the Log field with the base logger
		Config: lib.Config{
			ProjectID:  "12345678-1234-1234-1234-123456789012",
			RegionID:   "test-region",
			Timeout:    1,
			RetryCount: 1,
			LRPTimeout: 1,
			Agent: lib.AgentBlock{
				MBus: settings.MBus{
					URLs: []string{"mbus://some.nats:4222"},
				},
			},
		},
	}

	return cpi, req
}

func SetupMockClient(cpi *cpi.CPI, expectedResponses map[string]any) *httptest.Server {
	pathRetries := map[string]int{}
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if debug := os.Getenv("TEST_DEBUG"); debug != "" {
			fmt.Printf("%s_%s\n", r.Method, r.URL.Path)
		}
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		val, ok := expectedResponses[fmt.Sprintf("%s_%s", r.Method, path)]
		if !ok {
			val, ok = expectedResponses[fmt.Sprintf("%s_%s?details=true", r.Method, path)]
		}
		if ok {
			if debug := os.Getenv("TEST_DEBUG"); debug != "" {
				fmt.Println("matched path", path)
			}
			if stemcellPostResponse, ok := val.(iaas.ImageCreateResponse); ok {
				stemcellPostResponse.UploadUrl = stemcellPostResponse.UploadUrl
				_ = json.NewEncoder(w).Encode(stemcellPostResponse)
				return
			}
			// check if we're trying to test retry logic
			if retryData, ok := val.(AfterRetryResponse); ok {
				if pathRetries[path] < cpi.Config.RetryCount {
					pathRetries[path] += 1
					if openAPIErr, ok := retryData.FirstRespond.(oapierror.GenericOpenAPIError); ok {
						w.WriteHeader(openAPIErr.StatusCode)
					}
					json.NewEncoder(w).Encode(retryData.FirstRespond)
					return
				}

				if finalErr, ok := retryData.FinallyRespondWith.(oapierror.GenericOpenAPIError); ok {
					w.WriteHeader(finalErr.StatusCode)
				}
				if finalResp, ok := retryData.FinallyRespondWith.(FakeResponse); ok {
					w.WriteHeader(finalResp.StatusCode)
				}
				json.NewEncoder(w).Encode(retryData.FinallyRespondWith)
				return
			}
			if timeoutVal, ok := val.(FakeTimeout); ok {
				time.Sleep(time.Duration(timeoutVal.After) * time.Second)
			}
			// try to unwrap into a fake error
			if fakeErr, ok := val.(oapierror.GenericOpenAPIError); ok {
				w.WriteHeader(fakeErr.StatusCode)
				json.NewEncoder(w).Encode(fakeErr)
				return
			}
			if finalResp, ok := val.(FakeResponse); ok {
				w.WriteHeader(finalResp.StatusCode)
				w.Write(finalResp.Body)
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(val)
			return
		}
		fmt.Printf("unexpected %s to %s\n", r.Method, path)
		fmt.Println("expected paths:")
		for _, p := range slices.Collect(maps.Keys(expectedResponses)) {
			fmt.Println(p)
		}
		http.Error(w, "you shouldn't be here", http.StatusTeapot)
	}))

	sClient, err := lib.NewStackitClient(cpi.Config, cpi.Log.ChildLogger("api_client"), []config.ConfigurationOption{config.WithoutAuthentication(), config.WithEndpoint(mockServer.URL)})
	if err != nil {
		panic(err)
	}
	if cpi.Config.RetryCount == 0 {
		cpi.SClient = sClient
	} else {
		cpi.SClient = lib.NewRetryableStackitClient(sClient, cpi.Config.RetryCount, cpi.Log, lib.WithDelayBaseInSeconds(0))
	}
	return mockServer
}

var CreateVMTestArgs = []any{
	"test-agent-id",    // Agent ID
	"test-stemcell-id", // Stemcell ID
	map[string]any{ // Cloud properties
		"availability_zone": "test-zone",
		"instance_type":     "valid",
		"root_disk": map[string]any{
			"size": 30720, // 30GB in MB
		},
	},
	map[string]any{ // Networks
		"vip": map[string]any{
			"type": "vip",
			"ip":   TestPublicIPAddr,
		},
		"manual": map[string]any{
			"type":    "manual",
			"ip":      "10.0.0.10",
			"netmask": "255.255.255.0",
			"gateway": "10.0.0.1",
			"dns":     []string{"8.8.8.8", "8.8.4.4"},
			"default": []string{"dns"},
			"cloud_properties": map[string]any{
				"net_id": TestNetworkID,
			},
		},
		"dynamic": map[string]any{
			"type":    "dynamic",
			"ip":      "10.0.0.10",
			"netmask": "255.255.255.0",
			"gateway": "10.0.0.1",
			"dns":     []string{},
			"default": []string{"gateway"},
			"cloud_properties": map[string]any{
				"net_id": TestSecondNetworkID,
			},
		},
	},
	[]string{}, // Disk IDs
	map[string]any{ // Environment
		"bosh": map[string]any{
			"password": "test-password",
			"mbus": map[string]any{
				"cert": map[string]any{
					"ca":          "test-ca",
					"certificate": "test-cert",
					"private_key": "test-key",
				},
			},
			"blobstores": []map[string]any{
				{
					"provider": "local",
					"options": map[string]any{
						"endpoint": "https://10.0.0.2:25250",
						"user":     "agent",
						"password": "agent-password",
					},
				},
			},
			"ntp": []string{"0.pool.ntp.org", "1.pool.ntp.org"},
		},
		"groups": []string{"test-group"},
		"group":  "test-instance-group",
		"tags": map[string]any{
			"deployment": "test-deployment",
			"job":        "test-job",
		},
	},
}

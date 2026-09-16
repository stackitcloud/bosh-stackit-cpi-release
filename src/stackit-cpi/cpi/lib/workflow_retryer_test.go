package lib_test

import (
	"encoding/json"
	"errors"

	"github.com/cloudfoundry/bosh-agent/v2/settings"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"

	"github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-cpi/cpi/lib/libfakes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

var _ = Describe("WorkflowRetryer", func() {
	var req lib.RPCRequest
	var testCPI cpi.CPI
	var fakeClient libfakes.FakeIaasClient
	Context("blanket retryable errors create_vm errors", func() {
		logger := NewTestLogger(GinkgoTB(), req)
		BeforeEach(func() {
			fakeClient = libfakes.FakeIaasClient{}
			argBytes, err := json.Marshal(CreateVMTestArgs)
			Expect(err).ToNot(HaveOccurred())
			req = lib.RPCRequest{
				Method:     "create_vm",
				Arguments:  argBytes,
				APIVersion: 2,
				Context: lib.RPCContext{
					DirectorUUID: uuid.New().String(),
					RequestID:    uuid.New().String(),
				},
			}
			testCPI = cpi.CPI{
				Config: lib.Config{
					Agent: lib.AgentBlock{
						MBus: settings.MBus{
							URLs: []string{"test.com"},
						},
					},
					RegionID:                    "fake",
					ProjectID:                   "fake",
					HumanReadableVMNames:        false,
					AutoFixNameAndLabels:        false,
					Timeout:                     1,
					LRPTimeout:                  2,
					LogFile:                     "/tmp/test.log",
					RetryCount:                  2,
					LogLevel:                    "Debug",
					ServiceAccountJSON:          "{}",
					StageProfileJSON:            "{}",
					DefaultRootVolumeType:       "fake",
					DefaultPersistentVolumeType: "fake",
					DefaultSecurityGroups:       []string{"present"},
					DefaultSSHKeyName:           "fake",
				},
				Log:     logger,
				SClient: &fakeClient,
			}
		})
		It("terminally fails retrying non generic build aborted network errors", func() {
			fakeServer := &iaas.Server{Id: utils.Ptr("fake-id")}
			fakeClient.CreateVMReturns(fakeServer, nil)
			fakeClient.CreateNicReturns(&iaas.NIC{Id: utils.Ptr("test")}, nil)
			fakeClient.ListSecurityGroupsReturns([]iaas.SecurityGroup{{Name: "present"}}, nil)
			fakeClient.WaitForVMReturns(errors.New(`found non-GenericOpenApiError: create failed for server with id f23afdee-bc5a-4bf7-ad57-02f2e5073dcb: Build of instance f23afdee-bc5a-4bf7-ad57-02f2e5073dcb aborted: Failed to allocate the network(s), not rescheduling.`))
			_, err := testCPI.CreateVM(req)
			// retries + initial attempt
			for i := range testCPI.Config.RetryCount {
				Expect(fakeClient.DeleteVMArgsForCall(i)).To(Equal(*fakeServer.Id))
			}
			// here we want to see that the nics will be cleaned up because we terminally failed
			Expect(fakeClient.DeleteNicCallCount()).ToNot(Equal(0))
			Expect(fakeClient.CreateVMCallCount()).To(Equal(testCPI.Config.RetryCount + 1))
			Expect(fakeClient.WaitForVMCallCount()).To(Equal(testCPI.Config.RetryCount + 1))
			Expect(fakeClient.DeleteVMCallCount()).To(Equal(testCPI.Config.RetryCount + 1))
			Expect(err).To(HaveOccurred())
		})
		It("retries non generic build aborted network errors", func() {
			fakeServer := &iaas.Server{Id: utils.Ptr("fake-id")}
			fakeClient.CreateVMReturns(fakeServer, nil)
			fakeClient.CreateNicReturns(&iaas.NIC{}, nil)
			fakeClient.ListSecurityGroupsReturns([]iaas.SecurityGroup{{Name: "present"}}, nil)
			for i := range testCPI.Config.RetryCount {
				fakeClient.WaitForVMReturnsOnCall(i, errors.New(`found non-GenericOpenApiError: create failed for server with id f23afdee-bc5a-4bf7-ad57-02f2e5073dcb: Build of instance f23afdee-bc5a-4bf7-ad57-02f2e5073dcb aborted: Failed to allocate the network(s), not rescheduling.`))
			}
			_, err := testCPI.CreateVM(req)
			// retries + initial attempt
			for i := range testCPI.Config.RetryCount {
				Expect(fakeClient.DeleteVMArgsForCall(i)).To(Equal(*fakeServer.Id))
			}
			Expect(fakeClient.DeleteNicCallCount()).To(Equal(0))
			Expect(fakeClient.CreateVMCallCount()).To(Equal(testCPI.Config.RetryCount + 1))
			Expect(fakeClient.WaitForVMCallCount()).To(Equal(testCPI.Config.RetryCount + 1))
			Expect(fakeClient.DeleteVMCallCount()).To(Equal(testCPI.Config.RetryCount))
			Expect(err).To(BeNil())
		})
	})
})

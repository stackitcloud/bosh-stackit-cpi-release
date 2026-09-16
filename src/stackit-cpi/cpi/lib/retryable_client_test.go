package lib_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-cpi/cpi/lib/libfakes"
	. "github.com/stackitcloud/stackit-cpi/cpi/lib/test"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

var _ = Describe("RetryableClient", func() {
	var req lib.RPCRequest
	var logger *lib.CPILogger
	retries := 3
	JustBeforeEach(func() {
		logger = NewTestLogger(GinkgoTB(), req)
	})
	Context("blanket retryable errors", func() {
		BeforeEach(func() {
			req = lib.RPCRequest{
				Method:     "fake",
				Arguments:  json.RawMessage{},
				APIVersion: 2,
				Context: lib.RPCContext{
					DirectorUUID: uuid.New().String(),
					RequestID:    uuid.New().String(),
				},
			}
		})
		It("doesn't retry when there are no errors", func() {
			var fakeClient libfakes.FakeIaasClient
			retryClient := lib.NewRetryableStackitClient(&fakeClient, retries, logger, lib.WithDelayBaseInSeconds(0))
			fakeClient.AttachDiskToVMReturns(nil)
			Expect(retryClient.AttachDiskToVM("fakeVolume", "fakeVM")).To(Succeed())
			Expect(fakeClient.AttachDiskToVMCallCount()).To(Equal(1))
		})
		It("doesn't retry more than necessary", func() {
			var fakeClient libfakes.FakeIaasClient
			retryClient := lib.NewRetryableStackitClient(&fakeClient, retries, logger, lib.WithDelayBaseInSeconds(0))
			fakeClient.AttachDiskToVMReturnsOnCall(0, oapierror.GenericOpenAPIError{StatusCode: http.StatusInternalServerError})
			fakeClient.AttachDiskToVMReturnsOnCall(1, nil)
			Expect(retryClient.AttachDiskToVM("fakeVolume", "fakeVM")).To(Succeed())
			Expect(fakeClient.AttachDiskToVMCallCount()).To(Equal(2))
		})
		It("stops at max attempts reached", func() {
			var fakeClient libfakes.FakeIaasClient
			retryClient := lib.NewRetryableStackitClient(&fakeClient, retries, logger, lib.WithDelayBaseInSeconds(0))
			fakeClient.AttachDiskToVMReturns(oapierror.GenericOpenAPIError{StatusCode: http.StatusInternalServerError, Body: []byte("Internal Server Error")})
			err := retryClient.AttachDiskToVM("fakeVolume", "fakeVM")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(MatchRegexp("Internal Server Error")))
			Expect(fakeClient.AttachDiskToVMCallCount()).To(Equal(retries + 1))
		})
		It("retries even if the error changes over time as long as it is retryable", func() {
			var fakeClient libfakes.FakeIaasClient
			retryClient := lib.NewRetryableStackitClient(&fakeClient, retries, logger, lib.WithDelayBaseInSeconds(0))
			fakeClient.AttachDiskToVMReturnsOnCall(0, oapierror.GenericOpenAPIError{StatusCode: http.StatusInternalServerError})
			fakeClient.AttachDiskToVMReturnsOnCall(1, oapierror.GenericOpenAPIError{StatusCode: http.StatusGatewayTimeout})
			fakeClient.AttachDiskToVMReturnsOnCall(2, oapierror.GenericOpenAPIError{StatusCode: http.StatusBadGateway})
			fakeClient.AttachDiskToVMReturnsOnCall(3, nil)
			Expect(retryClient.AttachDiskToVM("fakeVolume", "fakeVM")).To(Succeed())
			Expect(fakeClient.AttachDiskToVMCallCount()).To(Equal(4))
		})
		It("properly passes the arguments received", func() {
			var fakeClient libfakes.FakeIaasClient
			retryClient := lib.NewRetryableStackitClient(&fakeClient, retries, logger, lib.WithDelayBaseInSeconds(0))
			fakeClient.AttachDiskToVMReturns(nil)
			Expect(retryClient.AttachDiskToVM("fakeVolume", "fakeVM")).To(Succeed())
			volumeArg, vmArg := fakeClient.AttachDiskToVMArgsForCall(0)
			Expect(volumeArg).To(Equal("fakeVolume"))
			Expect(vmArg).To(Equal("fakeVM"))
		})
		It("doesn't retry unretriable errors", func() {
			var fakeClient libfakes.FakeIaasClient
			retryClient := lib.NewRetryableStackitClient(&fakeClient, retries, logger, lib.WithDelayBaseInSeconds(0))
			fakeClient.AttachDiskToVMReturns(oapierror.GenericOpenAPIError{StatusCode: http.StatusNotImplemented})
			Expect(retryClient.AttachDiskToVM("fakeVolume", "fakeVM")).ToNot(Succeed())
			Expect(fakeClient.AttachDiskToVMCallCount()).To(Equal(1))
		})
	})
	// TODO add tests for logging via mocked logger interface
	Context("partially created vms with non generic build errors", func() {
		It("will not retry the creation or clean up the partial vm on similar errors", func() {
			var fakeClient libfakes.FakeIaasClient
			retryClient := lib.NewRetryableStackitClient(&fakeClient, retries, logger, lib.WithDelayBaseInSeconds(0))
			uuid := uuid.New().String()

			server := &iaas.Server{
				Id: &uuid,
			}
			buildError := fmt.Errorf("Build of instance %s aborted: Failed to allocate the memory, not rescheduling.", uuid)

			fakeClient.CreateVMReturnsOnCall(0, server, buildError)
			payload := iaas.CreateServerPayload{}
			_, err := retryClient.CreateVM(payload)
			Expect(err).To(HaveOccurred())
			// check for attempted cleanup
			Expect(fakeClient.DeleteVMCallCount()).To(Equal(0))
			// failed attempt + retry attemp
			Expect(fakeClient.CreateVMCallCount()).To(Equal(1))
		})
		It("will retry the creation after trying to clean up the partial vm", func() {
			var fakeClient libfakes.FakeIaasClient
			retryClient := lib.NewRetryableStackitClient(&fakeClient, retries, logger, lib.WithDelayBaseInSeconds(0))
			uuid := uuid.New().String()

			server := &iaas.Server{
				Id: &uuid,
			}
			buildError := fmt.Errorf("Build of instance %s aborted: Failed to allocate the network(s), not rescheduling.", uuid)

			fakeClient.CreateVMReturnsOnCall(0, server, buildError)
			fakeClient.CreateVMReturnsOnCall(1, server, nil)
			payload := iaas.CreateServerPayload{}
			fakeClient.DeleteVMReturns(nil)
			_, err := retryClient.CreateVM(payload)
			Expect(err).ToNot(HaveOccurred())
			// check for attempted cleanup
			Expect(fakeClient.DeleteVMCallCount()).To(Equal(1))
			// failed attempt + retry attemp
			Expect(fakeClient.CreateVMCallCount()).To(Equal(2))
		})
	})
})

package lib_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

var _ = Describe("Errors", func() {
	Describe("non generic build aborted errors", func() {
		It("recognizes them", func() {
			err := errors.New(`found non-GenericOpenApiError: create failed for server with id f23afdee-bc5a-4bf7-ad57-02f2e5073dcb: Build of instance f23afdee-bc5a-4bf7-ad57-02f2e5073dcb aborted: Failed to allocate the network(s), not rescheduling.`)

			Expect(lib.IsNonGenericBuildAbortedNetworkError(err)).To(BeTrue())
		})
		It("ignores similar looking errors", func() {
			// the below is completely made up. It has not been seen in the wild.
			err := errors.New(`found non-GenericOpenApiError: create failed for LoadBalancer with id f23afdee-bc5a-4bf7-ad57-02f2e5073dcb: Build of Loadbalancer f23afdee-bc5a-4bf7-ad57-02f2e5073dcb aborted: Failed to allocate the network(s), not rescheduling.`)

			Expect(lib.IsNonGenericBuildAbortedNetworkError(err)).To(BeFalse())
		})
	})
})

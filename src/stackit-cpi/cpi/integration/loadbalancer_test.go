package integration

import (
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/integration/helpers"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

var _ = Describe("handling Loadbalancer", func() {
	var testCPI *cpi.CPI
	var createVMArgs lib.CreateVMArgs
	var createdVMs map[string]string
	var req lib.RPCRequest
	var lbSDKClient *loadbalancer.APIClient
	var iaasSDKClient *iaas.APIClient
	var wg sync.WaitGroup
	var lb *loadbalancer.LoadBalancer
	var secGrp *iaas.SecurityGroup
	var testKey helpers.TestSSHKey
	var pubIP *iaas.PublicIp
	var vmID string
	Describe("TCP Loadbalancing", Ordered, func() {
		AfterAll(func() {
			for _, vmID := range slices.Collect(maps.Keys(createdVMs)) {
				helpers.CleanupVM(vmID, *testCPI)
			}
			Eventually(func() error {
				_, err := lbSDKClient.DefaultAPI.DeleteLoadBalancer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, lb.GetName()).Execute()
				if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
					return nil
				}
				return err
			}, "2m").Should(Succeed())
			Eventually(func() error {
				err := iaasSDKClient.DefaultAPI.DeletePublicIP(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, pubIP.GetId()).Execute()
				if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
					return nil
				}
				return err
			}, "2m").Should(Succeed())
			helpers.CleanupSecGrp(secGrp, testCPI, iaasSDKClient)
			testKey.Delete()
		})
		BeforeAll(func() {
			createdVMs = map[string]string{}
			testCPI = conf.Prod.DefaultTestCPI()
			testKey = conf.Prod.GenerateSSHKey()
			testCPI.Config.DefaultSSHKeyName = testKey.Name
			lbSDKClient = conf.Prod.GetStackitLoadBalancerClient()
			iaasSDKClient = conf.Prod.GetStackitIaasClient()

			var err error
			pubIP, err = iaasSDKClient.DefaultAPI.CreatePublicIP(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID).CreatePublicIPPayload(iaas.CreatePublicIPPayload{AdditionalProperties: pubIpTags}).Execute()
			Expect(err).ToNot(HaveOccurred())

			secGrp = conf.Prod.GenerateTestSecurityGroup()
			createVMArgs = conf.Prod.DefaultCreateVMArgs(secGrp)
			lbName := utils.Ptr(uuid.NewString())
			lb, err = lbSDKClient.DefaultAPI.CreateLoadBalancer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID).CreateLoadBalancerPayload(loadbalancer.CreateLoadBalancerPayload{
				Name: lbName,
				Networks: []loadbalancer.Network{{
					NetworkId: utils.Ptr(conf.Prod.Networks.Default.ID),
					Role:      loadbalancer.NETWORKROLE_ROLE_LISTENERS_AND_TARGETS.Ptr(),
				}},
				ExternalAddress: pubIP.Ip,
				Listeners: []loadbalancer.Listener{{
					DisplayName: lbName,
					Port:        utils.Ptr(int32(22)),
					Protocol:    loadbalancer.LISTENERPROTOCOL_PROTOCOL_TCP.Ptr(),
					TargetPool:  lbName,
				}},
				TargetPools: []loadbalancer.TargetPool{{
					Name:       lbName,
					TargetPort: utils.Ptr(int32(22)),
					Targets: []loadbalancer.Target{{
						DisplayName: lbName,
						Ip:          utils.Ptr("255.255.255.254"),
					}},
				}},
			}).Execute()
			Expect(err).ToNot(HaveOccurred())

			createVMArgs.Properties.LoadBalancer.Name = *lbName
			createVMArgs.Properties.LoadBalancer.TargetPools = []lib.TargetPool{
				{Name: *lbName},
			}
		})

		It("Attaches vms to loadbalancers on creation", func() {
			req = helpers.GenerateRequest(lib.CreateVM, createVMArgs)
			resp, err := testCPI.CreateVM(req)
			Expect(err).ToNot(HaveOccurred())
			resultList := resp.Result.([]any)
			vmID = resultList[0].(string)
			networks := resultList[1].(lib.Networks)
			createdVMs[vmID] = networks[helpers.DefaultNetName].IP

			lb, err := lbSDKClient.DefaultAPI.GetLoadBalancer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, createVMArgs.Properties.LoadBalancer.Name).Execute()
			Expect(err).ToNot(HaveOccurred())
			if !lib.HasTargetInGroup(lb, createVMArgs.Properties.LoadBalancer.TargetPools[0].Name, createdVMs[vmID]) {
				Fail(fmt.Sprintf("Expected %s to be target of %s in group: %s", createdVMs[vmID], lb.GetName(), createVMArgs.Properties.LoadBalancer.TargetPools[0].Name))
			}
		})
		It("does not attach duplicate targets", func() {
			lb, err := lbSDKClient.DefaultAPI.GetLoadBalancer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, createVMArgs.Properties.LoadBalancer.Name).Execute()
			Expect(err).ToNot(HaveOccurred())
			pools := lb.GetTargetPools()
			Expect(pools).ToNot(HaveLen(0))
			targets := pools[0].GetTargets()
			// dummy ip + just created VM
			Expect(err).ToNot(HaveOccurred())
			Expect(len(targets)).To(BeNumerically(">", 1))
			vm, err := testCPI.SClient.GetVM(vmID)
			Expect(err).ToNot(HaveOccurred())
			nics := vm.GetNics()
			Expect(nics).ToNot(HaveLen(0))
			nic, err := testCPI.SClient.GetNic(nics[0].GetNetworkId(), nics[0].GetNicId())
			Expect(err).ToNot(HaveOccurred())

			lb, err = lbSDKClient.DefaultAPI.GetLoadBalancer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, createVMArgs.Properties.LoadBalancer.Name).Execute()
			Expect(err).ToNot(HaveOccurred())
			newPools := lb.GetTargetPools()
			Expect(newPools[0].GetTargets()).To(Equal(pools[0].GetTargets()))

			Expect(testCPI.SClient.AttachNICToLB(createVMArgs.Properties.LoadBalancer.TargetPools[0], *nic, lb)).To(Succeed())
		})
		It("adds instances in parallel to an already configured target group without losing targets", func() {
			for range 4 {
				m := sync.Mutex{}
				wg.Go(func() {
					testCPI.Config.RetryCount = 2
					createVMArgs.AgentID = uuid.NewString()
					createVMArgs.Properties.AvailabilityZone = fmt.Sprintf("%s-%v", testCPI.Config.RegionID, rand.IntN(3-1)+1)
					defer GinkgoRecover()

					req = helpers.GenerateRequest(lib.CreateVM, createVMArgs)
					resp, err := testCPI.CreateVM(req)
					Expect(err).ToNot(HaveOccurred())
					resultList := resp.Result.([]any)
					vmID := resultList[0].(string)
					networks := resultList[1].(lib.Networks)
					m.Lock()
					createdVMs[vmID] = networks[helpers.DefaultNetName].IP
					m.Unlock()

					lb, err := lbSDKClient.DefaultAPI.GetLoadBalancer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, createVMArgs.Properties.LoadBalancer.Name).Execute()
					Expect(err).ToNot(HaveOccurred())
					Expect(lib.HasTargetInGroup(lb, createVMArgs.Properties.LoadBalancer.TargetPools[0].Name, createdVMs[vmID])).To(BeTrue())
				})
			}
			// wait for all vms to finish creating before running the checks
			wg.Wait()
			lb, err := lbSDKClient.DefaultAPI.GetLoadBalancer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, createVMArgs.Properties.LoadBalancer.Name).Execute()
			Expect(err).ToNot(HaveOccurred())
			pool := lb.GetTargetPools()[0]
			// we need to consider the dummy interface that we need to create to be able to create the loadbalancer.
			Expect(len(createdVMs) + 1).To(Equal(len(pool.GetTargets())))
			for _, ip := range slices.Collect(maps.Values(createdVMs)) {
				Expect(lib.HasTargetInGroup(lb, createVMArgs.Properties.LoadBalancer.TargetPools[0].Name, ip)).To(BeTrue())
			}
		})
		It("detaches vms from loadbalancers on deletion in parallel and leaves expected targets", func() {
			wg := sync.WaitGroup{}

			for _, vmID := range slices.Collect(maps.Keys(createdVMs)) {
				wg.Go(func() {
					defer GinkgoRecover()
					req = helpers.GenerateRequest(lib.DeleteVM, []string{vmID})
					_, err := testCPI.DeleteVM(req)
					Expect(err).ToNot(HaveOccurred())
				})
			}
			wg.Wait()
			lb, err := lbSDKClient.DefaultAPI.GetLoadBalancer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, createVMArgs.Properties.LoadBalancer.Name).Execute()
			Expect(err).ToNot(HaveOccurred())
			pool := lb.GetTargetPools()[0]
			// the dummy target
			Expect(pool.GetTargets()).To(HaveLen(1))
		})
	})
})

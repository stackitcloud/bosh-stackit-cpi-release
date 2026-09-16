package integration

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry/bosh-agent/v2/settings"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi"
	. "github.com/stackitcloud/stackit-cpi/cpi/integration/helpers"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

var _ = Describe("creating and deleting VMs", func() {
	Describe("deploying into regions", Ordered, func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		var network *iaas.Network
		var project TestProject
		var secGrp *iaas.SecurityGroup
		var testKey TestSSHKey
		var vmArgs lib.CreateVMArgs
		var vmID string
		var stemcellID string
		BeforeAll(func() {
			var err error
			project = conf.Prod
			project.Region = "eu02"
			testCPI = project.DefaultTestCPI()

			testKey = project.GenerateSSHKey()
			secGrp = project.GenerateTestSecurityGroup()

			iaasSDKClient = project.GetStackitIaasClient()
			testCPI.Config.DefaultSSHKeyName = testKey.Name

			network, err = iaasSDKClient.DefaultAPI.CreateNetwork(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID).CreateNetworkPayload(iaas.CreateNetworkPayload{
				Name: uuid.NewString(),
				Ipv4: &iaas.CreateNetworkIPv4{
					CreateNetworkIPv4WithPrefixLength: &iaas.CreateNetworkIPv4WithPrefixLength{
						Nameservers:  []string{"8.8.8.8"},
						PrefixLength: int64(24),
					},
				},
			}).Execute()

			Expect(err).ToNot(HaveOccurred())
		})
		AfterAll(func() {
			Eventually(iaasSDKClient.DefaultAPI.DeleteNetwork(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, network.GetId()).Execute, "2m").Should(Succeed())
			CleanupSecGrp(secGrp, testCPI, iaasSDKClient)
			testKey.Delete()
		})
		It("creates stemcells", func() {
			stemcellDownload, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			defer stemcellDownload.Close()
			resp, err := http.Get(conf.Meta.StemcellURL)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			// stemcells are double compressed. The tarball from bosh.io contains a file named `image`
			// image is a tarball itself.
			// usually, the bosh CLI / bosh director unpacks the outer tarball and provides the path to the `image` file for the create_stemcell dall
			gzipReader, err := gzip.NewReader(resp.Body)
			if err == io.EOF {
				err = nil
			}
			Expect(err).ToNot(HaveOccurred())
			tarReader := tar.NewReader(gzipReader)
			innerTarball, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			for {
				header, err := tarReader.Next()
				if err == io.EOF {
					err = nil
					break
				}
				Expect(err).ToNot(HaveOccurred())
				if strings.Contains(header.Name, "image") {
					defer innerTarball.Close()
					_, err = io.Copy(innerTarball, tarReader)
					Expect(err).ToNot(HaveOccurred())
					break
				}
			}

			stemcellArgs := []any{
				innerTarball.Name(),
				map[string]any{
					"name":         "bosh-test-stemcell",
					"version":      "0.666",
					"os_type":      "linux",
					"os_distro":    "ubuntu",
					"architecture": "x86_64",
					"disk_format":  "qcow2",
				},
			}
			args, err := json.Marshal(stemcellArgs)
			Expect(err).ToNot(HaveOccurred())

			cpiResp, err := testCPI.CreateStemcell(lib.RPCRequest{
				Method:     "create_stemcell",
				Arguments:  args,
				APIVersion: 1,
				Context:    lib.RPCContext{},
			})
			Expect(err).ToNot(HaveOccurred())
			stemcellID = cpiResp.Result.(string)
			Expect(stemcellID).ToNot(BeEmpty())
		})
		It("creates VMs", func() {
			vmArgs = project.DefaultCreateVMArgs(secGrp)
			vmArgs.Properties.InstanceType = "t3i.1"
			vmArgs.StemcellID = stemcellID
			vmArgs.Networks["vip"] = lib.Network{
				Network: settings.Network{
					Type: "vip",
				},
			}
			req := GenerateRequest(lib.CreateVM, vmArgs)
			err := json.Unmarshal(req.Arguments, &vmArgs)
			Expect(err).ToNot(HaveOccurred())

			boshNetwork := vmArgs.Networks["default"]
			boshNetwork.Properties.NetID = network.GetId()
			boshNetwork.Prefix = project.Networks.Default.CIDR
			boshNetwork.Gateway = project.Networks.Default.Gateway
			vmArgs.Networks["default"] = boshNetwork
			vmArgs.Properties.AvailabilityZone = "eu02-1"

			bytes, err := json.Marshal(vmArgs)
			Expect(err).ToNot(HaveOccurred())

			req.Arguments = bytes

			resp, err := testCPI.CreateVM(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Result).ToNot(BeEmpty())
			resultArray := resp.Result.([]any)
			vmID = resultArray[0].(string)
			vm, err := iaasSDKClient.DefaultAPI.GetServer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, vmID).Details(true).Execute()
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.GetAvailabilityZone()).To(Equal(vmArgs.Properties.AvailabilityZone))
		})
		It("deletes VMs", func() {
			req := GenerateRequest(lib.DeleteVM, []string{vmID})

			resp, err := testCPI.DeleteVM(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Result.(bool)).To(BeTrue())
		})
		It("deletes stemcells", func() {
			req := GenerateRequest(lib.DeleteStemcell, []string{stemcellID})
			_, err := testCPI.DeleteStemcell(req)
			Expect(err).ToNot(HaveOccurred())
			_, err = testCPI.SClient.GetStemcell(stemcellID)
			Expect(lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound)).To(BeTrue())
		})
	})
	Describe("a vip network with static ip is provided", Ordered, func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		var secGrp *iaas.SecurityGroup
		var testKey TestSSHKey
		var vmArgs lib.CreateVMArgs
		var vmID string
		var pubIP *iaas.PublicIp
		BeforeAll(func() {
			var err error
			testCPI = conf.Qa.DefaultTestCPI()
			testKey = conf.Qa.GenerateSSHKey()
			secGrp = conf.Qa.GenerateTestSecurityGroup()
			iaasSDKClient = conf.Qa.GetStackitIaasClient()
			testCPI.Config.DefaultSSHKeyName = testKey.Name

			vmArgs = conf.Qa.DefaultCreateVMArgs(secGrp)
			pubIP, err = iaasSDKClient.DefaultAPI.CreatePublicIP(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID).CreatePublicIPPayload(iaas.CreatePublicIPPayload{AdditionalProperties: pubIpTags}).Execute()
			Expect(err).ToNot(HaveOccurred())
		})
		AfterAll(func() {
			// the last test should've deleted the vm. But in case we didn't get there, we should double check
			vm, err := testCPI.SClient.GetVM(vmID)
			// check if it's already gone
			if err != nil {
				if !lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
					Expect(testCPI.DeleteServer(vm)).To(Succeed())
				}
			}
			CleanupVM(vmID, *testCPI)
			testKey.Delete()
			Eventually(func() error {
				err = iaasSDKClient.DefaultAPI.DeletePublicIP(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, pubIP.GetId()).Execute()
				if err != nil && lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
					return nil
				}
				return err
			}, "2m").Should(Succeed())
			CleanupSecGrp(secGrp, testCPI, iaasSDKClient)
		})

		It("creates VMs with static public_ips", func() {
			vmArgs.Networks["vip"] = lib.Network{
				Network: settings.Network{
					Type: "vip",
					IP:   pubIP.GetIp(),
				},
			}
			req := GenerateRequest(lib.CreateVM, vmArgs)
			resp, err := testCPI.CreateVM(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Result).ToNot(BeEmpty())
			resultArray := resp.Result.([]any)
			vmID = resultArray[0].(string)
			Expect(vmID).ToNot(BeEmpty())

			networksMap := resultArray[1].(lib.Networks)
			netID := networksMap[DefaultNetName].Properties.NetID

			Expect(netID).To(Equal(vmArgs.Networks[DefaultNetName].Properties.NetID))

			Eventually(func() error {
				connection, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", pubIP.GetIp()), 30*time.Second)
				if err != nil {
					return nil
				}
				defer connection.Close()
				Expect(connection).ToNot(BeNil())
				return err
			}).Should(Succeed())
			vm, err := testCPI.SClient.GetVM(vmID)
			Expect(err).ToNot(HaveOccurred())
			userDataString, ok := vm.GetUserDataOk()
			Expect(ok).To(BeTrue())

			userDataBytes, err := base64.StdEncoding.DecodeString(*userDataString)
			Expect(err).ToNot(HaveOccurred())
			settings := map[string]any{}
			json.Unmarshal(userDataBytes, &settings)
			networkSettings, ok := settings["networks"].(map[string]any)
			Expect(ok).To(BeTrue())

			vipNetwork, ok := networkSettings[ExternalNetName].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(vipNetwork["ip"].(string)).To(Equal(pubIP.GetIp()))
			Expect(vipNetwork["type"].(string)).To(Equal("vip"))
			Expect(vipNetwork["use_dhcp"]).To(BeTrue())

			mgmtNetwork, ok := networkSettings[DefaultNetName].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(mgmtNetwork["mac"]).ToNot(BeEmpty())
			Expect(mgmtNetwork["type"].(string)).To(Equal("dynamic"))
			Expect(mgmtNetwork["use_dhcp"]).To(BeTrue())
		})
		It("keeps static public ips on vm deletion", func() {
			// double check the ordered describe is working ( meaning we have an existing vmID from the previous test )
			Expect(vmID).ToNot(BeEmpty())
			args := lib.DeleteVMArgs{
				ServerId: vmID,
			}
			req := GenerateRequest(lib.DeleteVM, args)

			_, err := testCPI.DeleteVM(req)
			Expect(err).ToNot(HaveOccurred())

			_, err = iaasSDKClient.DefaultAPI.GetPublicIP(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, pubIP.GetId()).Execute()
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Describe("a vip network is provided", Ordered, func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		var secGrp *iaas.SecurityGroup
		var testKey TestSSHKey
		var vmArgs lib.CreateVMArgs
		var vmID string
		BeforeAll(func() {
			testCPI = conf.Qa.DefaultTestCPI()
			testKey = conf.Qa.GenerateSSHKey()
			secGrp = conf.Qa.GenerateTestSecurityGroup()
			iaasSDKClient = conf.Qa.GetStackitIaasClient()
			testCPI.Config.DefaultSSHKeyName = testKey.Name

			vmArgs = conf.Qa.DefaultCreateVMArgs(secGrp)
		})
		AfterAll(func() {
			CleanupVM(vmID, *testCPI)
			CleanupSecGrp(secGrp, testCPI, iaasSDKClient)
			testKey.Delete()
		})
		It("creates VMs with dynamic public_ips", func() {
			vmArgs.Networks["vip"] = lib.Network{
				Network: settings.Network{
					Type: "vip",
				},
			}
			req := GenerateRequest(lib.CreateVM, vmArgs)
			resp, err := testCPI.CreateVM(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Result).ToNot(BeEmpty())
			resultArray := resp.Result.([]any)
			vmID = resultArray[0].(string)
			Expect(vmID).ToNot(BeEmpty())

			networks := resultArray[1].(lib.Networks)
			netID := networks[DefaultNetName].Properties.NetID

			Expect(netID).To(Equal(vmArgs.Networks[DefaultNetName].Properties.NetID))

			vm, err := iaasSDKClient.DefaultAPI.GetServer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, vmID).Details(true).Execute()

			Expect(err).ToNot(HaveOccurred())
			nics := vm.GetNics()
			Expect(nics).To(HaveLen(1))
			Expect(nics[0].GetPublicIp()).ToNot(BeEmpty())

			// retry to adjust for flakes
			var connection net.Conn
			for range 3 {
				connection, err = net.DialTimeout("tcp", fmt.Sprintf("%s:22", nics[0].GetPublicIp()), 30*time.Second)
				if err == nil {
					break
				}
				lib.JitterWait(5000)
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(connection).ToNot(BeNil())
			defer connection.Close()
		})
		It("deletes dynamic public ips on vm deletion", func() {
			// double check the ordered describe is working ( meaning we have an existing vmID from the previous test )
			Expect(vmID).ToNot(BeEmpty())
			args := lib.DeleteVMArgs{
				ServerId: vmID,
			}
			req := GenerateRequest(lib.DeleteVM, args)

			vm, err := iaasSDKClient.DefaultAPI.GetServer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, vmID).Details(true).Execute()

			Expect(err).ToNot(HaveOccurred())
			nics := vm.GetNics()
			Expect(nics).To(HaveLen(1))
			Expect(nics[0].GetPublicIp()).ToNot(BeEmpty())

			pubIP, err := testCPI.SClient.GetPublicIPByAddr(nics[0].GetPublicIp())
			Expect(err).ToNot(HaveOccurred())

			_, err = testCPI.DeleteVM(req)
			Expect(err).ToNot(HaveOccurred())

			_, err = iaasSDKClient.DefaultAPI.GetPublicIP(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, pubIP.GetId()).Execute()
			Expect(lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound)).To(BeTrue())
		})
	})
	Describe("targetting a default stackit stage", func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		var secGrp *iaas.SecurityGroup
		var testKey TestSSHKey
		var vmArgs lib.CreateVMArgs
		var vm *iaas.Server
		BeforeEach(func() {
			testCPI = conf.Prod.DefaultTestCPI()
			testKey = conf.Prod.GenerateSSHKey()
			secGrp = conf.Prod.GenerateTestSecurityGroup()
			iaasSDKClient = conf.Prod.GetStackitIaasClient()
			testCPI.Config.DefaultSSHKeyName = testKey.Name
			vmArgs = conf.Prod.DefaultCreateVMArgs(secGrp)
		})

		AfterEach(func() {
			if vm != nil {
				CleanupVM(vm.GetId(), *testCPI)
			}
			CleanupSecGrp(secGrp, testCPI, iaasSDKClient)
			testKey.Delete()
		})

		Describe("root_disk size is provided", func() {
			It("has the expected root disk size", func() {
				vmArgs.Properties.RootDisk.Size = 64
				vmArgs.Properties.SecurityGroups = []string{secGrp.GetId()}
				vmArgs.Env.Bosh.Mbus.URLs = []string{"http://user:test@0.0.0.0:6868"}
				vmArgs.Env.Bosh.NTP = testCPI.Config.NTPConfig

				req := GenerateRequest(lib.CreateVM, vmArgs)
				resp, err := testCPI.CreateVM(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Result).ToNot(BeEmpty())
				resultArray := resp.Result.([]any)
				vmID := resultArray[0].(string)

				vm, err = iaasSDKClient.DefaultAPI.GetServer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, vmID).Details(true).Execute()
				Expect(err).ToNot(HaveOccurred())

				Expect(vmID).ToNot(BeEmpty())
				networks := resultArray[1].(lib.Networks)
				vmIP := networks[DefaultNetName].IP
				ssh := conf.Prod.SSHTunnelConnection(vmIP, testKey.PKPath)
				session, err := ssh.NewSession()
				Expect(err).ToNot(HaveOccurred())
				out, err := session.Output("lsblk -b --output SIZE -n -d /dev/vda")
				Expect(err).ToNot(HaveOccurred())

				expectedSize := vmArgs.Properties.RootDisk.Size * 1024 * 1024 * 1024 // kb -> mb -> gb
				Expect(string(out)).To(ContainSubstring(fmt.Sprint(expectedSize)))
			})
		})
	})

	Describe("targetting a non default stackit stage", func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		var secGrp *iaas.SecurityGroup
		var testKey TestSSHKey
		var vmArgs lib.CreateVMArgs
		var vm *iaas.Server
		BeforeEach(func() {
			testCPI = conf.Qa.DefaultTestCPI()
			testKey = conf.Qa.GenerateSSHKey()
			secGrp = conf.Qa.GenerateTestSecurityGroup()
			iaasSDKClient = conf.Qa.GetStackitIaasClient()
			testCPI.Config.DefaultSSHKeyName = testKey.Name
			vmArgs = conf.Qa.DefaultCreateVMArgs(secGrp)
		})

		AfterEach(func() {
			if vm != nil {
				CleanupVM(vm.GetId(), *testCPI)
			}
			CleanupSecGrp(secGrp, testCPI, iaasSDKClient)
			testKey.Delete()
		})

		Describe("root_disk size is provided", func() {
			It("has the expected root disk size", func() {
				vmArgs.Properties.RootDisk.Size = 64
				vmArgs.Properties.SecurityGroups = []string{secGrp.GetId()}
				vmArgs.Env.Bosh.Mbus.URLs = []string{"http://user:test@0.0.0.0:6868"}
				vmArgs.Env.Bosh.NTP = testCPI.Config.NTPConfig

				req := GenerateRequest(lib.CreateVM, vmArgs)
				resp, err := testCPI.CreateVM(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Result).ToNot(BeEmpty())
				resultArray := resp.Result.([]any)
				vmID := resultArray[0].(string)
				Expect(vmID).ToNot(BeEmpty())

				vm, err = iaasSDKClient.DefaultAPI.GetServer(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, vmID).Details(true).Execute()
				Expect(err).ToNot(HaveOccurred())

				networks := resultArray[1].(lib.Networks)
				vmIP := networks[DefaultNetName].IP

				ssh := conf.Qa.SSHTunnelConnection(vmIP, testKey.PKPath)
				session, err := ssh.NewSession()
				Expect(err).ToNot(HaveOccurred())
				out, err := session.Output("lsblk -b --output SIZE -n -d /dev/vda")
				Expect(err).ToNot(HaveOccurred())

				expectedSize := vmArgs.Properties.RootDisk.Size * 1024 * 1024 * 1024 // kb -> mb -> gb
				Expect(string(out)).To(ContainSubstring(fmt.Sprint(expectedSize)))
				Expect(testCPI.DeleteServer(vm)).To(Succeed())
				testCPI.AttemptToCleanOrphanedNICS(lib.NicData{Networks: vm.GetNics()})
			})
		})
	})
	Describe("using security_groups", func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		var secGrp *iaas.SecurityGroup
		var testKey TestSSHKey
		var vmArgs lib.CreateVMArgs
		var vm *iaas.Server
		var testSecGrps []*iaas.SecurityGroup
		const (
			networkField = "network"
			vmField      = "vm"
			mixedFields  = "all"
		)
		BeforeEach(func() {
			testCPI = conf.Prod.DefaultTestCPI()
			testKey = conf.Prod.GenerateSSHKey()
			secGrp = conf.Prod.GenerateTestSecurityGroup()
			testSecGrps = append(testSecGrps, secGrp)
			iaasSDKClient = conf.Prod.GetStackitIaasClient()
			testCPI.Config.DefaultSSHKeyName = testKey.Name

			vmArgs = conf.Prod.DefaultCreateVMArgs(secGrp)
		})
		runTest := func(specifyIn string, useName bool) bool {
			secGrp = conf.Prod.GenerateTestSecurityGroup()
			testSecGrps = append(testSecGrps, secGrp)
			var secGrp1, secGrp2, secGrp3, secGrp4, secGrp5 *iaas.SecurityGroup
			switch specifyIn {
			case mixedFields:
				secGrp1 = conf.Prod.GenerateTestSecurityGroup()
				testSecGrps = append(testSecGrps, secGrp1)
				secGrp2 = conf.Prod.GenerateTestSecurityGroup()
				testSecGrps = append(testSecGrps, secGrp2)
				secGrp3 = conf.Prod.GenerateTestSecurityGroup()
				testSecGrps = append(testSecGrps, secGrp3)
				secGrp4 = conf.Prod.GenerateTestSecurityGroup()
				testSecGrps = append(testSecGrps, secGrp4)
				secGrp5 = conf.Prod.GenerateTestSecurityGroup()
				testSecGrps = append(testSecGrps, secGrp5)
				testCPI.Config.DefaultSecurityGroups = []string{secGrp.GetName(), secGrp5.GetId()}

				for name, net := range vmArgs.Networks {
					net.Properties.SecurityGroups = []string{secGrp1.GetName(), secGrp2.GetId()}
					vmArgs.Networks[name] = net
				}
				vmArgs.Properties.SecurityGroups = []string{secGrp3.GetName(), secGrp4.GetId()}

			case networkField:

				vmArgs.Properties.SecurityGroups = []string{}

				for name, net := range vmArgs.Networks {
					if useName {
						net.Properties.SecurityGroups = []string{secGrp.GetName()}
					} else {
						net.Properties.SecurityGroups = []string{secGrp.GetId()}
					}
					vmArgs.Networks[name] = net
				}
			case vmField:
				for name, net := range vmArgs.Networks {
					net.Properties.SecurityGroups = []string{}
					vmArgs.Networks[name] = net
				}
				if useName {
					vmArgs.Properties.SecurityGroups = []string{secGrp.GetName()}
				} else {
					vmArgs.Properties.SecurityGroups = []string{secGrp.GetId()}
				}
			default:
				// shouldn't be here.
				Expect(true).To(BeFalse())
			}

			req := GenerateRequest(lib.CreateVM, vmArgs)
			resp, err := testCPI.CreateVM(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Result).ToNot(BeEmpty())
			resultArray := resp.Result.([]any)
			vmID := resultArray[0].(string)
			Expect(vmID).ToNot(BeEmpty())
			vm, err = testCPI.SClient.GetVM(vmID)
			Expect(err).ToNot(HaveOccurred())
			networks := vm.GetNics()

			secGrps, err := testCPI.ResolveSecurityGroups([]string{"default"})
			Expect(err).ToNot(HaveOccurred())
			defaultSecGrpID := secGrps[0]
			for _, net := range networks {
				Expect(net.GetSecurityGroups()).ToNot(ContainElement(defaultSecGrpID))
				Expect(net.GetSecurityGroups()).To(ContainElement(secGrp.GetId()))
				if specifyIn == mixedFields {
					Expect(net.GetSecurityGroups()).To(ContainElement(secGrp.GetId()))
					Expect(net.GetSecurityGroups()).To(ContainElement(secGrp1.GetId()))
					Expect(net.GetSecurityGroups()).To(ContainElement(secGrp2.GetId()))
					Expect(net.GetSecurityGroups()).To(ContainElement(secGrp3.GetId()))
					Expect(net.GetSecurityGroups()).To(ContainElement(secGrp4.GetId()))
					Expect(net.GetSecurityGroups()).To(ContainElement(secGrp5.GetId()))
				}
			}

			return true
		}
		AfterEach(func() {
			Eventually(testCPI.DeleteServer(vm), "2m").Should(Succeed())
			testCPI.AttemptToCleanOrphanedNICS(lib.NicData{Networks: vm.GetNics()})
			for _, sg := range testSecGrps {
				CleanupSecGrp(sg, testCPI, iaasSDKClient)
			}
			testKey.Delete()
		})
		Describe("mixing all sources for sec grp configs with names and ids", func() {
			It("will resolve and assign all sec grps from all sources", func() {
				Expect(runTest(mixedFields, true)).To(BeTrue())
			})
		})
		Describe("provided via default_security_groups", func() {
			It("works using IDs", func() {
				Expect(runTest(vmField, false)).To(BeTrue())
			})
			It("works using Names", func() {
				Expect(runTest(vmField, true)).To(BeTrue())
			})
		})
		Describe("provided via networks", func() {
			It("works using IDs", func() {
				Expect(runTest(networkField, false)).To(BeTrue())
			})
			It("works using Names", func() {
				Expect(runTest(networkField, true)).To(BeTrue())
			})
		})
	})
})

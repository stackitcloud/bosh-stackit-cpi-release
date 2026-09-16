package integration

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi"
	. "github.com/stackitcloud/stackit-cpi/cpi/integration/helpers"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

var _ = Describe("Disks", func() {
	Describe("max disk size", Ordered, func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		var testKey TestSSHKey
		var vmArgs lib.CreateVMArgs
		var diskArgs lib.CreateDiskArgs
		var diskID, vmID string
		var testSecGrp *iaas.SecurityGroup
		BeforeAll(func() {
			testCPI = conf.Prod.DefaultTestCPI()
			testKey = conf.Prod.GenerateSSHKey()
			testSecGrp = conf.Prod.GenerateTestSecurityGroup()
			testCPI.Config.DefaultSSHKeyName = testKey.Name
			iaasSDKClient = conf.Prod.GetStackitIaasClient()
			vmArgs = conf.Prod.DefaultCreateVMArgs(testSecGrp)
			req := GenerateRequest(lib.CreateVM, vmArgs)
			resp, err := testCPI.CreateVM(req)
			Expect(err).ToNot(HaveOccurred())
			resultList := resp.Result.([]any)
			vmID = resultList[0].(string)
		})
		AfterAll(func() {
			iaasSDKClient = conf.Prod.GetStackitIaasClient()
			CleanupVM(vmID, *testCPI)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			Expect(iaasSDKClient.DefaultAPI.DeleteVolume(ctx, testCPI.Config.ProjectID, testCPI.Config.RegionID, diskID).Execute()).To(Succeed())
			CleanupSecGrp(testSecGrp, testCPI, iaasSDKClient)
			testKey.Delete()
		})

		It("succeeds creating disks ", func() {
			diskArgs = lib.CreateDiskArgs{
				Size: lib.MaxDiskSizeMB / 2,
				Properties: map[string]any{
					"type": "storage_premium_perf10",
				},
				ServerId: vmID,
			}
			req := GenerateRequest(lib.CreateDisk, diskArgs)
			resp, err := testCPI.CreateDisk(req)
			Expect(err).ToNot(HaveOccurred())
			diskID = resp.Result.(string)
			Expect(diskID).ToNot(BeEmpty())
		})
		It("fails resizing the disk to a larger than allowed size", func() {
			resizeArgs := lib.ResizeDiskArgs{
				DiskId:  diskID,
				NewSize: lib.MaxDiskSizeMB + 1,
			}
			req := GenerateRequest(lib.ResizeDisk, resizeArgs)
			_, err := testCPI.ResizeDisk(req)
			Expect(err).To(MatchError(lib.ErrDiskSizeTooLarge))
		})
		It("succeeds resizing the disk within the MaxSizeLimit ", func() {
			resizeArgs := lib.ResizeDiskArgs{
				DiskId:  diskID,
				NewSize: lib.MaxDiskSizeMB,
			}
			req := GenerateRequest(lib.ResizeDisk, resizeArgs)
			_, err := testCPI.ResizeDisk(req)
			Expect(err).ToNot(HaveOccurred())
		})
		It("fails shrinking the disk", func() {
			resizeArgs := lib.ResizeDiskArgs{
				DiskId:  diskID,
				NewSize: lib.MaxDiskSizeMB / 2,
			}
			req := GenerateRequest(lib.ResizeDisk, resizeArgs)
			_, err := testCPI.ResizeDisk(req)
			// iaas limitation, that's an upstream error
			Expect(err).To(MatchError(ContainSubstring("new volume size must be greater than current size")))
		})
		It("fails creating disks larger than the allowed size", func() {
			diskArgs = lib.CreateDiskArgs{
				Size: lib.MaxDiskSizeMB + 1,
				Properties: map[string]any{
					"type": "storage_premium_perf10",
				},
				ServerId: vmID,
			}
			req := GenerateRequest(lib.CreateDisk, diskArgs)
			_, err := testCPI.CreateDisk(req)
			Expect(err).To(MatchError(lib.ErrDiskSizeTooLarge))
		})
		It("fails with disks larger than the allowed size", func() {
			diskArgs = lib.CreateDiskArgs{
				Size: lib.MaxDiskSizeMB + 1,
				Properties: map[string]any{
					"type": "storage_premium_perf10",
				},
				ServerId: vmID,
			}
			req := GenerateRequest(lib.CreateDisk, diskArgs)
			_, err := testCPI.CreateDisk(req)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(lib.ErrDiskSizeTooLarge))
		})
	})
	Describe("has_disk", func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		BeforeEach(func() {
			testCPI = conf.Prod.DefaultTestCPI()
			iaasSDKClient = conf.Prod.GetStackitIaasClient()
		})
		Describe("a disk exists", func() {
			var (
				disk *iaas.Volume
				err  error
			)
			BeforeEach(func() {
				disk, err = iaasSDKClient.DefaultAPI.CreateVolume(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID).CreateVolumePayload(iaas.CreateVolumePayload{
					AvailabilityZone: fmt.Sprintf("%s-1", conf.Prod.Region),
					Name:             utils.Ptr(fmt.Sprintf("test-%s", uuid.NewString())),
					Size:             utils.Ptr(int64(10)),
				}).Execute()
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				Eventually(iaasSDKClient.DefaultAPI.DeleteVolume(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, disk.GetId()).Execute, "2m", "5s").Should(Succeed())
			})
			It("will return a result with exists set to true", func() {
				req := GenerateRequest(lib.HasDisk, []string{disk.GetId()})

				resp, err := testCPI.HasDisk(req)

				Expect(err).ToNot(HaveOccurred())

				exists := resp.Result.(bool)

				Expect(exists).To(BeTrue())
			})
		})
		Describe("a disk doesn't exist", func() {
			It("will return a result with exists set to false", func() {
				// generate a fake ID which should not be known to the IaaS
				req := GenerateRequest(lib.HasDisk, []string{uuid.NewString()})

				resp, err := testCPI.HasDisk(req)

				Expect(err).ToNot(HaveOccurred())

				exists := resp.Result.(bool)

				Expect(exists).To(BeFalse())
			})
		})
	})
	Describe("disk lifecycle", Ordered, func() {
		var testCPI *cpi.CPI
		var iaasSDKClient *iaas.APIClient
		var secGrp *iaas.SecurityGroup
		var testKey TestSSHKey
		var vmArgs lib.CreateVMArgs
		var diskArgs lib.CreateDiskArgs
		var vmID, diskID, vmIP, diskHintPath string
		var outRealpath []byte
		BeforeAll(func() {
			testCPI = conf.Prod.DefaultTestCPI()
			testKey = conf.Prod.GenerateSSHKey()
			secGrp = conf.Prod.GenerateTestSecurityGroup()
			iaasSDKClient = conf.Prod.GetStackitIaasClient()
			testCPI.Config.DefaultSSHKeyName = testKey.Name

			vmArgs = conf.Prod.DefaultCreateVMArgs(secGrp)
			req := GenerateRequest(lib.CreateVM, vmArgs)
			resp, err := testCPI.CreateVM(req)
			Expect(err).ToNot(HaveOccurred())
			resultList := resp.Result.([]any)
			vmID = resultList[0].(string)
			Expect(vmID).ToNot(BeEmpty())
			networks := resultList[1].(lib.Networks)
			vmIP = networks[DefaultNetName].IP

			diskArgs = lib.CreateDiskArgs{
				Size: 10240,
				Properties: map[string]any{
					"type": "storage_premium_perf10",
				},
				ServerId: vmID,
			}
			req = GenerateRequest(lib.CreateDisk, diskArgs)
			resp, err = testCPI.CreateDisk(req)
			Expect(err).ToNot(HaveOccurred())
			diskID = resp.Result.(string)
			Expect(diskID).ToNot(BeEmpty())
		})
		AfterAll(func() {
			CleanupVM(vmID, *testCPI)
			Eventually(iaasSDKClient.DefaultAPI.DeleteVolume(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, diskID).Execute, "60s").Should(Succeed())
			testKey.Delete()
			CleanupSecGrp(secGrp, testCPI, iaasSDKClient)
		})
		When("getting disks on a fresh vm without attachments", func() {
			It("will return no disks", func() {
				req := GenerateRequest(lib.GetDisks, []string{vmID})
				resp, err := testCPI.GetDisks(req)

				Expect(err).ToNot(HaveOccurred())
				result := resp.Result.([]string)
				Expect(result).To(BeEmpty())
				ssh := conf.Prod.SSHTunnelConnection(vmIP, testKey.PKPath)
				session, err := ssh.NewSession()
				Expect(err).ToNot(HaveOccurred())
				out, err := session.Output("ls -la /dev; ip addr;")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(out)).ToNot(BeEmpty())
			})
		})
		When("attaching a disk", func() {
			It("will attach a disk to the vm", FlakeAttempts(3), func() {
				req := GenerateRequest(lib.AttachDisk, []string{vmID, diskID})
				resp, err := testCPI.AttachDisk(req)

				Expect(err).ToNot(HaveOccurred())
				result := resp.Result.(map[string]any)
				Expect(result["path"].(string)).To(Equal(fmt.Sprintf("/dev/disk/by-id/virtio-%s", diskID[:20])))
				Expect(result["volume_id"].(string)).To(Equal(fmt.Sprintf("/dev/disk/by-id/virtio-%s", diskID[:20])))
				diskHintPath = result["path"].(string)
			})
		})
		When("attaching an already attached disk", func() {
			It("does not error", FlakeAttempts(3), func() {
				req := GenerateRequest(lib.AttachDisk, []string{vmID, diskID})
				resp, err := testCPI.AttachDisk(req)

				Expect(err).ToNot(HaveOccurred())
				result := resp.Result.(map[string]any)
				Expect(result["path"].(string)).To(Equal(fmt.Sprintf("/dev/disk/by-id/virtio-%s", diskID[:20])))
				Expect(result["volume_id"].(string)).To(Equal(fmt.Sprintf("/dev/disk/by-id/virtio-%s", diskID[:20])))
				diskHintPath = result["path"].(string)
			})
		})
		When("getting disks again", func() {
			It("should return exactly one disk", func() {
				req := GenerateRequest(lib.GetDisks, []string{vmID})
				resp, err := testCPI.GetDisks(req)

				Expect(err).ToNot(HaveOccurred())
				result := resp.Result.([]string)
				Expect(result).To(HaveLen(1))
			})
		})
		When("checking the disk_hint on the vm", func() {
			It("should be pointing to an existing device", func() {
				Expect(diskHintPath).ToNot(BeEmpty())
				ssh := conf.Prod.SSHTunnelConnection(vmIP, testKey.PKPath)
				session, err := ssh.NewSession()
				Expect(err).ToNot(HaveOccurred())
				fmt.Printf("diskhintpath '%s'\n", diskHintPath)
				// Uses disk hint to check on which device the volume was provided
				outRealpath, err = session.Output(fmt.Sprintf("realpath %s", diskHintPath))

				Expect(string(outRealpath)).To(MatchRegexp("/dev/vd."))
				Expect(err).ToNot(HaveOccurred())
			})
		})
		When("checking the disk size", func() {
			It("should match the configured disk size", func() {
				ssh := conf.Prod.SSHTunnelConnection(vmIP, testKey.PKPath)
				session, err := ssh.NewSession()
				Expect(err).ToNot(HaveOccurred())
				out, err := session.Output(fmt.Sprintf("lsblk -b --output SIZE -n -d %s", string(outRealpath)))
				Expect(err).ToNot(HaveOccurred())

				expectedSize := diskArgs.Size * 1024 * 1024 // MB -> GB
				Expect(string(out)).To(ContainSubstring(fmt.Sprint(expectedSize)))
			})
		})
		When("detach disk is called", func() {
			It("should work", func() {
				req := GenerateRequest(lib.AttachDisk, []string{vmID, diskID})
				_, err := testCPI.DetachDisk(req)
				Expect(err).ToNot(HaveOccurred())
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()

				_, err = iaasSDKClient.DefaultAPI.GetAttachedVolume(ctx, testCPI.Config.ProjectID, testCPI.Config.RegionID, vmID, diskID).Execute()
				Expect(lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound)).To(BeTrue())
			})
		})
		When("detach disk is called again", func() {
			It("should not error", func() {
				req := GenerateRequest(lib.AttachDisk, []string{vmID, diskID})
				_, err := testCPI.DetachDisk(req)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

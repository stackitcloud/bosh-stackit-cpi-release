package lib

import (
	"fmt"
	"math"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

func NewRetryableStackitClient(client IaasClient, retries int, logger LoggerInterface, opts ...RetryOpt) RetryableClient {
	r := RetryableClient{
		c:         client,
		attempts:  retries + 1,
		logger:    logger,
		delayBase: 10000,
	}

	for _, f := range opts {
		f(&r)
	}
	return r
}

type (
	RetryOpt        func(r *RetryableClient)
	RetryableClient struct {
		c         IaasClient
		attempts  int
		logger    LoggerInterface
		delayBase int
	}
)

func WithDelayBaseInSeconds(delay int) RetryOpt {
	return func(r *RetryableClient) {
		r.delayBase = delay * 1000
	}
}

func retry(f func() error, delayRange, attempts int, logger LoggerInterface, specificErrors ...func(e error) bool) (err error) {
attemptRun:
	for attempt := range attempts {
		logger.Debugf("starting attempt: %d/%d", attempt+1, attempts)
		if attempt > 0 {
			delayBase := float64(delayRange) * math.Pow(float64(attempt), float64(attempt))
			logger.Debug("delaying api call")

			actual := JitterWait(int64(delayBase))
			logger.Debugf("waited for: %d seconds", actual/1000)
		}
		err = f()
		if err == nil {
			logger.Debugf("attempt: %d/%d succeeded", attempt+1, attempts)
			break
		}
		if IsRetryableOpenAPIError(err) {
			logger.Debugf("failed on attempt: %d/%d with retryable error: %s", attempt+1, attempts, err)
			continue
		}
		for _, check := range specificErrors {
			if check(err) {
				logger.Debugf("failed on attempt: %d/%d with specific retryable error: %s", attempt+1, attempts, err)
				continue attemptRun
			}
		}
		logger.Debugf("attempt %d/%d has non retryable failure: %s", attempt+1, attempts, err)
		return err
	}
	if err != nil {
		logger.Debugf("finally failed after %d attempts: %s", attempts, err)
	}
	return err
}

func (r RetryableClient) CreateNetwork(createNetworkPayload iaas.CreateNetworkPayload) (net *iaas.Network, err error) {
	err = retry(func() (err error) {
		net, err = r.c.CreateNetwork(createNetworkPayload)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForCreateNetwork(networkID string) (net *iaas.Network, err error) {
	err = retry(func() (err error) {
		net, err = r.c.WaitForCreateNetwork(networkID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetNetwork(networkID string) (net *iaas.Network, err error) {
	err = retry(func() (err error) {
		net, err = r.c.GetNetwork(networkID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) DeleteNetwork(networkID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.DeleteNetwork(networkID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForDeleteNetwork(networkID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.WaitForDeleteNetwork(networkID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetVM(vmID string) (server *iaas.Server, err error) {
	err = retry(func() (err error) {
		server, err = r.c.GetVM(vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) CreateVM(payload iaas.CreateServerPayload) (server *iaas.Server, err error) {
	err = retry(func() (err error) {
		server, err = r.c.CreateVM(payload)
		// we can retry this after deleting the partially created VM
		if IsNonGenericBuildAbortedNetworkError(err) {
			r.logger.Warnf("detected build aborted network not allocated error")
			delErr := r.c.DeleteVM(server.GetId())
			if delErr != nil {
				r.logger.Warn(fmt.Sprintf("failed deleting build aborted vm: %s", delErr), "server_id", server.GetId())
			}
		}
		return err
	}, r.delayBase, r.attempts, r.logger, IsNonGenericBuildAbortedNetworkError)
	return
}

func (r RetryableClient) UpdateVM(vmID string, payload iaas.UpdateServerPayload) (server *iaas.Server, err error) {
	err = retry(func() (err error) {
		server, err = r.c.UpdateVM(vmID, payload)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) DeleteVM(vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.DeleteVM(vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForVM(vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.WaitForVM(vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForVMDeletion(vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.WaitForVMDeletion(vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) RebootVM(vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.RebootVM(vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetAZForServer(vmID string) (az string, err error) {
	err = retry(func() (err error) {
		az, err = r.c.GetAZForServer(vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) ListNics(networkID string) (nics []iaas.NIC, err error) {
	err = retry(func() (err error) {
		nics, err = r.c.ListNics(networkID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) CreateNic(createNicPayload iaas.CreateNicPayload, networkID string) (nic *iaas.NIC, err error) {
	err = retry(func() (err error) {
		nic, err = r.c.CreateNic(createNicPayload, networkID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) DeleteNic(nicID, networkID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.DeleteNic(nicID, networkID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetNic(nicID, networkID string) (nic *iaas.NIC, err error) {
	err = retry(func() (err error) {
		nic, err = r.c.GetNic(nicID, networkID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) CreateDisk(createVolumePayload iaas.CreateVolumePayload) (volume *iaas.Volume, err error) {
	err = retry(func() (err error) {
		volume, err = r.c.CreateDisk(createVolumePayload)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForCreateDisk(volumeID string) (volume *iaas.Volume, err error) {
	err = retry(func() (err error) {
		volume, err = r.c.WaitForCreateDisk(volumeID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) DeleteDisk(volumeID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.DeleteDisk(volumeID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetDisk(volumeID string) (volume *iaas.Volume, err error) {
	err = retry(func() (err error) {
		volume, err = r.c.GetDisk(volumeID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForDeleteDisk(volumeID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.WaitForDeleteDisk(volumeID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) AttachDiskToVM(volumeID, vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.AttachDiskToVM(volumeID, vmID)
		return err
	}, r.delayBase, r.attempts, r.logger)
	return err
}

func (r RetryableClient) WaitForAttachDiskToVM(volumeID, vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.WaitForAttachDiskToVM(volumeID, vmID)
		return err
	}, r.delayBase, r.attempts, r.logger)
	return err
}

func (r RetryableClient) DetachDiskFromVM(volumeID, vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.DetachDiskFromVM(volumeID, vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForDetachDiskFromVM(volumeID, vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.WaitForDetachDiskFromVM(volumeID, vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) ResizeDisk(volumeID string, resizeVolumePayload iaas.ResizeVolumePayload) (err error) {
	err = retry(func() (err error) {
		err = r.c.ResizeDisk(volumeID, resizeVolumePayload)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) UpdateDisk(volumeID string, updateVolumePayload iaas.UpdateVolumePayload) (err error) {
	err = retry(func() (err error) {
		err = r.c.UpdateDisk(volumeID, updateVolumePayload)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) AttachSecGrpToVM(secGrp, vmID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.AttachSecGrpToVM(secGrp, vmID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) ListSecurityGroups() (secGrps []iaas.SecurityGroup, err error) {
	err = retry(func() (err error) {
		secGrps, err = r.c.ListSecurityGroups()
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) ListPublicIPs() (pubIPs []iaas.PublicIp, err error) {
	err = retry(func() (err error) {
		pubIPs, err = r.c.ListPublicIPs()
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) UpdatePublicIP(pubIPPayload iaas.UpdatePublicIPPayload, pubIPID string) (pubIP *iaas.PublicIp, err error) {
	err = retry(func() (err error) {
		pubIP, err = r.c.UpdatePublicIP(pubIPPayload, pubIPID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) CreatePublicIP(pubIPPayload iaas.CreatePublicIPPayload) (pubIP *iaas.PublicIp, err error) {
	err = retry(func() (err error) {
		pubIP, err = r.c.CreatePublicIP(pubIPPayload)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) DeletePublicIP(publicIPID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.DeletePublicIP(publicIPID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetPublicIPByAddr(publicIPAddr string) (pubIP *iaas.PublicIp, err error) {
	err = retry(func() (err error) {
		pubIP, err = r.c.GetPublicIPByAddr(publicIPAddr)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) CreateStemcell(createImagePayload iaas.CreateImagePayload) (image *iaas.ImageCreateResponse, err error) {
	err = retry(func() (err error) {
		image, err = r.c.CreateStemcell(createImagePayload)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetStemcell(imageID string) (image *iaas.Image, err error) {
	err = retry(func() (err error) {
		image, err = r.c.GetStemcell(imageID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForStemcellCreate(imageID string) (image *iaas.Image, err error) {
	err = retry(func() (err error) {
		image, err = r.c.WaitForStemcellCreate(imageID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) ListStemcells() (images []iaas.Image, err error) {
	err = retry(func() (err error) {
		images, err = r.c.ListStemcells()
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) DeleteStemcell(imageID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.DeleteStemcell(imageID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) WaitForStemcellDelete(imageID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.WaitForStemcellDelete(imageID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) CreateSnapshot(createSnaphotPayload *iaas.CreateSnapshotPayload) (snapshot *iaas.Snapshot, err error) {
	err = retry(func() (err error) {
		snapshot, err = r.c.CreateSnapshot(createSnaphotPayload)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) DeleteSnapshot(snapshotID string) (err error) {
	err = retry(func() (err error) {
		err = r.c.DeleteSnapshot(snapshotID)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetPublicKey(name string) (key string, err error) {
	err = retry(func() (err error) {
		key, err = r.c.GetPublicKey(name)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) ListMachineTypes() (machineTypes []iaas.MachineType, err error) {
	err = retry(func() (err error) {
		machineTypes, err = r.c.ListMachineTypes()
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) ListAZs() (azs []string, err error) {
	err = retry(func() (err error) {
		azs, err = r.c.ListAZs()
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) GetLB(name string) (lb *loadbalancer.LoadBalancer, err error) {
	err = retry(func() (err error) {
		lb, err = r.c.GetLB(name)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) AttachNICToLB(targetPoolProperties TargetPool, nic iaas.NIC, lb *loadbalancer.LoadBalancer) (err error) {
	err = retry(func() (err error) {
		err = r.c.AttachNICToLB(targetPoolProperties, nic, lb)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

func (r RetryableClient) DetachNICFromLB(nic iaas.NIC, lb *loadbalancer.LoadBalancer) (err error) {
	err = retry(func() (err error) {
		err = r.c.DetachNICFromLB(nic, lb)
		return
	}, r.delayBase, r.attempts, r.logger)
	return
}

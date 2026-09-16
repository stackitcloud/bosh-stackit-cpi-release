package lib

import (
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

// IaasClient interface defines the Public Methods for stackit api calls that the cpi expects to make.
// The interface is used to abstract away details of stackit sdk usage
//
//go:generate go tool counterfeiter . IaasClient
type IaasClient interface {
	// NETWORKS
	CreateNetwork(createNetworkPayload iaas.CreateNetworkPayload) (*iaas.Network, error)
	WaitForCreateNetwork(networkID string) (*iaas.Network, error)
	GetNetwork(networkID string) (*iaas.Network, error)
	DeleteNetwork(networkID string) error
	WaitForDeleteNetwork(networkID string) error

	// VMS
	GetVM(vmID string) (*iaas.Server, error)
	CreateVM(payload iaas.CreateServerPayload) (*iaas.Server, error)
	UpdateVM(vmID string, payload iaas.UpdateServerPayload) (*iaas.Server, error)
	DeleteVM(vmID string) error
	WaitForVM(vmID string) error
	WaitForVMDeletion(vmID string) error
	RebootVM(vmID string) error
	GetAZForServer(vmID string) (string, error)

	// NICS
	ListNics(networkID string) ([]iaas.NIC, error)
	CreateNic(createNicPayload iaas.CreateNicPayload, networkID string) (*iaas.NIC, error)
	DeleteNic(nicID, networkID string) error
	GetNic(networkID, netID string) (*iaas.NIC, error)

	// DISK
	CreateDisk(createVolumePayload iaas.CreateVolumePayload) (*iaas.Volume, error)
	WaitForCreateDisk(volumeID string) (*iaas.Volume, error)
	DeleteDisk(volumeID string) error
	GetDisk(volumeID string) (*iaas.Volume, error)
	WaitForDeleteDisk(volumeID string) error
	AttachDiskToVM(volumeID, vmID string) error
	WaitForAttachDiskToVM(volumeID, vmID string) error
	DetachDiskFromVM(volumeID, vmID string) error
	WaitForDetachDiskFromVM(volumeID, vmID string) error
	ResizeDisk(volumeID string, resizeVolumePayload iaas.ResizeVolumePayload) error
	UpdateDisk(volumeID string, updateVolumePayload iaas.UpdateVolumePayload) error

	// SECGRPS
	AttachSecGrpToVM(secGrp, vmID string) error
	ListSecurityGroups() ([]iaas.SecurityGroup, error)

	// PUBIPS
	ListPublicIPs() ([]iaas.PublicIp, error)
	UpdatePublicIP(pubIPPayload iaas.UpdatePublicIPPayload, pubIPID string) (*iaas.PublicIp, error)
	CreatePublicIP(pubIPPayload iaas.CreatePublicIPPayload) (*iaas.PublicIp, error)
	DeletePublicIP(publicIPID string) error
	GetPublicIPByAddr(publicIPAddr string) (*iaas.PublicIp, error)

	// IMAGES/STEMCELLS
	CreateStemcell(createImagePayload iaas.CreateImagePayload) (*iaas.ImageCreateResponse, error)
	GetStemcell(imageID string) (*iaas.Image, error)
	WaitForStemcellCreate(imageID string) (*iaas.Image, error)
	ListStemcells() ([]iaas.Image, error)
	DeleteStemcell(imageID string) error
	WaitForStemcellDelete(imageID string) error

	// SNAPSHOTS
	CreateSnapshot(createSnaphotPayload *iaas.CreateSnapshotPayload) (*iaas.Snapshot, error)
	DeleteSnapshot(snapshotID string) error

	// MISC
	GetPublicKey(name string) (string, error)
	ListMachineTypes() ([]iaas.MachineType, error)
	ListAZs() ([]string, error)

	// LOADBALANCERS
	GetLB(name string) (*loadbalancer.LoadBalancer, error)
	AttachNICToLB(targetPoolProperties TargetPool, nic iaas.NIC, lb *loadbalancer.LoadBalancer) error
	DetachNICFromLB(nic iaas.NIC, lb *loadbalancer.LoadBalancer) error
}

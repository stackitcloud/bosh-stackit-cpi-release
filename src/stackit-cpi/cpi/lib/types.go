package lib

import (
	agentsettings "github.com/cloudfoundry/bosh-agent/v2/settings"
)

type Stemcell struct {
	APIVersion int `json:"api_version"`
}
type VMContext struct {
	Stemcell Stemcell `json:"stemcell"`
}

/*
 * Stemcell Constructs
 */
type StemcellInfo struct {
	ID              string `json:"id,omitempty"`
	Path            string `json:"path,omitempty"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Infrastructure  string `json:"infrastructure"`
	Hypervisor      string `json:"hypervisor"`
	Disk            int    `json:"disk"`
	DiskFormat      string `json:"disk_format"`
	ContainerFormat string `json:"container_format"`
	OsType          string `json:"os_type"`
	OsDistro        string `json:"os_distro"`
	Architecture    string `json:"architecture"`
	AutoDiskConfig  bool   `json:"auto_disk_config"`
}

/*
 * Network Constructs
 */

type NetProperties struct {
	NetID          string   `json:"net_id"`
	SecurityGroups []string `json:"security_groups"`
	Nameservers    []string `json:"nameservers,omitempty"` // Nameservers for the network, if applicable
}

// VMProperties represents the Cloud Properties of a virtual machine
type VMProperties struct {
	AvailabilityZone string       `json:"availability_zone"`
	BootFromVolume   bool         `json:"boot_from_volume"`
	InstanceType     string       `json:"instance_type"`
	RootDisk         VMRootDisk   `json:"root_disk"`
	SecurityGroups   []string     `json:"security_groups,omitempty"` // Specifies the security groups associated with a vm_extension.
	LoadBalancer     LoadBalancer `json:"load_balancer"`
}
type LoadBalancer struct {
	Name        string       `json:"name"`
	TargetPools []TargetPool `json:"target_groups"`
}
type TargetPool struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

// VMRootDisk represents the root disk properties of a virtual machine
type VMRootDisk struct {
	Size             int64  `json:"size"`
	PerformanceClass string `json:"type,omitempty"` // Performance class for the root disk, if applicable
}

// Environment ironment represents the environment settings for a virtual machine
type Environment struct {
	Bosh   agentsettings.BoshEnv `json:"bosh"`
	Group  string                `json:"group"`
	Groups []string              `json:"groups"`
	Tags   map[string]string     `json:"tags"`
}

type DesiredInstanceSize struct {
	CPU               int64 `json:"cpu"`
	RAM               int64 `json:"ram"`                 // RAM is in MiB
	EphemeralDiskSize int64 `json:"ephemeral_disk_size"` // Size is in MB
}

/*
 * CPI Method Arguments Structs
 */
// CreateVMArguments represents the arguments required for the CreateVM method
type CreateVMArgs struct {
	AgentID    string       // The ID of the agent (BOSH Agent) for the VM
	StemcellID string       // The Cloud ID of the stemcell to use for the VM
	Properties VMProperties // Cloud properties for the VM, including availability zone, boot from volume, instance type, and root disk size
	Networks   Networks     // Network configuration for the VM, including default, manual, VIP, and dynamic networks
	DiskIds    []string     // List of Cloud IDs for disks to attach to the VM
	Env        Environment  // Environment settings for the VM, including BOSH configuration, group, groups, and tags
}
type Networks map[string]Network

type Network struct {
	agentsettings.Network
	Properties NetProperties `json:"cloud_properties"`
}

type HasVMArgs struct {
	ServerId string // The Cloud ID of the VM to check
}

type CalculateVMCloudPropertiesArgs struct {
	DesiredInstanceSize // Desired instance size for the CalculateVMCloudPropertiesArgs
}

// AttachDiskArgs represents the arguments required for the AttachDisk method
type AttachDiskArgs struct {
	ServerId string // The ID of the server (VM) to which the disk will be attached
	DiskId   string // The Cloud ID of the disk to attach
}

// DetachDiskArgs represents the arguments required for the DetachDisk method
type DetachDiskArgs struct {
	ServerId string // The ID of the server (VM) from which the disk will be detached
	DiskId   string // The Cloud ID of the disk to detach
}

// DeleteVMArgs represents the arguments required for the DeleteVM method
type DeleteDiskArgs struct {
	DiskId string // The Cloud ID of the disk to delete
}

type GetDisksArgs struct {
	ServerId string // The Cloud ID of the VM for which to get attached disks
}

type HasDiskArgs struct {
	DiskId string // The Cloud ID of the disk to check
}

// DeleteDiskArgs represents the arguments required for the DeleteDisk method
type DeleteStemcellArgs struct {
	StemcellId string // The Cloud ID of the stemcell to delete
}

// DeleteNetworkArgs represents the arguments required for the DeleteNetwork method
type DeleteNetworkArgs struct {
	NetworkId string // The Cloud ID of the network to delete
}

// HasVMArgs represents the arguments required for the HasVM method
type DeleteVMArgs struct {
	ServerId string // The Cloud ID of the VM to check
}

// HasDiskArgs represents the arguments required for the HasDisk method
type CreateStemcellArgs struct {
	Path string       // Path to the stemcell image file
	Info StemcellInfo // Metadata for the stemcell, including name, version, and other properties
}

// CreateDiskArgs represents the arguments required for the CreateDisk method
type CreateDiskArgs struct {
	Size       int64
	Properties map[string]any
	ServerId   string
}

// CreateNetworkArgs represents the arguments required for the CreateNetworkArgs method
type CreateNetworkArgs struct {
	Type        string        // Type of the network, e.g., "manual", "dynamic", etc.
	Properties  NetProperties `json:"cloud_properties"` // Cloud properties for the network, including net_id and security groups
	Range       string        // IP range for the network, if applicable
	Gateway     string        // Gateway for the network, if applicable
	NetmaskBits int           // Netmask bits for the network, if applicable
}

// RebootVMArgs represents the arguments required for the RebootVMArgs method
type RebootVMArgs struct {
	ServerId string // The Cloud ID of the VM to reboot
}

// SetVMMetadataArgs represents the arguments required for the SetVMMetadataArgs method
type SetVMMetadataArgs struct {
	ServerId string         // The Cloud ID of the VM to set metadata for
	Metadata map[string]any // Metadata to set for the VM
}

// SetDiskMetadataArgs represents the arguments required for the SetDiskMetadata method
type SetDiskMetadataArgs struct {
	DiskId   string         // The Cloud ID of the disk to set metadata for
	Metadata map[string]any // Metadata to set for the disk
}

// SnapshotDiskArgs represents the arguments required for the SnapshotDisk method
type SnapshotDiskArgs struct {
	DiskId   string         // The Cloud ID of the disk to snapshot
	Metadata map[string]any // Optional metadata for the snapshot
}

// DeleteSnapshotArgs represents the arguments required for the DeleteSnapshot method
type DeleteSnapshotArgs struct {
	SnapshotId string // The Cloud ID of the snapshot to delete
}

// UpdateDiskArgs represents the arguments required for the UpdateDisk method
type UpdateDiskArgs struct {
	DiskId     string         // The Cloud ID of the disk to update
	NewSize    int64          // New disk size in MiB
	Properties map[string]any // New cloud properties for the disk
}

// ResizeDiskArgs represents the arguments required for the ResizeDisk method
type ResizeDiskArgs struct {
	DiskId  string // The Cloud ID of the disk to resize
	NewSize int64  // New disk size in MiB
}

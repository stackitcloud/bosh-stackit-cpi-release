package lib

import "fmt"

const (

	// as of Oct. 10th 2025 the stackit sdk had no constants for nic STATUS. an unattached NIC
	// is shown as DOWN in the API.

	NICAvailableStatus = "DOWN"
	NICInUseStatus     = "ACTIVE"

	// These are the expected JSON attributes that we rely on from the Stackit Stage Profile
	TokenEndpointKey        = "token_custom_endpoint"
	IaasEndpointKey         = "iaas_custom_endpoint"
	LoadBalancerEndpointKey = "load_balancer_custom_endpoint"

	BoshTimeStampFormat = "2006-01-02T15:04:05Z"
	IaasTimestampFormat = "20060102_150405"

	CreatedByBoshLabelKey     = "created_by"
	CreatedByBoshLabelValue   = "stackit-bosh-cpi"
	BoshCreatedAtKeyName      = "created_at"
	BoshDirectorUUIDKeyName   = "director_uuid"
	AzTagKeyName              = "availability_zone"
	AttachedToLoadBalancerKey = "managed_lb"

	StemcellIDKeyName = "stemcell_id"
	AgentIDKeyName    = "agent_id"
	VMNameKey         = "name"
	DynamicVIPKey     = "dynamic_vip"
	BoshInstanceIDKey = "id"

	ServerNameMaxLength = 63
	TagKeyMaxLength     = 63
	TagValueMaxLength   = 63

	// Bosh Expected RPC error types to use in responses
	// https://github.com/cloudfoundry/bosh/blob/e79e1132314b9ff14312cc8ec6e03d44e6774252/src/bosh-director/lib/clouds/errors.rb#L8
	// As of 19.09.2025 the known types are

	CpiErrorType                 = "Bosh::Clouds::CpiError"
	NotImplementedErrorType      = "Bosh::Clouds::NotImplemented"
	NotSupportedErrorType        = "Bosh::Clouds::NotSupported"
	AttachDiskResponseErrorType  = "Bosh::Clouds::AttachDiskResponseError"
	CloudErrorType               = "Bosh::Clouds::CloudError"
	VMNotFoundErrorType          = "Bosh::Clouds::VMNotFound"
	NetworkNotFoundErrorType     = "Bosh::Clouds::NetworkNotFound"
	RetriableCloudErrorErrorType = "Bosh::Clouds::RetriableCloudError"
	NoDiskSpaceErrorType         = "Bosh::Clouds::NoDiskSpace"
	DiskNotAttachedErrorType     = "Bosh::Clouds::DiskNotAttached"
	DiskNotFoundErrorType        = "Bosh::Clouds::DiskNotFound"
	VMCreationFailedErrorType    = "Bosh::Clouds::VMCreationFailed"

	// Bosh known RPC Methods
	// https://bosh.io/docs/cpi-api-rpc/#methods
	Info                       = "info"
	CreateStemcell             = "create_stemcell"
	DeleteStemcell             = "delete_stemcell"
	CreateVM                   = "create_vm"
	DeleteVM                   = "delete_vm"
	HasVM                      = "has_vm"
	RebootVM                   = "reboot_vm"
	SetVMMetadata              = "set_vm_metadata"
	CalculateVMCloudProperties = "calculate_vm_cloud_properties"
	CreateDisk                 = "create_disk"
	DeleteDisk                 = "delete_disk"
	ResizeDisk                 = "resize_disk"
	UpdateDisk                 = "update_disk"
	HasDisk                    = "has_disk"
	AttachDisk                 = "attach_disk"
	DetachDisk                 = "detach_disk"
	SetDiskMetadata            = "set_disk_metadata"
	GetDisks                   = "get_disks"
	SnapshotDisk               = "snapshot_disk"
	DeleteSnapshot             = "delete_snapshot"
	CreateNetwork              = "create_network"
	DeleteNetwork              = "delete_network"

	// Minimum and maximum disk size in GiB (Cloud config is in MB https://bosh.io/docs/cloud-config/#disk-types )
	MinDiskSizeMB   int64 = 1024     // 1 GB
	MaxDiskSizeMB   int64 = 16384000 // 16 TB
	MinRootDiskSize int64 = 20
	StemcellMaxSize int64 = 20 * 1024 * 1024 * 1024 // 20GB

)

var (
	// Supported disk formats for stemcell images
	supportedDiskFormats = map[string]bool{
		"qcow2": true,
		"raw":   true,
		"iso":   true,
	}

	// Supported OS types for stemcell images
	supportedOSTypes = map[string]bool{
		"linux":   true,
		"windows": true,
	}

	MethodNotImplementedErrorMessage = fmt.Errorf("method not implemented")
	MethodNotSupportedErrorMessage   = fmt.Errorf("method not supported")
)

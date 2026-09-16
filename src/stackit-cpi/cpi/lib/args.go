package lib

import (
	"encoding/json"
	"fmt"
)

func (args CreateVMArgs) Validate() error {
	if args.AgentID == "" { // Cannot be empty, passed in by BOSH
		return fmt.Errorf("CreateVMArgs.Validate(): agent_id is required")
	}

	if args.StemcellID == "" { // Passed in by BOSH
		return fmt.Errorf("CreateVMArgs.Validate(): stemcell_cid is required")
	}

	if len(args.Networks) == 0 {
		return fmt.Errorf("CreateVMArgs.Validate(): at least one network must be specified")
	}

	if args.Properties.InstanceType == "" {
		return fmt.Errorf("instance_type must be specified")
	}

	if args.Properties.RootDisk.Size < 20 {
		args.Properties.RootDisk.Size = 20
	}

	if args.Properties.AvailabilityZone == "" {
		return fmt.Errorf("availability_zone must be specified")
	}

	return nil
}

// Custom UnmarshalJSON for CreateVMArgs to handle array-based RPC arguments
func (args *CreateVMArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	if len(arr) != 6 { // Verify we have the expected number of arguments
		return fmt.Errorf("expected 6 arguments for create_vm, got %d", len(arr))
	}

	// Unmarshal each element into the appropriate field
	if err := json.Unmarshal(arr[0], &args.AgentID); err != nil {
		return fmt.Errorf("failed to unmarshal agent_id: %w", err)
	}

	if err := json.Unmarshal(arr[1], &args.StemcellID); err != nil {
		return fmt.Errorf("failed to unmarshal stemcell_cid: %w", err)
	}

	if err := json.Unmarshal(arr[2], &args.Properties); err != nil {
		return fmt.Errorf("failed to unmarshal cloud_properties: %w", err)
	}

	if err := json.Unmarshal(arr[3], &args.Networks); err != nil {
		return fmt.Errorf("failed to unmarshal networks: %w", err)
	}

	if err := json.Unmarshal(arr[4], &args.DiskIds); err != nil {
		return fmt.Errorf("failed to unmarshal disk_cids: %w", err)
	}

	if err := json.Unmarshal(arr[5], &args.Env); err != nil {
		return fmt.Errorf("failed to unmarshal environment: %w", err)
	}
	return args.Validate()
}

func (args CreateVMArgs) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{
		args.AgentID,
		args.StemcellID,
		args.Properties,
		args.Networks,
		args.DiskIds,
		args.Env,
	})
}

func (args CreateStemcellArgs) Validate() error {
	// Validate required properties before unmarshaling to struct
	if args.Info.Name == "" {
		return fmt.Errorf("stemcell name is required")
	}
	if args.Info.Version == "" {
		return fmt.Errorf("stemcell version is required")
	}
	if args.Info.DiskFormat == "" {
		return fmt.Errorf("disk format is required")
	}
	if !supportedDiskFormats[args.Info.DiskFormat] {
		return fmt.Errorf("unsupported disk format: %s", args.Info.DiskFormat)
	}

	if args.Info.OsType == "" {
		return fmt.Errorf("os type is required")
	}
	if !supportedOSTypes[args.Info.OsType] {
		return fmt.Errorf("unsupported OS type: %s", args.Info.OsType)
	}

	return nil
}

func (args *CreateStemcellArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) == 0 {
		return fmt.Errorf("missing required arguments")
	}

	if len(arr) != 2 {
		return fmt.Errorf("expected 2 arguments for create_stemcell, got %d", len(arr))
	}

	// Unmarshal the first element (image path)
	if err := json.Unmarshal(arr[0], &args.Path); err != nil {
		return fmt.Errorf("first argument must be a string (path to image)")
	}

	// Try to unmarshal the second element as a map to check if it's the right type
	if err := json.Unmarshal(arr[1], &args.Info); err != nil {
		return fmt.Errorf("second argument must be a map of stemcell properties")
	}

	return args.Validate()
}

func (args *DeleteStemcellArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for delete_stemcell, got %d", len(arr))
	}

	// Unmarshal the first element (stemcell CID)
	if err := json.Unmarshal(arr[0], &args.StemcellId); err != nil {
		return fmt.Errorf("failed to unmarshal stemcell_cid: %w", err)
	}

	return nil
}

func (args DeleteDiskArgs) Validate() error {
	if args.DiskId == "" {
		return fmt.Errorf("disk ID cannot be empty")
	}
	return nil
}

func (args *DeleteDiskArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for delete_disk, got %d", len(arr))
	}

	// Unmarshal the first element (disk CID)
	if err := json.Unmarshal(arr[0], &args.DiskId); err != nil {
		return fmt.Errorf("failed to unmarshal disk_cid: %w", err)
	}

	return args.Validate()
}

func (args DeleteVMArgs) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{
		args.ServerId,
	})
}

func (args *DeleteVMArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for delete_vm, got %d", len(arr))
	}

	// Unmarshal the first element (VM CID)
	if err := json.Unmarshal(arr[0], &args.ServerId); err != nil {
		return fmt.Errorf("failed to unmarshal vm_cid: %w", err)
	}

	return nil
}

func (args *HasVMArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for has_vm, got %d", len(arr))
	}

	// Unmarshal the first element (VM CID)
	if err := json.Unmarshal(arr[0], &args.ServerId); err != nil {
		return fmt.Errorf("failed to unmarshal vm_cid: %w", err)
	}

	return nil
}

func (args HasDiskArgs) Validate() error {
	if args.DiskId == "" {
		return fmt.Errorf("disk_cid must be provided")
	}
	return nil
}

func (args *HasDiskArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for has_disk, got %d", len(arr))
	}

	// Unmarshal the first element (Disk CID)
	if err := json.Unmarshal(arr[0], &args.DiskId); err != nil {
		return fmt.Errorf("failed to unmarshal disk_cid: %w", err)
	}

	return args.Validate()
}

func (args CreateDiskArgs) Validate() error {
	if args.Size < MinDiskSizeMB {
		return fmt.Errorf("cpi.CreateDisk(): Disk size must be at least %d MiB", MinDiskSizeMB)
	}
	if args.Size > MaxDiskSizeMB {
		return ErrDiskSizeTooLarge
	}
	if args.ServerId == "" {
		return fmt.Errorf("cpi.CreateDisk(): VM CID cannot be empty")
	}
	return nil
}

func (args CreateDiskArgs) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{
		args.Size,
		args.Properties,
		args.ServerId,
	})
}

func (args *CreateDiskArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 3 {
		return fmt.Errorf("expected 3 arguments for create_disk, got %d", len(arr))
	}

	// Unmarshal each element into the appropriate field
	if err := json.Unmarshal(arr[0], &args.Size); err != nil {
		return fmt.Errorf("failed to unmarshal size: %w", err)
	}

	if err := json.Unmarshal(arr[1], &args.Properties); err != nil {
		return fmt.Errorf("failed to unmarshal cloud_properties: %w", err)
	}

	if err := json.Unmarshal(arr[2], &args.ServerId); err != nil {
		return fmt.Errorf("failed to unmarshal name: %w", err)
	}

	return args.Validate()
}

func (args *RebootVMArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for reboot_vm, got %d", len(arr))
	}

	// Unmarshal the first element (VM CID)
	if err := json.Unmarshal(arr[0], &args.ServerId); err != nil {
		return fmt.Errorf("failed to unmarshal vm_cid: %w", err)
	}

	return nil
}

func (args SetVMMetadataArgs) Validate() error {
	if !IsValidUUID(args.ServerId) {
		return fmt.Errorf("vm_cid must be a valid uuid, got: '%s'", args.ServerId)
	}
	if args.Metadata == nil {
		return fmt.Errorf("metadata must be provided")
	}
	return nil
}

func (args *SetVMMetadataArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 2 {
		return fmt.Errorf("expected 2 arguments for set_vm_metadata, got %d", len(arr))
	}

	// Unmarshal each element into the appropriate field
	if err := json.Unmarshal(arr[0], &args.ServerId); err != nil {
		return fmt.Errorf("failed to unmarshal vm_cid: %w", err)
	}

	if err := json.Unmarshal(arr[1], &args.Metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return args.Validate()
}

func (args SetDiskMetadataArgs) Validate() error {
	if args.DiskId == "" {
		return fmt.Errorf("disk_cid must be provided")
	}
	if args.Metadata == nil {
		return fmt.Errorf("metadata must be provided")
	}
	return nil
}

// UnmarshalJSON implements custom JSON unmarshalling for SetDiskMetadataArgs
func (args *SetDiskMetadataArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 2 {
		return fmt.Errorf("expected 2 arguments for set_disk_metadata, got %d", len(arr))
	}

	// Unmarshal each element into the appropriate field
	if err := json.Unmarshal(arr[0], &args.DiskId); err != nil {
		return fmt.Errorf("failed to unmarshal disk_cid: %w", err)
	}

	if err := json.Unmarshal(arr[1], &args.Metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return args.Validate()
}

func (args *SnapshotDiskArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) < 1 {
		return fmt.Errorf("expected 1 argument for snapshot_disk, got %d", len(arr))
	}

	// Unmarshal the first element (disk CID)
	if err := json.Unmarshal(arr[0], &args.DiskId); err != nil {
		return fmt.Errorf("failed to unmarshal disk_cid: %w", err)
	}

	// Unmarshal the second element (metadata) if provided
	if len(arr) > 1 {
		args.Metadata = make(map[string]any)
		if err := json.Unmarshal(arr[1], &args.Metadata); err != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return nil
}

func (args *DeleteSnapshotArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for delete_snapshot, got %d", len(arr))
	}

	// Unmarshal the first element (snapshot CID)
	if err := json.Unmarshal(arr[0], &args.SnapshotId); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot_cid: %w", err)
	}

	return nil
}

func (args ResizeDiskArgs) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{
		args.DiskId,
		args.NewSize,
	})
}

func (args *ResizeDiskArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 2 {
		return fmt.Errorf("expected 2 arguments for resize_disk, got %d", len(arr))
	}

	if err := json.Unmarshal(arr[0], &args.DiskId); err != nil {
		return fmt.Errorf("failed unmarshalling diskId `%s`: %w", string(data), err)
	}

	if err := json.Unmarshal(arr[1], &args.NewSize); err != nil {
		return fmt.Errorf("failed unmarshalling diskId `%s`: %w", string(data), err)
	}

	if args.NewSize > MaxDiskSizeMB {
		return ErrDiskSizeTooLarge
	}
	return nil
}

func (args *UpdateDiskArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 3 {
		return fmt.Errorf("expected 3 arguments for update_disk, got %d", len(arr))
	}

	// Unmarshal each element into the appropriate field
	if err := json.Unmarshal(arr[0], &args.DiskId); err != nil {
		return fmt.Errorf("failed to unmarshal disk_cid: %w", err)
	}

	if err := json.Unmarshal(arr[1], &args.NewSize); err != nil {
		return fmt.Errorf("failed to unmarshal new_size: %w", err)
	}

	if err := json.Unmarshal(arr[2], &args.Properties); err != nil {
		return fmt.Errorf("failed to unmarshal cloud_properties: %w", err)
	}

	return nil
}

func (args CreateNetworkArgs) Validate() error {
	if args.Type != "manual" && args.Type != "dynamic" {
		return fmt.Errorf("invalid network type '%s', expected 'manual' or 'dynamic'", args.Type)
	}
	if args.NetmaskBits < 0 || args.NetmaskBits > 32 {
		return fmt.Errorf("expected network `netmask_bits` key to be between 0 and 32, got %d", args.NetmaskBits)
	}
	return nil
}

func (args *CreateNetworkArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for create_network, got %d", len(arr))
	}

	// The network specification is a single object in the array
	var networkSpec struct {
		Type        string        `json:"type"`
		Properties  NetProperties `json:"cloud_properties"`
		Range       string        `json:"range"`
		Gateway     string        `json:"gateway"`
		NetmaskBits int           `json:"netmask_bits"`
	}

	if err := json.Unmarshal(arr[0], &networkSpec); err != nil {
		return fmt.Errorf("failed to unmarshal network specification: %w", err)
	}

	// Copy the values from the intermediate struct to our args struct
	args.Type = networkSpec.Type
	args.Properties = networkSpec.Properties
	args.Range = networkSpec.Range
	args.Gateway = networkSpec.Gateway
	args.NetmaskBits = networkSpec.NetmaskBits

	return args.Validate()
}

func (args *DeleteNetworkArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	if len(arr) != 1 { // Verify we have the expected number of arguments
		return fmt.Errorf("expected 1 argument for delete_network, got %d", len(arr))
	}

	// Unmarshal the first element (network CID)
	if err := json.Unmarshal(arr[0], &args.NetworkId); err != nil {
		return fmt.Errorf("failed to unmarshal network_cid: %w", err)
	}

	return nil
}

func (args AttachDiskArgs) Validate() error {
	if args.ServerId == "" {
		return fmt.Errorf("vm_cid must be provided")
	}
	if args.DiskId == "" {
		return fmt.Errorf("disk_cid must be provided")
	}
	return nil
}

func (args *AttachDiskArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 2 {
		return fmt.Errorf("expected 2 arguments for attach_disk, got %d", len(arr))
	}

	// Unmarshal each element into the appropriate field
	if err := json.Unmarshal(arr[0], &args.ServerId); err != nil {
		return fmt.Errorf("failed to unmarshal vm_cid: %w", err)
	}

	if err := json.Unmarshal(arr[1], &args.DiskId); err != nil {
		return fmt.Errorf("failed to unmarshal disk_cid: %w", err)
	}

	return args.Validate()
}

func (args *DetachDiskArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 2 {
		return fmt.Errorf("expected 2 arguments for detach_disk, got %d", len(arr))
	}

	// Unmarshal each element into the appropriate field
	if err := json.Unmarshal(arr[0], &args.ServerId); err != nil {
		return fmt.Errorf("failed to unmarshal vm_cid: %w", err)
	}

	if err := json.Unmarshal(arr[1], &args.DiskId); err != nil {
		return fmt.Errorf("failed to unmarshal disk_cid: %w", err)
	}

	return nil
}

func (args *GetDisksArgs) UnmarshalJSON(data []byte) error {
	// First check if we're getting an array (which is the expected format from BOSH CPI)
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("failed to unmarshal arguments array: %w", err)
	}

	// Verify we have the expected number of arguments
	if len(arr) != 1 {
		return fmt.Errorf("expected 1 argument for get_disks, got %d", len(arr))
	}

	// Unmarshal the first element (server ID)
	if err := json.Unmarshal(arr[0], &args.ServerId); err != nil {
		return fmt.Errorf("failed to unmarshal vm_cid: %w", err)
	}

	return nil
}

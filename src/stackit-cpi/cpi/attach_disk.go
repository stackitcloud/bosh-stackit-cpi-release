package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/*
Arguments:
  vm_cid [String]: Cloud ID of the VM.
  disk_cid [String]: Cloud ID of the disk.

Result:
  disk_hints [Hash or String]: Disks that are associated with the VM
*/

// AttachDisk handles the "attach_disk" method of the RPC
func (c *CPI) AttachDisk(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.AttachDiskArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed parsing arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	// Check if VM and disk are in the same availability zone
	err = c.ValidateDiskVMAvailabilityZone(args.ServerId, args.DiskId)
	if err != nil {
		c.Log.Error("Availability zone validation failed", "error", err)
		err = fmt.Errorf("availability zone validation failed: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	err = c.SClient.AttachDiskToVM(args.DiskId, args.ServerId)
	if err != nil {
		c.Log.Debug("Error details", "error", err)
		err = fmt.Errorf("[iaas API] Error when calling `AddVolumeToServer`: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Triggered attachment of volume", "disk_id", args.DiskId)
	c.Log.Debug("Waiting for attachment of volume to server", "disk_id", args.DiskId, "server_id", args.ServerId)

	err = c.SClient.WaitForAttachDiskToVM(args.DiskId, args.ServerId)
	if err != nil {
		err = fmt.Errorf("[iaas API] Error when waiting for attachment: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Volume has been successfully attached to the server", "disk_id: ", args.DiskId, "server_id: ", args.ServerId)

	return &lib.RPCResponse{
		Error: nil,
		Result: map[string]any{
			"path":      fmt.Sprintf("/dev/disk/by-id/virtio-%s", args.DiskId[:20]),
			"volume_id": fmt.Sprintf("/dev/disk/by-id/virtio-%s", args.DiskId[:20]),
		},
		Log: "success",
	}, nil
}

func (c *CPI) ValidateAttachDiskArgs(args *lib.AttachDiskArgs) error {
	return nil
}

// ValidateDiskVMAvailabilityZone checks if the VM and disk are in the same availability zone
func (c *CPI) ValidateDiskVMAvailabilityZone(serverID string, diskID string) error {
	// Get VM details
	c.Log.Debug("Getting server details", "server_id", serverID)
	vmDetails, err := c.SClient.GetVM(serverID)
	if err != nil {
		c.Log.Warn("Failed to get server details, skipping AZ validation", "error", err)
		// Skip validation if we can't get server details
		return nil
	}

	vmAZ := ""
	if vmDetails != nil {
		if az, ok := vmDetails.GetAvailabilityZoneOk(); ok && *az != "" {
			vmAZ = *az
		}
	}

	if vmAZ == "" {
		c.Log.Warn("VM has no availability zone set, skipping validation", "server_id", serverID)
		return nil
	}

	// Get disk details
	c.Log.Debug("Getting volume details", "disk_id", diskID)
	diskDetails, err := c.SClient.GetDisk(diskID)
	if err != nil {
		c.Log.Warn("Failed to get disk details, skipping AZ validation", "error", err)
		// Skip validation if we can't get disk details
		return nil
	}

	diskAZ := ""
	if diskDetails != nil {
		if az, ok := diskDetails.GetAvailabilityZoneOk(); ok && *az != "" {
			diskAZ = *az
		}
	}

	if diskAZ == "" {
		c.Log.Warn("Disk has no availability zone set, skipping validation", "disk_id", diskID)
		return nil
	}

	// Compare availability zones
	c.Log.Debug("Comparing VM AZ with disk AZ", "vm_az", vmAZ, "disk_az", diskAZ)
	if vmAZ != diskAZ {
		return fmt.Errorf("cannot attach disk: VM and disk are in different availability zones - VM is in AZ '%s' and disk is in AZ '%s'. Disks can only be attached to VMs in the same AZ", vmAZ, diskAZ)
	}

	c.Log.Debug("VM and disk are both in availability zone", "az", vmAZ)
	return nil
}

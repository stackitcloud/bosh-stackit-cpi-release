package cpi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/*
Arguments:
  size [Integer]: Size of the disk in MiB.
  cloud_properties [Hash]: Cloud properties hash specified in the deployment manifest under the disk pool.
  vm_cid [String]: Cloud ID of the VM created disk will most likely be attached; it could be used to .optimize disk placement so that disk is located near the VM.

Result:
  disk_cid [String]: Cloud ID of the created disk.
*/

// Supported volume types
var supportedVolumePerformanceClasss = map[string]bool{
	"storage_premium_perf0":  true,
	"storage_premium_perf1":  true,
	"storage_premium_perf2":  true,
	"storage_premium_perf4":  true,
	"storage_premium_perf6":  true,
	"storage_premium_perf8":  true,
	"storage_premium_perf10": true,
	"storage_premium_perf12": true,
	"storage_premium_perf14": true,
	"storage_premium_perf16": true,
	"storage_premium_perf18": true,
	"storage_premium_perf20": true,
}

// CreateDisk handles the "create_disk" method of the RPC
func (c *CPI) CreateDisk(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.CreateDiskArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("cpi.CreateDisk(): Error unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	availabilityZone, err := c.SClient.GetAZForServer(args.ServerId)
	if err != nil {
		err = fmt.Errorf("failed determinig az for server '%s': %w", args.ServerId, err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	// Extract optional cloud properties
	var performanceClass string
	performanceClass = c.Config.DefaultPersistentVolumeType // Default from CPI config
	if classVal, ok := args.Properties["type"].(string); ok && classVal != "" {
		if !supportedVolumePerformanceClasss[classVal] {
			c.Log.Warn("Unsupported volume performance class, using default performance class", "class", classVal)
		} else {
			performanceClass = classVal
		}
	}

	// Generate a meaningful disk name that fits within the 63 character limit

	diskName := fmt.Sprintf("disk-%s", args.ServerId)

	diskSizeGB := max(int64(args.Size/1024), 1)

	c.Log.Debug("Converting disk size from MB to GB", "size_mb", args.Size, "size_gb", diskSizeGB)

	// Create the volume payload
	createVolumePayload := iaas.CreateVolumePayload{
		Name:             utils.Ptr(diskName),
		AvailabilityZone: availabilityZone,
		Size:             utils.Ptr(diskSizeGB),
		PerformanceClass: utils.Ptr(performanceClass),
		Description:      utils.Ptr(fmt.Sprintf("BOSH disk created by STACKIT CPI director_uuid: %s", req.Context.DirectorUUID)),
	}

	// Add Labels (e.g., "tags")
	tags := map[string]any{
		lib.CreatedByBoshLabelKey: lib.CreatedByBoshLabelValue,
		"associated_vm_cid":       args.ServerId,
		"creation_date":           time.Now().UTC().Format("20060102_150405"),
	}
	createVolumePayload.Labels = tags

	c.Log.Debug("Creating disk", "name", diskName, "size_gb", diskSizeGB, "type", performanceClass, "vm_cid", args.ServerId, "performance_class", performanceClass, "region", c.Config.RegionID, "az", availabilityZone)

	// Attempt to create volume with retries
	volume, err := c.createVolume(createVolumePayload)
	if err != nil {
		c.Log.Error("Failed to create volume", "error", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("Disk created successfully", "name", diskName, "volume_id", volume.GetId())

	// Return the disk ID
	return &lib.RPCResponse{
		Error:  nil,
		Result: volume.GetId(),
		Log:    "success",
	}, nil
}

func (c *CPI) createVolume(payload iaas.CreateVolumePayload) (volume *iaas.Volume, err error) {
	c.Log.Info("starting create")
	volume, err = c.SClient.CreateDisk(payload)
	if err != nil {
		c.Log.Error("Volume creation attempt failed", "error", err)
		return nil, err
	}

	c.Log.Info("wiating for create to finish")
	volume, err = c.SClient.WaitForCreateDisk(volume.GetId())
	if err != nil {
		c.Log.Error("failed waiting for creation to finish", "error", err)
		return nil, err
	}

	return volume, nil
}

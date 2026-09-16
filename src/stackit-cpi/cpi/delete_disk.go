package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/* Deletes disk. Assume that disk was detached from all VMs.
 * To avoid losing track of disks, make sure to raise an error if disk deletion is not absolutely certain.

Arguments:
  disk_cid [String]: Cloud ID of the disk to delete; returned from create_disk.

Result:
  true/false - No return value expected
*/

// DeleteDisk handles the "delete_disk" method of the RPC
func (c *CPI) DeleteDisk(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.DeleteDiskArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failedunmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("Deleting disk", "disk_id", args.DiskId)

	// Delete the disk with retry mechanism
	err = c.deleteDisk(args.DiskId)
	if err != nil {
		err = fmt.Errorf("failed to delete disk: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("Disk has been successfully deleted", "disk_id", args.DiskId)
	return &lib.RPCResponse{
		Error:  nil,
		Result: true,
		Log:    "success",
	}, nil
}

func (c *CPI) deleteDisk(diskID string) error {
	// Trigger the deletion
	err := c.SClient.DeleteDisk(diskID)
	if err != nil {
		return fmt.Errorf("error triggering disk deletion: %w", err)
	}

	c.Log.Debug("Deleting volume", "disk_id", diskID)
	err = c.SClient.WaitForDeleteDisk(diskID)
	if err != nil {
		return fmt.Errorf("error waiting for disk deletion: %w", err)
	}

	return nil
}

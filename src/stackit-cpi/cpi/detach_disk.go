package cpi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/*
Arguments:
  vm_cid [String]: Cloud ID of the VM.
  disk_cid [String]: Cloud ID of the disk.

Result:
  true/false
*/

// DetachDisk handles the "detatch_disk" method of the RPC
func (c *CPI) DetachDisk(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.DetachDiskArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	err = c.validateDetachDiskArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Detaching volume from server", "disk_id", args.DiskId, "server_id", args.ServerId)
	err = c.SClient.DetachDiskFromVM(args.DiskId, args.ServerId)
	if err != nil {
		// Check if the error is that the volume-attachment is already detached (404)
		if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
			return &lib.RPCResponse{
				Error:  nil,
				Result: true,
				Log:    "success",
			}, nil
		}
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Triggered detachment of volume from server", "disk_id", args.DiskId, "server_id", args.ServerId)
	err = c.SClient.WaitForDetachDiskFromVM(args.DiskId, args.ServerId)

	if err == nil {
		c.Log.Debug("Volume has been successfully detached from server", "disk_id", args.DiskId, "server_id", args.ServerId)
		return &lib.RPCResponse{
			Error:  nil,
			Result: true,
			Log:    "success",
		}, nil
	}

	c.Log.Debugf("Failed waiting for volume detachment after %d retries: %v", c.Config.RetryCount, err.Error())
	err = fmt.Errorf("[iaas API] Error when waiting for volume %q to be detatched from server %q after %d retries: %w", args.DiskId, args.ServerId, c.Config.RetryCount, err)
	return lib.WrapErrorInResponse(err, req.GetLoggingContext())
}

func (c *CPI) validateDetachDiskArgs(args *lib.DetachDiskArgs) error {
	if args.ServerId == "" {
		return fmt.Errorf("vm_cid must be provided")
	}
	if args.DiskId == "" {
		return fmt.Errorf("disk_cid must be provided")
	}
	return nil
}

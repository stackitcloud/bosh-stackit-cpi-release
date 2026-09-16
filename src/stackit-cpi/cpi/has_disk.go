package cpi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/*
Checks for disk (volume) presence in the IaaS.
This method is mostly used by the consistency check tool (cloudcheck) to determine if the disk still exists.

Arguments:
  disk_cid [String]: Cloud ID of the disk to check; returned from create_disk.

Result:
  exists [Boolean]: True if disk is present.
*/

// HasDisk handles the "has_disk" method of the RPC
func (c *CPI) HasDisk(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.HasDiskArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Checking for presence of disk", "disk_id", args.DiskId)
	volume, err := c.SClient.GetDisk(args.DiskId)
	if err != nil {
		if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
			return &lib.RPCResponse{
				Error:  nil,
				Result: false,
				Log:    "",
			}, nil
		}
		c.Log.Error("Error when calling GetVolume", "error", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	resulValue := volume.GetId() == args.DiskId
	return &lib.RPCResponse{
		Error:  nil,
		Result: resulValue,
		Log:    "success",
	}, nil
}

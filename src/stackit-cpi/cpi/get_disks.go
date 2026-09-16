package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/*
Arguments:
  vm_cid [String]: Cloud ID of the VM.

Result:
  disk_cids [Array of strings]: Array of disk_cids that are currently attached to the VM.
*/

// GetDisks handles the "get_disks" method of the RPC
func (c *CPI) GetDisks(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.GetDisksArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("Error unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	err = c.validateGetDisksArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	server, err := c.SClient.GetVM(args.ServerId)
	if err != nil {
		err = fmt.Errorf("failed finding volumes of server '%s': %w", args.ServerId, err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	attachedVolumeIds := server.GetVolumes()

	// do not chain GetId() on the return of GetBootVolume(). It panics for some reason.
	bootVolume := server.GetBootVolume()
	bootVolumeID := bootVolume.GetId()

	respValue := []string{}

	for _, volumeID := range attachedVolumeIds {
		if volumeID == bootVolumeID {
			continue
		}
		respValue = append(respValue, volumeID)
	}
	return &lib.RPCResponse{
		Error:  nil,
		Result: respValue,
		Log:    "sucess",
	}, nil
}

func (c *CPI) validateGetDisksArgs(args *lib.GetDisksArgs) error {
	if args.ServerId == "" {
		return fmt.Errorf("cpi.validateGetDisksArgs(): vm_cid must be provided")
	}
	return nil
}

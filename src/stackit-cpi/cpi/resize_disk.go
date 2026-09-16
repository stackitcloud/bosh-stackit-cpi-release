package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/*
Arguments:
  disk_cid [String]: Cloud ID of the disk to resize; returned from create_disk.
  new_size [Integer]: New disk size in MiB.

Result:
  true/false - No return value expected
*/

// ResizeDisk handles the "reboot_disk" method of the RPC
func (c *CPI) ResizeDisk(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.ResizeDiskArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("error unmarshalling Arguments JSON `%s`: %w", string(req.Arguments), err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	err = c.validateResizeDiskArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	diskSizeInGB := max(args.NewSize/1024, 1)
	resizeVolumePayload := iaas.ResizeVolumePayload{
		Size: diskSizeInGB,
	}

	c.Log.Debug("Requesting resize of volume", "disk_id", args.DiskId, "new_size_mib", args.NewSize)

	err = c.SClient.ResizeDisk(args.DiskId, resizeVolumePayload)
	if err != nil {
		err = fmt.Errorf("cpi.ResizeDisk(): [iaas API] Error when calling `ResizeVolume`: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Disk has been successfully resized", "disk_id", args.DiskId)

	return &lib.RPCResponse{
		Error:  nil,
		Result: true,
		Log:    "success",
	}, nil
}

func (c *CPI) validateResizeDiskArgs(args *lib.ResizeDiskArgs) error {
	if args.DiskId == "" {
		return fmt.Errorf("disk_cid must be provided")
	}
	if args.NewSize <= 0 {
		return fmt.Errorf("new_size must be a positive integer representing MiB")
	}
	if args.NewSize > lib.MaxDiskSizeMB {
		return lib.ErrDiskSizeTooLarge
	}
	return nil
}

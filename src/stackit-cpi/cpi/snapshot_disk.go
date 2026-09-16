package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/* Takes a snapshot of the given volume (disk).

Arguments:
  disk_cid [String]: Cloud ID of the disk.
  metadata [Hash]: Collection of key-value pairs. CPI should not rely on presence of specific keys.

Result:
  snapshot_cid [String]: Cloud ID of the disk snapshot.
*/

// CreateSnapshot handles the "snapshot_disk" method of the RPC
func (c *CPI) SnapshotDisk(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.SnapshotDiskArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("error unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	err = c.validateSnapshotDiskArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	payload := iaas.NewCreateSnapshotPayload(args.DiskId)
	payload.Labels = args.Metadata
	snapshot, err := c.SClient.CreateSnapshot(payload)
	if err != nil {
		err = fmt.Errorf("[iaas API] Error when calling `CreateSnapshot`: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())

	}
	c.Log.Debug("Snapshot taken for disk", "snapshot_id", *snapshot.Id, "disk_id", args.DiskId)
	return &lib.RPCResponse{
		Error:  nil,
		Result: snapshot.GetId(),
		Log:    "success",
	}, nil
}

func (c *CPI) validateSnapshotDiskArgs(args *lib.SnapshotDiskArgs) error {
	if args.DiskId == "" {
		return fmt.Errorf("disk_cid must be provided")
	}
	if args.Metadata == nil {
		return fmt.Errorf("metadata must be provided")
	}
	return nil
}

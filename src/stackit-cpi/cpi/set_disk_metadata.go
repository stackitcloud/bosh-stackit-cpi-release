package cpi

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/*
Sets disk's metadata to make it easier for operators to categorize disks when looking at the IaaS management console.
Disk metadata is written when the disk is attached to a VM.
Metadata is not removed when disk is detached or VM is deleted.
What is called “metadata” here is actually implemented as tags by most CPIs.
The following default disk tags are forced by Bosh, whatever happens.
These cannot be overridden.

Arguments:
  disk_cid [String]: Cloud ID of the disk to modify; returned from create_disk.
  metadata [Hash]: Collection of key-value pairs. CPI should not rely on presence of specific keys.

Result:
  true/false - No response is expected

Example API Request:
[
  "vol-3475945",
  {
    "director": "director-784430",
    "deployment": "redis",
    "instance_id": "ce7d2040-212e-4d5a-a62d-952a12c50741",
    "job": "redis",
    "instance_index": "1",
    "instance_name": "redis/ce7d2040-212e-4d5a-a62d-952a12c50741",
    "attached_at": "2017-08-10T12:03:32Z"
  }
]
*/

// SetDiskMetadata handles the "set_disk_metadata" method of the RPC
func (c *CPI) SetDiskMetadata(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.SetDiskMetadataArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	// labels key/value format must pass regex: ^(-|_|[a-z0-9]){0,63}$ -- provided by BOSH
	// update args.Metadata so "attached_at" is formatted as as only numbers
	if attachedAt, ok := args.Metadata["attached_at"]; ok && attachedAt != nil && attachedAt != "" {
		if attachedAtStr, ok := attachedAt.(string); ok {
			args.Metadata["attached_at"] = removeNonNumeric(attachedAtStr)
		}
	}
	payload := iaas.UpdateVolumePayload{
		Labels: args.Metadata,
	}

	c.Log.Debug("Updating metadata for volume", "disk_id", args.DiskId, "payload", payload)
	err = c.SClient.UpdateDisk(args.DiskId, payload)
	if err != nil {
		err = fmt.Errorf("[iaas API] Error when calling `UpdateVolume`: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	return &lib.RPCResponse{
		Error:  nil,
		Result: true,
		Log:    "success",
	}, nil
}

// removeNonNumeric removes all non-numeric characters from a string
func removeNonNumeric(s string) string {
	reg := regexp.MustCompile("[^0-9]+")
	return reg.ReplaceAllString(s, "")
}

package cpi

import (
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/*
Arguments:
  disk_cid [String]: Cloud ID of the disk to update; returned from create_disk.
  new_size [Integer]: New disk size in MiB.
  cloud_properties [Hash]: New cloud properties for the disk. The properties are specific to the IaaS and are opaque to the Director. The Director does not validate the properties and passes them to the CPI as-is.

Result:
  true/false - No return value expected
*/

// UpdateDisk handles the "update_disk" method of the RPC
func (c *CPI) UpdateDisk(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	return lib.WrapErrorInResponse(lib.MethodNotSupportedErrorMessage, req.GetLoggingContext())
}

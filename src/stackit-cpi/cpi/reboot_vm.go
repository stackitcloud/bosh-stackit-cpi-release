package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/*
Reboots the VM.
Assume that VM can be either be powered on or off at the time of the call.
Waiting for the VM to finish rebooting is not required because the Director waits until the Agent on the VM responds back.

Arguments:
  vm_cid [String]: Cloud ID of the VM to reboot; returned from create_vm.

Result:
  true/false - No result is expected.
*/

// RebootVM handles the "reboot_vm" method of the RPC
func (c *CPI) RebootVM(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.RebootVMArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("Error unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	err = c.validateRebootVMArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Rebooting server", "server_id", args.ServerId)
	err = c.SClient.RebootVM(args.ServerId)
	if err != nil {
		err = fmt.Errorf("cpi.RebootVM(): [iaas API] Error when calling `RebootServer`: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	c.Log.Debug("Triggered reboot of server", "server_id", args.ServerId)

	return &lib.RPCResponse{
		Error:  nil,
		Result: true,
		Log:    "success",
	}, nil
}

func (c *CPI) validateRebootVMArgs(args *lib.RebootVMArgs) error {
	if args.ServerId == "" {
		return fmt.Errorf("vm_cid must be provided")
	}
	return nil
}

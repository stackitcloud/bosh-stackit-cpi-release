package cpi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/*
Checks for VM presence in the IaaS.
This method is mostly used by the consistency check tool (cloudcheck) to determine if the VM still exists.

Arguments:
  vm_cid [String]: Cloud ID of the VM to check; returned from create_vm.

Result:
  exists [Boolean]: True if VM is present.
*/

// HasVM handles the "has_vm" method of the RPC
func (c *CPI) HasVM(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.HasVMArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	err = c.validateHasVMArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Checking for presence of server", "server_id", args.ServerId)
	server, err := c.SClient.GetVM(args.ServerId)
	if err != nil {
		c.Log.Error("Error when calling GetServer", "error", err)
		if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
			c.Log.Debug("VM returned 404 from IaaS", "server_id", args.ServerId)
			return &lib.RPCResponse{
				Error:  nil,
				Result: false,
				Log:    "success",
			}, nil
		} else {
			err = fmt.Errorf("cpi.HasVM(): [iaas API] Error when calling `GetServer`: %w", err)
			return lib.WrapErrorInResponse(err, req.GetLoggingContext())
		}
	}
	_, resultValue := server.GetIdOk()

	return &lib.RPCResponse{
		Error:  nil,
		Result: resultValue,
		Log:    "success",
	}, nil
}

func (c *CPI) validateHasVMArgs(args *lib.HasVMArgs) error {
	if args.ServerId == "" {
		return fmt.Errorf("vm_cid must be provided")
	}
	return nil
}

package cpi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/* Deletes VM.
 * To avoid losing track of VMs, make sure to raise an error if VM deletion is not absolutely certain.

Arguments:
  vm_cid [String]: Cloud ID of the VM to delete; returned from create_vm.

Result:
  No return value
*/

// DeleteVM handles the "delete_vm" method of the RPC
func (c *CPI) DeleteVM(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.DeleteVMArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	err = c.validateDeleteVMArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	vm, err := c.SClient.GetVM(args.ServerId)
	if err != nil {
		if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
			c.Log.Debugf("vm with id: %s not found, assuming nothing to do", args.ServerId)
			return &lib.RPCResponse{
				Error:  nil,
				Result: true,
				Log:    "success",
			}, nil
		}
		err = fmt.Errorf("failed to get VM info VM: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("Deleting VM", "server_id", args.ServerId)
	err = c.DeleteServer(vm)
	if err != nil {
		err = fmt.Errorf("failed to delete VM: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	c.AttemptToCleanOrphanedNICS(lib.NicData{Networks: vm.GetNics()})
	c.Log.Info("VM has been successfully deleted", "server_id", args.ServerId)

	return &lib.RPCResponse{
		Error:  nil,
		Result: true,
		Log:    "success",
	}, nil
}

func (c *CPI) DeleteServer(vm *iaas.Server) error {
	err := c.SClient.DeleteVM(vm.GetId())
	if err != nil {
		// Check for specific error types that might indicate the VM is already gone
		return fmt.Errorf("error triggering VM deletion: %w", err)
	}

	c.Log.Debugf("waiting for server: '%s' to be deleted", vm.GetId())
	// Wait for deletion to complete
	err = c.SClient.WaitForVMDeletion(vm.GetId())
	if err != nil {
		return fmt.Errorf("error waiting for VM deletion: %w", err)
	}

	return err
}

func (c *CPI) validateDeleteVMArgs(args *lib.DeleteVMArgs) error {
	if args.ServerId == "" {
		return fmt.Errorf("cpi.DeleteVM(): vm_cid (server id) must be provided")
	}
	return nil
}

package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/* Deletes a network.
 * To avoid losing track of networks, make sure to raise an error if network deletion is not absolutely certain.

Arguments:
  network_cid [String]: Cloud ID of the network to delete; returned from create_network.

Result:
  true/false - No return value expected
*/

// DeleteNetwork handles the "delete_network" method of the RPC
func (c *CPI) DeleteNetwork(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.DeleteNetworkArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("Error unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	err = c.validateDeleteNetworkArgs(&args)
	if err != nil {
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("Deleting network", "network_id", args.NetworkId)

	// Delete the network with retries
	err = c.deleteNetwork(args.NetworkId)
	if err != nil {
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("Network has been successfully deleted", "network_id", args.NetworkId)
	return &lib.RPCResponse{
		Error:  nil,
		Result: true,
	}, nil
}

func (c *CPI) deleteNetwork(networkID string) error {
	c.Log.Debug("Deleting network", "network_id", networkID)
	err := c.SClient.DeleteNetwork(networkID)
	if err != nil {
		return fmt.Errorf("[IaaS API] Error when calling `DeleteNetwork`: %w", err)
	}

	// Wait for the network to be deleted using the real wait handler
	c.Log.Debug("Waiting for network to be deleted", "network_id", networkID)
	err = c.SClient.WaitForDeleteNetwork(networkID)
	if err != nil {
		return fmt.Errorf("[IaaS API] Error when waiting for network deletion: %w", err)
	}

	return nil
}

func (c *CPI) validateDeleteNetworkArgs(args *lib.DeleteNetworkArgs) error {
	if args.NetworkId == "" {
		return fmt.Errorf("invalid arguments: network_id must be provided")
	}
	return nil
}

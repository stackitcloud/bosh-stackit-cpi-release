package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/* Creates a network that will be used to place VMs on.

Properties required for creating the network.
It may contain range and gateway keys.
A cloud_properties is required to provide information specific to the CPI and target IaaS.

Arguments:
  {
    type: String (required)
    cloud_properties: Hash (required)
    range: String (optional)
    gateway: String (optional)
    netmask_bits: Integer (optional)
  }
Result:
  Array with the following format: [network_id (string), addresses (hash), cloud properties (hash)]
*/

// CreateNetwork handles the "create_network" method of the RPC
func (c *CPI) CreateNetwork(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.CreateNetworkArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("invalid arguments, expected a hash for network: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	err = args.Validate()
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Creating network", "type", args.Type, "range", args.Range, "gateway", args.Gateway, "netmask_bits", args.NetmaskBits)
	c.Log.Debug("Network cloud properties", "properties", args.Properties)

	createNetworkPayload := iaas.CreateNetworkPayload{
		Name: args.Type,
		Dhcp: utils.Ptr(true),
		Ipv4: &iaas.CreateNetworkIPv4{
			CreateNetworkIPv4WithPrefix: &iaas.CreateNetworkIPv4WithPrefix{
				Prefix:  args.Range,
				Gateway: *iaas.NewNullableString(&args.Gateway),
			},
		},
	}
	if len(args.Properties.Nameservers) > 1 {
		ipv4 := createNetworkPayload.GetIpv4()
		ipv4.CreateNetworkIPv4WithPrefix.Nameservers = args.Properties.Nameservers
		createNetworkPayload.Ipv4 = &ipv4
	}

	network, err := c.SClient.CreateNetwork(createNetworkPayload)
	if err != nil {
		err = fmt.Errorf("[iaas API] Error when calling `CreateNetwork`: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	c.Log.Infof("Waiting for network '%s' to be created", network.GetId())

	network, err = c.SClient.WaitForCreateNetwork(network.GetId())
	if err != nil {
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debugf("Succesfully created network '%s'", network.GetId())

	return &lib.RPCResponse{
		Error: nil,
		Result: []any{
			network.GetId(),
			map[string]any{
				"range":    args.Range,
				"gateway":  *network.GetIpv4().Gateway.Get(),
				"reserved": []string{},
			},
			map[string]any{
				"dns": network.GetIpv4().Nameservers,
			},
		},
		Log: "success",
	}, nil
}

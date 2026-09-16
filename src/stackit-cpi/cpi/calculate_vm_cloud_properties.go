package cpi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/*
Returns a hash that can be used as VM cloud_properties when calling create_vm;
it describes the IaaS instance type closest to the arguments passed.
The cloud_properties returned are IaaS-specific.
For example, when querying the AWS CPI for a VM with the parameters
{ "cpu": 1, "ram": 512, "ephemeral_disk_size": 1024 }, it will return the following,
which includes a t2.nano instance type which has 1 CPU and 512MB RAM:
  {
    "instance_type": "t2.nano",
    "ephemeral_args.": { "size": 1024 }
  }
calculate_vm_cloud_properties returns the minimum resources that satisfy the parameters,
which may result in a larger machine than expected.
For example, when querying the AWS CPI for a VM with the parameters
{ "cpu": 1, "ram": 8192, "ephemeral_disk_size": 4096},
it will return an m4.large instance type (which has 2 CPUs) because it is the smallest instance type
which has at least 8 GiB RAM.

If a parameter is set to a value greater than what is available (e.g. 1024 CPUs), an error is raised.


Arguments:
  desired_instance_size [Hash]: Parameters of the desired size of the VM consisting of the following keys:
    cpu [Integer]: Number of virtual cores desired
    ram [Integer]: Amount of RAM, in MiB (i.e. 4096 for 4 GiB)
    ephemeral_disk_size [Integer]: Size of ephemeral disk, in MB

Result:
  cloud_properties [Hash]: an IaaS-specific set of cloud properties that define the size of the VM.

Example API Request:
{
  "ram": 1024,
  "cpu": 2,
  "ephemeral_disk_size": 2048
}
*/

// CalculateVMCloudProperties handles the "calculate_vm_cloud_properties" method of the RPC
func (c *CPI) CalculateVMCloudProperties(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.CalculateVMCloudPropertiesArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: expected a hash for desired_instance_size: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	err = c.validateCalculateVMCloudPropertiesArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	availableMachineTypes, err := c.SClient.ListMachineTypes()
	if err != nil {
		err = fmt.Errorf("failed querying the IaaS api for available machine types %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	matchingTypes := []iaas.MachineType{}
	deprecatedMatchingTypes := []iaas.MachineType{}
	// find all that match cpu & ram
	for _, availableType := range availableMachineTypes {
		if _, ok := availableType.GetExtraSpecs()["gpu"]; ok {
			// skip gpu types to avoid accidential cost, if someone wants a GPU type they better create a cloud config.
			continue
		}
		// if the arch isn't x86, we cannot use it. Bosh doesn't support arm stemcells
		cpuVal, ok := availableType.GetExtraSpecs()["cpu"]
		if !ok {
			// skip this one because we cannot know what cpu it has
			continue
		}
		cpuString, ok := cpuVal.(string)
		if !ok {
			continue
		}

		if strings.HasPrefix(strings.ToLower(cpuString), "amd") || strings.HasPrefix(strings.ToLower(cpuString), "intel") {
			if availableType.GetRam() == args.RAM && availableType.GetVcpus() == args.CPU {
				if strings.Contains(strings.ToLower(availableType.GetDescription()), "deprecated") {
					deprecatedMatchingTypes = append(deprecatedMatchingTypes, availableType)
					continue
				}
				matchingTypes = append(matchingTypes, availableType)
			}
		}
	}

	switch len(matchingTypes) {
	case 0:
		if len(deprecatedMatchingTypes) > 0 {
			deprecatedNames := []string{}
			for _, mType := range deprecatedMatchingTypes {
				deprecatedNames = append(deprecatedNames, mType.GetName())
			}
			c.Log.Debugf("found %d deprecated matching types: '%s', while these should not be used, they could unblock your request", len(deprecatedMatchingTypes), deprecatedNames)
			err = fmt.Errorf("could not find a matching vm_type with '%d' CPU and '%d' RAM but found '%d' deprecated matching types: '%s'", args.CPU, args.RAM, len(deprecatedNames), strings.Join(deprecatedNames, ","))
		} else {
			err = fmt.Errorf("could not find a matching vm_type with '%d' CPU and '%d' RAM", args.CPU, args.RAM)
		}
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())

	case 1:
		return &lib.RPCResponse{
			Error: nil,
			Log:   "success",
			Result: map[string]any{
				"instance_type":  matchingTypes[0].GetName(),
				"ephemeral_disk": map[string]any{"size": 1024},
			},
		}, nil

	default:
		// pick first dedicated
		for i, matchingType := range matchingTypes {
			if overCommit, ok := matchingType.GetExtraSpecs()["overcommit"]; ok {
				if overCommit == "1" {
					return &lib.RPCResponse{
						Error: nil,
						Log:   "success",
						Result: map[string]any{
							"instance_type":  matchingTypes[i].GetName(),
							"ephemeral_disk": map[string]any{"size": 1024},
						},
					}, nil
				}
			}
		}

	}
	// pick first available if we didn't find a dedicated
	return &lib.RPCResponse{
		Error: nil,
		Log:   "success",
		Result: map[string]any{
			"instance_type":  matchingTypes[0].GetName(),
			"ephemeral_disk": map[string]any{"size": 1024},
		},
	}, nil
}

func (c *CPI) validateCalculateVMCloudPropertiesArgs(args *lib.CalculateVMCloudPropertiesArgs) error {
	// max value of cpu & ram will be determined by IaaS
	if args.CPU < 1 {
		return fmt.Errorf("desired_instance_size.cpu must be > 0")
	}
	if args.RAM < 1 {
		return fmt.Errorf("desired_instance_size.ram must be > 0")
	}
	if args.EphemeralDiskSize < 1 {
		return fmt.Errorf("desired_instance_size.ephemeral_disk_size must be > 0")
	}
	return nil
}

package cpi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/*
Sets VM's metadata to make it easier for operators to categorize VM-based resources when looking at the IaaS management console.
We recommend to set VM name based on the passed name key (note name may be missing).
What is called “metadata” here is actually implemented as tags by most CPIs.
The following default tags are forced by Bosh, whatever happens.
These cannot be overridden.
{
  "director":       "director name", // the director name, as shown by `bosh env`
  "deployment":     "traefik",       // the name of deployment to which the 'instance_group' belongs
  "instance_group": "traefik",       // the name of the group to which the (VM) instance belongs
  "job":            "traefik",       // 'job' is an old alias for 'instance_group', here it holds the exact same value
  "id":    "c50f32a0-65f4-40a2-9dde-bff12560c14d",         // the stable, unique ID of the (VM) instance in its group
  "name":  "traefik/c50f32a0-65f4-40a2-9dde-bff12560c14d", // the full name of the (VM) instance, composed of '<instance_group>/<instance_id>'
  "index": "1",                                            // the human-readable identifier of the (VM) instance in its instance group
  "created_at": "YYYY-MM-DDThh:mm:ssZ"                     // the last time at which `set_vm_metadata` method has been called on the (VM) instance
}

Arguments:
  vm_cid [String]: Cloud ID of the VM to modify; returned from create_vm.
  metadata [Hash]: Collection of key-value pairs, including the top-level tags in the deployment manifest. CPI should not rely on presence of specific keys.

*/

// SetVMMetadata handles the "set_vm_metadata" method of the RPC
func (c *CPI) SetVMMetadata(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.SetVMMetadataArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	if args.Metadata != nil && len(args.Metadata) == 0 {
		// no metadata values, nothing to do
		return &lib.RPCResponse{
			Error:  nil,
			Log:    "success",
			Result: nil,
		}, nil
	}
	err = args.Validate()
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.sanitizeVMMetadata(&args)
	name := args.Metadata[lib.VMNameKey]
	payload := iaas.UpdateServerPayload{
		Labels: args.Metadata,
	}
	if name != nil {
		nameString, ok := name.(string)
		if ok {
			payload.SetName(nameString)
		} else {
			c.Log.Debug("could not convert value from %v to string", name)
		}
	}

	c.Log.Debug("Updating metadata for server", "server_id", args.ServerId, "payload", payload)

	_, err = c.SClient.UpdateVM(args.ServerId, payload)
	if err != nil {
		err = fmt.Errorf("[iaas API] Error when calling `UpdateServer`: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Debug("Metadata for server has been successfully updated", "server_id", args.ServerId)

	return &lib.RPCResponse{
		Error:  nil,
		Result: nil,
		Log:    "success",
	}, nil
}

func (c *CPI) sanitizeVMMetadata(args *lib.SetVMMetadataArgs) {
	// Process metadata
	c.Log.Debug("Setting metadata for server", "server_id", args.ServerId, "metadata", args.Metadata)

	// if bosh provided a timestamp, reformat it in a way accepted by the iaas.
	if timestamp, ok := args.Metadata[lib.BoshCreatedAtKeyName].(string); ok {
		t, err := time.Parse(lib.BoshTimeStampFormat, timestamp)
		if err != nil {
			c.Log.Warnf("failed converting bosh provided timestamp: '%s' to iaas accepted format: '%s': %s", timestamp, lib.IaasTimestampFormat, err)
		}
		args.Metadata[lib.BoshCreatedAtKeyName] = t.UTC().Format(lib.IaasTimestampFormat)
	}
	var name string
	if c.Config.HumanReadableVMNames {
		var ok bool
		name, ok = args.Metadata[lib.VMNameKey].(string)
		if !ok {
			c.Log.Errorf("tag value for key: %s could not parsed as string", lib.VMNameKey)
		} else {
			cleanName := strings.ReplaceAll(name, "/", "-")
			c.Log.Infof("VMName was modified from '%s' to '%s'", name, cleanName)
			name = cleanName
			// always shorten to 63 chars
			if len(name) > lib.ServerNameMaxLength {
				name = name[:lib.ServerNameMaxLength-1]
				c.Log.Infof("shortened name to %d chars", lib.ServerNameMaxLength)
			}

		}
	} else {
		// this is the bosh instance id, not the agent id. this id matches the id you see in the instance name of `bosh is` output
		name = fmt.Sprintf("vm-%s", args.Metadata[lib.BoshInstanceIDKey])
	}
	args.Metadata[lib.VMNameKey] = name

	if c.Config.AutoFixNameAndLabels {
		cleanName := lib.MakeIaasCompatibleServerNameString(name)
		if name != cleanName {
			c.Log.Infof("VMName was modified from '%s' to '%s'", name, cleanName)
			args.Metadata[lib.VMNameKey] = cleanName
		}

		// loop through all strings in metadata and limit them to 63 characters for the IaaS regex, update to 128 in the future
		for key, v := range args.Metadata {
			// skip name, it was already handled separately
			if key == lib.VMNameKey {
				continue
			}
			if valueString, ok := v.(string); ok {
				cleanKey := lib.MakeIaasCompatibleTagKeyString(key)
				if key != cleanKey {
					c.Log.Infof("Metadata key was modified from '%s' to '%s'", key, cleanKey)
				}
				cleanValue := lib.MakeIaasCompatibleTagValueString(valueString)
				if cleanValue != valueString {
					c.Log.Infof("Metadata value for '%s' was modified from '%s' to '%s'", cleanKey, valueString, cleanValue)
				}
			}
		}
	}
}

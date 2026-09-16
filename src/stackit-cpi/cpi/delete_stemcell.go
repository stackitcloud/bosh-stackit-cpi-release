package cpi

import (
	"encoding/json"
	"fmt"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

/*
STDIN RPC Request Example / Signature:
{
  "method": "delete_stemcell",
  "arguments": [
    "647e6336-f6d9-44a3-a74f-f6594d58738b"
  ],
  "context": {
    "director_uuid": "ae5a9879-2fe8-4692-b374-5228f9be8bc5",
    "request_id": "cpi-900830"
  },
  "api_version": 2
}

STDOUT RPC Response Example / Signature:
{
  "result": true,
  "error": null,
  "log": ""
}

DeleteStemcell deletes a stemcell image from the infrastructure.
The method expects a single argument:
  - stemcell_cid [String]: Cloud ID of the stemcell to delete.
*/

// DeleteStemcell handles the "delete_stemcell" method of the RPC
func (c *CPI) DeleteStemcell(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.DeleteStemcellArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed Unmarshalling Arguments JSON Array: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	err = c.validateDeleteStemcellArgs(&args)
	if err != nil {
		err = fmt.Errorf("invalid arguments: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	// Create context with timeout based on config
	c.Log.Info("Deleting stemcell", "stemcell_id", args.StemcellId)

	err = c.deleteStemcell(args.StemcellId)
	if err != nil {
		err = fmt.Errorf("failed to delete stemcell: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("Stemcell has been successfully deleted", "stemcell_id", args.StemcellId)

	return &lib.RPCResponse{
		Error:  nil,
		Result: true,
		Log:    "success",
	}, nil
}

func (c *CPI) deleteStemcell(imageID string) error {
	err := c.SClient.DeleteStemcell(imageID)
	if err != nil {
		return fmt.Errorf("deleteStemcellAttempt(): error triggering deletion: %w", err)
	}

	c.Log.Debug("Deleting image", "image_id", imageID)

	err = c.SClient.WaitForStemcellDelete(imageID)
	if err != nil {
		return fmt.Errorf("cpi.deleteStemcellAttempt(): error waiting for stemcell deletion: %w", err)
	}

	return nil
}

func (c *CPI) validateDeleteStemcellArgs(args *lib.DeleteStemcellArgs) error {
	if !lib.IsValidUUID(args.StemcellId) {
		return fmt.Errorf("deleteStemcellAttempt(): imageId: '%s' is not a valid uuid", args.StemcellId)
	}
	return nil
}

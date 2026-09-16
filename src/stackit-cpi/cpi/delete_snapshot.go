// Package cpi provides BOSH CPI implementation for STACKIT Cloud
package cpi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

// DeleteSnapshot deletes a snapshot with the provided snapshot_cid
// Arguments: [snapshot_cid]
// Response: true on success
func (c *CPI) DeleteSnapshot(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	// Parse arguments
	var args lib.DeleteSnapshotArgs
	if err = json.Unmarshal(req.Arguments, &args); err != nil {
		err = fmt.Errorf("failed unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	// Validate arguments
	snapshotID := args.SnapshotId
	if snapshotID == "" {
		err = errors.New("invalid arguments: snapshot_cid must be provided")
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("Deleting snapshot", "snapshot_id", snapshotID)

	// Delete snapshot
	err = c.SClient.DeleteSnapshot(snapshotID)
	if err != nil {
		if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusConflict) {
			err = fmt.Errorf("snapshot %s is in use and cannot be deleted: %w", snapshotID, err)
			return lib.WrapErrorInResponse(err, req.GetLoggingContext())
		}

		err = fmt.Errorf("failed to delete snapshot %s: %w", snapshotID, err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	// If we get here, deletion was successful
	c.Log.Info("Successfully deleted snapshot", "snapshot_id", snapshotID)
	return &lib.RPCResponse{
		Error:  nil,
		Result: true,
		Log:    "success",
	}, nil
}

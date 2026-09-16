package cpi

import "github.com/stackitcloud/stackit-cpi/cpi/lib"

/*
STDIN RPC Request Example / Signature:
{
  "method": "info",
  "arguments": [],
  "context": {
    "director_uuid": "ae5a9879-2fe8-4692-b374-5228f9be8bc5",
    "request_id": "cpi-162766"
  },
  "api_version": 2
}
STDOUT RPC Response Example / Signature:
{
  "result": {
    "api_version": 2,
    "stemcell_formats": [
      "openstack-raw",
      "openstack-qcow2",
      "openstack-light"
    ]
  },
  "error": null,
  "log": ""
}
*/

// Info handles the "info" method of the RPC
func (c *CPI) Info(request lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	// Populate the response object of the information from the config, plus API version 2, and stemcell formats of stackit-qcow2.
	return &lib.RPCResponse{
		Result: map[string]any{
			"api_version": 2,
			"stemcell_formats": []string{
				"qcow2",
				"raw",
				"iso",
				"openstack-qcow2",
			},
			"config": map[string]any{
				"region_id":   c.Config.RegionID,
				"project_id":  c.Config.ProjectID,
				"timeout":     c.Config.Timeout,
				"retry_count": c.Config.RetryCount,
				"log_level":   c.Config.LogLevel,
			},
			"cpi_info": map[string]any{
				"build_project":     BuildProject,
				"build_version":     getCpiVersion(),
				"build_date":        getCpiZuluTimeStamp(BuildDate),
				"build_vcs_id":      BuildVcsId,
				"build_vcs_id_date": getCpiZuluTimeStamp(BuildVcsIdDate),
				"build_vcs_url":     BuildVcsUrl,
				"build_go_arch":     BuildGoArch,
				"build_go_os":       BuildGoOs,
				"build_go_version":  getCpiGoVersion(),
			},
			"context": map[string]any{
				"director_uuid": request.Context.DirectorUUID,
				"request_id":    request.Context.RequestID,
			},
		},
	}, nil
}

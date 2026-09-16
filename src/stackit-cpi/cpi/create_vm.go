package cpi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	agentinfra "github.com/cloudfoundry/bosh-agent/v2/infrastructure"
	agentsettings "github.com/cloudfoundry/bosh-agent/v2/settings"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

/*
Arguments:
- agent_id [String]: ID selected by the Director for the VM's agent.
- stemcell_cid [String]: Cloud ID of the stemcell to use as a base image for new VM.
- cloud_properties [Hash]: Cloud properties hash specified in the deployment manifest under the VM's resource pool.
- networks [Hash]: Networks hash that specifies which VM networks must be configured.
- disk_cids [Array of strings] Array of disk cloud IDs for the disks that the created VM will most likely attach.
- environment [Hash]: Resource pool's env hash specified in the deployment manifest, including initial properties.

Result:
  Array of results
  vm_cid [String]: Cloud ID of the created VM.
  networks [Hash]: Networks associated with the VM.
*/

func getDefaultTags(directorUUID string, args lib.CreateVMArgs) map[string]any {
	// on initial creation the bosh director will not provide any contextual info for the vm ( e.g.
	// instance group name, jobs, etc). but we want some debug information in case something between
	// creating the vm and setting the metadata goes wrong and a manual cleanup is required.
	// since the actual vm-name will only be computable when set_vm_metadata runs, we just use the
	// agent-id prefixed by `vm-` because that satisfies the iaas naming requirements
	tags := map[string]any{
		lib.VMNameKey:               fmt.Sprintf("vm-%s", args.AgentID),
		lib.CreatedByBoshLabelKey:   lib.CreatedByBoshLabelValue,
		lib.BoshDirectorUUIDKeyName: directorUUID,
		lib.BoshCreatedAtKeyName:    time.Now().UTC().Format(lib.IaasTimestampFormat),
		lib.AgentIDKeyName:          args.AgentID,
		lib.StemcellIDKeyName:       args.StemcellID,
		lib.AzTagKeyName:            args.Properties.AvailabilityZone,
	}
	if args.Properties.LoadBalancer.Name != "" && len(args.Properties.LoadBalancer.TargetPools) > 0 {
		tags[lib.AttachedToLoadBalancerKey] = args.Properties.LoadBalancer.Name
	}

	// Add user-defined tags
	for k, v := range args.Env.Tags {
		tags[k] = v
	}

	return tags
}

// CreateVM handles the "create_vm" method of the RPC
func (c *CPI) CreateVM(req lib.RPCRequest) (resp *lib.RPCResponse, err error) {
	var args lib.CreateVMArgs
	err = json.Unmarshal(req.Arguments, &args)
	if err != nil {
		err = fmt.Errorf("failed unmarshalling Arguments JSON: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	tags := getDefaultTags(req.Context.DirectorUUID, args)
	nics, networks, err := c.createAllRequiredNICs(args, tags)
	if err != nil {
		c.AttemptToCleanOrphanedNICS(lib.NicData{Nics: slices.Collect(maps.Values(nics))})
		err = fmt.Errorf("failed to create nics: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	vmUserData, err := c.GetAgentSettings(tags["name"].(string), args, agentsettings.Disks{
		System:     "/dev/vda",
		Ephemeral:  nil,
		Persistent: nil,
	}, networks)
	if err != nil {
		err = fmt.Errorf("failed to generate agent settings: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	userDataBytes, err := json.Marshal(vmUserData)
	if err != nil {
		err = fmt.Errorf("failed to marshal agent settings to bytes: %w", err)
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}
	userDataBase64String := base64.StdEncoding.EncodeToString(userDataBytes)

	var perfClass string
	if args.Properties.RootDisk.PerformanceClass != "" {
		perfClass = args.Properties.RootDisk.PerformanceClass
	} else {
		perfClass = c.Config.DefaultRootVolumeType // Use default performance class if not specified
	}
	if args.Properties.RootDisk.Size < lib.MinRootDiskSize {
		c.Log.Debugf("increasing root disk size to from: %d minimum value: %d", args.Properties.RootDisk.Size, lib.MinRootDiskSize)
		args.Properties.RootDisk.Size = lib.MinRootDiskSize
	}

	nicIds := []string{}
	for netName, nic := range nics {
		nicIds = append(nicIds, nic.GetId())
		net := args.Networks[netName]
		net.IP = nic.GetIpv4()
		args.Networks[netName] = net
	}

	c.Log.Debugf("Generated nics for server: %s", nicIds)
	// Create the VM server payload

	secGrps := append(c.Config.DefaultSecurityGroups, args.Properties.SecurityGroups...)
	secGrps = append(secGrps, networks.CollectSecGrps()...)
	resolvedSecGrpIDs, err := c.ResolveSecurityGroups(secGrps)
	if err != nil {
		c.Log.Debug("failed resolving secGrps: %s, %s", secGrps, err)
		err = fmt.Errorf("failed resolving secGrps: %s, %w", secGrps, err)
		c.AttemptToCleanOrphanedNICS(lib.NicData{Nics: slices.Collect(maps.Values(nics))})
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())

	}

	c.Log.Infof(`Creating VM: '%s' with stemcell: '%s' instance_type: '%s' in zone: '%s'`, tags["name"].(string), args.StemcellID, args.Properties.InstanceType, args.Properties.AvailabilityZone)
	createServerPayload := iaas.CreateServerPayload{
		// Substitute '/' in name with '_', also the regex rules for '-' don't allow multiple in UUID
		Name:             tags["name"].(string), // Ensure name is valid for API
		AvailabilityZone: &args.Properties.AvailabilityZone,
		MachineType:      args.Properties.InstanceType,
		Labels:           tags,
		SecurityGroups:   resolvedSecGrpIDs,
		BootVolume: &iaas.BootVolume{
			Size: utils.Ptr(int64(args.Properties.RootDisk.Size)),
			Source: &iaas.BootVolumeSource{
				Id:   args.StemcellID, // Use stemcell_cid as source
				Type: "image",
			},
			PerformanceClass:    &perfClass,
			DeleteOnTermination: utils.Ptr(true),
		},
		Networking: iaas.CreateServerPayloadAllOfNetworking{
			CreateServerNetworkingWithNics: &iaas.CreateServerNetworkingWithNics{
				NicIds: nicIds,
			},
		},
		UserData: utils.Ptr(userDataBase64String),
	}

	if c.Config.DefaultSSHKeyName != "" {
		createServerPayload.KeypairName = &c.Config.DefaultSSHKeyName
	}

	c.Log.Debug("Generated iaas CreateServer data", "payload", createServerPayload)
	var server *iaas.Server

	retryer := &lib.Retryer{
		Workflow: func() error {
			server, err = c.createServer(createServerPayload)
			return err
		},
		OnFail: func() error {
			return c.DeleteServer(server)
		},
		Attempts: c.Config.RetryCount + 1,
		RetryAbleErrorCheck: []func(error) bool{
			lib.IsNonGenericBuildAbortedNetworkError,
		},
		Logger: c.Log.ChildLogger("create_vm_retryer"),
	}

	err = retryer.Do()
	if err != nil {
		c.Log.Debug("VM creation failed", "error", err)
		err = fmt.Errorf("failed to create VM: %w", err)
		c.AttemptToCleanOrphanedNICS(lib.NicData{Nics: slices.Collect(maps.Values(nics))})
		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	c.Log.Info("VM created successfully", "server_id", server.GetId())

	primaryNetworkName := lib.FindDefaultNetworkName(networks)
	var primaryNic *iaas.NIC
	if primaryNetworkName == "" {
		// technically this should never be the case because bosh would error if an instance
		// has more than one network configured, but none is default.
		// fall back to first interface we get
		c.Log.Debug("could not find default network", "networks", networks)
		primaryNic = slices.Collect(maps.Values(nics))[0]
		c.Log.Debug("falling back to nicId: '%s' nicIP: '%s'", primaryNic.GetId(), primaryNic.GetIpv4())
	} else {
		primaryNic = nics[primaryNetworkName]
	}
	err = c.attachToLoadBalancerWithRetries(*primaryNic, args.Properties.LoadBalancer)
	if err != nil {
		delErr := c.DeleteServer(server)
		if delErr != nil {
			c.Log.Warn(fmt.Sprintf("failed cleaning partially created vm with id: %s", server.GetId()), "error", err)
		}
		c.AttemptToCleanOrphanedNICS(lib.NicData{Networks: server.GetNics()})

		return lib.WrapErrorInResponse(err, req.GetLoggingContext())
	}

	// Set API response with server ID and networks (formatted as required by BOSH CPI v2)
	return &lib.RPCResponse{
		Error:  nil,
		Result: []any{server.GetId(), args.Networks},
		Log:    "success",
	}, nil
}

func (c *CPI) attachToLoadBalancerWithRetries(nic iaas.NIC, lbProperties lib.LoadBalancer) error {
	for _, targetPoolProperties := range lbProperties.TargetPools {
		// add additional retries to account for race conditions when configuring multiple targets on the
		// same lb with concurrent calls to the cpi by the director.
		c.Log.Debugf("getting LB with name: '%s' for attachment", lbProperties.Name)
		lb, err := c.SClient.GetLB(lbProperties.Name)
		if err != nil {
			return err
		}

		for attempt := range c.Config.RetryCount {

			c.Log.Debugf("attemp %v/%v attaching IP: %s to LB with name: '%s'", attempt+1, c.Config.RetryCount+1, nic.GetIpv4(), lbProperties.Name)
			err = c.SClient.AttachNICToLB(targetPoolProperties, nic, lb)
			if err != nil {
				return err
			}
			// because race conditions and the way the loadbalancer API works, we should confirm our call succeeded.
			c.Log.Debugf("waiting for %v ms before confirming targets of %s", lib.JitterWait(2000), lbProperties.Name)
			lb, err = c.SClient.GetLB(lbProperties.Name)
			if err != nil {
				return err
			}
			if lib.HasTargetInGroup(lb, targetPoolProperties.Name, nic.GetIpv4()) {
				break
			}
			// try again
			c.Log.Debug("attachment race condition, failed to confirm target presence. desync with %s ms", "nic", nic.GetId(), "lb", lbProperties.Name, "attempt", attempt, lib.JitterWait(2000))
			continue
		}
	}

	return nil
}

func (c *CPI) createServer(createServerPayload iaas.CreateServerPayload) (server *iaas.Server, err error) {
	c.Log.Info("starting create server")
	server, err = c.SClient.CreateVM(createServerPayload)
	if err != nil {
		c.Log.Error("failed creating server %s", err)
		return nil, err
	}

	c.Log.Info("waiting for server to finish creating", "server_id", server.GetId())
	err = c.SClient.WaitForVM(server.GetId())
	if err != nil {
		c.Log.Errorf("failed waiting for server: %s", err)
		return server, err
	}

	c.Log.Info("ensuring security groups are attached for server", "server_id", server.GetId())
	// technically this should not be necessary. But it was observerd that *sometimes* the created VMs
	// did not have their sec groups configured. This ensures they are attached.
	err = c.attachAllSecurityGroups(createServerPayload.GetSecurityGroups(), server.GetId())
	if err != nil {
		c.Log.Error("failed ensuring security groups for server %s: %s", server.GetId(), err)
		return server, err
	}
	return server, nil
}

func (c *CPI) attachAllSecurityGroups(secGrpIDs []string, vmID string) error {
	for _, secGrpID := range secGrpIDs {
		err := c.SClient.AttachSecGrpToVM(secGrpID, vmID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CPI) createAllRequiredNICs(
	args lib.CreateVMArgs,
	tags map[string]any,
) (
	nics map[string]*iaas.NIC,
	networks lib.Networks,
	err error,
) {
	var (
		vipNet, primaryNet         *lib.Network
		vipNetName, primaryNetName string
	)
	nics = make(map[string]*iaas.NIC)

	networks = args.Networks
	for netName, network := range networks {
		if err != nil {
			return nil, nil, err
		}
		if network.IsVIP() {
			vipNet = &network
			vipNetName = netName
			continue
		}
		if network.IsDefaultFor("gateway") {
			primaryNet = &network
			primaryNetName = netName
		}
		nicPayload := iaas.CreateNicPayload{
			Labels:         tags,
			Name:           utils.Ptr(fmt.Sprintf("%s-%s", netName, args.AgentID)),
			SecurityGroups: []string{},
		}

		// TODO figure out v6 support
		withIP := ""
		if network.IP != "" {
			nicPayload.Ipv4 = utils.Ptr(network.IP)
			withIP = fmt.Sprintf("with ip :%s ", network.IP)
		}
		var nic *iaas.NIC
		c.Log.Debugf("Creating nic %s for network %s", withIP, network.Properties.NetID)
		nic, err = c.SClient.CreateNic(nicPayload, network.Properties.NetID)
		if err != nil {
			// if we see a 409 here, an interface with the specified IP already exists. We can check if it's currently unattached
			if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusConflict) {
				c.Log.Debugf("NIC for IP %s couldn't be created because the IP is already in use. Checking if the existing NIC is available", network.IP)
				nic, err = c.findNic(network.IP, network.Properties.NetID)
			}
			if err != nil {
				return nil, nil, err
			}
		}
		c.Log.Debug("updating network", "name", netName, "nic", nic)
		network.IP = nic.GetIpv4()
		nics[netName] = nic
		network.Mac = nic.GetMac()
		networks[netName] = network
	}

	if vipNet != nil {
		c.Log.Infof("configuring vip network for %s", tags["name"])
		vipNet.IP, err = c.AttachPubIPToNIC(nics[primaryNetName], *vipNet, *primaryNet, tags)
		networks[vipNetName] = *vipNet
	}
	return nics, networks, err
}

func (c *CPI) findNic(ip, netID string) (*iaas.NIC, error) {
	netNics, err := c.SClient.ListNics(netID)
	if err != nil {
		c.Log.Debugf("Failed listing nics if NIC is available: %s", err.Error())
		return nil, err
	}
	for _, nic := range netNics {
		if nic.GetIpv4() == ip && nic.GetStatus() == lib.NICAvailableStatus {
			return &nic, nil
		}
	}

	return nil, fmt.Errorf("no matching NIC with ip: '%s' in network with id: '%s' and status: '%s' found", ip, netID, lib.NICAvailableStatus)
}

func (c *CPI) AttachPubIPToNIC(nic *iaas.NIC, vipNet, primaryNetwork lib.Network, tags map[string]any) (string, error) {
	var pubIP *iaas.PublicIp
	var err error
	if vipNet.IP != "" {
		c.Log.Debugf("attaching provided ip: %s", vipNet.IP)

		pubIP, err = c.SClient.GetPublicIPByAddr(vipNet.IP)
		if err != nil {
			return "", err
		}
		tags[lib.DynamicVIPKey] = "false"

		if pubIPInterface, ok := pubIP.GetNetworkInterfaceOk(); ok {
			if pubIPInterface == nic.Id {
				return pubIP.GetIp(), nil
			}
		}
		pubIPPayload := iaas.UpdatePublicIPPayload{
			Labels:           tags,
			NetworkInterface: *iaas.NewNullableString(nic.Id),
		}
		_, err = c.SClient.UpdatePublicIP(pubIPPayload, pubIP.GetId())
		if err != nil {
			return "", err
		}
		return pubIP.GetIp(), nil
	}
	c.Log.Debugf("requesting public ip for nic: %s", nic.GetId())
	// if bosh does manage the IP, we should clean it up on vm deletion.
	// update the tags to leave a cleanup note
	tags[lib.DynamicVIPKey] = "true"

	pubIPPayload := iaas.CreatePublicIPPayload{
		Labels:           tags,
		NetworkInterface: *iaas.NewNullableString(nic.Id),
	}
	pubIP, err = c.SClient.CreatePublicIP(pubIPPayload)
	if err != nil {
		return "", err
	}
	c.Log.Debugf("succesfully created public ip: '%s' for nic: '%s'", pubIP.GetId(), nic.GetId())

	return pubIP.GetIp(), nil
}

func (c *CPI) GetAgentSettings(
	serverName string,
	args lib.CreateVMArgs,
	disks agentsettings.Disks,
	networks lib.Networks,
) (agentinfra.UserDataContentsType, error) {
	type server struct {
		Name string
	}
	type dns struct {
		Nameserver []string
	}
	// this will only have the URLs set. It is set on job level of the cpi in the director
	// deployment.

	if len(args.Env.Bosh.Blobstores) == 0 {
		return agentinfra.UserDataContentsType{}, lib.ErrNoBlobstoreInAgentSettings
	}
	mbusConfig := c.Config.Agent.MBus
	if len(mbusConfig.URLs) == 0 {
		return agentinfra.UserDataContentsType{}, lib.ErrNoMbusURLInAgentSettings
	}
	agentNetworks := networks.ToAgentNetworks()
	userData := agentinfra.UserDataContentsType{
		Server: server{
			Name: serverName,
		},
		DNS: dns{
			Nameserver: networks.CollectDNSServers(),
		},
		Settings: agentsettings.Settings{
			AgentID:   args.AgentID,
			Blobstore: args.Env.Bosh.Blobstores[0],
			Disks:     disks,
			Mbus:      mbusConfig.URLs[0],
			Networks:  agentNetworks,
			NTP:       c.Config.NTPConfig,
			VM:        agentsettings.VM{Name: serverName},
			Env: agentsettings.Env{
				Bosh: args.Env.Bosh,
			},
		},
	}
	// we need to check the args.env data from the director for certs that the agent on the VM should use
	if args.Env.Bosh.Mbus.Cert.Certificate != "" && args.Env.Bosh.Mbus.Cert.CA != "" && args.Env.Bosh.Mbus.Cert.PrivateKey != "" {
		mbusConfig.Cert = args.Env.Bosh.Mbus.Cert
	}

	if c.Config.DefaultSSHKeyName != "" {
		c.Log.Debugf("getting configured default ssh key: %s", c.Config.DefaultSSHKeyName)
		authorizedKey, err := c.SClient.GetPublicKey(c.Config.DefaultSSHKeyName)
		if err != nil {
			return userData, fmt.Errorf("failed getting configured ssh key '%s': %w", c.Config.DefaultSSHKeyName, err)
		}
		userData.Env.Bosh.AuthorizedKeys = append(userData.Env.Bosh.AuthorizedKeys, authorizedKey)
		c.Log.Debugf("finished configuring default ssh key: %s", c.Config.DefaultSSHKeyName)
	}

	return userData, nil
}

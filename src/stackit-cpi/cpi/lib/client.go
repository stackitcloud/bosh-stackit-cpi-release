package lib

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api/wait"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

type StackitClient struct {
	loadbalancer loadbalancer.APIClient
	server       iaas.APIClient
	projectID    string
	region       string
	timeout      int
	waitTimeout  int
	logger       LoggerInterface
}

func NewStackitClient(cpiConfig Config, logger LoggerInterface, opts []config.ConfigurationOption) (StackitClient, error) {
	serverClient, err := iaas.NewAPIClient(opts...)
	if err != nil {
		return StackitClient{}, err
	}
	lbClient, err := loadbalancer.NewAPIClient(opts...)
	if err != nil {
		return StackitClient{}, err
	}
	return StackitClient{
		projectID:    cpiConfig.ProjectID,
		region:       cpiConfig.RegionID,
		timeout:      cpiConfig.Timeout,
		waitTimeout:  cpiConfig.LRPTimeout,
		server:       *serverClient,
		loadbalancer: *lbClient,
		logger:       logger,
	}, nil
}

func (client StackitClient) ListAZs() ([]string, error) {
	ctx, cancel := client.getConfiguredContext()
	var err error
	var azs *iaas.AvailabilityZoneListResponse
	defer cancel()
	azs, err = client.server.DefaultAPI.ListAvailabilityZones(ctx, client.region).Execute()
	if err != nil {
		return nil, err
	}
	if items, ok := azs.GetItemsOk(); ok {
		return items, nil
	}
	return nil, ErrItemsNotOK
}

func (client StackitClient) ListMachineTypes() ([]iaas.MachineType, error) {
	ctx, cancel := client.getConfiguredContext()
	var machinetypes *iaas.MachineTypeListResponse
	var err error
	defer cancel()
	machinetypes, err = client.server.DefaultAPI.ListMachineTypes(ctx, client.projectID, client.region).Execute()
	if err != nil {
		return nil, err
	}
	if items, ok := machinetypes.GetItemsOk(); ok {
		return items, nil
	}
	return nil, ErrItemsNotOK
}

func (client StackitClient) GetPublicKey(name string) (string, error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	var key *iaas.Keypair
	var err error
	key, err = client.server.DefaultAPI.GetKeyPair(ctx, name).Execute()
	if err != nil {
		return "", err
	}
	if key.GetPublicKey() == "" {
		return "", fmt.Errorf("iaas returned an empty public key for key with name: '%s'", name)
	}
	return key.GetPublicKey(), nil
}

func (client StackitClient) CreatePublicIP(pubIPPayload iaas.CreatePublicIPPayload) (ip *iaas.PublicIp, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.CreatePublicIP(ctx, client.projectID, client.region).CreatePublicIPPayload(pubIPPayload).Execute()
}

func (client StackitClient) UpdatePublicIP(pubIPPayload iaas.UpdatePublicIPPayload, pubIPID string) (ip *iaas.PublicIp, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.UpdatePublicIP(ctx, client.projectID, client.region, pubIPID).UpdatePublicIPPayload(pubIPPayload).Execute()
}

func (client StackitClient) ListNics(networkID string) ([]iaas.NIC, error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	var err error
	var nics *iaas.NICListResponse

	nics, err = client.server.DefaultAPI.ListNics(ctx, client.projectID, client.region, networkID).Execute()
	if err != nil {
		return nil, err
	}

	if items, ok := nics.GetItemsOk(); ok {
		return items, nil
	}

	return nil, ErrItemsNotOK
}

func (client StackitClient) CreateNic(createNicPayload iaas.CreateNicPayload, networkID string) (nic *iaas.NIC, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	return client.server.DefaultAPI.CreateNic(ctx, client.projectID, client.region, networkID).CreateNicPayload(createNicPayload).Execute()
}

func (client StackitClient) AttachSecGrpToVM(secGrpID, vmID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	err = client.server.DefaultAPI.AddSecurityGroupToServer(ctx, client.projectID, client.region, vmID, secGrpID).Execute()

	// a 409 indicates the secGrp is already attached.
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusConflict) {
		return nil
	}
	return err
}

func (client StackitClient) WaitForVMDeletion(vmID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	_, err = wait.DeleteServerWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, vmID).WaitWithContext(ctx)
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) WaitForVM(vmID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	client.logger.Debug("starting to wait for", "server_id", vmID, "region", client.region)
	_, err = wait.CreateServerWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, vmID).WaitWithContext(ctx)
	return err
}

func (client StackitClient) UpdateVM(vmID string, updateServerPayload iaas.UpdateServerPayload) (server *iaas.Server, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.UpdateServer(ctx, client.projectID, client.region, vmID).UpdateServerPayload(updateServerPayload).Execute()
}

func (client StackitClient) CreateVM(createServerPayload iaas.CreateServerPayload) (server *iaas.Server, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.CreateServer(ctx, client.projectID, client.region).CreateServerPayload(createServerPayload).Execute()
}

func (client StackitClient) DeleteVM(vmID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	err = client.server.DefaultAPI.DeleteServer(ctx, client.projectID, client.region, vmID).Execute()
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) WaitForDeleteDisk(volumeID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	_, err = wait.DeleteVolumeWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, volumeID).WaitWithContext(ctx)
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) WaitForCreateDisk(volumeID string) (volume *iaas.Volume, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	return wait.CreateVolumeWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, volumeID).WaitWithContext(ctx)
}

func (client StackitClient) GetDisk(volumeID string) (volume *iaas.Volume, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.GetVolume(ctx, client.projectID, client.region, volumeID).Execute()
}

func (client StackitClient) DeleteDisk(volumeID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	err = client.server.DefaultAPI.DeleteVolume(ctx, client.projectID, client.region, volumeID).Execute()
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) CreateDisk(createVolumePayload iaas.CreateVolumePayload) (volume *iaas.Volume, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.CreateVolume(ctx, client.projectID, client.region).CreateVolumePayload(createVolumePayload).Execute()
}

func (client StackitClient) DetachDiskFromVM(volumeID, vmID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	attached, err := client.server.DefaultAPI.ListAttachedVolumes(ctx, client.projectID, client.region, vmID).Execute()
	if err != nil {
		// no vm nothing to detach
		if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
			return nil
		}
		return fmt.Errorf("failed discovering attached volumes: %w", err)
	}
	for _, currentVol := range attached.GetItems() {
		if currentVol.GetVolumeId() == volumeID {
			return client.server.DefaultAPI.RemoveVolumeFromServer(ctx, client.projectID, client.region, vmID, volumeID).Execute()
		}
	}
	return nil
}

func (client StackitClient) WaitForDetachDiskFromVM(volumeID, vmID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	_, err = wait.RemoveVolumeFromServerWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, vmID, volumeID).WaitWithContext(ctx)
	return err
}

func (client StackitClient) UpdateDisk(volumeID string, updateVolumePayload iaas.UpdateVolumePayload) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	_, err = client.server.DefaultAPI.UpdateVolume(ctx, client.projectID, client.region, volumeID).UpdateVolumePayload(updateVolumePayload).Execute()
	return err
}

func (client StackitClient) ResizeDisk(volumeID string, resizeVolumePayload iaas.ResizeVolumePayload) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	return client.server.DefaultAPI.ResizeVolume(ctx, client.projectID, client.region, volumeID).ResizeVolumePayload(resizeVolumePayload).Execute()
}

func (client StackitClient) AttachDiskToVM(volumeID, vmID string) error {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	_, err := client.server.DefaultAPI.AddVolumeToServer(ctx, client.projectID, client.region, vmID, volumeID).AddVolumeToServerPayload(iaas.AddVolumeToServerPayload{}).Execute()
	return err
}

func (client StackitClient) WaitForAttachDiskToVM(volumeID, vmID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	volAttach, err := wait.AddVolumeToServerWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, vmID, volumeID).WaitWithContext(ctx)
	if err != nil {
		return err
	}

	if volAttach.GetServerId() == "" || volAttach.GetVolumeId() == "" {
		return fmt.Errorf("the response is missing the volume_id: '%s', or the server_id: '%s'", volAttach.GetVolumeId(), volAttach.GetServerId())
	}

	return nil
}

func (client StackitClient) GetPublicIPByAddr(addr string) (*iaas.PublicIp, error) {
	var publicIPs []iaas.PublicIp
	var err error
	publicIPs, err = client.ListPublicIPs()
	if err != nil {
		return nil, err
	}
	for _, publicIP := range publicIPs {
		if publicIP.GetIp() == addr {
			return &publicIP, nil
		}
	}
	return nil, fmt.Errorf("no public IP with addr: '%s' found: %w", addr, err)
}

func (client StackitClient) DeletePublicIP(publicIPID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	err = client.server.DefaultAPI.DeletePublicIP(ctx, client.projectID, client.region, publicIPID).Execute()
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) GetNic(networkID, nicID string) (nic *iaas.NIC, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.GetNic(ctx, client.projectID, client.region, networkID, nicID).Execute()
}

func (client StackitClient) DeleteNic(networkID, nicID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	err = client.server.DefaultAPI.DeleteNic(ctx, client.projectID, client.region, networkID, nicID).Execute()
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) getConfiguredLRPContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(client.waitTimeout)*time.Second)
}

func (client StackitClient) getConfiguredContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(client.timeout)*time.Second)
}

func (client StackitClient) RebootVM(vmID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.RebootServer(ctx, client.projectID, client.region, vmID).Execute()
}

func (client StackitClient) GetVM(vmID string) (server *iaas.Server, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.GetServer(ctx, client.projectID, client.region, vmID).Details(true).Execute()
}

func (client StackitClient) ListPublicIPs() (ips []iaas.PublicIp, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	publicIPs, err := client.server.DefaultAPI.ListPublicIPs(ctx, client.projectID, client.region).Execute()
	if err != nil {
		return nil, err
	}

	if items, ok := publicIPs.GetItemsOk(); ok {
		return items, nil
	}

	return nil, ErrItemsNotOK
}

func (client StackitClient) GetServer(serverID string) (server *iaas.Server, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.GetServer(ctx, client.projectID, client.region, serverID).Details(true).Execute()
}

func (client StackitClient) GetAZForServer(vmID string) (azName string, err error) {
	var server *iaas.Server
	server, err = client.GetServer(vmID)
	if err != nil {
		return "", err
	}
	if azName, ok := server.GetAvailabilityZoneOk(); ok {
		return *azName, nil
	}
	return "", ErrItemsNotOK
}

func (client StackitClient) ListSecurityGroups() (secGrps []iaas.SecurityGroup, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	list, err := client.server.DefaultAPI.ListSecurityGroups(ctx, client.projectID, client.region).Execute()
	if err != nil {
		return nil, err
	}
	var ok bool
	if secGrps, ok = list.GetItemsOk(); ok {
		return secGrps, nil
	}
	return nil, ErrItemsNotOK
}

func (client StackitClient) DeleteStemcell(imageID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	err = client.server.DefaultAPI.DeleteImage(ctx, client.projectID, client.region, imageID).Execute()
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) GetStemcell(imageID string) (image *iaas.Image, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.GetImage(ctx, client.projectID, client.region, imageID).Execute()
}

func (client StackitClient) ListStemcells() (images []iaas.Image, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	var imageListResp *iaas.ImageListResponse

	imageListResp, err = client.server.DefaultAPI.ListImages(ctx, client.projectID, client.region).LabelSelector(fmt.Sprintf("%s=%s", CreatedByBoshLabelKey, CreatedByBoshLabelValue)).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	if images, ok := imageListResp.GetItemsOk(); ok {
		return images, nil
	}

	return nil, ErrItemsNotOK
}

func (client StackitClient) WaitForStemcellDelete(imageID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	_, err = wait.DeleteImageWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, imageID).WaitWithContext(ctx)
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) WaitForStemcellCreate(imageID string) (image *iaas.Image, err error) {
	ctx, cancel := client.getConfiguredLRPContext()
	defer cancel()
	return wait.UploadImageWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, imageID).WaitWithContext(ctx)
}

func (client StackitClient) CreateStemcell(createImagePayload iaas.CreateImagePayload) (image *iaas.ImageCreateResponse, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.CreateImage(ctx, client.projectID, client.region).CreateImagePayload(createImagePayload).Execute()
}

func (client StackitClient) CreateSnapshot(createSnaphotPayload *iaas.CreateSnapshotPayload) (snapshot *iaas.Snapshot, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.CreateSnapshot(ctx, client.projectID, client.region).CreateSnapshotPayload(*createSnaphotPayload).Execute()
}

func (client StackitClient) DeleteSnapshot(snapshotID string) error {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	err := client.server.DefaultAPI.DeleteSnapshot(ctx, client.projectID, client.region, snapshotID).Execute()
	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) WaitForCreateNetwork(networkID string) (network *iaas.Network, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return wait.CreateNetworkWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, networkID).WaitWithContext(ctx)
}

func (client StackitClient) CreateNetwork(createNetworkPayload iaas.CreateNetworkPayload) (network *iaas.Network, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.CreateNetwork(ctx, client.projectID, client.region).CreateNetworkPayload(createNetworkPayload).Execute()
}

func (client StackitClient) GetNetwork(networkID string) (network *iaas.Network, err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	return client.server.DefaultAPI.GetNetwork(ctx, client.projectID, client.region, networkID).Execute()
}

func (client StackitClient) DeleteNetwork(networkID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	err = client.server.DefaultAPI.DeleteNetwork(ctx, client.projectID, client.region, networkID).Execute()

	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

func (client StackitClient) WaitForDeleteNetwork(networkID string) (err error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()
	_, err = wait.DeleteNetworkWaitHandler(ctx, client.server.DefaultAPI, client.projectID, client.region, networkID).WaitWithContext(ctx)

	if ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return nil
	}
	return err
}

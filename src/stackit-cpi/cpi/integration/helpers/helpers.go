// Package helpers contains testhelpers
package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	agentsettings "github.com/cloudfoundry/bosh-agent/v2/settings"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

const (
	DefaultNetName       = "default"
	ExternalNetName      = "vip"
	DefaultManualNetName = "manual"
	DefaultMachinetype   = "t2i.1"
)

type Network struct {
	ID      string
	Gateway string
	CIDR    string
}
type TestProject struct {
	Region           string            `json:"region"`
	ProjectID        string            `json:"project_id"`
	SaKeyJSON        map[string]any    `json:"sa_key_json"`
	StageProfileJSON map[string]string `json:"stage_profile_json"`
	StemcellID       string            `json:"stemcell_id"`
	Networks         struct {
		Default Network `json:"default"`
		Manual  Network `json:"manual"`
	} `json:"networks"`
	Jumpbox struct {
		PrivateKeyPemBlock string `json:"private_ssh_key"`
		PublicIP           string `json:"public_ip"`
	} `json:"jumphost"`
}
type Metadata struct {
	StemcellURL string `json:"stemcell_url"`
}
type TestConfig struct {
	Prod TestProject `json:"prod"`
	Qa   TestProject `json:"qa"`
	Meta Metadata    `json:"meta"`
}

func (p TestProject) GetSaKeyString() string {
	bytes, _ := json.Marshal(p.SaKeyJSON)
	return string(bytes)
}

func (p TestProject) GetStageProfileString() string {
	bytes, _ := json.Marshal(p.StageProfileJSON)
	return string(bytes)
}

func (p TestProject) DefaultCreateVMArgs(secGrp *iaas.SecurityGroup) lib.CreateVMArgs {
	return lib.CreateVMArgs{
		AgentID:    uuid.NewString(),
		StemcellID: p.StemcellID,
		Env: lib.Environment{
			Bosh: agentsettings.BoshEnv{
				Blobstores: []agentsettings.Blobstore{
					{
						Type: "local",
						Options: map[string]any{
							"blobstore_path": "/var/vcap/micro_bosh/data/cache",
						},
					},
				},
			},
		},
		Properties: lib.VMProperties{
			AvailabilityZone: fmt.Sprintf("%s-%v", p.Region, rand.Intn(3)+1),
			InstanceType:     DefaultMachinetype,
			RootDisk: lib.VMRootDisk{
				Size:             32,
				PerformanceClass: "storage_premium_perf2",
			},
			SecurityGroups: []string{secGrp.GetId()},
		},

		Networks: lib.Networks{
			DefaultNetName: lib.Network{
				Network: agentsettings.Network{
					Type:    "dynamic",
					UseDHCP: true,
					Default: []string{"gateway", "dns"},
					DNS:     []string{"8.8.8.8"},
				},
				Properties: lib.NetProperties{
					NetID:          p.Networks.Default.ID,
					SecurityGroups: []string{secGrp.GetId()},
					Nameservers:    []string{"8.8.8.8"},
				},
			},
		},
		DiskIds: []string{},
	}
}

func (p TestProject) GenerateTestSecurityGroup() *iaas.SecurityGroup {
	client := p.GetStackitIaasClient()
	secGrp, err := client.DefaultAPI.CreateSecurityGroup(context.Background(), p.ProjectID, p.Region).CreateSecurityGroupPayload(iaas.CreateSecurityGroupPayload{
		Name:     fmt.Sprintf("test-%s", uuid.NewString()),
		Stateful: utils.Ptr(true),
	}).Execute()
	Expect(err).ToNot(HaveOccurred())

	_, err = client.DefaultAPI.CreateSecurityGroupRule(context.Background(), p.ProjectID, p.Region, secGrp.GetId()).CreateSecurityGroupRulePayload(iaas.CreateSecurityGroupRulePayload{
		Description: utils.Ptr("test-ssh"),
		Direction:   "ingress",
		PortRange: &iaas.PortRange{
			Min: int64(22),
			Max: int64(22),
		},
		Ethertype: utils.Ptr("IPv4"),
		Protocol: &iaas.CreateProtocol{
			String: utils.Ptr("tcp"),
		},
	}).Execute()
	Expect(err).ToNot(HaveOccurred())
	_, err = client.DefaultAPI.CreateSecurityGroupRule(context.Background(), p.ProjectID, p.Region, secGrp.GetId()).CreateSecurityGroupRulePayload(iaas.CreateSecurityGroupRulePayload{
		Description: utils.Ptr("test-agent-create-env"),
		Direction:   "ingress",
		PortRange: &iaas.PortRange{
			Min: int64(6868),
			Max: int64(6868),
		},
		Ethertype: utils.Ptr("IPv4"),
		Protocol: &iaas.CreateProtocol{
			String: utils.Ptr("tcp"),
		},
	}).Execute()
	Expect(err).ToNot(HaveOccurred())

	return secGrp
}

func (p TestProject) DefaultTestCPI() *cpi.CPI {
	c := CustomCPI(
		WithDefaults(),
		WithProject(p),
		WithDefaultLogger(),
	)
	Expect(c.InitAPIClient()).To(Succeed())

	return c
}

func GenerateRequest(method string, args any) lib.RPCRequest {
	argsBytes, err := json.Marshal(args)
	Expect(err).ToNot(HaveOccurred())
	return lib.RPCRequest{
		Method:     method,
		Arguments:  argsBytes,
		APIVersion: 2,
		Context: lib.RPCContext{
			DirectorUUID: uuid.NewString(),
			RequestID:    uuid.NewString(),
			Config:       lib.Config{},
			VM:           lib.VMContext{},
		},
	}
}

func CleanupSecGrp(testSecGrp *iaas.SecurityGroup, testCPI *cpi.CPI, iaasSDKClient *iaas.APIClient) {
	Eventually(func() error {
		err := iaasSDKClient.DefaultAPI.DeleteSecurityGroup(context.Background(), testCPI.Config.ProjectID, testCPI.Config.RegionID, testSecGrp.GetId()).Execute()
		if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
			return nil
		}
		return err
	},
		"2m").Should(Succeed())
}

func CleanupVM(vmID string, testCPI cpi.CPI) {
	if vmID == "" {
		return
	}
	vm, err := testCPI.SClient.GetVM(vmID)
	if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
		return
	}
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		testCPI.AttemptToCleanOrphanedNICS(lib.NicData{Networks: vm.GetNics()})
		return testCPI.DeleteServer(vm)
	}, "2m").Should(Succeed())
}

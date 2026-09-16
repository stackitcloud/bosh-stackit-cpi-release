package lib

import (
	"slices"

	agentsettings "github.com/cloudfoundry/bosh-agent/v2/settings"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

type NicData struct {
	Nics     []*iaas.NIC
	Networks []iaas.ServerNetwork
}

func (n Networks) CollectSecGrps() []string {
	secGrps := []string{}
	for _, net := range n {
		secGrps = append(secGrps, net.Properties.SecurityGroups...)
	}
	slices.Sort(secGrps)
	return slices.Compact(secGrps)
}

func (n Networks) CollectDNSServers() []string {
	for _, network := range n {
		if network.IsDefaultFor("dns") {
			return network.DNS
		}
	}
	return []string{}
}

func (n Networks) ToAgentNetworks() agentsettings.Networks {
	agentNetworks := agentsettings.Networks{}
	for name, net := range n {
		net.UseDHCP = true
		agentNetworks[name] = net.Network
	}
	return agentNetworks
}

func FindDefaultNetworkName(networks Networks) string {
	for name, network := range networks {
		if network.IsDefaultFor("gateway") {
			return name
		}
	}
	return ""
}

func FindVipNetwork(networks Networks) *Network {
	for _, network := range networks {
		if network.IsVIP() {
			return &network
		}
	}
	return nil
}

func HasTargetInGroup(lb *loadbalancer.LoadBalancer, targetGroupName string, nicIP string) bool {
	for _, pool := range lb.GetTargetPools() {
		for _, target := range pool.GetTargets() {
			if target.GetIp() == nicIP {
				return true
			}
		}
	}
	return false
}

func HasTarget(lb *loadbalancer.LoadBalancer, nicIP string) bool {
	for _, pool := range lb.GetTargetPools() {
		for _, target := range pool.GetTargets() {
			if target.GetIp() == nicIP {
				return true
			}
		}
	}
	return false
}

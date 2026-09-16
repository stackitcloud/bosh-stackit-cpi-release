package lib

import (
	"fmt"

	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

func (client StackitClient) GetLB(name string) (*loadbalancer.LoadBalancer, error) {
	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	return client.loadbalancer.DefaultAPI.GetLoadBalancer(ctx, client.projectID, client.region, name).Execute()
}

func (client StackitClient) DetachNICFromLB(nic iaas.NIC, lb *loadbalancer.LoadBalancer) error {
	var err error

	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	targetPools := lb.GetTargetPools()

	for _, pool := range targetPools {

		var newTargets []loadbalancer.Target
		for _, target := range pool.GetTargets() {
			if target.GetIp() == nic.GetIpv4() {
				continue
			}
			newTargets = append(newTargets, target)
		}
		if len(newTargets) != len(pool.GetTargets()) {

			_, err = client.loadbalancer.DefaultAPI.UpdateTargetPool(ctx, client.projectID, client.region, lb.GetName(), pool.GetName()).UpdateTargetPoolPayload(loadbalancer.UpdateTargetPoolPayload{
				ActiveHealthCheck:  pool.ActiveHealthCheck,
				Name:               pool.Name,
				SessionPersistence: pool.SessionPersistence,
				TargetPort:         pool.TargetPort,
				Targets:            newTargets,
			}).Execute()
			if err != nil {
				return fmt.Errorf("failed detaching nic '%s' from pool '%s' on lb '%s': %w", nic.GetId(), pool.GetName(), lb.GetName(), err)
			}
		}
	}

	return nil
}

func (client StackitClient) AttachNICToLB(targetPoolProperties TargetPool, nic iaas.NIC, lb *loadbalancer.LoadBalancer) error {
	matchingPoolFound := false

	ctx, cancel := client.getConfiguredContext()
	defer cancel()

	targetPools := lb.GetTargetPools()
outer:
	for _, pool := range targetPools {
		if pool.GetName() != targetPoolProperties.Name {
			continue
		}
		matchingPoolFound = true
		for _, target := range pool.GetTargets() {
			if target.GetIp() == nic.GetIpv4() {
				// nothing to do, target ip already exists
				continue outer
			}
		}

		newTargets := append(pool.GetTargets(), loadbalancer.Target{
			DisplayName: nic.Id,
			Ip:          nic.Ipv4,
		})

		pool.SetTargets(newTargets)

		_, err := client.loadbalancer.DefaultAPI.UpdateTargetPool(ctx, client.projectID, client.region, lb.GetName(), pool.GetName()).UpdateTargetPoolPayload(loadbalancer.UpdateTargetPoolPayload{
			ActiveHealthCheck:  pool.ActiveHealthCheck,
			Targets:            newTargets,
			Name:               pool.Name,
			SessionPersistence: pool.SessionPersistence,
			TargetPort:         pool.TargetPort,
		}).Execute()
		if err != nil {
			return err
		}
	}
	if !matchingPoolFound {
		return fmt.Errorf("target pool with name: '%s' could not be found on lb '%s'", targetPoolProperties.Name, lb.GetName())
	}
	return nil
}

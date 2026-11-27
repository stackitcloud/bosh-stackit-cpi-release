package integration

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/stackitcloud/stackit-cpi/cpi/integration/helpers"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

var _ = AfterSuite(func() {
	sweepTestRunResources()
})

// sweepTestRunResources queries the Stackit API for every resource tagged with
// TESTRUN_UUID (direct SDK-created resources) or director_uuid (CPI-created
// resources) and best-effort deletes them. Cleanup failures are logged but
// never fail the suite; this is a safety net for leftovers from failed tests.
func sweepTestRunResources() {
	if TestRunUUID == "" {
		fmt.Println("WARN: sweep skipped, TestRunUUID not set")
		return
	}
	projects := []TestProject{conf.Prod, conf.Qa}
	regions := dedupe([]string{conf.Prod.Region, conf.Qa.Region, "eu02"})

	for _, project := range projects {
		iaasClient := newIaasClient(project)
		if iaasClient == nil {
			continue
		}
		lbClient := newLBClient(project)
		for _, region := range regions {
			fmt.Printf("sweeping %s/%s for TESTRUN_UUID=%s\n", project.ProjectID, region, TestRunUUID)
			sweepSecurityGroups(iaasClient, project.ProjectID, region)
			sweepServers(iaasClient, project.ProjectID, region)
			sweepVolumes(iaasClient, project.ProjectID, region)
			sweepPublicIPs(iaasClient, project.ProjectID, region)
			sweepNetworks(iaasClient, project.ProjectID, region)
			sweepKeyPairs(iaasClient, project.ProjectID, region)
			sweepImages(iaasClient, project.ProjectID, region)
			if lbClient != nil {
				sweepLoadBalancers(lbClient, project.ProjectID, region)
			}
		}
	}
}

func newIaasClient(project TestProject) *iaas.APIClient {
	saKeyBytes, err := json.Marshal(project.SaKeyJSON)
	if err != nil {
		fmt.Printf("WARN: failed to marshal SA key for sweep: %v\n", err)
		return nil
	}
	opts := []config.ConfigurationOption{config.WithServiceAccountKey(string(saKeyBytes))}
	if len(project.StageProfileJSON) != 0 {
		opts = append(opts, config.WithEndpoint(project.StageProfileJSON[lib.IaasEndpointKey]), config.WithTokenEndpoint(project.StageProfileJSON[lib.TokenEndpointKey]))
	}
	client, err := iaas.NewAPIClient(opts...)
	if err != nil {
		fmt.Printf("WARN: failed to create iaas client for sweep: %v\n", err)
		return nil
	}
	return client
}

func newLBClient(project TestProject) *loadbalancer.APIClient {
	saKeyBytes, err := json.Marshal(project.SaKeyJSON)
	if err != nil {
		fmt.Printf("WARN: failed to marshal SA key for LB sweep: %v\n", err)
		return nil
	}
	opts := []config.ConfigurationOption{config.WithServiceAccountKey(string(saKeyBytes))}
	if len(project.StageProfileJSON) != 0 {
		opts = append(opts, config.WithEndpoint(project.StageProfileJSON[lib.IaasEndpointKey]), config.WithTokenEndpoint(project.StageProfileJSON[lib.TokenEndpointKey]))
	}
	client, err := loadbalancer.NewAPIClient(opts...)
	if err != nil {
		fmt.Printf("WARN: failed to create loadbalancer client for sweep: %v\n", err)
		return nil
	}
	return client
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func sweepSecurityGroups(client *iaas.APIClient, projectID, region string) {
	for _, selector := range []string{testRunSelector(), directorSelector()} {
		groups, err := client.DefaultAPI.ListSecurityGroups(context.Background(), projectID, region).LabelSelector(selector).Execute()
		if err != nil {
			fmt.Printf("WARN: failed to list security groups (%s): %v\n", selector, err)
			continue
		}
		for _, g := range groups.GetItems() {
			if err := client.DefaultAPI.DeleteSecurityGroup(context.Background(), projectID, region, g.GetId()).Execute(); err != nil {
				fmt.Printf("WARN: failed to delete security group %s: %v\n", g.GetId(), err)
			} else {
				fmt.Printf("sweep: deleted security group %s\n", g.GetId())
			}
		}
	}
}

func sweepServers(client *iaas.APIClient, projectID, region string) {
	for _, selector := range []string{testRunSelector(), directorSelector()} {
		servers, err := client.DefaultAPI.ListServers(context.Background(), projectID, region).LabelSelector(selector).Execute()
		if err != nil {
			fmt.Printf("WARN: failed to list servers (%s): %v\n", selector, err)
			continue
		}
		for _, s := range servers.GetItems() {
			if err := client.DefaultAPI.DeleteServer(context.Background(), projectID, region, s.GetId()).Execute(); err != nil {
				fmt.Printf("WARN: failed to delete server %s: %v\n", s.GetId(), err)
			} else {
				fmt.Printf("sweep: deleted server %s\n", s.GetId())
			}
		}
	}
}

func sweepVolumes(client *iaas.APIClient, projectID, region string) {
	for _, selector := range []string{testRunSelector(), directorSelector()} {
		volumes, err := client.DefaultAPI.ListVolumes(context.Background(), projectID, region).LabelSelector(selector).Execute()
		if err != nil {
			fmt.Printf("WARN: failed to list volumes (%s): %v\n", selector, err)
			continue
		}
		for _, v := range volumes.GetItems() {
			if err := client.DefaultAPI.DeleteVolume(context.Background(), projectID, region, v.GetId()).Execute(); err != nil {
				fmt.Printf("WARN: failed to delete volume %s: %v\n", v.GetId(), err)
			} else {
				fmt.Printf("sweep: deleted volume %s\n", v.GetId())
			}
		}
	}
}

func sweepPublicIPs(client *iaas.APIClient, projectID, region string) {
	for _, selector := range []string{testRunSelector(), directorSelector()} {
		ips, err := client.DefaultAPI.ListPublicIPs(context.Background(), projectID, region).LabelSelector(selector).Execute()
		if err != nil {
			fmt.Printf("WARN: failed to list public IPs (%s): %v\n", selector, err)
			continue
		}
		for _, ip := range ips.GetItems() {
			if err := client.DefaultAPI.DeletePublicIP(context.Background(), projectID, region, ip.GetId()).Execute(); err != nil {
				fmt.Printf("WARN: failed to delete public IP %s: %v\n", ip.GetId(), err)
			} else {
				fmt.Printf("sweep: deleted public IP %s\n", ip.GetId())
			}
		}
	}
}

func sweepNetworks(client *iaas.APIClient, projectID, region string) {
	for _, selector := range []string{testRunSelector(), directorSelector()} {
		networks, err := client.DefaultAPI.ListNetworks(context.Background(), projectID, region).LabelSelector(selector).Execute()
		if err != nil {
			fmt.Printf("WARN: failed to list networks (%s): %v\n", selector, err)
			continue
		}
		for _, n := range networks.GetItems() {
			if err := client.DefaultAPI.DeleteNetwork(context.Background(), projectID, region, n.GetId()).Execute(); err != nil {
				fmt.Printf("WARN: failed to delete network %s: %v\n", n.GetId(), err)
			} else {
				fmt.Printf("sweep: deleted network %s\n", n.GetId())
			}
		}
	}
}

func sweepKeyPairs(client *iaas.APIClient, projectID, region string) {
	for _, selector := range []string{testRunSelector(), directorSelector()} {
		keys, err := client.DefaultAPI.ListKeyPairs(context.Background()).LabelSelector(selector).Execute()
		if err != nil {
			fmt.Printf("WARN: failed to list key pairs (%s): %v\n", selector, err)
			continue
		}
		for _, k := range keys.GetItems() {
			if err := client.DefaultAPI.DeleteKeyPair(context.Background(), k.GetName()).Execute(); err != nil {
				fmt.Printf("WARN: failed to delete key pair %s: %v\n", k.GetName(), err)
			} else {
				fmt.Printf("sweep: deleted key pair %s\n", k.GetName())
			}
		}
	}
}

func sweepImages(client *iaas.APIClient, projectID, region string) {
	images, err := client.DefaultAPI.ListImages(context.Background(), projectID, region).LabelSelector(directorSelector()).Execute()
	if err != nil {
		fmt.Printf("WARN: failed to list images: %v\n", err)
		return
	}
	for _, img := range images.GetItems() {
		if err := client.DefaultAPI.DeleteImage(context.Background(), projectID, region, img.GetId()).Execute(); err != nil {
			fmt.Printf("WARN: failed to delete image %s: %v\n", img.GetId(), err)
		} else {
			fmt.Printf("sweep: deleted image %s\n", img.GetId())
		}
	}
}

func sweepLoadBalancers(client *loadbalancer.APIClient, projectID, region string) {
	lbs, err := client.DefaultAPI.ListLoadBalancers(context.Background(), projectID, region).Execute()
	if err != nil {
		fmt.Printf("WARN: failed to list load balancers: %v\n", err)
		return
	}
	for _, lb := range lbs.GetLoadBalancers() {
		if !lbHasTestRunLabel(lb) {
			continue
		}
		if _, err := client.DefaultAPI.DeleteLoadBalancer(context.Background(), projectID, region, lb.GetName()).Execute(); err != nil {
			fmt.Printf("WARN: failed to delete load balancer %s: %v\n", lb.GetName(), err)
		} else {
			fmt.Printf("sweep: deleted load balancer %s\n", lb.GetName())
		}
	}
}

func lbHasTestRunLabel(lb loadbalancer.LoadBalancer) bool {
	for _, selector := range []string{TestRunUUIDLabelKey, lib.BoshDirectorUUIDKeyName} {
		if v, ok := lb.GetLabels()[selector]; ok && v == TestRunUUID {
			return true
		}
	}
	return false
}

func testRunSelector() string {
	return fmt.Sprintf("%s=%s", TestRunUUIDLabelKey, TestRunUUID)
}

func directorSelector() string {
	return fmt.Sprintf("%s=%s", lib.BoshDirectorUUIDKeyName, TestRunUUID)
}

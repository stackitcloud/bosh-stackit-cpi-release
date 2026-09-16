package helpers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/config"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	loadbalancer "github.com/stackitcloud/stackit-sdk-go/services/loadbalancer/v2api"
)

type CPIOpt func(c *cpi.CPI)

func WithProject(t TestProject) CPIOpt {
	return func(c *cpi.CPI) {
		c.Config.ProjectID = t.ProjectID
		c.Config.RegionID = t.Region
		c.Config.ServiceAccountJSON = t.GetSaKeyString()
		if t.StageProfileJSON != nil {
			c.Config.StageProfileJSON = t.GetStageProfileString()
		} else {
			c.Config.StageProfileJSON = ""
		}
	}
}

func WithRegion(region string) CPIOpt {
	return func(c *cpi.CPI) {
		c.Config.RegionID = region
	}
}

func WithDefaultLogger() CPIOpt {
	return func(c *cpi.CPI) {
		logFile, err := os.CreateTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		fmt.Println("log location", logFile.Name())

		l, err := lib.NewCPILogger(logFile.Name(), slog.LevelDebug, 1, 1, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(l).ToNot(BeNil())
		c.Log = l
	}
}

func WithSpecificLogger(l *lib.CPILogger) CPIOpt {
	return func(c *cpi.CPI) {
		c.Log = l
	}
}

func WithDefaults() CPIOpt {
	return func(c *cpi.CPI) {
		c.Config.RetryCount = 3
		c.Config.Timeout = 300
		c.Config.LRPTimeout = 3600
		c.Config.NTPConfig = []string{"de.pool.ntp.org"}
		c.Config.DefaultSSHKeyName = TestSSHKeyName
		c.Config.HumanReadableVMNames = true
		c.Config.LogLevel = slog.LevelDebug.String()
		c.Config.Agent.MBus.URLs = []string{"http://user:test@0.0.0.0:6868"}
	}
}

func (p TestProject) GetStackitLoadBalancerClient() *loadbalancer.APIClient {
	saKeyBytes, err := json.Marshal(p.SaKeyJSON)

	opts := []config.ConfigurationOption{
		config.WithServiceAccountKey(string(saKeyBytes)),
	}

	if len(p.StageProfileJSON) != 0 {
		opts = append(opts, config.WithEndpoint(p.StageProfileJSON[lib.IaasEndpointKey]), config.WithTokenEndpoint(p.StageProfileJSON[lib.TokenEndpointKey]))
	}
	Expect(err).ToNot(HaveOccurred())
	c, err := loadbalancer.NewAPIClient(opts...)
	Expect(err).ToNot(HaveOccurred())
	return c
}

func (p TestProject) GetStackitIaasClient() *iaas.APIClient {
	saKeyBytes, err := json.Marshal(p.SaKeyJSON)

	opts := []config.ConfigurationOption{
		config.WithServiceAccountKey(string(saKeyBytes)),
	}

	if len(p.StageProfileJSON) != 0 {
		opts = append(opts, config.WithEndpoint(p.StageProfileJSON[lib.IaasEndpointKey]), config.WithTokenEndpoint(p.StageProfileJSON[lib.TokenEndpointKey]))
	}
	Expect(err).ToNot(HaveOccurred())
	c, err := iaas.NewAPIClient(opts...)
	Expect(err).ToNot(HaveOccurred())
	return c
}

// returns a configured but not fully initialized cpi
func CustomCPI(opts ...CPIOpt) *cpi.CPI {
	c := &cpi.CPI{
		Config: lib.Config{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

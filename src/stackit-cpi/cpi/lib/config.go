package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	agentsettings "github.com/cloudfoundry/bosh-agent/v2/settings"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	RegionID                    string     `yaml:"region"`                                                                     // Region for the CPI, ex: eu01, eu02
	ProjectID                   string     `yaml:"project_id"`                                                                 // Project ID for the CPI
	HumanReadableVMNames        bool       `yaml:"human_readable_vm_names,omitempty"`                                          // Whether to use human-readable VM names, default is false
	AutoFixNameAndLabels        bool       `yaml:"auto_fix_name_and_labels,omitempty"`                                         // Whether to attempt to fix input values in Metadata to be accepted by the IaaS API, default is false
	Timeout                     int        `yaml:"timeout,omitempty"`                                                          // Timeout for the CPI in seconds, default is 600 seconds (10 minutes)
	LRPTimeout                  int        `yaml:"lrp_timeout,omitempty"`                                                      // Timeout for the long running processes in the CPI in seconds ( stemcell uploads ), default is 3600 seconds (1 h). The iaas takes a long time for the stemcell to become ready after uploading
	LogFile                     string     `yaml:"log_file,omitempty"`                                                         // Log file path for the CPI, if logging to a file
	RetryCount                  int        `yaml:"retry_count,omitempty"`                                                      // Number of retries for the CPI, default is 3 retries
	LogLevel                    string     `yaml:"log_level,omitempty"`                                                        // Log level for the CPI, default is "info"
	ServiceAccountJSON          string     `yaml:"service_account_json,omitempty"`                                             // The service account key data, if using service account authentication
	StageProfileJSON            string     `yaml:"stage_profile_json,omitempty" json:"stage_profile_json,omitempty"`           // The StageProfileJSON to use to connect to non prod or custom stackit environments
	DefaultRootVolumeType       string     `yaml:"default_root_volume_type,omitempty"`                                         // Default volume type for the CPI, if applicable
	DefaultPersistentVolumeType string     `yaml:"default_persistent_volume_type,omitempty"`                                   // Default volume type for the CPI, if applicable
	DefaultSecurityGroups       []string   `yaml:"default_security_groups,omitempty" json:"default_security_groups,omitempty"` // Default security groups to apply to all VMs (names or UUIDs)
	DefaultSSHKeyName           string     `yaml:"default_ssh_key_name,omitempty" json:"default_ssh_key_name,omitempty"`       // Default public key name to use when creating VMs.
	Agent                       AgentBlock `yaml:"agent,omitempty"`                                                            // Cloud properties for the CPI, if applicable
	NTPConfig                   []string   `yaml:"ntp,omitempty"`                                                              // NTP configuration for the BOSH Agent, if applicable
}

func NewConfig(configFileName string) (cpiConfig *Config, err error) {
	// Validate the config file path
	configFileName = filepath.Clean(configFileName)

	data, err := os.ReadFile(configFileName)
	if err != nil {
		return nil, fmt.Errorf("cpi.Config: failed to read config file: %w", err)
	}

	// Unmarshal the config file
	yamlConfig := &ConfigFile{}

	err = yaml.Unmarshal(data, yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("cpi.Config: failed to unmarshal config file: %w", err)
	}
	cpiConfig = &yamlConfig.CPI
	cpiConfig.Agent = yamlConfig.Agent
	cpiConfig.NTPConfig = yamlConfig.NTPConfig

	// required values
	if cpiConfig.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	if cpiConfig.ServiceAccountJSON == "" {
		return nil, fmt.Errorf("cpi.Config: service_account_json is required")
	}

	if cpiConfig.RegionID == "" {
		cpiConfig.RegionID = "eu01"
	}

	// optional values
	if cpiConfig.LogLevel == "" {
		cpiConfig.LogLevel = "info"
	}

	if cpiConfig.Timeout == 0 {
		cpiConfig.Timeout = 600
	}

	if cpiConfig.LRPTimeout == 0 {
		cpiConfig.LRPTimeout = 3600
	}

	if cpiConfig.RetryCount == 0 {
		cpiConfig.RetryCount = 3
	}

	if cpiConfig.DefaultRootVolumeType == "" {
		// If PerformanceClass is not set, default to "storage_premium_perf2" for root disks
		cpiConfig.DefaultRootVolumeType = "storage_premium_perf2"
	}

	if cpiConfig.DefaultPersistentVolumeType == "" {
		// If PerformanceClass is not set, default to "storage_premium_perf2" for persistent disks
		cpiConfig.DefaultPersistentVolumeType = "storage_premium_perf2"
	}

	// Validate security groups if present
	if len(cpiConfig.DefaultSecurityGroups) > 0 {
		if err := validateSecurityGroups(cpiConfig.DefaultSecurityGroups); err != nil {
			return nil, fmt.Errorf("cpi.Config: invalid default_security_groups: %w", err)
		}
	}

	return cpiConfig, nil
}

// Context key types to avoid using built-in string types as keys
type contextKey string

const (
	requestIDKey    contextKey = "request_id"
	cpiMethodKey    contextKey = "method"
	functionKey     string     = "function"
	directorUUIDKey contextKey = "director_uuid"
)

type AgentBlock struct {
	MBus agentsettings.MBus `yaml:"mbus,omitempty"` // MBus configuration for the BOSH Agent, if applicable
}

type ConfigFile struct {
	CPI       Config     `yaml:"stackit"`
	Agent     AgentBlock `yaml:"agent,omitempty"` // Cloud properties for the CPI, if applicable
	NTPConfig []string   `yaml:"ntp,omitempty"`   // NTP configuration for the BOSH Agent, if applicable
}

func (c Config) String() string {
	c.ServiceAccountJSON = "<redacted>"
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

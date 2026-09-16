package cpi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	config "github.com/stackitcloud/stackit-sdk-go/core/config"
)

/* CPI implements the cpi rpc interface
 * RPC (Remote Procedure Call) structure for BOSH CPI
 */
type CPI struct {
	Config  lib.Config          `json:"cpi_config"` // Configuration for the CPI, RegionID, StageProfileJSON, ProjectID, and ServiceAccountJson
	Log     lib.LoggerInterface `json:"-"`          // Request-scoped logger with context
	SClient lib.IaasClient
}

// ResolveSecurityGroups takes a list of security group identifiers (could be names or UUIDs)
// and returns a list of UUIDs. If a security group is already a UUID, it's returned as-is.
// If it's a name, the function will look it up and return the corresponding UUID.
func (c *CPI) ResolveSecurityGroups(securityGroups []string) ([]string, error) {
	if len(securityGroups) == 0 {
		return []string{}, nil
	}
	c.Log.Debugf("resolving configured security groups: %v", securityGroups)
	// Filter out empty strings and trim whitespace
	filteredGroups := make([]string, 0, len(securityGroups))
	for _, sg := range securityGroups {
		trimmed := strings.TrimSpace(sg)
		if trimmed != "" {
			filteredGroups = append(filteredGroups, trimmed)
		}
	}
	filteredGroups = lib.UniqueArray(filteredGroups)
	c.Log.Debugf("cleaned configured security groups to: %v", filteredGroups)
	if len(filteredGroups) == 0 {
		return []string{}, nil
	}

	resolvedGroups := make([]string, 0, len(filteredGroups))

	// Get all security groups for the project
	c.Log.Debug("discovering all security groups")
	secGroups, err := c.SClient.ListSecurityGroups()
	if err != nil {
		err = fmt.Errorf("failed to list security groups: %w", err)
		c.Log.Debug(err.Error())
		return nil, err
	}

	if len(filteredGroups) > len(secGroups) {
		err = fmt.Errorf("not enough security groups found. %d exist in project, but %d were requested", len(secGroups), len(filteredGroups))

		c.Log.Debug(err.Error())
		return nil, err
	}
	missing := []string{}
	for _, requiredNameOrID := range filteredGroups {
		c.Log.Debugf("resolving security group id for: '%s'", requiredNameOrID)
		found := false
	inner:
		for _, secGrp := range secGroups {
			if secGrp.GetId() == requiredNameOrID || secGrp.GetName() == requiredNameOrID {
				resolvedGroups = append(resolvedGroups, secGrp.GetId())
				found = true
				break inner
			}
		}
		if !found {
			c.Log.Debugf("failed finding security group id for: '%s'", requiredNameOrID)
			missing = append(missing, requiredNameOrID)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("couldn't resolve all required security groups. requested: %s, missing: %s", filteredGroups, missing)
	}

	return lib.UniqueArray(resolvedGroups), nil
}

func (c *CPI) GetUserData(agentConfig string) string {
	c.Log.Debug("agentConfig", "config", agentConfig)
	encoded := base64.StdEncoding.EncodeToString([]byte(agentConfig)) // Encode the agentConfig as base64 for cloud-init

	return encoded
}

// AttemptToCleanOrphanedNICS is used after a VM is already deleted. At this point we could return an error
// and fail the deployment. That would make someone retry the deployment and that would pass because the VM is already
// gone. Once we VM is gone, we have no great way to lookup the NICs that the missing VM had.
// We have no great way of ensuring the nics are gone outside of retrying.
func (c *CPI) AttemptToCleanOrphanedNICS(data lib.NicData) {
	c.Log.Debugf("cleaning %v nic(s)", len(data.Networks)+len(data.Nics))
	allNics := data.Nics

	for _, network := range data.Networks {
		if network.GetPublicIp() != "" {
			c.Log.Debugf("nic: %s has public IP: %s attached", network.GetNicId(), network.GetPublicIp())
			pubIP, err := c.SClient.GetPublicIPByAddr(network.GetPublicIp())
			if err != nil {
				c.Log.Warnf("failed finding public IP ID for '%s': %s", pubIP.GetIp(), err)
			} else {
				labels := pubIP.GetLabels()
				dyn, ok := labels[lib.DynamicVIPKey].(string)
				if !ok {
					c.Log.Warnf("failed checking tags on public ip: %s", pubIP.GetId())
				}

				if dyn == "true" && ok {
					c.Log.Debugf("found %s tag, cleaninup up publicIP: %s", lib.DynamicVIPKey, pubIP.GetIp())
					err = c.SClient.DeletePublicIP(pubIP.GetId())
					if err != nil {
						c.Log.Warnf("failed cleaning public IP: %s: %s", pubIP.GetId(), err)
					}
				}
			}
		}
		nic, err := c.SClient.GetNic(network.GetNetworkId(), network.GetNicId())
		if err != nil {
			if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
				// nothing to delete
				continue
			}
			c.Log.Debugf("failed getting nic %s: %s", network.GetNicId(), err)
		}
		allNics = append(allNics, nic)
	}

all_nics:
	for _, nic := range allNics {
		if lbTagValue, ok := nic.GetLabels()[lib.AttachedToLoadBalancerKey]; ok {
			attempt := 0
		detach_nic:
			for attempt <= 3 {
				attempt += 1
				lb, err := c.SClient.GetLB(lbTagValue.(string))
				if err != nil {
					if lib.ContainsOpenAPIErrorWithStatusCode(err, http.StatusNotFound) {
						// no LB nothing to detach
						c.Log.Warn("failed finding LB from nic tag: %s", err, "nic_id", nic.GetId(), "loadbalancer_name", lbTagValue.(string))
						continue all_nics
					}
				}
				err = c.SClient.DetachNICFromLB(*nic, lb)
				if err != nil {
					c.Log.Warnf(fmt.Sprintf("failed detaching nic '%s' from LB: '%s': %s", nic.GetId(), lbTagValue.(string), err), "attempt", attempt)
					continue detach_nic
				}
				c.Log.Debugf("desync with %v before confirming targets of %s", lib.JitterWait(2000), lbTagValue.(string))
				lb, err = c.SClient.GetLB(lbTagValue.(string))
				if err != nil {
					c.Log.Warnf(fmt.Sprintf("failed confirming nic'%s' detached from LB: '%s': %s", nic.GetId(), lbTagValue.(string), err), "attempt", attempt)
					continue detach_nic
				}
				// confirm that the target is in fact gone. the loadbalancer api allows for race conditions to happen when
				// concurrent calls to the same targetpool are made
				if lib.HasTarget(lb, nic.GetIpv4()) {
					// still configured, retry
					c.Log.Warnf("race condition detaching nic '%s' from LB: '%s', still in targets list. attempt: %d", nic.GetId(), lbTagValue.(string), attempt)
					lib.JitterWait(5000)
					attempt -= 1
					continue detach_nic
				}
				c.Log.Debugf("detached nic %s from LB: %s", nic.GetId(), lbTagValue.(string))
				break
			}
		}
		err := c.SClient.DeleteNic(nic.GetNetworkId(), nic.GetId())
		if err != nil {
			c.Log.Warnf(fmt.Sprintf("failed cleaning nic: %s", err), "nic_id", nic.GetId())
		}
	}
}

// New Returns a new CPI instance with the configuration loaded
func New(configPath string) (c *CPI, err error) {
	c = &CPI{}

	err = c.initLogger()
	if err != nil {
		return c, fmt.Errorf("cpi.New(): failed to initialize logger: %w", err)
	}

	c.Log.Debug("CPI Version Info", "version", c.Version())

	err = c.InitConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("cpi.New(): failed to initialize CPI config: %w", err)
	}

	if c.Config.LogFile != "" {
		// Slog handles log file path during initialization, not after
		c.Log.Debug("Log file path configured during initialization", "path", c.Config.LogFile)
	}
	err = c.InitAPIClient()
	if err != nil {
		return nil, fmt.Errorf("cpi.New(): failed to initialize API client: %w", err)
	}
	return c, nil
}

func ParseStdIn() (req lib.RPCRequest, err error) {
	req = lib.RPCRequest{}
	stdinBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return
	}
	err = json.Unmarshal(stdinBytes, &req)
	if err != nil {
		err = fmt.Errorf("bad input: %w", err)
		return
	}
	if req.Context.RequestID == "" {
		req.Context.RequestID = uuid.NewString()
	}
	return
}

// initLogger initializes the logger for the CPI
func (c *CPI) initLogger() (err error) {
	cwd := os.Getenv("PWD")
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "."
		}
	}

	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = fmt.Sprintf("%s/%s", cwd, "cpi.log")
	}

	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		logLevelStr = "debug"
	}

	// Convert string level to slog level
	logLevel := lib.MapLogLevel(logLevelStr)

	logger, err := lib.NewCPILogger(logFile, logLevel, 100, 10, true) // 100MB, 10 files, compression enabled
	if err != nil {
		return fmt.Errorf("cpi.initLogger(): failed to create slog logger. CPI Version Info: '%s'. Error: %w", c.Version(), err)
	}

	// Initialize Log field with the base logger
	c.Log = logger

	c.Log.Debug("Logger initialized", "log_file", logFile)

	return nil
}

// InitConfig() Loads the configuration from the file specified in the first command line argument
func (c *CPI) InitConfig(path string) (err error) {
	c.Log.Debugf("Initializing CPI Configuration at '%s'", path)
	cpiConfig, err := lib.NewConfig(path)
	if err != nil {
		return fmt.Errorf("cpi.initConfig(): failed to load CPI configuration at: '%s': %w", path, err)
	}
	c.Config = *cpiConfig
	return
}

// initClient() Initializes the IaaS API Client
func (c *CPI) InitAPIClient() (err error) {
	c.Log.Debug("Initializing IaaS API Client")
	c.Log.Debug("Using Service Account Key for API Authentication")

	stackitClientConfigOpts := []config.ConfigurationOption{
		config.WithServiceAccountKey(c.Config.ServiceAccountJSON),
	}

	if c.Config.StageProfileJSON != "" && c.Config.StageProfileJSON != "{}" {
		var endpoints map[string]any
		err = json.Unmarshal([]byte(c.Config.StageProfileJSON), &endpoints)
		if err != nil {
			return fmt.Errorf("cpi.initConfig(): Error converting config to json: %w ", err)
		}
		tokenEndpoint, ok := endpoints[lib.TokenEndpointKey].(string)
		if !ok {
			return fmt.Errorf("expected non empty value for `%s` in stackit profile json", lib.TokenEndpointKey)
		}
		iaasEndpoint, ok := endpoints[lib.IaasEndpointKey].(string)
		if !ok {
			return fmt.Errorf("expected non empty value for `%s` in stackit profile json", lib.IaasEndpointKey)
		}
		stackitClientConfigOpts = append(stackitClientConfigOpts, config.WithTokenEndpoint(tokenEndpoint))
		stackitClientConfigOpts = append(stackitClientConfigOpts, config.WithEndpoint(iaasEndpoint))
	}
	clientLogger := c.Log.ChildLogger("api_client")
	sClient, err := lib.NewStackitClient(c.Config, clientLogger, stackitClientConfigOpts)
	if err != nil {
		return fmt.Errorf("cpi.initAPIClient(): failed to create stackit-go-sdk iaas.NewClient(%s): %+v", c.Config.RegionID, err)
	}
	if c.Config.RetryCount != 0 {
		c.SClient = lib.NewRetryableStackitClient(sClient, c.Config.RetryCount, c.Log)
	} else {
		c.SClient = sClient
	}
	c.Log.Debug("Initialized IaaS API Client")

	return
}

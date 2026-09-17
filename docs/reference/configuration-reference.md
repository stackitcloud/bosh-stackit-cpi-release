# Configuration Reference

This document lists the STACKIT CPI configuration values and cloud properties.

## Configuration Overview

The STACKIT CPI can be configured through:

1. Configuration file
2. BOSH manifest properties

Configuration precedence, from lowest to highest:

1. Default values
2. Configuration file
3. BOSH manifest properties

## Required Parameters

### `project_id`

- **Type**: String
- **Required**: Yes
- **Description**: STACKIT project ID where resources are created

```yaml
project_id: "87654321-4321-4321-4321-210987654321"
```

### `service_account_json`

- **Type**: String
- **Required**: Yes
- **Description**: STACKIT service account JSON

```yaml
service_account_json: |
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "privateKey": "-----BEGIN RSA PRIVATE KEY-----\n...",
    "credentials": {
      "aud": "https://stackit.cloud/service-accounts",
      "iss": "https://sa.stackit.cloud",
      "sub": "550e8400-e29b-41d4-a716-446655440000"
    }
  }
```

Follow the
<sup>**↗**</sup>[STACKIT CLI authentication guide](https://github.com/stackitcloud/stackit-cli/blob/main/AUTHENTICATION.md)
to create the service account JSON.

## Regional Configuration

### `region`

- **Type**: String
- **Required**: No
- **Default**: `eu01`
- **Description**: STACKIT region for resource creation

```yaml
region: eu01
```

Check the
<sup>**↗**</sup>[STACKIT regions documentation](https://docs.stackit.cloud/stackit/en/regions-and-availability-zones-135137675.html)
for current regions.

### `stage_profile_json`

- **Type**: String
- **Required**: No
- **Description**: Stage profile JSON for non-production or custom STACKIT stages

```yaml
stage_profile_json: ((stackit_stage_profile_json))
```

## Performance Configuration

### `timeout`

- **Type**: Integer, seconds
- **Required**: No
- **Default**: `600`
- **Description**: Global timeout for API operations

```yaml
timeout: 1800
```

### `retry_count`

- **Type**: Integer
- **Required**: No
- **Default**: `3`
- **Description**: Number of retries for failed operations

```yaml
retry_count: 5
```

### `lrp_timeout`

- **Type**: Integer, seconds
- **Required**: No
- **Default**: `3600`
- **Description**: Timeout for long-running operations such as stemcell uploads

```yaml
lrp_timeout: 3600
```

## Resource Defaults

### `default_root_volume_type`

- **Type**: String
- **Required**: No
- **Default**: `storage_premium_perf2`
- **Description**: Default performance class for root disks

```yaml
default_root_volume_type: storage_premium_perf2
```

### `default_persistent_volume_type`

- **Type**: String
- **Required**: No
- **Default**: `storage_premium_perf2`
- **Description**: Default performance class for persistent disks

```yaml
default_persistent_volume_type: storage_premium_perf2
```

For the currently available performance classes, including their IOPS and
throughput limits, see the [STACKIT Block Storage service plans
documentation](https://docs.stackit.cloud/products/storage/block-storage/basics/service-plans/#currently-available-service-plans-performance-classes).

### `default_security_groups`

- **Type**: Array of strings
- **Required**: No
- **Default**: None
- **Description**: Default security groups to apply to all VMs. Names and UUIDs are accepted.

```yaml
default_security_groups:
- default
- bosh-managed
```

### `human_readable_vm_names`

- **Type**: Boolean
- **Required**: No
- **Default**: `false`
- **Description**: Use human-readable VM names in STACKIT instead of BOSH-generated UUID names

```yaml
human_readable_vm_names: true
```

### `auto_fix_name_and_labels`

- **Type**: Boolean
- **Required**: No
- **Default**: `false`
- **Description**: Adjust VM names and label keys so they satisfy STACKIT API restrictions

```yaml
auto_fix_name_and_labels: true
```

### `default_ssh_key_name`

- **Type**: String
- **Required**: No
- **Description**: Default STACKIT SSH key name to use when creating VMs

```yaml
default_ssh_key_name: bosh-default
```

## Logging Configuration

### `log_level`

- **Type**: String
- **Required**: No
- **Default**: `info`
- **Description**: Logging verbosity

```yaml
log_level: debug
```

### `log_file`

- **Type**: String
- **Required**: No
- **Default**: None, logs to stdout
- **Description**: Path to a CPI log file

```yaml
log_file: /var/log/bosh/stackit-cpi.log
```

## Cloud Properties

These properties are used in BOSH cloud config.

### VM Types

```yaml
vm_types:
- name: default
  cloud_properties:
    instance_type: g2i.2
    root_disk:
      size: 50
      type: storage_premium_perf2
    availability_zone: eu01-1
    security_groups:
    - default
```

| Property | Type | Required | Description |
| -------- | ---- | -------- | ----------- |
| `instance_type` | String | Yes | STACKIT machine type |
| `root_disk` | Object | No | Root disk settings, including `size` in GB and optional `type` |
| `availability_zone` | String | No | Specific availability zone |
| `security_groups` | Array | No | Security groups to apply in addition to defaults |

`root_disk.size` is configured in GB. BOSH `disk_types[].disk_size` values are
configured in MiB; the CPI converts persistent disk sizes from MiB to GB before
calling the STACKIT API.

Check the
<sup>**↗**</sup>[STACKIT machine types documentation](https://docs.stackit.cloud/stackit/en/server-machine-types-75137231.html)
for current machine types.

### Disk Types

```yaml
disk_types:
- name: small
  disk_size: 10_240
  cloud_properties:
    type: storage_premium_perf2
```

`disk_size` is a BOSH value in MiB. For example, `10_240` is 10 GiB.

| Property | Type | Required | Description |
| -------- | ---- | -------- | ----------- |
| `type` | String | No | Storage performance class. Defaults to the CPI config value. |

### Networks

```yaml
networks:
- name: default
  type: manual
  subnets:
  - range: 10.0.0.0/24
    gateway: 10.0.0.1
    cloud_properties:
      net_id: "net-12345678-1234-1234-1234-123456789012"
      security_groups:
      - default
```

| Property | Type | Required | Description |
| -------- | ---- | -------- | ----------- |
| `net_id` | String | Yes, for manual networks | STACKIT network UUID |
| `security_groups` | Array | No | Security groups for the network interface |

## Configuration Examples

### Minimal

```yaml
stackit:
  project_id: "87654321-4321-4321-4321-210987654321"
  service_account_json: ((stackit_service_account_json))
```

### With Common Defaults

```yaml
stackit:
  project_id: "87654321-4321-4321-4321-210987654321"
  service_account_json: ((stackit_service_account_json))
  region: eu01
  timeout: 600
  retry_count: 3
  default_root_volume_type: storage_premium_perf2
  default_persistent_volume_type: storage_premium_perf2
  default_security_groups:
  - bosh-sg
  log_level: info
```

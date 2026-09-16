# STACKIT BOSH CPI Release

This repository contains the BOSH Cloud Provider Interface (CPI) release for
STACKIT. It lets BOSH create and manage STACKIT VMs, disks, networks, and
stemcells.

## Getting Started

For a tested walkthrough that creates a BOSH Director on STACKIT, configures
cloud config, uploads a stemcell, and deploys a smoke-test VM, use the
[quickstart](docs/getting-started/quickstart.md).

## Requirements

- STACKIT project with IaaS access
- STACKIT service account JSON
- Network and security group for BOSH-managed VMs
- BOSH CLI v2
- Go 1.24+ for development

## CPI Configuration

The CPI job is configured through the `stackit` property block.

```yaml
stackit:
  project_id: ((stackit_project_id))
  service_account_json: ((stackit_service_account_json))
  region: eu01
  default_security_groups: [bosh-sg]
```

Required values:

- `project_id`
- `service_account_json`

Common optional values:

- `region`, default `eu01`
- `default_security_groups`, names or UUIDs
- `default_root_volume_type`, default `storage_premium_perf2`
- `default_persistent_volume_type`, default `storage_premium_perf2`
- `timeout`, default `600`
- `lrp_timeout`, default `3600`
- `retry_count`, default `3`
- `log_level`, default `info`

See the [configuration reference](docs/reference/configuration-reference.md)
for cloud properties and the full option list.

## Basic Cloud Config

Use this as a minimal starting point after your director is available. Replace
the variables through `--vars-file` or your usual BOSH variable source.

```yaml
azs:
- name: z1
  cloud_properties:
    availability_zone: ((stackit_zone))

vm_types:
- name: default
  cloud_properties:
    instance_type: g2i.2
- name: large
  cloud_properties:
    instance_type: g2i.8

disk_types:
- name: small
  disk_size: 5_120 # MiB
  cloud_properties:
    type: storage_premium_perf0
- name: medium
  disk_size: 10_240 # MiB
  cloud_properties:
    type: storage_premium_perf2

networks:
- name: default
  type: manual
  subnets:
  - range: ((internal_cidr))
    gateway: ((internal_gw))
    dns: ((internal_dns))
    reserved: [10.0.0.1-10.0.0.10]
    az: z1
    cloud_properties:
      net_id: ((stackit_network_id))
      security_groups: ((default_security_groups))

compilation:
  workers: 3
  reuse_compilation_vms: true
  az: z1
  vm_type: default
  network: default
```

Apply it with:

```bash
bosh update-cloud-config cloud-config.yml --vars-file=stackit-vars.yml
```

## Stemcells

Use standard BOSH OpenStack KVM full stemcells. For example:

```bash
bosh upload-stemcell \
  --sha1 2c1715b4926ff895e779e1eaa738621887bcef1e \
  "https://bosh.io/d/stemcells/bosh-openstack-kvm-ubuntu-noble?v=1.562"
```

## Development

The `develop` branch is the trunk branch. The `main` branch is used for release
engineering.

Build the CPI from the repository root:

```bash
cd src/stackit-cpi
GOFLAGS=-mod=mod go build -o ../../bin/stackit-cpi .
```

Run unit tests from the repository root:

```bash
cd src/stackit-cpi
GOFLAGS=-mod=mod go test -p 12 -v ./cpi ./cpi/lib
```

Create a development release from the repository root:

```bash
mkdir -p builds
bosh create-release --force --timestamp-version --tarball builds/dev.tgz
```

Run integration tests from the repository root when STACKIT credentials are available:

```bash
cd src/stackit-cpi/cpi/integration
GOFLAGS=-mod=mod go run github.com/onsi/ginkgo/v2/ginkgo -p -v --flake-attempts 3 --randomize-all --fail-fast
```

## Support

Open a GitHub issue and include:

- CPI release version
- BOSH Director version
- The failing BOSH task output
- Relevant CPI logs with secrets removed

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

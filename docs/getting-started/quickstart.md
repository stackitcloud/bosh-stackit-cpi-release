# Quickstart: BOSH Director on STACKIT

This guide deploys a BOSH Director on STACKIT with the STACKIT CPI.

## Prerequisites

Before starting, make sure you have:

- A STACKIT project with IaaS access
- STACKIT CLI access to the project
- STACKIT service account JSON
- BOSH CLI v2
- A STACKIT network for the director
- A security group for the director VM
- Enough quota for at least one VM and a 50 GB root disk

The STACKIT CPI supports service account authentication. Follow the
<sup>**↗**</sup>[STACKIT CLI authentication guide](https://github.com/stackitcloud/stackit-cli/blob/main/AUTHENTICATION.md)
to create the service account JSON used as `service_account_json`.

## Prepare Files

Create a working directory and clone `bosh-deployment`:

```bash
mkdir ~/bosh-on-stackit
cd ~/bosh-on-stackit
git clone https://github.com/cloudfoundry/bosh-deployment.git
```

Collect the STACKIT values used by the variables file:

```bash
stackit project list

stackit network list \
  --project-id YOUR_PROJECT_ID \
  --region eu01

stackit security-group list \
  --project-id YOUR_PROJECT_ID \
  --region eu01
```

If you do not already have a security group for the director, create one:

```bash
stackit security-group create \
  --project-id YOUR_PROJECT_ID \
  --region eu01 \
  --name bosh-sg \
  --stateful

export BOSH_SG_ID=YOUR_SECURITY_GROUP_ID
export ADMIN_CIDR=YOUR_ADMIN_CIDR

stackit security-group rule create \
  --project-id YOUR_PROJECT_ID \
  --region eu01 \
  --security-group-id "$BOSH_SG_ID" \
  --direction ingress \
  --protocol-name tcp \
  --port-range-min 22 \
  --port-range-max 22 \
  --ip-range "$ADMIN_CIDR"

stackit security-group rule create \
  --project-id YOUR_PROJECT_ID \
  --region eu01 \
  --security-group-id "$BOSH_SG_ID" \
  --direction ingress \
  --protocol-name tcp \
  --port-range-min 25555 \
  --port-range-max 25555 \
  --ip-range "$ADMIN_CIDR"

stackit security-group rule create \
  --project-id YOUR_PROJECT_ID \
  --region eu01 \
  --security-group-id "$BOSH_SG_ID" \
  --direction ingress \
  --protocol-name tcp \
  --port-range-min 8443 \
  --port-range-max 8443 \
  --ip-range "$ADMIN_CIDR"

stackit security-group rule create \
  --project-id YOUR_PROJECT_ID \
  --region eu01 \
  --security-group-id "$BOSH_SG_ID" \
  --direction ingress \
  --remote-security-group-id "$BOSH_SG_ID"
```

Use an availability zone that exists in your STACKIT region. The examples below
use `eu01-1`.

Create `stackit-vars.yml`:

```yaml
director_name: bosh-stackit

internal_cidr: 10.0.0.0/24
internal_gw: 10.0.0.1
internal_ip: 10.0.0.6
internal_dns: [8.8.8.8, 8.8.4.4]

stackit_project_id: YOUR_PROJECT_ID
stackit_service_account: |
  YOUR_SERVICE_ACCOUNT_JSON
stackit_region: eu01
stackit_zone: eu01-1
stackit_network_id: YOUR_NETWORK_ID

vm_machine_type: g2i.2
root_disk_size: 50           # GB
persistent_disk_size: 25_000 # MiB
default_root_volume_type: storage_premium_perf2
default_persistent_volume_type: storage_premium_perf2
default_security_groups: [bosh-sg]

bosh_stackit_cpi_release_url: https://github.com/stackitcloud/bosh-stackit-cpi-release/releases/latest/download/bosh-stackit-cpi-release-0.1.30.tgz
bosh_stackit_cpi_release_sha1: SHA1_OF_BOSH_STACKIT_CPI_RELEASE
director_stemcell_url: https://bosh.io/d/stemcells/bosh-openstack-kvm-ubuntu-noble?v=1.562
director_stemcell_sha1: 2c1715b4926ff895e779e1eaa738621887bcef1e
```

For a director with an external IP, also set:

```yaml
external_ip: X.X.X.X
```

For `bosh-deployment/misc/ntp.yml`, also set:

```yaml
internal_ntp: [0.pool.ntp.org, 1.pool.ntp.org]
```

BOSH requires `sha1` for HTTP(S) release URLs. For a local development release,
build a tarball from this repository, point `bosh_stackit_cpi_release_url` at a
`file://` URL, and set the SHA1 from:

```bash
shasum -a 1 builds/dev.tgz
```

## Create The CPI Ops File

Create `ops/stackit-cpi.yml`:

```bash
mkdir -p ops
```

```yaml
- type: replace
  path: /releases/-
  value:
    name: bosh-stackit-cpi
    url: ((bosh_stackit_cpi_release_url))
    sha1: ((bosh_stackit_cpi_release_sha1))

- type: replace
  path: /resource_pools/name=vms/stemcell?
  value:
    url: ((director_stemcell_url))
    sha1: ((director_stemcell_sha1))

- type: replace
  path: /resource_pools/name=vms/cloud_properties?
  value:
    availability_zone: ((stackit_zone))
    instance_type: ((vm_machine_type))
    root_disk:
      size: ((root_disk_size))

- type: replace
  path: /networks/name=default/subnets/0/cloud_properties?
  value:
    net_id: ((stackit_network_id))

- type: replace
  path: /instance_groups/name=bosh/jobs/-
  value:
    name: stackit_cpi
    release: bosh-stackit-cpi

- type: replace
  path: /instance_groups/name=bosh/properties/director/cpi_job?
  value: stackit_cpi

- type: replace
  path: /instance_groups/name=bosh/properties/agent/mbus?
  value:
    urls: ["nats://((internal_ip)):4222"]

- type: replace
  path: /cloud_provider/template?
  value:
    name: stackit_cpi
    release: bosh-stackit-cpi

- type: replace
  path: /cloud_provider/properties/agent/mbus
  value:
    urls: [https://mbus:((mbus_bootstrap_password))@0.0.0.0:6868]
    cert: ((mbus_bootstrap_ssl))

- type: replace
  path: /instance_groups/name=bosh/properties/stackit?
  value:
    region: ((stackit_region))
    project_id: ((stackit_project_id))
    service_account_json: ((stackit_service_account))
    default_security_groups: ((default_security_groups))
    default_root_volume_type: ((default_root_volume_type))
    default_persistent_volume_type: ((default_persistent_volume_type))
    human_readable_vm_names: true

- type: replace
  path: /cloud_provider/properties/stackit?
  value:
    region: ((stackit_region))
    project_id: ((stackit_project_id))
    service_account_json: ((stackit_service_account))
    default_security_groups: ((default_security_groups))
    default_root_volume_type: ((default_root_volume_type))
    default_persistent_volume_type: ((default_persistent_volume_type))
    human_readable_vm_names: true
```

Use a standard BOSH OpenStack KVM full stemcell, such as
`bosh-openstack-kvm-ubuntu-noble`.

## Deploy The Director

For an internal-only director:

```bash
bosh create-env bosh-deployment/bosh.yml \
  --state=state.json \
  --vars-store=creds.yml \
  --vars-file=stackit-vars.yml \
  -o bosh-deployment/uaa.yml \
  -o bosh-deployment/credhub.yml \
  -o bosh-deployment/jumpbox-user.yml \
  -o ops/stackit-cpi.yml
```

For a director with an external IP:

```bash
bosh create-env bosh-deployment/bosh.yml \
  --state=state.json \
  --vars-store=creds.yml \
  --vars-file=stackit-vars.yml \
  -o bosh-deployment/external-ip-not-recommended.yml \
  -o bosh-deployment/uaa.yml \
  -o bosh-deployment/credhub.yml \
  -o bosh-deployment/jumpbox-user.yml \
  -o ops/stackit-cpi.yml
```

To configure NTP, add `-o bosh-deployment/misc/ntp.yml` and set `internal_ntp`
in `stackit-vars.yml`.

## Target The Director

```bash
export BOSH_CLIENT=admin
export BOSH_CLIENT_SECRET=$(bosh int ./creds.yml --path /admin_password)

bosh alias-env stackit-bosh -e 10.0.0.6 \
  --ca-cert="$(bosh int ./creds.yml --path /director_ssl/ca)"

export BOSH_ENVIRONMENT=stackit-bosh
bosh env
```

For an external IP, use that address in `bosh alias-env` instead of `10.0.0.6`.

## Configure Cloud Config

Create `cloud-config.yml`:

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
- name: large
  disk_size: 50_000 # MiB
  cloud_properties:
    type: storage_premium_perf4

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
- name: vip
  type: vip

compilation:
  workers: 3
  reuse_compilation_vms: true
  az: z1
  vm_type: default
  network: default
```

Apply it:

```bash
bosh update-cloud-config cloud-config.yml \
  --vars-file=stackit-vars.yml
```

## Upload A Stemcell

Upload the same stemcell family used for the director:

```bash
bosh upload-stemcell \
  --sha1 2c1715b4926ff895e779e1eaa738621887bcef1e \
  https://bosh.io/d/stemcells/bosh-openstack-kvm-ubuntu-noble?v=1.562

bosh stemcells
```

## Deploy A Smoke Test

Create `nginx-smoke-test.yml`:

```yaml
name: nginx-smoke-test

releases:
- name: nginx
  version: 1.21.6
  url: https://bosh.io/d/github.com/cloudfoundry-community/nginx-release?v=1.21.6
  sha1: sha256:f12c44d1eb49629ed65c703f1c81289780d0c5738608597d57ecf93e908a69f5

stemcells:
- alias: default
  os: ubuntu-noble
  version: "1.562"

update:
  canaries: 1
  max_in_flight: 1
  canary_watch_time: 5000-30000
  update_watch_time: 5000-30000

instance_groups:
- name: web
  instances: 1
  azs: [z1]
  vm_type: default
  stemcell: default
  networks:
  - name: default
  jobs:
  - name: nginx
    release: nginx
    properties:
      nginx_conf: |
        worker_processes auto;
        error_log /var/vcap/sys/log/nginx/error.log;
        pid /var/vcap/sys/run/nginx/nginx.pid;

        events {
          worker_connections 1024;
        }

        http {
          server {
            listen 80;
            server_name _;

            location /health {
              return 200 "healthy\n";
              add_header Content-Type text/plain;
            }
          }
        }
```

Deploy it and check the health endpoint:

```bash
bosh -d nginx-smoke-test deploy nginx-smoke-test.yml
bosh -d nginx-smoke-test instances
bosh -d nginx-smoke-test ssh web/0 -c "curl -fsS http://127.0.0.1/health"
```

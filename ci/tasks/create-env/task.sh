#!/usr/bin/env bash
set -eEo pipefail

CLOUD_CONFIG="$(jq .cloud_config <<< "${TERRAFORM_OUTPUT}")"

JUMPBOX_IP="$(jq .jumphost.public_ip -r <<< "${TERRAFORM_OUTPUT}")"
JUMPBOX_KEY_PATH=$(mktemp)
JUMPBOX_KEY_DATA=$(jq .jumphost.private_ssh_key -r <<< "${TERRAFORM_OUTPUT}")
echo -e "${JUMPBOX_KEY_DATA}" > "${JUMPBOX_KEY_PATH}"
OPS_FILES=$(echo "${OPS}" | jq '.[] | .="-o "+.' -r | xargs)
VAR_FILES=$(echo "${VAR_FILES}" | jq '.[] | .="-l "+.' -r | xargs)
VARS=$(echo "${VARS}" | jq '.[] | .="-v "+.' -r | xargs)
echo "${TERRAFORM_OUTPUT}" | jq .bosh -r > bosh-vars.json

#we're relying on word splitting to build the --flags provided=to-bosh
#shellcheck disable=2068
bosh create-env ${MANIFEST[@]} ${OPS_FILES[@]} ${VAR_FILES[@]} ${VARS[@]} --state "${STATE_FILE_PATH}" --vars-store "${CREDS_FILE_PATH}"

cat << EOF > "${BOSH_RC_FILE_PATH}"
JUMPBOX_KEY_DATA="$(cat "${JUMPBOX_KEY_PATH}")"
JUMPBOX_KEY_PATH=\$(mktemp)
echo -e "$JUMPBOX_KEY_DATA" > "\${JUMPBOX_KEY_PATH}"
export BOSH_CA_CERT="$(bosh  int "${CREDS_FILE_PATH}" --path /default_ca/ca)"
export BOSH_ENVIRONMENT=$(bosh int bosh-vars.json --path /internal_ip)
export BOSH_CLIENT=admin
export BOSH_CLIENT_SECRET="$(bosh int "${CREDS_FILE_PATH}" --path /admin_password)"
export BOSH_ALL_PROXY="ssh+socks5://ubuntu@${JUMPBOX_IP}:22?private-key=\${JUMPBOX_KEY_PATH}"
export CREDHUB_PROXY="\${BOSH_ALL_PROXY}"
export CREDHUB_SERVER=https://$(bosh int bosh-vars.json --path /internal_ip):8844
export CREDHUB_CA_CERT="$( bosh interpolate "${CREDS_FILE_PATH}" --path=/credhub_tls/ca )\n$( bosh interpolate "${CREDS_FILE_PATH}" --path=/uaa_ssl/ca )"
export CREDHUB_CLIENT=director_to_credhub
export CREDHUB_SECRET="$(bosh int "${CREDS_FILE_PATH}" --path /uaa_clients_director_to_credhub)"
EOF

#shellcheck disable=2001
cat << EOF > "${BOSH_RESOURCE_FILE_PATH}"
target: https://$(bosh int bosh-vars.json --path /internal_ip):25555/
client: admin
client_secret: "$(bosh int "${CREDS_FILE_PATH}" --path /admin_password)"
ca_cert: |
$(bosh  int "${CREDS_FILE_PATH}" --path /default_ca/ca | sed 's/^/  /g')
jumpbox_url: "${JUMPBOX_IP}:22"
jumpbox_ssh_key: |
$(echo "${JUMPBOX_KEY_DATA}" | sed 's/^/  /g')
jumpbox_username: ubuntu
EOF

#shellcheck disable=1091
. data/bosh.rc

BOSH_ALL_PROXY="ssh+socks5://ubuntu@${JUMPBOX_IP}:22?private-key=${JUMPBOX_KEY_PATH}" \
  bosh ucc <(echo "${CLOUD_CONFIG}") -n

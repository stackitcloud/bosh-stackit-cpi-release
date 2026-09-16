#!/usr/bin/env bash

set -Eeo pipefail


VAULT_ADDR=$( jq -r .addr <<< "${VAULT}" )
VAULT_USER=$( jq -r .user <<< "${VAULT}" )
VAULT_PASS=$( jq -r .pass <<< "${VAULT}" )
VAULT_KV_MOUNTPATH=$( jq -r .kv_mount_path <<< "${VAULT}" )

export VAULT_ADDR
vault login -method=userpass username="${VAULT_USER}" password="${VAULT_PASS}" &> /dev/null
VALUES=${VALUES:-()}
INDEX=0
LENGTH=$(yq length <<< "$VALUES")
while [[ ${INDEX} -lt ${LENGTH} ]]; do
  vault kv put -mount="${VAULT_KV_MOUNTPATH}" \
    "$(yq ".[${INDEX}].vault_path" <<< "${VALUES}" )" \
    value="$(cat "$(yq ".[${INDEX}].value_path" <<< "${VALUES}")" )"
  INDEX=$((INDEX+1))
done

#!/usr/bin/env bash
set -eu -o pipefail

pushd repo
  echo -e "${PRIVATE_YML}" > config/private.yml
  bosh sync-blobs
  bosh create-release --force --version "${VERSION}" --tarball ../builds/dev.tgz
popd

ls -al builds/


#!/usr/bin/env bash
set -eu -o pipefail
chown "$(whoami):$(whoami)" -R repo
pushd repo
  echo -e "${PRIVATE_YML}" > config/private.yml
  bosh sync-blobs
  if [[ -f "releases/bosh-stackit-cpi/bosh-stackit-cpi-${VERSION}.yml" ]]; then
    echo "found existing release, overwriting"
    rm "releases/bosh-stackit-cpi/bosh-stackit-cpi-${VERSION}.yml"
    RELEASE_KEY=$(yq releases/bosh-stackit-cpi/index.yml -o json | jq ".builds | to_entries[] | select(.value.version  == \"${VERSION}\").key" -r )
    yq -i "del(.builds.$RELEASE_KEY)" releases/bosh-stackit-cpi/index.yml
    git tag -d "v${VERSION}"
  fi
  bosh finalize-release --version "${VERSION}" ../release/*.tgz --force

  git config --global user.name "${GIT_USER}"
  git config --global user.email "${GIT_EMAIL}"
  git add .final_builds/
  git add "releases/bosh-stackit-cpi/bosh-stackit-cpi-${VERSION}.yml"
  git add "releases/bosh-stackit-cpi/index.yml"
  git commit -m "create final release ${VERSION}"
  bosh create-release "releases/bosh-stackit-cpi/bosh-stackit-cpi-${VERSION}.yml" --tarball "../builds/bosh-stackit-cpi-release-${VERSION}.tgz"


popd

ls -al builds/
echo "v${VERSION}" > builds/tag-for-release

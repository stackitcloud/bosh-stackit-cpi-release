#!/usr/bin/env bash
set -eu -o pipefail

TEST_CONFIG_PATH=$(pwd)/test-config.json
export TEST_CONFIG_PATH
echo "${TEST_CONFIG}" > "${TEST_CONFIG_PATH}"
pushd repo/src/stackit-cpi/cpi/integration
  # run all units in parallel
  go run github.com/onsi/ginkgo/v2/ginkgo -p -v --flake-attempts 3 --randomize-all --fail-fast
popd

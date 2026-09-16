#!/usr/bin/env bash
set -eu -o pipefail

pushd repo/src/stackit-cpi/cpi
  # run all units in parallel
  go test -p 12 -v . ./lib
popd

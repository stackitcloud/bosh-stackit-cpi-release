#!/usr/bin/env bash
set -eu -o pipefail

pushd repo/src/stackit-cpi/cpi
  go run "github.com/securego/gosec/v2/cmd/gosec@${GOSEC_VERSION}" . ./lib
popd

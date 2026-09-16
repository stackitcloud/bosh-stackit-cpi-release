#!/usr/bin/env bash
set -eu -o pipefail

pushd repo/src/stackit-cpi
  go run golang.org/x/vuln/cmd/govulncheck@latest .
popd

#!/usr/bin/env bash
set -eu -o pipefail

pushd repo || false
  gh pr create --title "${TITLE}" --body "${BODY}" --base "${TARGET_BRANCH}" --head "${SOURCE_BRANCH}"
popd

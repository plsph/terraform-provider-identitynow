#!/usr/bin/env bash

set -e

echo "==>Running acceptance testing..."

source "$(dirname "$0")/gotestacc_vars.sh"
if [ -f "$(dirname "$0")/../.secrets/gotestacc_vars.sh" ]; then
  source "$(dirname "$0")/../.secrets/gotestacc_vars.sh"
fi

TF_ACC=1 go test -cover ./... -v -timeout 120m

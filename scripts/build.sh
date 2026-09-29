#!/bin/bash
set -euo pipefail

# Builds the provider and installs it into the implied local mirror directory, so Terraform uses it
# instead of the registry version: https://developer.hashicorp.com/terraform/cli/config/config-file#implied-local-mirror-directories
# Use a version that is not published, e.g. ./scripts/build.sh 0.0.1-dev
# Remove ~/.terraform.d/plugins/registry.terraform.io/plsph/identitynow when finished to use the registry version again.
VERSION=${1:-${VERSION:-0.0.1-dev}}
OS_ARCH="$(go env GOOS)_$(go env GOARCH)"
PLUGIN_DIR=~/.terraform.d/plugins/registry.terraform.io/plsph/identitynow/${VERSION}/${OS_ARCH}

mkdir -p "${PLUGIN_DIR}"
go build -ldflags "-X main.version=${VERSION}" -o "${PLUGIN_DIR}/terraform-provider-identitynow_v${VERSION}"
echo "Installed ${PLUGIN_DIR}/terraform-provider-identitynow_v${VERSION}"

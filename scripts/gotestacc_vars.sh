#!/usr/bin/env bash

# Environment for acceptance tests. Do not commit real values: put them in .secrets/gotestacc_vars.sh
# (ignored by git) or export them before running make testacc.

set -e

export IDENTITYNOW_URL="${IDENTITYNOW_URL:-<identitynow_url>}"
export IDENTITYNOW_CLIENT_ID="${IDENTITYNOW_CLIENT_ID:-<client_id>}"
export IDENTITYNOW_CLIENT_SECRET="${IDENTITYNOW_CLIENT_SECRET:-<client_secret>}"
export IDENTITYNOW_OWNER_ID="${IDENTITYNOW_OWNER_ID:-<owner_id>}"
export IDENTITYNOW_OWNER_NAME="${IDENTITYNOW_OWNER_NAME:-<owner_name>}"
export IDENTITYNOW_EXTERNAL_OWNER_ID="${IDENTITYNOW_EXTERNAL_OWNER_ID:-<external_owner_id>}"
export IDENTITYNOW_CLUSTER_ID="${IDENTITYNOW_CLUSTER_ID:-<cluster_id>}"
export IDENTITYNOW_CLUSTER_NAME="${IDENTITYNOW_CLUSTER_NAME:-<cluster_name>}"

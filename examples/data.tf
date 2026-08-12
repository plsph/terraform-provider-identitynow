data "identitynow_identity" "john_doe" {
  alias = "A12BCDE3F" #user identity alias in IdN
}

data "identitynow_source_entitlement" "aad_operator" {
  source_id = identitynow_source.azure_ad_source[0].id
  name      = "<AZURE_ACTIVE_DIRECTORY_GROUP_NAME>"
}

data "identitynow_source_entitlement" "ad_developer" {
  source_id = identitynow_source.active_directory_source[0].id
  name      = "<ACTIVE_DIRECTORY_GROUP_NAME>"
}

# If multiple entitlements share the same `name`, use `attribute` and/or `value` to disambiguate, or
# read the computed `entitlements` list to inspect all matches:
#
# Example: reference all matching entitlement IDs
# locals {
#   aad_entitlement_ids = data.identitynow_source_entitlement.aad_operator.entitlements[*].id
# }
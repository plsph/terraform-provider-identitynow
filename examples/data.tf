data "identitynow_identity" "john_doe" {
  alias = "A12BCDE3F" #user identity alias in IdN
}

# Entitlements are available after the source has been aggregated.
data "identitynow_source_entitlement" "aad_operator" {
  source_id = identitynow_source.azure_ad_source.id
  name      = "<AZURE_ACTIVE_DIRECTORY_GROUP_NAME>"
}

data "identitynow_source_entitlement" "ad_developer" {
  source_id = identitynow_source.active_directory_source.id
  name      = "<ACTIVE_DIRECTORY_GROUP_NAME>"
}

# All entitlements of the source with the given name are returned in the `entitlements` list,
# e.g. reference all matching entitlement IDs:
locals {
  aad_operator_entitlement_ids = data.identitynow_source_entitlement.aad_operator.entitlements[*].id
}

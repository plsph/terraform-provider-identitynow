data "identitynow_source" "active_directory" {
  name = "Active Directory"
}

data "identitynow_source_entitlement" "developers" {
  source_id = data.identitynow_source.active_directory.id
  name      = "Developers"
}

output "developers_entitlement_id" {
  value = data.identitynow_source_entitlement.developers.entitlements[0].id
}

output "all_matching_entitlement_ids" {
  value = data.identitynow_source_entitlement.developers.entitlements[*].id
}

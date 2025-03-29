resource "identitynow_role" "operator_developer_role" {
  access_profile_ids = [
    identitynow_access_profile.aad_access_profile_operators.id,
    identitynow_access_profile.ad_access_profile_developers[count.index].id
  ]
  description      = "Developer Operator Role Description"
  name             = "Developer Operator Role"
  approval_schemes = "none"
  disabled         = false
  requestable      = true
  owner            = data.identitynow_identity.john_doe.alias
  lifecycle {
    ignore_changes = [
      name,
      display_name,
      identity_count
    ]
  }
}

resource "identitynow_custom_user_level" "identity_managers" {
  name        = "Identity Managers"
  description = "Manage identities"
  right_sets  = ["idn:ui-identity-management-read"]
  publish     = true

  owner {
    id = "2c9180835d2e5168015d32f890ca1581"
  }
}

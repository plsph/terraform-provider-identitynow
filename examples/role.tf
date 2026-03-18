resource "identitynow_role" "operator_developer_role" {
  name        = "Developer Operator Role"
  description = "Developer Operator Role Description"

  owner {
    id   = data.identitynow_identity.john_doe.id
    type = "IDENTITY"
    name = data.identitynow_identity.john_doe.name
  }

  access_profiles {
    id   = identitynow_access_profile.aad_access_profile_operators.id
    type = "ACCESS_PROFILE"
    name = identitynow_access_profile.aad_access_profile_operators.name
  }

  requestable = true
  enabled     = true
}

data "identitynow_role" "example" {
  id = "2c91808a7813090a017813b6301f1234"
}

output "role_name" {
  value = data.identitynow_role.example.name
}

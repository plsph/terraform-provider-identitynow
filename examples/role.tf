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

  access_profiles {
    id   = identitynow_access_profile.ad_access_profile_developers.id
    type = "ACCESS_PROFILE"
    name = identitynow_access_profile.ad_access_profile_developers.name
  }

  access_model_metadata {
    attributes {
      key         = "iscPrivacy"
      name        = "Privacy"
      multiselect = false
      status      = "active"
      type        = "custom"

      values {
        value  = "public"
        name   = "Public"
        status = "active"
      }

      values {
        value  = "internal"
        name   = "Internal"
        status = "active"
      }
    }
  }

  requestable = true
  enabled     = true
}

data "identitynow_role" "operator_developer_role" {
  id = identitynow_role.operator_developer_role.id
}

output "role_access_profile_names" {
  value = data.identitynow_role.operator_developer_role.access_profiles[*].name
}


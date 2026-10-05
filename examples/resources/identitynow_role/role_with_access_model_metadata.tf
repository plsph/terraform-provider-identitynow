resource "identitynow_role" "with_metadata" {
  name        = "Metadata Role"
  description = "A role with access model metadata"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  access_profiles {
    id   = "2c91808a7813090a017813b6301f0044"
    type = "ACCESS_PROFILE"
    name = "Example Access Profile"
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

resource "identitynow_role" "standard_membership" {
  name        = "Department Role"
  description = "Automatically assigned based on department"

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

  membership {
    type = "STANDARD"

    criteria {
      operation    = "EQUALS"
      string_value = "Engineering"

      key {
        type     = "IDENTITY"
        property = "attribute.department"
      }
    }
  }

  enabled = true
}

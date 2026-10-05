resource "identitynow_role" "example" {
  name        = "Sales Role"
  description = "Dimensional role for the sales department"
  dimensional = true

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }
}

resource "identitynow_dimension" "example" {
  role_id     = identitynow_role.example.id
  name        = "Example Dimension"
  description = "An example dimension"

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
      string_value = "Sales"

      key {
        type     = "IDENTITY"
        property = "attribute.department"
      }
    }
  }
}

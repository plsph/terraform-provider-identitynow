resource "identitynow_role" "compound_membership" {
  name        = "Compound Membership Role"
  description = "Assigned based on multiple criteria"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  membership {
    type = "STANDARD"

    criteria {
      operation = "AND"

      children {
        operation    = "EQUALS"
        string_value = "Engineering"

        key {
          type     = "IDENTITY"
          property = "attribute.department"
        }
      }

      children {
        operation    = "EQUALS"
        string_value = "US"

        key {
          type     = "IDENTITY"
          property = "attribute.location"
        }
      }
    }
  }

  enabled = true
}

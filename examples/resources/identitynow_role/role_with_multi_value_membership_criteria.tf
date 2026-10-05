resource "identitynow_role" "multivalue_membership" {
  name        = "Multi-Value Membership Role"
  description = "Assigned based on multiple job codes"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  membership {
    type = "STANDARD"

    criteria {
      operation = "OR"

      children {
        operation = "AND"

        children {
          operation = "EQUALS"
          values    = ["active"]

          key {
            type     = "IDENTITY"
            property = "attribute.cloudLifecycleState"
          }
        }

        children {
          operation = "EQUALS"
          values    = ["G8244", "G8243", "G8242", "G6644"]

          key {
            type     = "IDENTITY"
            property = "attribute.jobcode"
          }
        }
      }
    }
  }

  enabled = true
}

resource "identitynow_role" "dimensional_approval" {
  name        = "Dimensional Approval Role"
  description = "A dimensional role with per-dimension approval schemas"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  access_request_config {
    comments_required        = true
    denial_comments_required = false

    approval_schemes {
      approver_type = "MANAGER"
    }

    dimension_schema {
      dimension_attributes {
        name         = "eqLocation"
        display_name = "EQ Location"
        derived      = true
      }
    }
  }

  requestable = true
  enabled     = true
  dimensional = true
}

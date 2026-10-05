resource "identitynow_role" "with_approval" {
  name        = "Approval Role"
  description = "A role with access request configuration"

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

  access_request_config {
    comments_required        = true
    denial_comments_required = true

    approval_schemes {
      approver_type = "MANAGER"
    }

    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = "2c91808a7813090a017813b6301faaaa"
    }
  }

  requestable = true
  enabled     = true
}

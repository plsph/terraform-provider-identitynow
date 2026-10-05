data "identitynow_identity" "owner" {
  alias = "john.doe"
}

data "identitynow_source" "active_directory" {
  name = "Active Directory"
}

data "identitynow_source_entitlement" "developers" {
  source_id = data.identitynow_source.active_directory.id
  name      = "Developers"
}

resource "identitynow_access_profile" "developers" {
  name        = "AD Developers"
  description = "Developer access in Active Directory"
  requestable = true
  enabled     = true
  segments    = ["f7b1b8a3-5fed-4fd4-ad29-82014e137e19"]

  owner {
    id   = data.identitynow_identity.owner.id
    name = data.identitynow_identity.owner.name
    type = "IDENTITY"
  }

  source {
    id   = data.identitynow_source.active_directory.id
    name = data.identitynow_source.active_directory.name
    type = "SOURCE"
  }

  entitlements {
    id   = data.identitynow_source_entitlement.developers.entitlements[0].id
    name = data.identitynow_source_entitlement.developers.entitlements[0].name
    type = "ENTITLEMENT"
  }

  access_request_config {
    comments_required        = true
    denial_comments_required = true
    reauthorization_required = false
    require_end_date         = true

    approval_schemes {
      approver_type = "MANAGER"
    }

    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = "46c79819-a69f-49a2-becb-12c971ae66c6"
    }

    max_permitted_access_duration {
      value     = 6
      time_unit = "MONTHS"
    }
  }

  revocation_request_config {
    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = "46c79819-a69f-49a2-becb-12c971ae66c6"
    }
  }

  additional_owners {
    id   = "46c79819-a69f-49a2-becb-12c971ae66c6"
    name = "Access Approvers"
    type = "GOVERNANCE_GROUP"
  }

  provisioning_criteria {
    operation = "OR"

    children {
      operation = "EQUALS"
      attribute = "accountType"
      value     = "developer"
    }

    children {
      operation = "EQUALS"
      attribute = "accountType"
      value     = "admin"
    }
  }
}

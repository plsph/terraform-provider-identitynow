locals {
  manager = [{
    approver_type = "MANAGER"
    approver_id   = null
  }]
  governance_group = [for elem in var.governance_groups :
    {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = elem
    }
  ]
}

resource "identitynow_access_profile" "this" {
  name        = var.name
  description = var.description
  requestable = true
  enabled     = true
  segments    = ["f7b1b8a3-5fed-4fd4-ad29-82014e137e19"]

  entitlements {
    id   = var.entitlement_id
    name = var.entitlement_name
    type = "ENTITLEMENT"
  }

  source {
    id   = var.src_id
    name = var.src_name
    type = "SOURCE"
  }

  owner {
    id   = var.owner_id
    name = var.owner_name
    type = "IDENTITY"
  }

  access_request_config {
    comments_required        = true
    denial_comments_required = true
    reauthorization_required = true
    require_end_date        = true

    dynamic "approval_schemes" {
      for_each = concat(local.manager, local.governance_group)
      content {
        approver_type = approval_schemes.value.approver_type
        approver_id   = approval_schemes.value.approver_id
      }
    }

    max_permitted_access_duration {
      value    = 6
      time_unit = "MONTHS"
    }
  }

  revocation_request_config {
    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = var.governance_group_id
    }
  }
}

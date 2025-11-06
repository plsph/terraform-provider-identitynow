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
    dynamic "approval_schemes" {
      for_each = concat(local.manager, local.governance_group)
      content {
        approver_type = approval_schemes.value.approver_type
        approver_id   = approval_schemes.value.approver_id
      }
    }
  }
}

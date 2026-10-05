locals {
  owner_email   = "jane.doe@example.com"
  member_emails = ["john.doe@example.com", "mary.major@example.com"]
}

# Fetch identity details
data "identitynow_identity" "owner" {
  email_address = local.owner_email
}

data "identitynow_identity" "members" {
  for_each      = toset(local.member_emails)
  email_address = each.key
}

resource "identitynow_governance_group" "example" {
  name        = "Access Approvers"
  description = "Approves access requests"

  owner {
    id   = data.identitynow_identity.owner.id
    name = data.identitynow_identity.owner.name
    type = "IDENTITY"
  }
}

resource "identitynow_governance_group_members" "example" {
  governance_group_id = identitynow_governance_group.example.id

  dynamic "members" {
    for_each = data.identitynow_identity.members
    content {
      id   = members.value.id
      name = members.value.name
      type = "IDENTITY"
    }
  }
}

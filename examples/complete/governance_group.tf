resource "identitynow_governance_group" "approvers" {
  name        = "Access Approvers"
  description = "Approves access requests for operator access profiles"

  owner {
    id   = data.identitynow_identity.john_doe.id
    name = data.identitynow_identity.john_doe.name
    type = "IDENTITY"
  }
}

resource "identitynow_governance_group_members" "approvers" {
  governance_group_id = identitynow_governance_group.approvers.id

  members {
    id   = data.identitynow_identity.john_doe.id
    name = data.identitynow_identity.john_doe.name
    type = "IDENTITY"
  }
}

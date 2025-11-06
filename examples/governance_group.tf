resource "identitynow_governance_group" "this" {
  name        = var.name
  description = var.description
  owner {
    id   = var.owner_id
    name = var.owner_name
    type = "IDENTITY"
  }
}

resource "identitynow_governance_group" "this" {
  name        = "Access Approvers"
  description = "Approves access requests"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }
}

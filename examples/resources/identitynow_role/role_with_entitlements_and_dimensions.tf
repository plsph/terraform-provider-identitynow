resource "identitynow_role" "advanced" {
  name        = "Advanced Role"
  description = "A role with entitlements and dimensions"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  entitlements {
    id   = "2c91808a7813090a017813b6301fabcd"
    type = "ENTITLEMENT"
    name = "Example Entitlement"
  }

  requestable = true
  enabled     = true
  dimensional = true
}

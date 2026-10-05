resource "identitynow_access_profile" "with_metadata" {
  name        = "AD Operators"
  description = "Operator access in Active Directory"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }

  source {
    id   = "2c9180835d191a86015d28455b4a2329"
    name = "Active Directory"
    type = "SOURCE"
  }

  entitlements {
    id   = "2c91808a7813090a017813b6301fabcd"
    name = "Operators"
  }

  access_model_metadata {
    attributes {
      key  = "iscPrivacy"
      name = "Privacy"

      values {
        value = "internal"
      }
    }
  }
}

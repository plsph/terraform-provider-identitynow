data "identitynow_identity" "owner" {
  alias = "john.doe"
}

resource "identitynow_identity_profile" "employees" {
  name        = "Employees"
  description = "Identities from the HR system"
  priority    = 10

  owner {
    id = data.identitynow_identity.owner.id
  }

  authoritative_source {
    id = "2c9180835d191a86015d28455b4a2329"
  }

  identity_attribute_config_enabled = true
  attribute_config_json = jsonencode([
    {
      identityAttributeName = "email"
      transformDefinition = {
        type = "accountAttribute"
        attributes = {
          sourceName    = "HR"
          attributeName = "mail"
        }
      }
    }
  ])
}

resource "identitynow_account_schema" "hr_account" {
  source_id          = "2c9180835d191a86015d28455b4a2329"
  schema_id          = "2c9180835d191a86015d28455b4a2331"
  identity_attribute = "employeeId"
  display_attribute  = "employeeId"

  # All attributes of the schema must be listed, unlisted attributes are removed.
  attributes {
    name        = "employeeId"
    type        = "STRING"
    description = "Employee ID"
  }

  attributes {
    name        = "email"
    type        = "STRING"
    description = "Work email address"
  }

  attributes {
    name            = "groups"
    type            = "STRING"
    description     = "Groups of the account"
    is_multi_valued = true
    is_entitlement  = true
    is_group        = true

    schema {
      id   = "2c9180835d191a86015d28455b4a2332"
      name = "group"
      type = "CONNECTOR_SCHEMA"
    }
  }
}

resource "identitynow_account_schema" "active_directory_account" {
  source_id          = "2c9180835d191a86015d28455b4a2329"
  schema_id          = "2c9180835d191a86015d28455b4a2330"
  identity_attribute = "distinguishedName"
  display_attribute  = "sAMAccountName"
}

resource "identitynow_source_provisioning_policy" "create" {
  source_id   = "2c9180835d191a86015d28455b4a2329"
  usage_type  = "CREATE"
  name        = "Account"
  description = "Attributes of new accounts"
  fields_json = jsonencode([
    {
      name = "userName"
      type = "string"
      transform = {
        type       = "identityAttribute"
        attributes = { name = "uid" }
      }
    },
    {
      name          = "groups"
      type          = "string"
      isMultiValued = true
    }
  ])
}

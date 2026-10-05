variable "db_password_jwe" {
  description = "Private fields of the database credential, JWE encrypted."
  type        = string
  sensitive   = true
}

resource "identitynow_parameter" "db_credential" {
  name        = "Database service account"
  description = "Credential of the HR database connector"
  type        = "password"
  owner_id    = "2c9180835d2e5168015d32f890ca1581"
  public_fields_json = jsonencode({
    username = "svc-hr"
  })
  private_fields = var.db_password_jwe
}

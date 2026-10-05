resource "identitynow_password_policy" "example" {
  name                    = "Primary Password Policy"
  description             = "Password policy for the Active Directory source"
  min_length              = 14
  max_length              = 64
  min_alpha               = 1
  min_lower               = 1
  min_upper               = 1
  min_numeric             = 1
  min_special             = 1
  use_history             = 12
  use_account_attributes  = true
  use_identity_attributes = true

  enable_password_expiration = true
  password_expiration        = 90
  first_expiration_reminder  = 14

  source_ids = ["2c9180835d191a86015d28455b4a2329"]
}

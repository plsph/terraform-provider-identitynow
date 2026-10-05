resource "identitynow_password_policy" "password_policy" {
  name                    = "Primary Password Policy"
  description             = "Password Policy for Azure Active Directory and Active Directory Sources"
  min_alpha               = 1
  min_length              = 14
  min_lower               = 1
  min_numeric             = 1
  min_special             = 1
  min_upper               = 1
  use_account_attributes  = true #if true it prevents the use of account attributes
  use_identity_attributes = true #if true it prevents the use of identity attributes
  use_history             = 12

  source_ids = [
    identitynow_source.active_directory_source.id,
    identitynow_source.azure_ad_source.id,
  ]
}

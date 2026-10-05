resource "identitynow_source_app" "azure_ad_app" {
  name               = "Azure AD Operators App"
  description        = "Application for requesting Azure AD operator access"
  enabled            = true
  match_all_accounts = false

  source {
    id   = identitynow_source.azure_ad_source.id
    name = identitynow_source.azure_ad_source.name
    type = "SOURCE"
  }
}

resource "identitynow_access_profile_attachment" "azure_ad_app" {
  source_app_id   = identitynow_source_app.azure_ad_app.id
  access_profiles = [identitynow_access_profile.aad_access_profile_operators.id]
}

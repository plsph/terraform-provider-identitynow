resource "identitynow_source_app" "example" {
  name        = "Active Directory Developers"
  description = "Application for requesting developer access"

  source {
    id   = "2c9180835d191a86015d28455b4a2329"
    name = "Active Directory"
  }
}

resource "identitynow_access_profile_attachment" "example" {
  source_app_id = identitynow_source_app.example.id
  access_profiles = [
    "2c91808a7813090a017813b6301f0044",
    "2c91808a7813090a017813b6301f0045",
  ]
}

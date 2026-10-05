resource "identitynow_source_app" "example" {
  name               = "Active Directory Developers"
  description        = "Application for requesting developer access"
  enabled            = true
  match_all_accounts = true

  source {
    id   = "2c9180835d191a86015d28455b4a2329"
    name = "Active Directory"
    type = "SOURCE"
  }
}

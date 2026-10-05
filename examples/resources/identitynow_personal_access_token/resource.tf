resource "identitynow_personal_access_token" "reporting" {
  name                          = "reporting"
  scope                         = ["sp:scopes:all"]
  access_token_validity_seconds = 3600
  expiration_date               = "2027-12-31T23:59:59Z"
}

# A token that never expires must be acknowledged explicitly.
resource "identitynow_personal_access_token" "service" {
  name                           = "service"
  user_aware_token_never_expires = true
}

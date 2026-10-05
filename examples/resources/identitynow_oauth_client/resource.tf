resource "identitynow_oauth_client" "ci" {
  name                          = "CI pipeline"
  description                   = "Used by the CI pipeline to deploy configuration"
  access_token_validity_seconds = 750
  grant_types                   = ["CLIENT_CREDENTIALS"]
  access_type                   = "OFFLINE"
  scope                         = ["sp:scopes:all"]
}

output "ci_client_secret" {
  value     = identitynow_oauth_client.ci.secret
  sensitive = true
}

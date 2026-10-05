resource "identitynow_oauth_client" "ci" {
  name                          = "CI pipeline"
  description                   = "Used by the CI pipeline to deploy configuration"
  access_token_validity_seconds = 750
  grant_types                   = ["CLIENT_CREDENTIALS"]
  access_type                   = "OFFLINE"
  scope                         = ["sp:scopes:all"]
}

# The secret is only returned when the client is created.
output "ci_client_secret" {
  value     = identitynow_oauth_client.ci.secret
  sensitive = true
}

resource "identitynow_personal_access_token" "reporting" {
  name                          = "reporting"
  scope                         = ["sp:scopes:all"]
  access_token_validity_seconds = 3600
  expiration_date               = "2027-12-31T23:59:59Z"
}

# Experimental API.
data "identitynow_authorization_right_sets" "identity" {
  category = "identity"
}

# Experimental API.
resource "identitynow_custom_user_level" "identity_viewers" {
  name        = "Identity Viewers"
  description = "Read only access to identities"
  right_sets  = [for r in data.identitynow_authorization_right_sets.identity.right_sets : r.id if r.depth == 0]
  publish     = true

  owner {
    id = data.identitynow_identity.john_doe.id
  }
}

resource "identitynow_parameter" "db_credential" {
  name        = "Database service account"
  description = "Credential of the HR database connector"
  type        = "password"
  owner_id    = data.identitynow_identity.john_doe.id
  public_fields_json = jsonencode({
    username = "svc-hr"
  })
  # JWE encrypted private fields, see the parameter storage attestation endpoint.
  private_fields = "<JWE_ENCRYPTED_PRIVATE_FIELDS>"
}

# Experimental API.
resource "identitynow_custom_password_instruction" "reset_password" {
  page_id      = "reset-password:enter-password"
  page_content = "See the company password policy <a href=\"https://intranet.example.com/passwords\" target=\"_blank\">here</a>."
}

resource "identitynow_branding" "corporate" {
  name                        = "corporate"
  product_name                = "Corporate Identity Portal"
  action_button_color         = "0074D9"
  navigation_color            = "011E69"
  login_informational_message = "Use your corporate account to sign in."
}

resource "identitynow_launcher" "onboarding" {
  name        = "Start onboarding"
  description = "Starts the onboarding workflow"
  config_json = jsonencode({
    workflowId = "6b42d9be-61b6-46af-827e-ea29ba8aa3d9"
  })

  reference {
    id = "6b42d9be-61b6-46af-827e-ea29ba8aa3d9"
  }
}

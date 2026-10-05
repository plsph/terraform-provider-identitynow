variable "okta_api_token" {
  description = "Okta API token."
  type        = string
  sensitive   = true
}

resource "identitynow_mfa_okta_config" "this" {
  enabled            = true
  host               = "example.okta.com"
  identity_attribute = "email"
  access_key         = var.okta_api_token
}

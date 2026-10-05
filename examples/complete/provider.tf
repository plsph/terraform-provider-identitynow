terraform {
  required_providers {
    identitynow = {
      source  = "plsph/identitynow"
      version = "~> 0.17"
    }
  }
}

# api_url, client_id and client_secret can also be set with the IDENTITYNOW_URL,
# IDENTITYNOW_CLIENT_ID and IDENTITYNOW_CLIENT_SECRET environment variables.
provider "identitynow" {
  api_url       = "https://example.api.identitynow.com"
  client_id     = var.api_client_id
  client_secret = var.api_client_secret
}

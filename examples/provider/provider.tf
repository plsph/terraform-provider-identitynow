# We strongly recommend using the required_providers block to set the
# IdentityNow Provider source and version being used
terraform {
  required_providers {
    identitynow = {
      source  = "plsph/identitynow"
      version = "~> 0.17"
    }
  }
}

# Configure the IdentityNow Provider
provider "identitynow" {
  api_url                   = "https://<org_name>.api.identitynow.com"
  client_id                 = "<client_id>"
  client_secret             = "<client_secret>"
  max_client_pool_size      = 1
  default_client_pool_size  = 1
  client_request_rate_limit = 10
}

# An additional provider configuration using a pool of two API clients
provider "identitynow" {
  alias   = "pool"
  api_url = "https://<org_name>.api.identitynow.com"
  credentials = [
    { client_id = "<client_id1>", client_secret = "<client_secret1>" },
    { client_id = "<client_id2>", client_secret = "<client_secret2>" },
  ]
  max_client_pool_size      = 2
  default_client_pool_size  = 2
  client_request_rate_limit = 10
}

# Create a governance group
resource "identitynow_governance_group" "example" {
  name        = "Example Approvers"
  description = "Example governance group"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }
}

# Create a governance group with the aliased provider configuration
resource "identitynow_governance_group" "pooled" {
  provider    = identitynow.pool
  name        = "Example Reviewers"
  description = "Example governance group created with the client pool"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }
}

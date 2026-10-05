# Connector attributes (IQService, forest and domain settings, search DNs, ...) are not managed by
# the identitynow_source resource. Configure them in IdentityNow, they are kept when Terraform updates the source.
resource "identitynow_source" "active_directory_source" {
  name             = "Active Directory Source"
  description      = "The Active Directory connector created by terraform"
  connector        = "active-directory"
  authoritative    = false
  delete_threshold = 10

  owner {
    id   = data.identitynow_identity.john_doe.id
    name = data.identitynow_identity.john_doe.name
    type = "IDENTITY"
  }

  cluster {
    id   = "<CLUSTER_ID>"
    name = "<CLUSTER_NAME>"
    type = "CLUSTER"
  }
}

# Manages settings of the existing account schema of the source. Without attributes blocks the
# schema attributes are left unchanged.
resource "identitynow_account_schema" "active_directory_account" {
  source_id          = identitynow_source.active_directory_source.id
  schema_id          = "<ACCOUNT_SCHEMA_ID>"
  identity_attribute = "distinguishedName"
  display_attribute  = "sAMAccountName"
}

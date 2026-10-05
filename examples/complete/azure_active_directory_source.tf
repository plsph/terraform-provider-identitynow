resource "identitynow_source" "azure_ad_source" {
  name             = "Azure Active Directory Source"
  description      = "The Azure Active Directory connector created by terraform"
  connector        = "azure-active-directory"
  authoritative    = false
  delete_threshold = 10

  owner {
    id   = data.identitynow_identity.john_doe.id
    name = data.identitynow_identity.john_doe.name
    type = "IDENTITY"
  }
}

# To schedule an account aggregation. The source must be configured and its connection tested first.
resource "identitynow_schedule_account_aggregation" "azure_ad_aggregation" {
  source_id        = "<SOURCE_CLOUD_EXTERNAL_ID>" # legacy (cc) ID of the source, see the resource documentation
  cron_expressions = ["0 0 * * * ?"]              # aggregate every hour
}

# Account create policy of the Active Directory source.
resource "identitynow_source_provisioning_policy" "ad_create" {
  source_id   = identitynow_source.active_directory_source.id
  usage_type  = "CREATE"
  name        = "Account"
  description = "Attributes of new Active Directory accounts"
  fields_json = jsonencode([
    {
      name = "sAMAccountName"
      type = "string"
      transform = {
        type       = "identityAttribute"
        attributes = { name = "uid" }
      }
    },
    {
      name = "mail"
      type = "string"
      transform = {
        type       = "identityAttribute"
        attributes = { name = "email" }
      }
    }
  ])
}

# Group aggregation every day at 02:00.
resource "identitynow_source_schedule" "ad_group_aggregation" {
  source_id       = identitynow_source.active_directory_source.id
  type            = "GROUP_AGGREGATION"
  cron_expression = "0 0 2 * * ?"
}

resource "identitynow_source_subtype" "ad_service_account" {
  source_id      = identitynow_source.active_directory_source.id
  technical_name = "service_account"
  display_name   = "Service Account"
  description    = "Active Directory service accounts"
}

resource "identitynow_connector_rule" "before_create" {
  name        = "AD Before Create"
  description = "Logs account creation"
  type        = "ConnectorBeforeCreate"

  source_code {
    version = "1.0"
    script  = <<-EOT
      log.info("Creating account for " + application.getName());
    EOT
  }
}

resource "identitynow_connector" "custom" {
  name       = "My Custom Connector"
  class_name = "sailpoint.connector.OpenConnectorAdapter"
  status     = "DEVELOPMENT"
}

resource "identitynow_connector_customizer" "custom" {
  name = "My Connector Customizer"
}

resource "identitynow_multihost" "sql_servers" {
  name        = "SQL Servers"
  description = "Microsoft SQL Server databases"
  connector   = "multihost-microsoft-sql-server"
  connector_attributes_json = jsonencode({
    authType = "SQLAuthentication"
    user     = "svc_identitynow"
    password = "<SQL_SERVER_PASSWORD>"
  })
  max_sources_per_agg_group = 10

  owner {
    id = data.identitynow_identity.john_doe.id
  }

  management_workgroup {
    id = identitynow_governance_group.approvers.id
  }
}

resource "identitynow_password_sync_group" "ad_and_entra" {
  name               = "AD and Entra ID"
  password_policy_id = identitynow_password_policy.password_policy.id
  source_ids = [
    identitynow_source.active_directory_source.id,
    identitynow_source.azure_ad_source.id,
  ]
}

data "identitynow_connector_rule" "before_create" {
  name = identitynow_connector_rule.before_create.name
}

data "identitynow_source_schedule" "ad_account_aggregation" {
  source_id = identitynow_source.active_directory_source.id
  type      = "ACCOUNT_AGGREGATION"
}

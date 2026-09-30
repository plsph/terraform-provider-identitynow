# Virtual appliance cluster and one of its virtual appliances.
resource "identitynow_managed_cluster" "va_cluster" {
  name        = "Primary VA Cluster"
  type        = "idn"
  description = "Virtual appliances in the primary data center"
  configuration = {
    gmtOffset = "-5"
  }
}

resource "identitynow_managed_client" "va_1" {
  cluster_id  = identitynow_managed_cluster.va_cluster.id
  name        = "VA 1"
  description = "First virtual appliance of the primary cluster"
  type        = "VA"
}

resource "identitynow_managed_cluster_type" "custom" {
  type                = "custom-cluster"
  pod                 = "<POD>"
  org                 = "<ORG>"
  managed_process_ids = ["<MANAGED_PROCESS_ID>"]
}

# Supported service desk integration types, e.g. ServiceNowSDIM.
data "identitynow_service_desk_integration_types" "all" {}

resource "identitynow_service_desk_integration" "servicenow" {
  name            = "ServiceNow"
  description     = "Creates ServiceNow tickets for Active Directory provisioning"
  type            = "ServiceNowSDIM"
  managed_sources = [identitynow_source.active_directory_source.id]

  provisioning_config_json = jsonencode({
    noProvisioningRequests        = false
    provisioningRequestExpiration = 7
  })

  attributes_json = jsonencode({
    url      = "https://<INSTANCE>.service-now.com"
    username = "<SERVICENOW_USER>"
    password = "<SERVICENOW_PASSWORD>"
  })

  owner_ref {
    id = data.identitynow_identity.john_doe.id
  }

  cluster_ref {
    id = identitynow_managed_cluster.va_cluster.id
  }
}

resource "identitynow_sim_integration" "servicenow_sim" {
  name        = "ServiceNow SIM"
  description = "Service integration module for ServiceNow"
  type        = "ServiceNow Service Desk"
  sources     = [identitynow_source.active_directory_source.id]
  cluster     = identitynow_managed_cluster.va_cluster.id

  status_map_json = jsonencode({
    closed_complete  = "Committed"
    closed_cancelled = "Failed"
    in_process       = "Queued"
  })

  request_json = jsonencode({
    short_description = "SailPoint Access Request $!plan.arguments.identityRequestId"
  })

  attributes_json = jsonencode({
    url      = "https://<INSTANCE>.service-now.com"
    username = "<SERVICENOW_USER>"
    password = "<SERVICENOW_PASSWORD>"
  })
}

# Data Access Security: a monitored file share and a weekly crawl of it.
resource "identitynow_das_application" "file_share" {
  name             = "Finance File Share"
  description      = "Finance department file share"
  application_type = 8

  application_crawler_settings_json = jsonencode({
    isEnabled = true
    clusterId = "<DAS_CLUSTER_ID>"
  })

  permission_collector_settings_json = jsonencode({
    isEnabled                     = true
    clusterId                     = "<DAS_CLUSTER_ID>"
    calculateEffectivePermissions = true
  })
}

resource "identitynow_das_task_schedule" "file_share_crawl" {
  task_type_name = "Crawl"
  schedule_type  = "Weekly"
  interval       = 1
  days_of_week   = ["Monday"]
  active         = true
  application_id = identitynow_das_application.file_share.id
}

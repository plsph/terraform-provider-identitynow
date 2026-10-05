resource "identitynow_sim_integration" "servicenow_sim" {
  name        = "ServiceNow SIM"
  description = "Service integration module for ServiceNow"
  type        = "ServiceNow Service Desk"
  sources     = ["<SOURCE_ID>"]
  cluster     = "<CLUSTER_ID>"

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

  before_provisioning_rule {
    id = "<RULE_ID>"
  }
}

data "identitynow_identity" "john_doe" {
  alias = "A12BCDE3F"
}

resource "identitynow_service_desk_integration" "servicenow" {
  name            = "ServiceNow"
  description     = "Creates ServiceNow tickets for Active Directory provisioning"
  type            = "ServiceNowSDIM"
  managed_sources = ["<SOURCE_ID>"]

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
    id = "<CLUSTER_ID>"
  }

  before_provisioning_rule {
    id = "<RULE_ID>"
  }
}

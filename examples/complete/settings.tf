# Tenant settings. Each resource manages a tenant-wide singleton: only the configured settings
# are managed, the other settings keep their current value. Destroying a resource only removes
# it from Terraform state.

data "identitynow_tenant" "current" {}

data "identitynow_org_config" "current" {}

data "identitynow_valid_time_zones" "all" {}

resource "identitynow_org_config" "this" {
  time_zone = "Europe/Warsaw"

  lifecycle {
    precondition {
      condition     = contains(data.identitynow_valid_time_zones.all.time_zones, "Europe/Warsaw")
      error_message = "Europe/Warsaw is not a valid time zone."
    }
  }
}

resource "identitynow_access_request_config" "this" {
  request_on_behalf_of_employee_by_manager = true
  request_on_behalf_of_anyone_by_anyone    = false

  # Merged key by key into the current entitlement request configuration.
  entitlement_request_config_json = jsonencode({
    accessRequestConfig = {
      requestCommentRequired = true
      denialCommentRequired  = true
      approvalSchemes = [
        { approverType = "MANAGER", approverId = null }
      ]
    }
  })
}

resource "identitynow_lockout_config" "this" {
  maximum_attempts = 5
  lockout_duration = 15
  lockout_window   = 5
}

resource "identitynow_session_config" "this" {
  max_idle_time    = 15
  max_session_time = 480
  remember_me      = false
}

resource "identitynow_network_config" "this" {
  range       = ["10.0.0.0/8"]
  geolocation = ["PL", "DE"]
  whitelisted = true
}

resource "identitynow_service_provider_config" "this" {
  # Enable SAML sign-in once the identity provider settings are verified.
  enabled                = false
  idp_entity_id          = "http://www.okta.com/exk0000000000000000"
  idp_binding            = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
  idp_name_id            = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
  idp_mapping_attribute  = "email"
  idp_login_url_post     = "https://example.okta.com/app/example/sso/saml"
  idp_login_url_redirect = "https://example.okta.com/app/example/sso/saml"
  idp_cert               = "<BASE64_ENCODED_IDP_CERTIFICATE>"
}

resource "identitynow_role_propagation_config" "this" {
  enabled = true
}

resource "identitynow_reassignment_tenant_config" "this" {
  disabled = false
}

resource "identitynow_mfa_duo_config" "this" {
  enabled            = true
  host               = "api-00000000.duosecurity.com"
  identity_attribute = "email"
  access_key         = "<DUO_ACCESS_KEY>"
  config_properties_json = jsonencode({
    ikey = "<DUO_INTEGRATION_KEY>"
    skey = "<DUO_SECRET_KEY>"
  })
}

resource "identitynow_mfa_okta_config" "this" {
  enabled            = false
  host               = "example.okta.com"
  identity_attribute = "email"
  access_key         = "<OKTA_API_TOKEN>"
}

resource "identitynow_campaign_reports_config" "this" {
  identity_attribute_columns = ["department", "location"]
}

resource "identitynow_recommendations_config" "this" {
  recommender_features            = ["jobTitle", "department", "location"]
  peer_group_percentage_threshold = 0.5
}

resource "identitynow_ai_access_request_recommendations_config" "this" {
  score_threshold           = 0.5
  restriction_attribute     = "location"
  use_restriction_attribute = true
}

resource "identitynow_sdi_status_check_config" "this" {
  provisioning_status_check_interval_minutes = 30
  provisioning_max_status_check_days         = 2
}

resource "identitynow_ui_metadata" "this" {
  username_label      = "Work email"
  username_empty_text = "Please provide your work email address"
}

output "tenant_region" {
  value = data.identitynow_tenant.current.region
}

output "org_name" {
  value = data.identitynow_org_config.current.org_name
}

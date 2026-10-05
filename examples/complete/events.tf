# Event triggers: deliver "identity created" events to a webhook and to AWS EventBridge.
data "identitynow_trigger" "identity_created" {
  id = "idn:identity-created"
}

resource "identitynow_trigger_subscription" "identity_created_webhook" {
  name        = "Identity created webhook"
  description = "Notifies the HR integration about new engineering identities"
  trigger_id  = data.identitynow_trigger.identity_created.id
  type        = "HTTP"
  filter      = "$[?($.attributes.department == \"Engineering\")]"

  http_config {
    url                = "https://hooks.example.com/identity-created"
    http_dispatch_mode = "ASYNC"
  }
}

resource "identitynow_trigger_subscription" "identity_created_eventbridge" {
  name       = "Identity created to EventBridge"
  trigger_id = data.identitynow_trigger.identity_created.id
  type       = "EVENTBRIDGE"
  enabled    = false

  event_bridge_config {
    aws_account = "123456789012"
    aws_region  = "eu-central-1"
  }
}

# Notifications: a verified sender address and a customized email template.
resource "identitynow_verified_from_address" "no_reply" {
  email = "no-reply@example.com"
}

resource "identitynow_notification_template" "work_item_summary" {
  key     = "cloud_manual_work_item_summary"
  medium  = "EMAIL"
  locale  = "en"
  subject = "You have $numItems pending work items"
  body    = "<p>Hello $recipient.name,</p><p>please review your pending work items.</p>"
  from    = identitynow_verified_from_address.no_reply.email
}

# Non-Employee Lifecycle Management: a source for contractors with a custom attribute.
resource "identitynow_non_employee_source" "contractors" {
  name             = "Contractors"
  description      = "External contractors"
  approvers        = [data.identitynow_identity.john_doe.id]
  account_managers = [data.identitynow_identity.john_doe.id]

  owner {
    id = data.identitynow_identity.john_doe.id
  }
}

resource "identitynow_non_employee_schema_attribute" "cost_center" {
  non_employee_source_id = identitynow_non_employee_source.contractors.id
  technical_name         = "costCenter"
  label                  = "Cost center"
  help_text              = "Cost center that pays the contractor"
  required               = true
}

# Work reassignment: forward John's access request approvals during his vacation.
resource "identitynow_reassignment_configuration" "john_doe_access_requests" {
  identity_id      = data.identitynow_identity.john_doe.id
  config_type      = "ACCESS_REQUESTS"
  reassigned_to_id = "2c9180867624cbd7017642d8c8c81f67"
  start_date       = "2026-10-01T00:00:00Z"
  end_date         = "2026-10-15T00:00:00Z"
}

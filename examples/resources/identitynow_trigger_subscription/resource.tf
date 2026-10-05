variable "webhook_token" {
  type      = string
  sensitive = true
}

resource "identitynow_trigger_subscription" "identity_created_webhook" {
  name        = "Identity created webhook"
  description = "Notifies the HR integration about new identities"
  trigger_id  = "idn:identity-created"
  type        = "HTTP"
  filter      = "$[?($.attributes.department == \"Engineering\")]"

  http_config {
    url                      = "https://hooks.example.com/identity-created"
    http_dispatch_mode       = "SYNC"
    http_authentication_type = "BEARER_TOKEN"
    bearer_token             = var.webhook_token
  }
}

resource "identitynow_trigger_subscription" "identity_created_eventbridge" {
  name       = "Identity created to EventBridge"
  trigger_id = "idn:identity-created"
  type       = "EVENTBRIDGE"

  event_bridge_config {
    aws_account = "123456789012"
    aws_region  = "eu-central-1"
  }
}

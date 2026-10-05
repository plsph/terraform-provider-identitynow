resource "identitynow_lifecycle_state" "inactive" {
  identity_profile_id = "2b838de9-db9b-abcf-e646-d4f274ad4238"
  name                = "Inactive"
  technical_name      = "inactive"
  description         = "Identities that left the organization"
  enabled             = true
  identity_state      = "INACTIVE_LONG_TERM"

  account_actions_json = jsonencode([
    {
      action     = "DISABLE"
      allSources = true
    }
  ])
  remove_all_access_enabled = true

  email_notification_option {
    notify_managers       = true
    notify_specific_users = true
    email_address_list    = ["it-security@example.com"]
  }
}

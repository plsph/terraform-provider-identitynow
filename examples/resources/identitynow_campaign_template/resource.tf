resource "identitynow_campaign_template" "quarterly_managers" {
  name              = "Quarterly manager certification"
  description       = "Managers review the access of their reports every quarter."
  deadline_duration = "P2W"

  campaign_json = jsonencode({
    name                     = "Manager certification %TIME_PERIOD%"
    description              = "Review the access of your reports"
    type                     = "MANAGER"
    emailNotificationEnabled = true
    autoRevokeAllowed        = false
    recommendationsEnabled   = true
  })
}

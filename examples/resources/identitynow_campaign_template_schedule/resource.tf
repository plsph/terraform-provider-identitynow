resource "identitynow_campaign_template" "quarterly_managers" {
  name        = "Quarterly manager certification"
  description = "Managers review the access of their reports every quarter."

  campaign_json = jsonencode({
    name        = "Manager certification"
    description = "Review the access of your reports"
    type        = "MANAGER"
  })
}

# Runs at 9 AM on the first day of January, April, July and October.
resource "identitynow_campaign_template_schedule" "quarterly_managers" {
  campaign_template_id = identitynow_campaign_template.quarterly_managers.id
  type                 = "ANNUALLY"
  time_zone_id         = "Europe/Warsaw"

  months {
    type     = "LIST"
    values   = ["1"]
    interval = 3
  }

  days {
    type   = "LIST"
    values = ["1"]
  }

  hours {
    type   = "LIST"
    values = ["9"]
  }
}

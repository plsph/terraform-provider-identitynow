resource "identitynow_notification_template" "work_item_summary" {
  key     = "cloud_manual_work_item_summary"
  medium  = "EMAIL"
  locale  = "en"
  subject = "You have $numItems pending work items"
  body    = "<p>Hello $recipient.name,</p><p>please review your pending work items.</p>"
  from    = "no-reply@example.com"
}

resource "identitynow_notification_template" "work_item_summary_slack" {
  key    = "cloud_manual_work_item_summary"
  medium = "SLACK"
  locale = "en"
  slack_template_json = jsonencode({
    text = "You have $numItems pending work items"
  })
}

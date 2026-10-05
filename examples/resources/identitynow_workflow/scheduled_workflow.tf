resource "identitynow_workflow" "scheduled_report" {
  name        = "Weekly Compliance Report"
  description = "Generate a weekly compliance report."
  enabled     = false

  owner {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
    name = "William Wilson"
  }

  trigger {
    type = "SCHEDULED"
    attributes_json = jsonencode({
      cronString = "0 0 9 ? * MON"
    })
  }

  definition {
    start = "Generate Report"
    steps_json = jsonencode({
      "Generate Report" = {
        actionId = "sp:generate-report"
        attributes = {
          reportType = "compliance"
        }
        nextStep = "success"
        type     = "ACTION"
      }
      "success" = {
        type = "success"
      }
    })
  }
}

resource "identitynow_sod_policy_schedule" "weekly" {
  policy_id           = "0f11f2a4-7c94-4bf3-a2bd-742580fe3bde"
  name                = "Weekly SOD report"
  description         = "Sent every Monday at 8 AM"
  email_empty_results = false

  schedule_json = jsonencode({
    type       = "WEEKLY"
    days       = { type = "LIST", values = ["MON"] }
    hours      = { type = "LIST", values = ["8"] }
    timeZoneId = "Europe/Warsaw"
  })

  recipient {
    id = "2c91808568c529c60168cca6f90c1313"
  }
}

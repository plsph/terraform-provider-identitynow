resource "identitynow_source_schedule" "account_aggregation" {
  source_id       = "2c9180835d191a86015d28455b4a2329"
  type            = "ACCOUNT_AGGREGATION"
  cron_expression = "0 0 12 1/1 * ? *" # every day at 12:00
}

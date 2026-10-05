resource "identitynow_schedule_account_aggregation" "active_directory" {
  source_id        = "123456"
  cron_expressions = ["0 0 * * * ?"] # aggregate every hour
}

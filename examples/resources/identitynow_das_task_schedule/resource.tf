resource "identitynow_das_task_schedule" "file_share_crawl" {
  task_type_name = "Crawl"
  schedule_type  = "Weekly"
  interval       = 1
  days_of_week   = ["Monday"]
  active         = true
  application_id = 42
}

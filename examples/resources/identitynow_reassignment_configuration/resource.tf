resource "identitynow_reassignment_configuration" "vacation_access_requests" {
  identity_id      = "2c9180867624cbd7017642d8c8c81f67"
  config_type      = "ACCESS_REQUESTS"
  reassigned_to_id = "2c9180867624cbd7017642d8c8c81f68"
  start_date       = "2026-10-01T00:00:00Z"
  end_date         = "2026-10-15T00:00:00Z"
}

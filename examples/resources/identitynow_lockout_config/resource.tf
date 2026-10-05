resource "identitynow_lockout_config" "this" {
  maximum_attempts = 5
  lockout_duration = 15
  lockout_window   = 5
}

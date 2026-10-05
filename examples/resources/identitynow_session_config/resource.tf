resource "identitynow_session_config" "this" {
  max_idle_time    = 15
  max_session_time = 480
  remember_me      = false
}

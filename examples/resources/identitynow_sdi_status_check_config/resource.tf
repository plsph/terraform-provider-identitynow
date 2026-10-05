resource "identitynow_sdi_status_check_config" "this" {
  provisioning_status_check_interval_minutes = 30
  provisioning_max_status_check_days         = 2
}

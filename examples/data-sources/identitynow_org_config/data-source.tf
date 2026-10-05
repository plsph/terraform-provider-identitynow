data "identitynow_org_config" "current" {}

output "org_time_zone" {
  value = data.identitynow_org_config.current.time_zone
}

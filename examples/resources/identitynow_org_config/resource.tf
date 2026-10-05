data "identitynow_valid_time_zones" "all" {}

resource "identitynow_org_config" "this" {
  time_zone = "Europe/Warsaw"
}

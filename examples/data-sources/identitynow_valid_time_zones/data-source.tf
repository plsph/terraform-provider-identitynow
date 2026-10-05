data "identitynow_valid_time_zones" "all" {}

resource "identitynow_org_config" "this" {
  time_zone = "Europe/Warsaw"

  lifecycle {
    precondition {
      condition     = contains(data.identitynow_valid_time_zones.all.time_zones, "Europe/Warsaw")
      error_message = "Europe/Warsaw is not a valid time zone."
    }
  }
}

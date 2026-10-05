resource "identitynow_managed_cluster" "va_cluster" {
  name        = "Primary VA Cluster"
  type        = "idn"
  description = "Virtual appliances in the primary data center"
  configuration = {
    gmtOffset = "-5"
  }
}

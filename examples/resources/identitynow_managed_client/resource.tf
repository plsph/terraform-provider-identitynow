resource "identitynow_managed_client" "va_1" {
  cluster_id  = "<CLUSTER_ID>"
  name        = "VA 1"
  description = "First virtual appliance of the primary cluster"
  type        = "VA"
}

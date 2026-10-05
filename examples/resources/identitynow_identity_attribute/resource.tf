resource "identitynow_identity_attribute" "cost_center" {
  name         = "costCenter"
  display_name = "Cost Center"
  type         = "string"
  searchable   = true
}

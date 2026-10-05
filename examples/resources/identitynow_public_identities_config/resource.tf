resource "identitynow_public_identities_config" "this" {
  attribute {
    key  = "country"
    name = "Country"
  }

  attribute {
    key  = "department"
    name = "Department"
  }
}

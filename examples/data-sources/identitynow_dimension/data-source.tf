data "identitynow_dimension" "example" {
  id      = "2c91808a7813090a017813b6301f1234"
  role_id = "2c91808a7813090a017813b6301fabcd"
}

output "identitynow_dimension_name" {
  value = data.identitynow_dimension.example.name
}

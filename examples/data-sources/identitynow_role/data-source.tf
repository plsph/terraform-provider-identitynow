data "identitynow_role" "example" {
  id = "2c91808a7813090a017813b6301f1234"
}

output "identitynow_role_name" {
  value = data.identitynow_role.example.name
}

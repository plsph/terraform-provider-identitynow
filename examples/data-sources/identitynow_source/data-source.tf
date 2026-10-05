data "identitynow_source" "example" {
  name = "Active Directory"
}

output "identitynow_source_description" {
  value = data.identitynow_source.example.description
}

output "identitynow_source_owner_name" {
  value = data.identitynow_source.example.owner[0].name
}

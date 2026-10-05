data "identitynow_source_app" "example" {
  name = "Active Directory Developers"
}

output "identitynow_source_app_description" {
  value = data.identitynow_source_app.example.description
}

output "identitynow_source_app_source_id" {
  value = data.identitynow_source_app.example.source[0].id
}

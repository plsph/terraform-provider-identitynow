data "identitynow_service_desk_integration_types" "all" {}

output "service_desk_integration_types" {
  value = data.identitynow_service_desk_integration_types.all.types[*].type
}

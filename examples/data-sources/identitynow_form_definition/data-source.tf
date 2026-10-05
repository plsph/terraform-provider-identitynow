data "identitynow_form_definition" "access_request" {
  name = "Access Request Justification"
}

output "form_definition_id" {
  value = data.identitynow_form_definition.access_request.id
}

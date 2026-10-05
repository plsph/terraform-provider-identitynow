data "identitynow_workflow" "example" {
  name = "Send Email on Manager Change"
}

output "workflow_id" {
  value = data.identitynow_workflow.example.id
}

output "workflow_enabled" {
  value = data.identitynow_workflow.example.enabled
}

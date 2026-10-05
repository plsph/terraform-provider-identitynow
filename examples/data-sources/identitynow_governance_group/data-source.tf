data "identitynow_governance_group" "example" {
  name = "Access Approvers"
}

output "identitynow_group_description" {
  value = data.identitynow_governance_group.example.description
}

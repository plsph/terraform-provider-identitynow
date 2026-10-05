data "identitynow_identity" "by_alias" {
  alias = "john.doe"
}

data "identitynow_identity" "by_email" {
  email_address = "john.doe@example.com"
}

output "identitynow_identity_name" {
  value = data.identitynow_identity.by_alias.name
}

output "identitynow_identity_email" {
  value = data.identitynow_identity.by_email.attributes[0].email
}

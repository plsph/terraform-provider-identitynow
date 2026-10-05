data "identitynow_access_profile" "example" {
  name = "AD Developers"
}

output "identitynow_ap_desc" {
  value = data.identitynow_access_profile.example.description
}

output "identitynow_ap_segments" {
  value = data.identitynow_access_profile.example.segments
}

output "identitynow_ap_source_id" {
  value = data.identitynow_access_profile.example.source[0].id
}

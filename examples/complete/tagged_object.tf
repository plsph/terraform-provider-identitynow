# Each resource manages the full tag set of its objects, so manage every object with one resource only.

# Tag multiple Access Profiles with the same tags
resource "identitynow_tagged_object" "access_profile_tags" {
  object_type = "ACCESS_PROFILE"
  object_ids = [
    identitynow_access_profile.aad_access_profile_operators.id,
    identitynow_access_profile.ad_access_profile_developers.id,
  ]
  tags = ["production", "quarterly-review"]
}

# Tag a Role
resource "identitynow_tagged_object" "role_tags" {
  object_type = "ROLE"
  object_ids  = [identitynow_role.operator_developer_role.id]
  tags        = ["critical", "audit-required"]
}

# Tag a Source
resource "identitynow_tagged_object" "source_tags" {
  object_type = "SOURCE"
  object_ids  = [identitynow_source.active_directory_source.id]
  tags        = ["active-directory", "hr-system"]
}

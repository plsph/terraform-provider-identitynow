# Tag an Access Profile
resource "identitynow_tagged_object" "access_profile_tags" {
  object_type = "ACCESS_PROFILE"
  object_ids  = [identitynow_access_profile.example.id]
  tags        = ["production", "finance"]
}

# Tag a Role
resource "identitynow_tagged_object" "role_tags" {
  object_type = "ROLE"
  object_ids  = [identitynow_role.example.id]
  tags        = ["critical", "audit-required"]
}

# Tag a Source
resource "identitynow_tagged_object" "source_tags" {
  object_type = "SOURCE"
  object_ids  = [identitynow_source.example.id]
  tags        = ["active-directory", "hr-system"]
}

# Tag multiple Access Profiles with the same tags
resource "identitynow_tagged_object" "finance_access_profiles" {
  object_type = "ACCESS_PROFILE"
  object_ids  = [
    identitynow_access_profile.ap1.id,
    identitynow_access_profile.ap2.id,
    identitynow_access_profile.ap3.id,
  ]
  tags = ["finance", "quarterly-review"]
}

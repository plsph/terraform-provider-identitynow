resource "identitynow_tagged_object" "access_profile_tags" {
  object_type = "ACCESS_PROFILE"
  object_ids  = ["2c91808568c529c60168cca6f90c1313"]
  tags        = ["production", "finance"]
}

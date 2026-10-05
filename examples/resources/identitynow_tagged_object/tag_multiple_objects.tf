resource "identitynow_tagged_object" "finance_access_profiles" {
  object_type = "ACCESS_PROFILE"
  object_ids = [
    "2c91808568c529c60168cca6f90c1314",
    "2c91808568c529c60168cca6f90c1315",
    "2c91808568c529c60168cca6f90c1316",
  ]
  tags = ["finance", "quarterly-review"]
}

resource "identitynow_sod_policy" "admins" {
  name         = "Privileged administrators"
  policy_query = "@access(name:\"Domain Admins\") AND @access(name:\"Security Auditors\")"

  owner_ref {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
  }
}

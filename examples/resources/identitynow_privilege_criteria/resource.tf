resource "identitynow_privilege_criteria" "admins" {
  source_id       = "c42c45d8d7c04d2da64d215cd8c32f21"
  operator        = "AND"
  privilege_level = "HIGH"

  groups_json = jsonencode([
    {
      operator = "OR"
      criteriaItems = [
        {
          targetType = "group"
          property   = "displayName"
          operator   = "CONTAINS"
          values     = ["admin", "superuser"]
          ignoreCase = true
        }
      ]
    }
  ])
}

resource "identitynow_data_segment" "emea" {
  name        = "EMEA"
  description = "Members only see EMEA entitlements."
  membership  = "FILTER"
  enabled     = true
  publish     = true

  member_filter_json = jsonencode({
    expression = {
      operator  = "EQUALS"
      attribute = "location"
      value = {
        type  = "STRING"
        value = "EMEA"
      }
    }
  })

  scopes_json = jsonencode([
    {
      scope      = "ENTITLEMENT"
      visibility = "FILTER"
      scopeFilter = {
        expression = {
          operator  = "EQUALS"
          attribute = "region"
          value = {
            type  = "STRING"
            value = "EMEA"
          }
        }
      }
    }
  ])
}

resource "identitynow_sod_policy" "payables_receivables" {
  name                  = "Payables vs Receivables"
  description           = "Nobody may both create and approve payments."
  type                  = "CONFLICTING_ACCESS_BASED"
  state                 = "ENFORCED"
  compensating_controls = "Monthly payment review"
  correction_advice     = "Remove one of the conflicting entitlements."
  tags                  = ["SOX"]

  owner_ref {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
  }

  violation_owner_assignment_config {
    assignment_rule = "STATIC"

    owner_ref {
      id   = "2c9180867624cbd7017642d8c8c81f67"
      type = "GOVERNANCE_GROUP"
    }
  }

  conflicting_access_criteria_json = jsonencode({
    leftCriteria = {
      name = "Payables"
      criteriaList = [
        { type = "ENTITLEMENT", id = "2c9180866166b5b0016167c32ef31a66" }
      ]
    }
    rightCriteria = {
      name = "Receivables"
      criteriaList = [
        { type = "ENTITLEMENT", id = "2c9180866166b5b0016167c32ef31a67" }
      ]
    }
  })
}

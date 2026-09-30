# Separation of duties: nobody may hold both entitlements.
resource "identitynow_sod_policy" "payables_receivables" {
  name              = "Payables vs Receivables"
  description       = "Nobody may both create and approve payments."
  type              = "CONFLICTING_ACCESS_BASED"
  state             = "ENFORCED"
  correction_advice = "Remove one of the conflicting entitlements."
  tags              = ["SOX"]

  owner_ref {
    id   = data.identitynow_identity.john_doe.id
    type = "IDENTITY"
  }

  violation_owner_assignment_config {
    assignment_rule = "MANAGER"
  }

  conflicting_access_criteria_json = jsonencode({
    leftCriteria = {
      name = "Payables"
      criteriaList = [
        { type = "ENTITLEMENT", id = "<PAYABLES_ENTITLEMENT_ID>" }
      ]
    }
    rightCriteria = {
      name = "Receivables"
      criteriaList = [
        { type = "ENTITLEMENT", id = "<RECEIVABLES_ENTITLEMENT_ID>" }
      ]
    }
  })
}

resource "identitynow_sod_policy_schedule" "payables_receivables" {
  policy_id = identitynow_sod_policy.payables_receivables.id
  name      = "Weekly SOD report"

  schedule_json = jsonencode({
    type  = "WEEKLY"
    days  = { type = "LIST", values = ["MON"] }
    hours = { type = "LIST", values = ["8"] }
  })

  recipient {
    id = data.identitynow_identity.john_doe.id
  }
}

# Quarterly manager certification.
resource "identitynow_campaign_template" "quarterly_managers" {
  name              = "Quarterly manager certification"
  description       = "Managers review the access of their reports every quarter."
  deadline_duration = "P2W"

  campaign_json = jsonencode({
    name                     = "Manager certification"
    description              = "Review the access of your reports"
    type                     = "MANAGER"
    emailNotificationEnabled = true
  })
}

resource "identitynow_campaign_template_schedule" "quarterly_managers" {
  campaign_template_id = identitynow_campaign_template.quarterly_managers.id
  type                 = "ANNUALLY"

  months {
    type     = "LIST"
    values   = ["1"]
    interval = 3
  }

  days {
    type   = "LIST"
    values = ["1"]
  }

  hours {
    type   = "LIST"
    values = ["9"]
  }
}

# Entitlements with "admin" in their name are highly privileged.
resource "identitynow_privilege_criteria" "admins" {
  source_id       = "<SOURCE_ID>"
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
          values     = ["admin"]
          ignoreCase = true
        }
      ]
    }
  ])
}

# Members located in EMEA only see EMEA entitlements.
resource "identitynow_data_segment" "emea" {
  name       = "EMEA"
  membership = "FILTER"
  enabled    = true
  publish    = true

  member_filter_json = jsonencode({
    expression = {
      operator  = "EQUALS"
      attribute = "location"
      value     = { type = "STRING", value = "EMEA" }
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
          value     = { type = "STRING", value = "EMEA" }
        }
      }
    }
  ])
}

# Daily email with the identities that have a disabled account.
resource "identitynow_saved_search" "disabled_accounts" {
  name    = "Identities with disabled accounts"
  indices = ["identities"]
  query   = "@accounts(disabled:true)"
  sort    = ["displayName"]
}

resource "identitynow_scheduled_search" "disabled_accounts_daily" {
  name            = "Daily disabled accounts report"
  saved_search_id = identitynow_saved_search.disabled_accounts.id
  enabled         = true

  schedule_json = jsonencode({
    type  = "DAILY"
    hours = { type = "LIST", values = ["9"] }
  })

  recipient {
    id = data.identitynow_identity.john_doe.id
  }
}

data "identitynow_sod_policy" "payables_receivables" {
  name = identitynow_sod_policy.payables_receivables.name
}

data "identitynow_campaign_template" "quarterly_managers" {
  id = identitynow_campaign_template.quarterly_managers.id
}

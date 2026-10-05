resource "identitynow_saved_search" "disabled_accounts" {
  name    = "Identities with disabled accounts"
  indices = ["identities"]
  query   = "@accounts(disabled:true)"
}

resource "identitynow_scheduled_search" "disabled_accounts_daily" {
  name                  = "Daily disabled accounts report"
  saved_search_id       = identitynow_saved_search.disabled_accounts.id
  enabled               = true
  email_empty_results   = false
  display_query_details = false

  schedule_json = jsonencode({
    type       = "DAILY"
    hours      = { type = "LIST", values = ["9"] }
    timeZoneId = "Europe/Warsaw"
  })

  recipient {
    id = "2c91808568c529c60168cca6f90c1313"
  }
}

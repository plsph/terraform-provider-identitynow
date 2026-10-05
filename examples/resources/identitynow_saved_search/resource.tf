resource "identitynow_saved_search" "disabled_accounts" {
  name        = "Identities with disabled accounts"
  description = "Identities that have at least one disabled account."
  indices     = ["identities"]
  query       = "@accounts(disabled:true)"
  sort        = ["displayName"]

  order_by = {
    identity = ["lastName", "firstName"]
  }

  columns_json = jsonencode({
    identity = [
      { field = "displayName", header = "Display Name" },
      { field = "email", header = "Work Email" }
    ]
  })

  filters_json = jsonencode({
    "source.name" = {
      type    = "TERMS"
      terms   = ["HR Employees"]
      exclude = false
    }
  })
}

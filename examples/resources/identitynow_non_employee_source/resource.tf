resource "identitynow_non_employee_source" "contractors" {
  name                 = "Contractors"
  description          = "External contractors"
  management_workgroup = "2c9180867624cbd7017642d8c8c81f68"
  approvers            = ["2c9180867624cbd7017642d8c8c81f67"]
  account_managers     = ["2c9180867624cbd7017642d8c8c81f67"]

  owner {
    id = "2c9180867624cbd7017642d8c8c81f67"
  }
}

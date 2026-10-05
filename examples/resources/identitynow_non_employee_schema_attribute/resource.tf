resource "identitynow_non_employee_source" "contractors" {
  name        = "Contractors"
  description = "External contractors"

  owner {
    id = "2c9180867624cbd7017642d8c8c81f67"
  }
}

resource "identitynow_non_employee_schema_attribute" "cost_center" {
  non_employee_source_id = identitynow_non_employee_source.contractors.id
  technical_name         = "costCenter"
  label                  = "Cost center"
  help_text              = "Cost center that pays the contractor"
  placeholder            = "CC-0000"
  required               = true
}

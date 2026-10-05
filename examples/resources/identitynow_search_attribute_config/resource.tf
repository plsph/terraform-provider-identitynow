resource "identitynow_search_attribute_config" "employee_number" {
  name         = "employeeNumber"
  display_name = "Employee Number"
  application_attributes = {
    "2c9180835d191a86015d28455b4a2329" = "employeeID"
    "2c918083746f642c01746f990884012a" = "empNo"
  }
}

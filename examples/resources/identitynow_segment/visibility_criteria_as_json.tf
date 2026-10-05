resource "identitynow_segment" "austin_json" {
  name   = "Austin employees (JSON)"
  active = true

  visibility_criteria_json = jsonencode({
    expression = {
      operator  = "EQUALS"
      attribute = "location"
      value = {
        type  = "STRING"
        value = "Austin"
      }
    }
  })
}

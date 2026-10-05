resource "identitynow_segment" "austin" {
  name        = "Austin employees"
  description = "Employees whose location is Austin"
  active      = true

  owner {
    id   = "2c9180a46faadee4016fb4e018c20639"
    type = "IDENTITY"
    name = "support"
  }

  visibility_criteria {
    expression {
      operator  = "EQUALS"
      attribute = "location"

      value {
        type  = "STRING"
        value = "Austin"
      }
    }
  }
}

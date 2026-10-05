resource "identitynow_segment" "austin_engineering" {
  name   = "Austin engineering"
  active = true

  visibility_criteria {
    expression {
      operator = "AND"

      children {
        expression {
          operator  = "EQUALS"
          attribute = "location"

          value {
            type  = "STRING"
            value = "Austin"
          }
        }
      }

      children {
        expression {
          operator  = "EQUALS"
          attribute = "department"

          value {
            type  = "STRING"
            value = "Engineering"
          }
        }
      }
    }
  }
}

resource "identitynow_segment" "austin" {
  name        = "Austin employees"
  description = "Employees whose location is Austin"
  active      = true

  owner {
    id   = var.owner_id
    type = "IDENTITY"
    name = var.owner_name
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

data "identitynow_segment" "existing" {
  name = "Existing segment"
}

output "existing_segment_id" {
  value = data.identitynow_segment.existing.id
}
resource "identitynow_form_definition" "access_request" {
  name        = "Access Request Justification"
  description = "Collects a business justification for an access request."

  owner {
    id   = data.identitynow_identity.john_doe.id
    type = "IDENTITY"
  }

  form_input {
    type        = "STRING"
    label       = "requestedFor"
    description = "Identity the access is requested for"
  }

  form_elements_json = jsonencode([
    {
      id          = "section1"
      elementType = "SECTION"
      config = {
        alignment  = "LEFT"
        label      = "Justification"
        labelStyle = "h2"
        showLabel  = true
        formElements = [
          {
            id          = "justification"
            key         = "justification"
            elementType = "TEXTAREA"
            config = {
              label    = "Business justification"
              helpText = "Explain why the access is needed"
              required = true
            }
            validations = [
              { validationType = "REQUIRED" }
            ]
          }
        ]
      }
    }
  ])
}

data "identitynow_form_definition" "existing" {
  name = identitynow_form_definition.access_request.name
}

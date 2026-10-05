resource "identitynow_form_definition" "access_request" {
  name        = "Access Request Justification"
  description = "Collects a business justification for an access request."

  owner {
    id   = "2c91808568c529c60168cca6f90c1313"
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
          },
          {
            id          = "urgent"
            key         = "urgent"
            elementType = "TOGGLE"
            config = {
              label = "Urgent request"
            }
            validations = []
          }
        ]
      }
    }
  ])

  form_conditions_json = jsonencode([
    {
      ruleOperator = "AND"
      rules = [
        {
          sourceType = "ELEMENT"
          source     = "urgent"
          operator   = "EQ"
          valueType  = "BOOLEAN"
          value      = "true"
        }
      ]
      effects = [
        {
          effectType = "REQUIRE"
          config = {
            element = "justification"
          }
        }
      ]
    }
  ])
}

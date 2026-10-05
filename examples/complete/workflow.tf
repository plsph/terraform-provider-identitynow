resource "identitynow_workflow" "email_on_manager_change" {
  name        = "Send Email on Manager Change"
  description = "Send an email to the identity when their manager attribute changes."
  enabled     = false

  owner {
    id   = data.identitynow_identity.john_doe.id
    type = "IDENTITY"
    name = data.identitynow_identity.john_doe.name
  }

  trigger {
    type = "EVENT"
    attributes_json = jsonencode({
      id         = "idn:identity-attributes-changed"
      "filter.$" = "$.changes[?(@.attribute == 'manager')]"
    })
  }

  definition {
    start = "Send Email"
    steps_json = jsonencode({
      "Send Email" = {
        actionId = "sp:send-email"
        attributes = {
          body            = "This is a test"
          from            = "sailpoint@sailpoint.com"
          "recipientId.$" = "$.identity.id"
          subject         = "test"
        }
        nextStep     = "success"
        selectResult = null
        type         = "ACTION"
      }
      "success" = {
        type = "success"
      }
    })
  }
}

data "identitynow_workflow" "existing" {
  name = identitynow_workflow.email_on_manager_change.name
}

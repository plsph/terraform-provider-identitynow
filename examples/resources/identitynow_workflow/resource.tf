resource "identitynow_workflow" "email_on_manager_change" {
  name        = "Send Email on Manager Change"
  description = "Send an email to the identity when their manager attribute changes."
  enabled     = false

  owner {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
    name = "William Wilson"
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
          body             = "This is a test"
          from             = "sailpoint@sailpoint.com"
          "recipientId.$"  = "$.identity.id"
          subject          = "test"
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

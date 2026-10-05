resource "identitynow_workflow" "external_trigger" {
  name        = "External Trigger Workflow"
  description = "A workflow triggered externally via API."
  enabled     = false

  owner {
    id   = "2c91808568c529c60168cca6f90c1313"
    type = "IDENTITY"
    name = "William Wilson"
  }

  trigger {
    type = "EXTERNAL"
  }

  definition {
    start = "Process Request"
    steps_json = jsonencode({
      "Process Request" = {
        actionId = "sp:send-email"
        attributes = {
          body    = "External trigger received"
          from    = "sailpoint@sailpoint.com"
          subject = "External Trigger"
        }
        nextStep = "success"
        type     = "ACTION"
      }
      "success" = {
        type = "success"
      }
    })
  }
}

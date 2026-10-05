resource "identitynow_launcher" "onboarding" {
  name        = "Start onboarding"
  description = "Starts the onboarding workflow"
  config_json = jsonencode({
    workflowId = "6b42d9be-61b6-46af-827e-ea29ba8aa3d9"
  })

  reference {
    id = "6b42d9be-61b6-46af-827e-ea29ba8aa3d9"
  }
}

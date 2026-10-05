resource "identitynow_connector_rule" "before_create" {
  name        = "AD Before Create"
  description = "Logs account creation"
  type        = "ConnectorBeforeCreate"
  signature_json = jsonencode({
    input = [
      { name = "plan", type = "ProvisioningPlan", description = "The provisioning plan" }
    ]
  })

  source_code {
    version = "1.0"
    script  = <<-EOT
      log.info("Creating account for " + application.getName());
    EOT
  }
}

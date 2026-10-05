resource "identitynow_connector" "custom" {
  name       = "My Custom Connector"
  class_name = "sailpoint.connector.OpenConnectorAdapter"
  status     = "DEVELOPMENT"
}

---
subcategory: "Connector"
layout: "identitynow"
page_title: "IdentityNow: identitynow_connector"
description: |-
  Manages an IdentityNow custom connector.
---

# identitynow_connector

Manages a custom connector. IdentityNow derives a unique `script_name` from the name, which is the ID of the connector.

If the create response does not contain the script name, the provider looks the connector up by its exact name in the connector list. The API documents that this list only contains connectors with the `RELEASED` status; when the connector cannot be found, the apply fails although the connector was created, and it has to be imported with its script name.

Only the connector metadata and the application, correlation config and source config XML can be updated; every other argument replaces the connector when changed. Uploading connector files (JAR files, source config and source template uploads) and translations is not supported by this resource.

## Example Usage

```hcl
resource "identitynow_connector" "custom" {
  name       = "My Custom Connector"
  class_name = "sailpoint.connector.OpenConnectorAdapter"
  status     = "DEVELOPMENT"
}
```

## Arguments Reference

* `name` - (Required) Connector name, unique in the tenant. Changing this forces a new connector to be created.
* `class_name` - (Required) Connector class name. Connectors that implement the open connector standard use `sailpoint.connector.OpenConnectorAdapter`. Changing this forces a new connector to be created.
* `type` - (Optional) Connector type. Defaults to `custom <name>`. Changing this forces a new connector to be created.
* `direct_connect` - (Optional) Whether sources of the connector are direct connect sources. Defaults to `true`. Changing this forces a new connector to be created.
* `status` - (Optional) Connector status, `DEVELOPMENT`, `DEMO` or `RELEASED`. Changing this forces a new connector to be created.
* `connector_metadata_json` - (Optional) UI metadata of the connector as a JSON object. Use `jsonencode()` for convenience. The value is compared semantically. When not set, the value is not managed.
* `application_xml` - (Optional) Application XML of the connector. When not set, the value is not managed.
* `correlation_config_xml` - (Optional) Correlation config XML of the connector. When not set, the value is not managed.
* `source_config_xml` - (Optional) Source config XML of the connector. When not set, the value is not managed.

Differences only in line endings or trailing whitespace of the XML values are ignored.

## Attributes Reference

* `id` - Connector ID, the same as `script_name`.
* `script_name` - Unique script name of the connector.
* `file_upload` - Whether sources of the connector are file upload sources.

## Import

Connectors can be imported using their script name:

```shell
terraform import identitynow_connector.example <script-name>
```

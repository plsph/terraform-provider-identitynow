---
subcategory: "Connector"
page_title: "IdentityNow: identitynow_connector_customizer"
description: |-
  Manages an IdentityNow connector customizer.
---

# identitynow_connector_customizer

Manages a connector customizer. The name is the only writable field; changes are sent with an update request. The API documentation is contradictory: the update request only contains the name, but the update description lists the name as immutable. The provider therefore reads the customizer back after an update and fails the apply, keeping the prior state, when IdentityNow did not apply the new name. In that case revert the name or replace the customizer (e.g. `terraform apply -replace`). Customizer versions (code uploads) are not managed by this resource, use the SailPoint CLI to deploy them. The image version and image ID are refreshed from IdentityNow.

## Example Usage

```terraform
resource "identitynow_connector_customizer" "custom" {
  name = "My Connector Customizer"
}
```

## Arguments Reference

* `name` - (Required) Connector customizer name. A rename is verified after the update, see above.

## Attributes Reference

* `id` - Connector customizer ID.
* `image_version` - Current image version of the customizer. Null until a version is created.
* `image_id` - Current image ID of the customizer. Null until a version is created.
* `tenant_id` - Tenant ID of the customizer.
* `created` - Creation date.

## Import

Connector customizers can be imported using their ID:

```shell
terraform import identitynow_connector_customizer.example <customizer-id>
```

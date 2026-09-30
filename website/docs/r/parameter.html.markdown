---
subcategory: "Parameter Storage"
layout: "identitynow"
page_title: "IdentityNow: identitynow_parameter"
description: |-
  Manages an IdentityNow parameter storage parameter.
---

# identitynow_parameter

Manages a parameter in parameter storage, for example a credential that connectors and workflows reference instead of storing the secret themselves.

~> **Note:** The private fields are secrets and must be sent as a JWE (AES256) encrypted blob, encrypted with the public key of the attestation document returned by `GET /parameter-storage/attestation`. The provider does not encrypt the value. The API never returns the private fields, so the planned value is kept in the Terraform state as a sensitive value and a changed value cannot be read back. Changes made outside Terraform are only detected through `private_fields_last_modified_at`: when it changes, the next plan sets the configured `private_fields` again. The timestamps are compared as points in time, so a different notation of the same timestamp is not treated as a change, and the provider reads the parameter back after each create and update to store the timestamp as the API reports it.

## Example Usage

```hcl
variable "db_password_jwe" {
  description = "Private fields of the database credential, JWE encrypted."
  type        = string
  sensitive   = true
}

resource "identitynow_parameter" "db_credential" {
  name        = "Database service account"
  description = "Credential of the HR database connector"
  type        = "password"
  owner_id    = "2c9180835d2e5168015d32f890ca1581"
  public_fields_json = jsonencode({
    username = "svc-hr"
  })
  private_fields = var.db_password_jwe
}
```

## Arguments Reference

* `name` - (Required) Human-readable name of the parameter.
* `type` - (Required) Parameter type, see `GET /parameter-storage/specification` for the types and their fields. Cannot be changed, changing it forces a new parameter to be created.
* `owner_id` - (Required) Identity ID of the parameter owner.
* `description` - (Optional) Description of the parameter.
* `public_fields_json` - (Optional) Public fields of the parameter as a JSON object, as defined by the type specification. The value is compared semantically.
* `private_fields` - (Optional, Sensitive) Private fields as a JWE encrypted blob containing a JSON object, as defined by the type specification. Write-only in the API. Removing the argument leaves the stored private fields unchanged.

All arguments except `type` are updated in place.

## Attributes Reference

* `id` - Parameter ID.
* `primary_field` - Name of the primary field in the public fields.
* `last_modified_at` - Date any field of the parameter was last changed.
* `private_fields_last_modified_at` - Date the private fields were last changed.

## Import

Parameters can be imported using their ID. The private fields cannot be imported, the next apply sets the configured value:

```shell
terraform import identitynow_parameter.example <parameter-id>
```

Parameters that are still referenced, for example by a source, cannot be deleted.

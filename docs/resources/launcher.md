---
subcategory: "Launcher"
page_title: "IdentityNow: identitynow_launcher"
description: |-
  Manages an IdentityNow launcher.
---

# identitynow_launcher

Manages a launcher, which lets users start an interactive process such as a workflow from the IdentityNow UI.

## Example Usage

```hcl
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
```

## Arguments Reference

* `name` - (Required) Name of the launcher, at most 255 characters.
* `description` - (Required) Description of the launcher, at most 2000 characters.
* `config_json` - (Required) Launcher configuration as a JSON object of at most 4 KB. Use `jsonencode()` for convenience. The value is compared semantically.
* `type` - (Optional) Launcher type. Defaults to `INTERACTIVE_PROCESS`, the only supported type.
* `disabled` - (Optional) Whether the launcher is disabled. Defaults to `false`.
* `reference` - (Optional) Object the launcher starts. At most one block. Contains:
  * `id` - (Required) ID of the referenced object, e.g. a workflow ID.
  * `type` - (Optional) Type of the referenced object. Defaults to `WORKFLOW`, the only supported type.

All arguments are updated in place; the launcher is replaced as a whole with the configured values.

## Attributes Reference

* `id` - Launcher ID.
* `owner` - Owner of the launcher, the identity that created it. Contains `id` and `type`.
* `created` - Creation date.
* `modified` - Last modification date.

## Import

Launchers can be imported using their ID:

```shell
terraform import identitynow_launcher.example <launcher-id>
```

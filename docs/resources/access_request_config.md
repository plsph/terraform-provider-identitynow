---
subcategory: "Tenant Settings"
page_title: "IdentityNow: identitynow_access_request_config"
description: |-
  Manages the tenant-wide access request configuration.
---

# identitynow_access_request_config

Manages the tenant-wide access request configuration: external approvals, requests on behalf of others and the default entitlement request configuration.

There is one access request configuration per tenant, it cannot be created or deleted. Creating the resource applies the configured settings to the existing access request configuration, and destroying it only removes it from Terraform state: the settings are left unchanged in IdentityNow.

Only the settings present in the configuration are managed. Settings that are not configured are never changed or reset; they show the current tenant value. Removing a setting from the configuration stops managing it and leaves its current value in place. The API replaces the whole access request configuration on update, so the provider reads the current access request configuration, replaces the configured settings and sends it back.

`entitlement_request_config_json` is merged key by key into the current entitlement request configuration: keys that are not in the JSON keep their value, and only the configured keys are compared on refresh. Arrays such as `approvalSchemes` are replaced as a whole. On refresh, arrays are compared element by element with the same rules, so keys that the API adds to the elements (e.g. `approverId: null` in approval schemes) and configured `false`, `0` or empty values that the API omits do not produce a diff.

## Example Usage

```terraform
resource "identitynow_access_request_config" "this" {
  request_on_behalf_of_employee_by_manager = true
  request_on_behalf_of_anyone_by_anyone    = false

  entitlement_request_config_json = jsonencode({
    accessRequestConfig = {
      requestCommentRequired = true
      denialCommentRequired  = true
      approvalSchemes = [
        { approverType = "MANAGER", approverId = null }
      ]
    }
  })
}
```

## Arguments Reference

All arguments are optional; settings that are not configured keep their current value.

* `approvals_must_be_external` - (Optional) Whether approvals must be processed by an external system. This blocks Request Center access requests for users who are not org admins.
* `reauthorization_enabled` - (Optional) Whether reauthorization is enforced for appropriately configured access items.
* `gov_group_visibility_enabled` - (Optional) Whether requesters and requested-for users can see the names of governance group members when a request awaits the group's approval.
* `request_on_behalf_of_anyone_by_anyone` - (Optional) Whether anyone can request access for anyone.
* `request_on_behalf_of_employee_by_manager` - (Optional) Whether managers can request access for their direct reports.
* `entitlement_request_config_json` - (Optional) Default entitlement request configuration with `accessRequestConfig` and `revocationRequestConfig` objects (`approvalSchemes`, `requestCommentRequired`, `denialCommentRequired`, `reauthorizationRequired`, `requireEndDate`, `maxPermittedAccessDuration`). A JSON object, e.g. built with `jsonencode`.

## Attributes Reference

In addition to the arguments, the following attributes are exported; arguments that are not configured show the current tenant value.

* `id` - Always `access-request-config`.

## Import

The settings can be imported with any ID; the ID is always set to `access-request-config`:

```shell
terraform import identitynow_access_request_config.this access-request-config
```

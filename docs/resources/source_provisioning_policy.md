---
subcategory: "Source"
page_title: "IdentityNow: identitynow_source_provisioning_policy"
description: |-
  Manages a provisioning policy of an IdentityNow source.
---

# identitynow_source_provisioning_policy

Manages the provisioning policy of a source for one usage type, e.g. the policy that defines the attributes of new accounts (`CREATE`). Each usage type of a source has at most one policy. Fields can use transforms, see [Transforms in Provisioning Policies](https://developer.sailpoint.com/docs/extensibility/transforms/guides/transforms-in-provisioning-policies).

Updates replace the whole policy, so the policy is fully managed by Terraform. If the source already has a policy for the usage type, import it instead of creating it.

## Example Usage

```terraform
resource "identitynow_source_provisioning_policy" "create" {
  source_id   = "2c9180835d191a86015d28455b4a2329"
  usage_type  = "CREATE"
  name        = "Account"
  description = "Attributes of new accounts"
  fields_json = jsonencode([
    {
      name = "userName"
      type = "string"
      transform = {
        type       = "identityAttribute"
        attributes = { name = "uid" }
      }
    },
    {
      name          = "groups"
      type          = "string"
      isMultiValued = true
    }
  ])
}
```

## Arguments Reference

* `source_id` - (Required) ID of the source. Changing this forces a new provisioning policy to be created.
* `usage_type` - (Required) Provisioning operation the policy applies to: `CREATE`, `UPDATE`, `ENABLE`, `DISABLE`, `DELETE`, `ASSIGN`, `UNASSIGN`, `CREATE_GROUP`, `UPDATE_GROUP`, `DELETE_GROUP`, `REGISTER`, `CREATE_IDENTITY`, `UPDATE_IDENTITY`, `EDIT_GROUP`, `UNLOCK` or `CHANGE_PASSWORD`. Changing this forces a new provisioning policy to be created.
* `name` - (Required) Provisioning policy name.
* `description` - (Optional) Provisioning policy description.
* `fields_json` - (Optional) Policy fields as a JSON array. Each field has a `name`, a `type` (`string`, `int`, `long`, `date`, `boolean` or `secret`) and optionally a `transform`, `attributes` and `isMultiValued`. Use `jsonencode()` for convenience. The value is compared semantically, so formatting and key order do not produce a diff, and keys that hold the API defaults (`transform` and `attributes` as empty objects, `isRequired` and `isMultiValued` set to false, null values) are ignored. When not set, the policy has no fields.

## Attributes Reference

* `id` - Resource ID in the format `<source_id>/<usage_type>`.

## Import

Provisioning policies can be imported using the source ID and the usage type:

```shell
terraform import identitynow_source_provisioning_policy.example <source-id>/<usage-type>
```

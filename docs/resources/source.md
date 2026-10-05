---
subcategory: "Source"
page_title: "IdentityNow: identitynow_source"
description: |-
  Manages an IdentityNow Source.
---

# identitynow_source

Manages an IdentityNow Source.

The resource manages the basic settings of the source: name, description, connector, owner, cluster, delete threshold and whether the source is authoritative. Connector attributes, account schemas and other source configuration are not managed by this resource. Updates are sent as a JSON Patch of the managed fields only, so connector attributes and schemas configured elsewhere (in the IdentityNow UI, the API, or with `identitynow_account_schema`) are kept.

## Example Usage

### Source with a Virtual Appliance Cluster

```hcl
resource "identitynow_source" "active_directory" {
  name             = "Active Directory"
  description      = "The Active Directory connector created by terraform"
  connector        = "active-directory"
  authoritative    = false
  delete_threshold = 10

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }

  cluster {
    id   = "2c9180887671ff8c01767b4671fc7d60"
    name = "Primary Cluster"
    type = "CLUSTER"
  }
}
```

### Direct Connect Source

```hcl
data "identitynow_identity" "owner" {
  alias = "john.doe"
}

resource "identitynow_source" "azure_ad" {
  name             = "Azure Active Directory"
  description      = "The Azure Active Directory connector created by terraform"
  connector        = "azure-active-directory"
  authoritative    = false
  delete_threshold = 10

  owner {
    id   = data.identitynow_identity.owner.id
    name = data.identitynow_identity.owner.name
    type = "IDENTITY"
  }
}
```

## Arguments Reference

The following arguments are supported:

As per developer guide: (https://developer.sailpoint.com/docs/api/v3/create-source)

* `name` - (Required) Source name. Can be updated in place.

* `description` - (Required) Source description.

* `connector` - (Required) Connector script name, e.g. `active-directory`, `azure-active-directory` or `aws`. Changing this forces a new resource to be created.

* `authoritative` - (Required) Whether the source is authoritative, i.e. a source of identities. Changing this forces a new resource to be created.

* `delete_threshold` - (Required) Maximum percentage of accounts that can be deleted during an aggregation.

* `owner` - (Required) Owner of the source. Exactly one block is required. Contains:
  * `id` - (Required) Owner identity ID.
  * `name` - (Required) Owner name.
  * `type` - (Required) Owner type, `IDENTITY`.

* `cluster` - (Optional) Virtual appliance cluster used by the source. At most one block is allowed. Contains:
  * `id` - (Required) Cluster ID.
  * `name` - (Required) Cluster name.
  * `type` - (Required) Cluster type, `CLUSTER`.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `id` - Source ID.

## Import

Sources can be imported using their ID:

```shell
terraform import identitynow_source.example <source-id>
```

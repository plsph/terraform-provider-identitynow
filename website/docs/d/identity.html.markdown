---
subcategory: "Identity"
layout: "identitynow"
page_title: "IdentityNow: Data Identity: identitynow_identity"
description: |-
  Gets information about an existing Identity.
---

# Data Source: identitynow_identity

Use this data source to access information about an existing Identity.

## Example Usage

```hcl
data "identitynow_identity" "example" {
  alias = "example"
}

data "identitynow_identity" "example" {
  email_address = "example"
}

output "identitynow_source_description" {
  value = data.identitynow_identity.example.description
}
```

## Arguments Reference

The following arguments are supported:

* `alias` - The identity's alternate unique identifier is equivalent to its Account Name on the authoritative source account schema.

* `email_address` - The email address of the identity.

## Attributes Reference

In addition to the Arguments listed above - the following Attributes are exported:

* `name` - The identity's name is equivalent to its Display Name attribute.

* `is_manager` - Whether this identity is a manager of another identity.

* `identity_status` - The identity's status in the system.

* `attributes` - A map with the identity attributes for the identity.


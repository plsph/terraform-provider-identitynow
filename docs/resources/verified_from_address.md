---
subcategory: "Notification"
page_title: "IdentityNow: identitynow_verified_from_address"
description: |-
  Manages an IdentityNow verified sender email address.
---

# identitynow_verified_from_address

Manages a sender ("From:") email address for notifications. Creating the resource starts the verification: AWS SES sends a verification email to the address, and the address can only be used after it is verified. The `verification_status` attribute is refreshed on every read.

## Example Usage

```hcl
resource "identitynow_verified_from_address" "no_reply" {
  email = "no-reply@example.com"
}
```

## Arguments Reference

* `email` - (Required) Sender email address. The address cannot be changed, changing this forces a new address to be created.

## Attributes Reference

* `id` - Sender address ID.
* `is_verified_by_domain` - Whether the address is verified by its domain.
* `verification_status` - Verification status: `PENDING`, `SUCCESS`, `FAILED` or `NA`.
* `region` - AWS SES region the address is associated with.

## Import

Verified from addresses can be imported using their ID:

```shell
terraform import identitynow_verified_from_address.example <address-id>
```

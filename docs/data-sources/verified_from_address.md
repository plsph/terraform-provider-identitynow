---
subcategory: "Notification"
page_title: "IdentityNow: Data Source: identitynow_verified_from_address"
description: |-
  Gets information about an IdentityNow verified sender email address.
---

# Data Source: identitynow_verified_from_address

Use this data source to look up a sender ("From:") email address and its verification status.

## Example Usage

```hcl
data "identitynow_verified_from_address" "no_reply" {
  email = "no-reply@example.com"
}
```

## Arguments Reference

* `email` - (Required) Sender email address, compared case-insensitively.

## Attributes Reference

* `id` - Sender address ID.
* `is_verified_by_domain` - Whether the address is verified by its domain.
* `verification_status` - Verification status: `PENDING`, `SUCCESS`, `FAILED` or `NA`.
* `region` - AWS SES region the address is associated with.

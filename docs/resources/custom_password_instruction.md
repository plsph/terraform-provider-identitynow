---
subcategory: "Password Policy"
page_title: "IdentityNow: identitynow_custom_password_instruction"
description: |-
  Manages IdentityNow custom password instructions.
---

# identitynow_custom_password_instruction

Manages custom instructions shown on a page of the password reset, change password, unlock account and forgotten username flows.

~> **Note:** This resource uses an experimental API that may change without notice.

The API cannot update instructions, so changing any argument deletes the instructions and creates new ones.

IdentityNow sanitizes the content. The configured content is kept in state instead of the sanitized content returned by the API, so sanitizing does not cause a replacement on every apply. As a consequence, changes of the content made outside Terraform are not detected; after an import the content returned by the API is used.

There is one set of instructions per page and locale, identified by the page ID and locale. Creating the resource takes over existing instructions of the same page and locale, and destroying it deletes them. Do not use `create_before_destroy` with this resource: the replacement would be created with the same page and locale and then deleted together with the old instructions.

## Example Usage

```terraform
resource "identitynow_custom_password_instruction" "reset_password" {
  page_id      = "reset-password:enter-password"
  page_content = "See the company password policy <a href=\"https://intranet.example.com/passwords\" target=\"_blank\">here</a>."
}
```

## Arguments Reference

* `page_id` - (Required) Page the instructions are shown on. One of `change-password:enter-password`, `change-password:finish`, `flow-selection:select`, `forget-username:user-email`, `mfa:enter-code`, `mfa:enter-kba`, `mfa:select`, `reset-password:enter-password`, `reset-password:enter-username`, `reset-password:finish`, `unlock-account:enter-username` or `unlock-account:finish`. Changing it forces new instructions to be created.
* `page_content` - (Required) Instructions in basic HTML, at most 1000 characters. Links open in the current page unless they use `target="_blank"`. Changing it forces new instructions to be created. Changes made outside Terraform are not detected.
* `locale` - (Optional) BCP 47 language tag of the instructions. Defaults to `default`. Changing it forces new instructions to be created.

## Attributes Reference

* `id` - Identifier in the form `<page_id>/<locale>`.

## Import

Custom password instructions can be imported using the page ID for the default locale, or the page ID and locale separated by `/`:

```shell
terraform import identitynow_custom_password_instruction.example reset-password:enter-password
terraform import identitynow_custom_password_instruction.example reset-password:enter-password/de-DE
```

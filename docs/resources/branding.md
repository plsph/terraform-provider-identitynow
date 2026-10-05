---
subcategory: "Branding"
page_title: "IdentityNow: identitynow_branding"
description: |-
  Manages an IdentityNow branding item.
---

# identitynow_branding

Manages a branding item: the product name, colors, email sender address and login message shown to users.

~> **Note:** The logo is not managed by this resource, upload it in the IdentityNow UI. Its URL is available in `standard_logo_url`. The API replaces the whole branding item on update (PUT) and the provider never sends the logo file (`fileStandard`). The API documentation does not say whether a missing file keeps the current logo; if IdentityNow treats it as a removal, every update of this resource clears the uploaded logo, and you have to upload it again afterwards.

The default branding item of the tenant is named `default` and already exists; manage additional branding items with this resource, or read the default one with the [identitynow_branding](../data-sources/branding) data source.

## Example Usage

```terraform
resource "identitynow_branding" "corporate" {
  name                        = "corporate"
  product_name                = "Corporate Identity Portal"
  action_button_color         = "0074D9"
  active_link_color           = "011E69"
  navigation_color            = "011E69"
  email_from_address          = "no-reply@example.com"
  login_informational_message = "Use your corporate account to sign in."
}
```

## Arguments Reference

* `name` - (Required) Name of the branding item. Changing it forces a new branding item to be created.
* `product_name` - (Required) Product name shown in the UI and in emails.
* `action_button_color` - (Optional) Hex color of action buttons, e.g. `0074D9`.
* `active_link_color` - (Optional) Hex color of links.
* `navigation_color` - (Optional) Hex color of the navigation bar.
* `email_from_address` - (Optional) Sender address of emails.
* `login_informational_message` - (Optional) Informational message shown on the login page.

Optional arguments that are not set are not sent to the API and are stored as null. Removing an optional argument from the configuration sends an empty value, which clears it. All arguments except `name` are updated in place.

## Attributes Reference

* `id` - Name of the branding item.
* `standard_logo_url` - URL of the standard logo. The logo is not managed by this resource, see the note above.

## Import

Branding items can be imported using their name:

```shell
terraform import identitynow_branding.example corporate
```

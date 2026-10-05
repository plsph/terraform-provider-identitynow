---
subcategory: "Branding"
page_title: "IdentityNow: Data Source: identitynow_branding"
description: |-
  Gets information about an existing IdentityNow branding item.
---

# Data Source: identitynow_branding

Use this data source to look up a branding item by name.

## Example Usage

```hcl
data "identitynow_branding" "default" {
  name = "default"
}
```

## Arguments Reference

* `name` - (Required) Name of the branding item, e.g. `default`.

## Attributes Reference

* `id` - Name of the branding item.
* `product_name` - Product name.
* `action_button_color` - Hex color of action buttons.
* `active_link_color` - Hex color of links.
* `navigation_color` - Hex color of the navigation bar.
* `email_from_address` - Sender address of emails.
* `login_informational_message` - Informational message shown on the login page.
* `standard_logo_url` - URL of the standard logo.

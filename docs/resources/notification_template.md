---
subcategory: "Notification"
page_title: "IdentityNow: identitynow_notification_template"
description: |-
  Manages an IdentityNow custom notification template.
---

# identitynow_notification_template

Manages a custom notification template. IdentityNow does not allow new template keys: the resource customizes an existing template for a `key`, `medium` and `locale`. Destroying the resource deletes the custom template, so the default template is used again.

Creating the resource fails when a custom template already exists for the key, medium and locale, because the API would silently overwrite it and destroying the resource would delete it. The error shows the ID of the existing template; import it with `terraform import` to manage it.

A new custom template starts as a copy of the default template (`GET /notification-template-defaults`): `name`, `subject`, `body`, `from`, `reply_to` and `description` that are not set take the value of the default template, so the new template is not empty.

## Example Usage

```terraform
resource "identitynow_notification_template" "work_item_summary" {
  key     = "cloud_manual_work_item_summary"
  medium  = "EMAIL"
  locale  = "en"
  subject = "You have $numItems pending work items"
  body    = "<p>Hello $recipient.name,</p><p>please review your pending work items.</p>"
  from    = "no-reply@example.com"
}

resource "identitynow_notification_template" "work_item_summary_slack" {
  key    = "cloud_manual_work_item_summary"
  medium = "SLACK"
  locale = "en"
  slack_template_json = jsonencode({
    text = "You have $numItems pending work items"
  })
}
```

## Arguments Reference

* `key` - (Required) Key of the template to customize, e.g. `cloud_manual_work_item_summary`. Changing this forces a new template to be created.
* `medium` - (Required) Message medium: `EMAIL`, `SLACK` or `TEAMS`. Changing this forces a new template to be created.
* `locale` - (Required) Locale of the message text as a BCP 47 language tag, e.g. `en`. Changing this forces a new template to be created.
* `name` - (Optional) Template name.
* `subject` - (Optional) Subject line. Supports Velocity template variables.
* `body` - (Optional) Message body. Supports Velocity template variables.
* `from` - (Optional) "From:" address, e.g. an [identitynow_verified_from_address](verified_from_address).
* `reply_to` - (Optional) "Reply To" address.
* `description` - (Optional) Template description.
* `slack_template_json` - (Optional) Slack template as a JSON object, e.g. with `text`, `blocks` and `attachments`.
* `teams_template_json` - (Optional) Microsoft Teams template as a JSON object, e.g. with `title`, `text` and `messageJSON`.

When `name`, `subject`, `body`, `from`, `reply_to` or `description` are not set, a new template uses the value of the default template, and later the value stored in IdentityNow is kept and shown in the state. `slack_template_json` and `teams_template_json` are not copied from the default template; set them for `SLACK` and `TEAMS` templates.

The JSON attributes are compared semantically. Null, `false`, empty string and empty values are ignored in the comparison, so defaults filled in by the API do not produce a diff.

Every change sends the full template again (`POST /notification-templates` with the same key, medium and locale).

## Attributes Reference

* `id` - Template ID.
* `header` - Deprecated header, read only. The header is now part of `body` and the API rejects non-null values, so it is never sent.
* `footer` - Deprecated footer, read only. The footer is now part of `body` and the API rejects non-null values, so it is never sent.
* `created` - Creation date.
* `modified` - Last modification date.

## Import

Custom notification templates can be imported using their ID:

```shell
terraform import identitynow_notification_template.example <template-id>
```

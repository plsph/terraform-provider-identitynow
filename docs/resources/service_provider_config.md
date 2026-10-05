---
subcategory: "Tenant Settings"
page_title: "IdentityNow: identitynow_service_provider_config"
description: |-
  Manages the IdentityNow SAML service provider configuration.
---

# identitynow_service_provider_config

Manages the tenant-wide SAML single sign-on configuration, in which IdentityNow is the service provider (SP) of an external identity provider (IdP).

There is one service provider configuration per tenant, it cannot be created or deleted. Creating the resource applies the configured settings to the existing service provider configuration, and destroying it only removes it from Terraform state: the settings are left unchanged in IdentityNow.

Only the settings present in the configuration are managed. Settings that are not configured are never changed or reset; they show the current tenant value. Removing a setting from the configuration stops managing it and leaves its current value in place. Updates are sent as JSON Patch operations for the configured settings that changed.

The identity provider settings (`idp_*`) belong to the IdP element of the SAML configuration. When any of them changes, the provider applies the configured settings to the current IdP element and replaces the element as a whole, so the API validates a complete element. The service provider settings (`sp_*`) cannot be changed and are read only. `idp_cert` is marked sensitive to keep the certificate out of the plan output. Enabling just-in-time provisioning (`idp_jit_enabled`) requires `idp_jit_source_id` and `idp_jit_source_attribute_mappings` with the `firstName`, `lastName` and `email` keys.

## Example Usage

```hcl
resource "identitynow_service_provider_config" "this" {
  enabled                = true
  bypass_idp             = false
  idp_entity_id          = "http://www.okta.com/exk0000000000000000"
  idp_binding            = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
  idp_name_id            = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
  idp_mapping_attribute  = "email"
  idp_login_url_post     = "https://example.okta.com/app/example/sso/saml"
  idp_login_url_redirect = "https://example.okta.com/app/example/sso/saml"
  idp_cert               = "<BASE64_ENCODED_IDP_CERTIFICATE>"
}
```

## Arguments Reference

All arguments are optional; settings that are not configured keep their current value.

* `enabled` - (Optional) Whether SAML authentication is enabled.
* `bypass_idp` - (Optional) Whether basic login with the `prompt=true` parameter is allowed, e.g. while debugging the SAML setup. When disabled, only org admins with MFA can bypass the identity provider.
* `idp_entity_id` - (Optional) Entity ID of the identity provider.
* `idp_binding` - (Optional) SAML binding of the identity provider, e.g. `urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST`.
* `idp_authn_context` - (Optional) SAML authentication context, e.g. `urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport`.
* `idp_include_authn_context` - (Optional) Whether the configured authentication context is used instead of the default.
* `idp_name_id` - (Optional) Name ID format, e.g. `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`.
* `idp_mapping_attribute` - (Optional) Identity attribute that is matched with the SAML name ID, e.g. `email`.
* `idp_login_url_post` - (Optional) Identity provider login URL for the HTTP-POST binding.
* `idp_login_url_redirect` - (Optional) Identity provider login URL for the HTTP-Redirect binding.
* `idp_logout_url` - (Optional) Identity provider logout URL.
* `idp_cert` - (Optional) Base64-encoded signing certificate of the identity provider. The value is sensitive.
* `idp_jit_enabled` - (Optional) Whether just-in-time provisioning of identities is enabled. It requires `idp_jit_source_id` and `idp_jit_source_attribute_mappings` with `firstName`, `lastName` and `email`.
* `idp_jit_source_id` - (Optional) ID of the source that just-in-time provisioned accounts are created on.
* `idp_jit_source_attribute_mappings` - (Optional) Map of identity profile attribute names to SAML assertion attribute names for just-in-time provisioning.

## Attributes Reference

In addition to the arguments, the following attributes are exported; arguments that are not configured show the current tenant value.

* `id` - Always `service-provider-config`.
* `saml_configuration_valid` - Whether the SAML configuration is valid.
* `idp_certificate_name` - Subject name of the identity provider certificate.
* `idp_certificate_expiration_date` - Expiration date of the identity provider certificate.
* `sp_entity_id` - Entity ID of IdentityNow as the service provider.
* `sp_alias` - Alias of the service provider.
* `sp_callback_url` - Assertion consumer (callback) URL of the service provider, to be configured in the identity provider.
* `sp_legacy_acs_url` - Legacy assertion consumer service URL of the service provider.

## Import

The settings can be imported with any ID; the ID is always set to `service-provider-config`:

```shell
terraform import identitynow_service_provider_config.this service-provider-config
```

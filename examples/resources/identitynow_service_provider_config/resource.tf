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

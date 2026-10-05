resource "identitynow_branding" "corporate" {
  name                        = "corporate"
  product_name                = "Corporate Identity Portal"
  action_button_color         = "0074D9"
  active_link_color           = "011E69"
  navigation_color            = "011E69"
  email_from_address          = "no-reply@example.com"
  login_informational_message = "Use your corporate account to sign in."
}

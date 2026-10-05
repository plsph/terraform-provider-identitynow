resource "identitynow_custom_password_instruction" "reset_password" {
  page_id      = "reset-password:enter-password"
  page_content = "See the company password policy <a href=\"https://intranet.example.com/passwords\" target=\"_blank\">here</a>."
}

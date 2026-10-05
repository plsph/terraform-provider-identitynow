variable "duo_access_key" {
  description = "Duo access key."
  type        = string
  sensitive   = true
}

variable "duo_secret_key" {
  description = "Duo secret key (skey)."
  type        = string
  sensitive   = true
}

resource "identitynow_mfa_duo_config" "this" {
  enabled            = true
  host               = "api-00000000.duosecurity.com"
  identity_attribute = "email"
  access_key         = var.duo_access_key
  config_properties_json = jsonencode({
    ikey = "DIXXXXXXXXXXXXXXXXXX"
    skey = var.duo_secret_key
  })
}

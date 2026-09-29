#Follow https://community.sailpoint.com/t5/Admin-Help/REST-API-to-request-IdentityNow-API-Client-ID-and-Client-key/ta-p/74408
# to learn how to request IdentityNow API Client ID and Client key with REST API
variable "api_client_id" {
  description = "Client ID of the IdentityNow API client used by the provider."
  type        = string
}

variable "api_client_secret" {
  description = "Client secret of the IdentityNow API client used by the provider."
  type        = string
  sensitive   = true
}

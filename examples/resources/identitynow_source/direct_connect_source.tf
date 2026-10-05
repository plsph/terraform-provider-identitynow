data "identitynow_identity" "owner" {
  alias = "john.doe"
}

resource "identitynow_source" "azure_ad" {
  name             = "Azure Active Directory"
  description      = "The Azure Active Directory connector created by terraform"
  connector        = "azure-active-directory"
  authoritative    = false
  delete_threshold = 10

  owner {
    id   = data.identitynow_identity.owner.id
    name = data.identitynow_identity.owner.name
    type = "IDENTITY"
  }
}

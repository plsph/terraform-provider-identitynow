resource "identitynow_access_profile" "aad_access_profile_operators" {
  name        = "Azure AD Operators"
  description = "Operator access in Azure Active Directory"
  requestable = true
  enabled     = true

  owner {
    id   = data.identitynow_identity.john_doe.id
    name = data.identitynow_identity.john_doe.name
    type = "IDENTITY"
  }

  source {
    id   = identitynow_source.azure_ad_source.id
    name = identitynow_source.azure_ad_source.name
    type = "SOURCE"
  }

  entitlements {
    id   = data.identitynow_source_entitlement.aad_operator.entitlements[0].id
    name = data.identitynow_source_entitlement.aad_operator.entitlements[0].name
    type = "ENTITLEMENT"
  }

  access_request_config {
    comments_required        = true
    denial_comments_required = true

    approval_schemes {
      approver_type = "MANAGER"
    }

    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = identitynow_governance_group.approvers.id
    }

    max_permitted_access_duration {
      value     = 6
      time_unit = "MONTHS"
    }
  }

  revocation_request_config {
    approval_schemes {
      approver_type = "GOVERNANCE_GROUP"
      approver_id   = identitynow_governance_group.approvers.id
    }
  }
}

resource "identitynow_access_profile" "ad_access_profile_developers" {
  name        = "AD Developers"
  description = "Developer access in Active Directory"
  requestable = true
  enabled     = true

  owner {
    id   = data.identitynow_identity.john_doe.id
    name = data.identitynow_identity.john_doe.name
    type = "IDENTITY"
  }

  source {
    id   = identitynow_source.active_directory_source.id
    name = identitynow_source.active_directory_source.name
    type = "SOURCE"
  }

  entitlements {
    id   = data.identitynow_source_entitlement.ad_developer.entitlements[0].id
    name = data.identitynow_source_entitlement.ad_developer.entitlements[0].name
    type = "ENTITLEMENT"
  }
}

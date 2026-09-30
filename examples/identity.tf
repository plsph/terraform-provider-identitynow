resource "identitynow_identity_profile" "employees" {
  name        = "Employees"
  description = "Identities from Active Directory"
  priority    = 10

  owner {
    id = data.identitynow_identity.john_doe.id
  }

  authoritative_source {
    id = identitynow_source.active_directory_source.id
  }

  identity_attribute_config_enabled = true
  attribute_config_json = jsonencode([
    {
      identityAttributeName = "email"
      transformDefinition = {
        type = "accountAttribute"
        attributes = {
          sourceName    = identitynow_source.active_directory_source.name
          attributeName = "mail"
        }
      }
    },
    {
      identityAttributeName = "costCenter"
      transformDefinition = {
        type = "accountAttribute"
        attributes = {
          sourceName    = identitynow_source.active_directory_source.name
          attributeName = "department"
        }
      }
    }
  ])

  depends_on = [identitynow_identity_attribute.cost_center]
}

resource "identitynow_lifecycle_state" "employees_inactive" {
  identity_profile_id = identitynow_identity_profile.employees.id
  name                = "Inactive"
  technical_name      = "inactive"
  description         = "Identities that left the organization"
  enabled             = true
  identity_state      = "INACTIVE_LONG_TERM"

  account_actions_json = jsonencode([
    {
      action     = "DISABLE"
      allSources = true
    }
  ])
  remove_all_access_enabled = true

  email_notification_option {
    notify_managers       = true
    notify_specific_users = true
    email_address_list    = ["it-security@example.com"]
  }
}

resource "identitynow_identity_attribute" "cost_center" {
  name         = "costCenter"
  display_name = "Cost Center"
  type         = "string"
  searchable   = true
}

# The API cannot delete metadata attributes: destroying only removes it from the state.
resource "identitynow_access_model_metadata_attribute" "data_sensitivity" {
  name         = "Data Sensitivity"
  description  = "Sensitivity of the data the access grants"
  type         = "governance"
  object_types = ["all"]

  values {
    value = "public"
    name  = "Public"
  }

  values {
    value = "confidential"
    name  = "Confidential"
  }
}

# Uses an experimental API.
resource "identitynow_search_attribute_config" "employee_number" {
  name         = "employeeNumber"
  display_name = "Employee Number"
  application_attributes = {
    (identitynow_source.active_directory_source.id) = "employeeID"
  }
}

data "identitynow_identity_profile" "employees" {
  name = identitynow_identity_profile.employees.name
}

data "identitynow_lifecycle_state" "employees_inactive" {
  identity_profile_id = identitynow_identity_profile.employees.id
  id                  = identitynow_lifecycle_state.employees_inactive.id
}

data "identitynow_identity_attribute" "department" {
  name = "department"
}

data "identitynow_access_model_metadata_attribute" "data_sensitivity" {
  key = identitynow_access_model_metadata_attribute.data_sensitivity.key
}

data "identitynow_search_attribute_config" "employee_number" {
  name = identitynow_search_attribute_config.employee_number.name
}

# Uses an experimental API.
data "identitynow_auth_profile" "default" {
  name = "Default"
}

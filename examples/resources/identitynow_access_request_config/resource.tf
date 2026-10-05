resource "identitynow_access_request_config" "this" {
  request_on_behalf_of_employee_by_manager = true
  request_on_behalf_of_anyone_by_anyone    = false

  entitlement_request_config_json = jsonencode({
    accessRequestConfig = {
      requestCommentRequired = true
      denialCommentRequired  = true
      approvalSchemes = [
        { approverType = "MANAGER", approverId = null }
      ]
    }
  })
}

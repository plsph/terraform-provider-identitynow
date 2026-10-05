The files in this directory form a single Terraform configuration. Replace the `<...>` placeholders
(cluster, entitlement names, account schema ID, ...) and the identity alias in `data.tf` with values from your tenant.

Example command to run plan and apply for resources:

`terraform plan -var="api_client_id=<CLIENT_ID>" -var="api_client_secret=<CLIENT_SECRET>"`

`terraform apply -var="api_client_id=<CLIENT_ID>" -var="api_client_secret=<CLIENT_SECRET>"`

The provider can also read the API URL and credentials from the `IDENTITYNOW_URL`, `IDENTITYNOW_CLIENT_ID` and
`IDENTITYNOW_CLIENT_SECRET` environment variables.

To list resources:

```
$ terraform state list
data.identitynow_form_definition.existing
data.identitynow_identity.john_doe
data.identitynow_role.operator_developer_role
data.identitynow_segment.existing
data.identitynow_source_entitlement.aad_operator
data.identitynow_source_entitlement.ad_developer
data.identitynow_workflow.existing
identitynow_access_profile.aad_access_profile_operators
identitynow_access_profile.ad_access_profile_developers
identitynow_access_profile_attachment.azure_ad_app
identitynow_account_schema.active_directory_account
identitynow_form_definition.access_request
identitynow_governance_group.approvers
identitynow_governance_group_members.approvers
identitynow_password_policy.password_policy
identitynow_role.operator_developer_role
identitynow_schedule_account_aggregation.azure_ad_aggregation
identitynow_segment.austin
identitynow_source.active_directory_source
identitynow_source.aws_iam_source
identitynow_source.azure_ad_source
identitynow_source_app.azure_ad_app
identitynow_tagged_object.access_profile_tags
identitynow_tagged_object.role_tags
identitynow_tagged_object.source_tags
identitynow_workflow.email_on_manager_change
```

Notes:

- The `identitynow_source_entitlement` data source returns all entitlements of the source with the given `name` in the computed `entitlements` list (an empty list when there is no match), e.g. `data.identitynow_source_entitlement.aad_operator.entitlements[0].id` or `data.identitynow_source_entitlement.aad_operator.entitlements[*].id`.
- Entitlements are only available after the source has been aggregated, and the account schema and aggregation schedule can only be managed after the source connection has been configured and tested.

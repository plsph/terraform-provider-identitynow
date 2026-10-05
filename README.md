To find how you can declare resources see [documentation](https://registry.terraform.io/providers/plsph/identitynow/latest/docs) and the [examples](examples).

### Resources

- `identitynow_access_model_metadata_attribute`
- `identitynow_access_profile`
- `identitynow_access_profile_attachment`
- `identitynow_access_request_config`
- `identitynow_account_schema`
- `identitynow_ai_access_request_recommendations_config`
- `identitynow_branding`
- `identitynow_campaign_reports_config`
- `identitynow_campaign_template`
- `identitynow_campaign_template_schedule`
- `identitynow_connector`
- `identitynow_connector_customizer`
- `identitynow_connector_rule`
- `identitynow_custom_password_instruction`
- `identitynow_custom_user_level`
- `identitynow_das_application`
- `identitynow_das_task_schedule`
- `identitynow_data_segment`
- `identitynow_dimension`
- `identitynow_form_definition`
- `identitynow_governance_group`
- `identitynow_governance_group_members`
- `identitynow_identity_attribute`
- `identitynow_identity_profile`
- `identitynow_launcher`
- `identitynow_lifecycle_state`
- `identitynow_lockout_config`
- `identitynow_managed_client`
- `identitynow_managed_cluster`
- `identitynow_managed_cluster_type`
- `identitynow_mfa_duo_config`
- `identitynow_mfa_okta_config`
- `identitynow_multihost`
- `identitynow_network_config`
- `identitynow_non_employee_schema_attribute`
- `identitynow_non_employee_source`
- `identitynow_notification_template`
- `identitynow_oauth_client`
- `identitynow_org_config`
- `identitynow_parameter`
- `identitynow_password_policy`
- `identitynow_password_sync_group`
- `identitynow_personal_access_token`
- `identitynow_privilege_criteria`
- `identitynow_public_identities_config`
- `identitynow_reassignment_configuration`
- `identitynow_reassignment_tenant_config`
- `identitynow_recommendations_config`
- `identitynow_role`
- `identitynow_role_propagation_config`
- `identitynow_saved_search`
- `identitynow_schedule_account_aggregation`
- `identitynow_scheduled_search`
- `identitynow_sdi_status_check_config`
- `identitynow_search_attribute_config`
- `identitynow_segment`
- `identitynow_service_desk_integration`
- `identitynow_service_provider_config`
- `identitynow_session_config`
- `identitynow_sim_integration`
- `identitynow_sod_policy`
- `identitynow_sod_policy_schedule`
- `identitynow_source`
- `identitynow_source_app`
- `identitynow_source_provisioning_policy`
- `identitynow_source_schedule`
- `identitynow_source_subtype`
- `identitynow_tag`
- `identitynow_tagged_object`
- `identitynow_transform`
- `identitynow_trigger_subscription`
- `identitynow_ui_metadata`
- `identitynow_verified_from_address`
- `identitynow_workflow`

### Data Sources

- `identitynow_access_model_metadata_attribute`
- `identitynow_access_profile`
- `identitynow_auth_profile`
- `identitynow_authorization_right_sets`
- `identitynow_branding`
- `identitynow_campaign_template`
- `identitynow_connector`
- `identitynow_connector_customizer`
- `identitynow_connector_rule`
- `identitynow_custom_password_instruction`
- `identitynow_custom_user_level`
- `identitynow_das_application`
- `identitynow_data_segment`
- `identitynow_dimension`
- `identitynow_form_definition`
- `identitynow_governance_group`
- `identitynow_identity`
- `identitynow_identity_attribute`
- `identitynow_identity_profile`
- `identitynow_launcher`
- `identitynow_lifecycle_state`
- `identitynow_managed_client`
- `identitynow_managed_cluster`
- `identitynow_managed_cluster_type`
- `identitynow_multihost`
- `identitynow_non_employee_schema_attribute`
- `identitynow_non_employee_source`
- `identitynow_notification_template`
- `identitynow_oauth_client`
- `identitynow_org_config`
- `identitynow_parameter`
- `identitynow_password_sync_group`
- `identitynow_personal_access_token`
- `identitynow_privilege_criteria`
- `identitynow_role`
- `identitynow_saved_search`
- `identitynow_scheduled_search`
- `identitynow_search_attribute_config`
- `identitynow_segment`
- `identitynow_service_desk_integration`
- `identitynow_service_desk_integration_types`
- `identitynow_sim_integration`
- `identitynow_sod_policy`
- `identitynow_source`
- `identitynow_source_app`
- `identitynow_source_entitlement`
- `identitynow_source_provisioning_policy`
- `identitynow_source_schedule`
- `identitynow_source_subtype`
- `identitynow_tag`
- `identitynow_tenant`
- `identitynow_transform`
- `identitynow_trigger`
- `identitynow_valid_time_zones`
- `identitynow_verified_from_address`
- `identitynow_workflow`

### Limitations:
- `identitynow_source` manages the basic settings of a source (name, description, connector, owner, cluster, delete threshold, authoritative). Connector attributes are not managed by the provider, configure them in IdentityNow; they are kept when the provider updates the source.

- Sources get created first, but an aggregation has to run before the rest of the plan can complete successfully because the entitlement data lookups will fail until the aggregation has pulled the entitlements from the source into IdentityNow.

- After creating the source, you also need to go into the UI and press the "Test Connection" button to verify the source. This unlocks the ability to apply `identitynow_account_schema` and `identitynow_schedule_account_aggregation`.

- Password policies can be created, but there is a bug in Idn that makes the association to the source (`source_ids`) not work. For now, you have to go into the UI and make the association. Sailpoint ticket: https://support.sailpoint.com/hc/en-us/requests/82917

- This is provider in development working on experimental api.

Note: The `identitynow_source_entitlement` data source returns all entitlements of the source with the given `name` in the computed `entitlements` list, e.g. `data.identitynow_source_entitlement.example.entitlements[0].id`.

# Development
The repository follows the layout of the [Terraform provider scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding-framework):

- `main.go` starts the provider server.
- `internal/provider` contains the provider, the API client, and one `<name>_resource.go` or `<name>_data_source.go` file per resource and data source, with tests next to them.
- `templates` contains the documentation templates: `index.md.tmpl`, `resources/<name>.md.tmpl` and `data-sources/<name>.md.tmpl`.
- `examples` contains the examples used in the documentation (`provider/provider.tf`, `resources/<type>/resource.tf` and `import.sh`, `data-sources/<type>/data-source.tf`) and a complete example configuration in `examples/complete`.
- `docs` contains the registry documentation generated from `templates` and `examples`. Do not edit it directly.
- `tools` contains the documentation tooling.

Edit the Go files that make up the provider, and rebuild the provider.

```bash
./scripts/build.sh
```

This script places the provider binary in an implied local mirror directory ($HOME/.terraform.d/plugins/). See build.sh
for more comments about ensuring that Terraform uses the local mirror rather than searching the remote registry. 

# Documentation

The documentation in `docs` is generated with [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). To change it, edit the templates in `templates` or the examples in `examples`, then run:

```sh
$ make generate
```

This formats the examples, renders the templates into `docs` and validates the result. It requires `terraform` on the `PATH`; if it is not found, tfplugindocs downloads it. A documentation template for a new resource or data source goes into `templates/resources/<name>.md.tmpl` or `templates/data-sources/<name>.md.tmpl`. Reference its example with `{{ tffile .ExampleFile }}` and its import command with `{{ codefile "shell" .ImportFile }}`. Without a template, tfplugindocs generates the page from the schema descriptions.

# Testing the Provider

In order to test the provider, you can simply run `make test`.
```sh
$ make test
```
In order to run the full suite of Acceptance tests the IdentityNow URL, client id and secret, owner name and id, external owner id, and cluster name and id are needed to make the API calls that create IdentityNow objects for the tests.

To run acceptance tests, provide these values as the environment variables listed in `scripts/gotestacc_vars.sh` (`IDENTITYNOW_URL`, `IDENTITYNOW_CLIENT_ID`, `IDENTITYNOW_CLIENT_SECRET`, `IDENTITYNOW_OWNER_ID`, `IDENTITYNOW_OWNER_NAME`, `IDENTITYNOW_EXTERNAL_OWNER_ID`, `IDENTITYNOW_CLUSTER_ID`, `IDENTITYNOW_CLUSTER_NAME`), either by exporting them or in `.secrets/gotestacc_vars.sh` (ignored by git), and then simply run `make testacc`. Do not commit real values.

```sh
$ make testacc
```

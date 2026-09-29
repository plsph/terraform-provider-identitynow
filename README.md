To find how you can declare resources see [documentation](https://registry.terraform.io/providers/plsph/identitynow/latest/docs) and the [examples](examples).

### Resources

- `identitynow_access_profile`
- `identitynow_access_profile_attachment`
- `identitynow_account_schema`
- `identitynow_dimension`
- `identitynow_form_definition`
- `identitynow_governance_group`
- `identitynow_governance_group_members`
- `identitynow_password_policy`
- `identitynow_role`
- `identitynow_schedule_account_aggregation`
- `identitynow_segment`
- `identitynow_source`
- `identitynow_source_app`
- `identitynow_tagged_object`
- `identitynow_workflow`

### Data Sources

- `identitynow_access_profile`
- `identitynow_dimension`
- `identitynow_form_definition`
- `identitynow_governance_group`
- `identitynow_identity`
- `identitynow_role`
- `identitynow_segment`
- `identitynow_source`
- `identitynow_source_app`
- `identitynow_source_entitlement`
- `identitynow_workflow`

### Limitations:
- `identitynow_source` manages the basic settings of a source (name, description, connector, owner, cluster, delete threshold, authoritative). Connector attributes are not managed by the provider, configure them in IdentityNow; they are kept when the provider updates the source.

- Sources get created first, but an aggregation has to run before the rest of the plan can complete successfully because the entitlement data lookups will fail until the aggregation has pulled the entitlements from the source into IdentityNow.

- After creating the source, you also need to go into the UI and press the "Test Connection" button to verify the source. This unlocks the ability to apply `identitynow_account_schema` and `identitynow_schedule_account_aggregation`.

- Password policies can be created, but there is a bug in Idn that makes the association to the source (`source_ids`) not work. For now, you have to go into the UI and make the association. Sailpoint ticket: https://support.sailpoint.com/hc/en-us/requests/82917

- This is provider in development working on experimental api.

Note: The `identitynow_source_entitlement` data source returns all entitlements of the source with the given `name` in the computed `entitlements` list, e.g. `data.identitynow_source_entitlement.example.entitlements[0].id`.

# Development
Edit the Go files that make up the provider, and rebuild the provider.

```bash
./scripts/build.sh
```

This script places the provider binary in an implied local mirror directory ($HOME/.terraform.d/plugins/). See build.sh
for more comments about ensuring that Terraform uses the local mirror rather than searching the remote registry. 

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

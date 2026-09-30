## Unreleased

FEATURES:
* resource/identitynow_form_definition, data-source/identitynow_form_definition: manage SailPoint custom forms
* new resources and data sources for the v2026 API configuration objects:
  * identity: identity profiles, lifecycle states, identity attributes, access model metadata attributes, search attribute config, auth profiles (data source)
  * sources and connectors: provisioning policies, v2026 source schedules, connector rules, custom connectors, connector customizers, source subtypes, multi-host integrations, password sync groups, transforms
  * security: OAuth clients, personal access tokens, custom user levels and right sets, parameter storage, custom password instructions, brandings, launchers
  * governance: SOD policies and schedules, campaign templates and schedules, privilege criteria, data segments, saved and scheduled searches, tags
  * integrations: service desk and SIM integrations, managed clusters, managed clients, managed cluster types, Data Access Security applications and task schedules
  * events and notifications: trigger subscriptions, triggers (data source), notification templates, verified from addresses, non-employee sources and schema attributes, work reassignment configurations
  * tenant settings (one resource per tenant, destroy only removes it from state): access request, org, lockout, session, service provider, network, role propagation, work reassignment, MFA Duo and Okta, campaign reports, recommendations, AI access request recommendations, service desk status check, UI metadata, public identities; tenant, org config and valid time zones data sources
* resource/identitynow_password_policy, resource/identitynow_account_schema, resource/identitynow_schedule_account_aggregation: import support (`<source_id>/<schema_id>` for account schemas)
* provider: retries on 429 (and 502/503/504 for GET, PUT and DELETE requests) with Retry-After support, token refresh on 401
* data sources: fill owner, source, cluster, access profiles, metadata, provisioning criteria and identity attributes that were declared but always empty

BREAKING CHANGES AND BEHAVIOUR CHANGES:
* provider: `api_url` is optional and falls back to `IDENTITYNOW_URL`; missing credentials fail at configure time; pool size and rate limit environment variables are parsed and must be at least 1
* provider: the registry manifest now declares protocol 6.0, the protocol the provider serves
* resource/identitynow_source: `connector` and `authoritative` force replacement, `name` is updated in place; exactly one `owner` block is required
* resource/identitynow_role, resource/identitynow_dimension: `name` is updated in place instead of forcing replacement
* resource/identitynow_account_schema: destroy only removes the resource from state; without `attributes` blocks existing attributes are left unchanged
* resource/identitynow_schedule_account_aggregation: exactly one cron expression is supported
* resource/identitynow_access_profile_attachment, resource/identitynow_governance_group_members: destroy removes only the items managed by the resource
* resource/identitynow_tagged_object: `tags` is no longer computed, removing it clears the tags
* resource/identitynow_access_profile: changing `access_model_metadata` of an existing access profile shows a warning, the access profile API only accepts metadata on creation
* data-source/identitynow_identity, data-source/identitynow_workflow, data-source/identitynow_segment: fail when several objects match instead of picking the first one

BUG FIXES:
* provider: fix divide-by-zero crash and hanging requests when pool size or rate limit environment variables are set
* provider: send OAuth client credentials in the request body, so they are not leaked in error messages and secrets with special characters work
* provider: fix a data race on the access token, refresh expired tokens for long running operations
* provider: detect 404 responses without a JSON body as not found, accept empty success responses
* provider: escape filter values in lookups by name, email and alias
* provider: stop logging connector attributes and response bodies that can contain secrets
* resource/identitynow_password_policy: fix update URL, fix crash on some delete errors, fix inconsistent results for settings not in the configuration
* resource/identitynow_schedule_account_aggregation: fix crash when a source has no schedule
* resource/identitynow_account_schema: fix `is_multi_valued` never being sent (wrong API field), keep `nativeName` and other unmanaged attribute fields, keep schema features and configuration on update
* resource/identitynow_source: update with JSON Patch of managed fields instead of a partial PUT that could clear the configuration; fix creating Microsoft Entra sources
* resource/identitynow_segment: fix destroy always failing, fix three level visibility criteria, detect changes to `visibility_criteria_json`, allow removing criteria and owner
* resource/identitynow_workflow: support `enabled = true` on create and destroying enabled workflows; compare `steps_json` and `attributes_json` semantically
* resource/identitynow_access_profile: patch all changed fields on update (including name), keep unset `segments` null, round trip three level provisioning criteria, do not send unset booleans
* resource/identitynow_role: do not send unset booleans, allow clearing description and membership, fix inconsistent results for access request config and metadata defaults
* resource/identitynow_source_app, resource/identitynow_governance_group, resource/identitynow_governance_group_members: fix unknown `type` values after apply, patch account source changes
* resource/identitynow_access_profile_attachment, resource/identitynow_governance_group_members: stable ordering, `source_app_id` and `governance_group_id` force replacement
* resource/identitynow_tagged_object: fix create without tags, detect drift on every object, de-duplicate tags that differ only in case

Changes between 0.4.0 and 0.17.1 were not recorded in this file, see the git history.

## 0.4.0
* upgrade: upgrade cc api
## 0.3.4

BUG FIXES:
* fix: build for darwin_arm64
* skip: 0.3.4 due to incorrecly release

## 0.3.2

BUG FIXES:
* fix: update type as required for updateSource ([#5](https://github.com/OpenAxon/terraform-provider-identitynow/pull/5))

## 0.3.1 (December 25, 2020)

Initial release

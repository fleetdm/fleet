# Migrate from Fleet 4 to Fleet 5

Fleet 5 removes deprecated features such as API endpoints, API parameters, and YAML (GitOps) keys. This guide lists every breaking change, what changed, and what you need to do.

Update to Fleet 4.95 first, then work through the list. Fleet will support Fleet 4.95 with security fixes through June 30, 2027.


## Breaking changes

- [Teams are now fleets](#teams-are-now-fleets)
- [SSO: Fleet role SAML attribute renamed](#sso-fleet-role-saml-attribute-renamed)
- [Policy endpoints merged](#policy-endpoints-merged)
- [One automation per policy](#one-automation-per-policy)
- [Schedule endpoints replaced by reports](#schedule-endpoints-replaced-by-reports)
- [Built-in labels can't target reports](#built-in-labels-cant-target-reports)
- [MDM endpoints moved out of /mdm](#mdm-endpoints-moved-out-of-mdm)
- [List MDM commands requires a host](#list-mdm-commands-requires-a-host)
- [Other endpoints moved](#other-endpoints-moved)
- [Apple Business Manager renamed to Apple Business](#apple-business-manager-renamed-to-apple-business)
- [macos_setup renamed to setup_experience](#macos_setup-renamed-to-setup_experience)
- [Setup experience script removed](#setup-experience-script-removed)
- [Setup experience and Recovery Lock settings renamed](#setup-experience-and-recovery-lock-settings-renamed)
- [Organization logo settings renamed](#organization-logo-settings-renamed)
- [all_linux_count removed](#all_linux_count-removed)
- [os_name and os_version are separate](#os_name-and-os_version-are-separate)
- [Windows wipe options moved](#windows-wipe-options-moved)
- [YAML-only lists removed from configuration responses](#yaml-only-lists-removed-from-configuration-responses)
- [detail_query_overrides removed](#detail_query_overrides-removed)
- [Software API changes](#software-api-changes)
- [Label platforms: centos and ubuntu replaced by linux](#label-platforms-centos-and-ubuntu-replaced-by-linux)
- [Host identifiers for personal and Android hosts](#host-identifiers-for-personal-and-android-hosts)
- [Android Lock resets the PIN](#android-lock-resets-the-pin)
- [SCIM user delete requires an admin](#scim-user-delete-requires-an-admin)
- [GitOps users can't run custom MDM commands](#gitops-users-cant-run-custom-mdm-commands)
- [Omitted GitOps settings reset to defaults](#omitted-gitops-settings-reset-to-defaults)
- [Manual labels don't accept hostname](#manual-labels-dont-accept-hostname)
- [scripts_disabled removed](#scripts_disabled-removed)
- [Package YAML files must be lists](#package-yaml-files-must-be-lists)
- [Certificate authority names must be unique](#certificate-authority-names-must-be-unique)
- [Fleet names must be unique](#fleet-names-must-be-unique)
- [Packs removed](#packs-removed)
- [Disk encryption activity types merged](#disk-encryption-activity-types-merged)
- [fleetd process name is Fleet](#fleetd-process-name-is-fleet)
- [fleetctl package file names changed](#fleetctl-package-file-names-changed)
- [Static SCEP challenge off by default](#static-scep-challenge-off-by-default)
- [Async host processing removed](#async-host-processing-removed)
- [osquery_min_software_last_opened_at_diff removed](#osquery_min_software_last_opened_at_diff-removed)


### Teams are now fleets

You'll be missed, and we'll keep saying "teams" out of muscle memory.

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/teams` | [`GET /api/v1/fleet/fleets`](https://fleetdm.com/docs/rest-api/rest-api#list-fleets) |
| `GET /api/v1/fleet/teams/:id` | [`GET /api/v1/fleet/fleets/:id`](https://fleetdm.com/docs/rest-api/rest-api#get-fleet) |
| `POST /api/v1/fleet/teams` | [`POST /api/v1/fleet/fleets`](https://fleetdm.com/docs/rest-api/rest-api#create-fleet) |
| `PATCH /api/v1/fleet/teams/:id` | [`PATCH /api/v1/fleet/fleets/:id`](https://fleetdm.com/docs/rest-api/rest-api#update-fleet) |
| `DELETE /api/v1/fleet/teams/:id` | [`DELETE /api/v1/fleet/fleets/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-fleet) |
| `GET /api/v1/fleet/teams/:id/secrets` | [`GET /api/v1/fleet/fleets/:id/secrets`](https://fleetdm.com/docs/rest-api/rest-api#get-fleet-level-enroll-secrets) |
| `PATCH /api/v1/fleet/teams/:id/secrets` | [`PATCH /api/v1/fleet/fleets/:id/secrets`](https://fleetdm.com/docs/rest-api/rest-api#update-fleet-level-enroll-secrets) |
| `POST /api/v1/fleet/teams/:id/agent_options` | [`POST /api/v1/fleet/fleets/:id/agent_options`](https://fleetdm.com/docs/rest-api/rest-api#update-fleet-level-agent-options) |
| `PATCH /api/v1/fleet/teams/:id/users` | [`PATCH /api/v1/fleet/fleets/:id/users`](https://fleetdm.com/docs/rest-api/rest-api#add-users-to-fleet) |
| `GET` and `DELETE /api/v1/fleet/teams/:id/users` | `GET` and `DELETE /api/v1/fleet/fleets/:id/users` |
| `POST /api/v1/fleet/spec/teams` | `POST /api/v1/fleet/spec/fleets` |
| `PATCH /api/v1/fleet/vpp_tokens/:id/teams` | [`PATCH /api/v1/fleet/vpp_tokens/:id/fleets`](https://fleetdm.com/docs/rest-api/rest-api#update-volume-purchasing-program-vpp-tokens-fleets) |
| `PATCH /api/v1/fleet/ab_tokens/:id/teams` (and `/abm_tokens/:id/teams` and `/abm_tokens/:id/fleets`) | `PATCH /api/v1/fleet/ab_tokens/:id/fleets` |
| `team_id`, `team_name`, `team`, `teams`, `team_ids`, `team_ids_by_name`, and `available_teams` in requests, responses, and query parameters | `fleet_id`, `fleet_name`, `fleet`, `fleets`, `fleet_ids`, `fleet_ids_by_name`, and `available_fleets` |
| `team_id` and `team_name` as `order_key` values | `fleet_id` and `fleet_name` |
| `team_id` and `team_name` columns in [Get hosts report in CSV](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-report-in-csv) | `fleet_id` and `fleet_name` columns |
| `macos_team_id`, `ios_team_id`, `ipados_team_id`, and `byod_team_id` when you assign fleets to an Apple Business token | `macos_fleet_id`, `ios_fleet_id`, `ipados_fleet_id`, and `byod_fleet_id` |

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `default_team`, `macos_team`, `ios_team`, `ipados_team`, and `byod_team` under `org_settings.mdm.apple_business` | `default_fleet`, `macos_fleet`, `ios_fleet`, `ipados_fleet`, and `byod_fleet` ([docs](https://fleetdm.com/docs/configuration/yaml-files#apple-business)) |

#### fleetctl

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get teams` | `fleetctl get fleets` |
| `fleetctl gitops --delete-other-teams` | `fleetctl gitops --delete-other-fleets` |
| `fleetctl apply --policies-team` | `fleetctl apply --policies-fleet` |
| `--team` on `fleetctl report`, `user create`, `hosts transfer`, `run-script`, `get reports`, `get labels`, `get hosts`, and `get software` | `--fleet` on the same commands |
| `fleetctl generate-gitops --team no-team` | `fleetctl generate-gitops --fleet unassigned` |

#### Activities

| Fleet 4 | Fleet 5 |
|:--|:--|
| `created_team`, `deleted_team`, `applied_spec_team`, `changed_user_team_role`, and `deleted_user_team_role` | `created_fleet`, `deleted_fleet`, `applied_spec_fleet`, `changed_user_fleet_role`, and `deleted_user_fleet_role` ([docs](https://fleetdm.com/docs/api/global-audit-logs)) |

What to do:

1. Search your scripts, integrations, and GitOps repo for the old names:
   ```sh
   grep -rnE '/fleet/teams|/teams\b|team_ids?\b|team_name|available_teams|default_team|(macos|ios|ipados|byod)_team|--team\b|--policies-team|--delete-other-teams|no-team|get teams' .
   ```
2. Replace each match with its Fleet 5 name. The `/fleets` endpoints and `--fleet` flags already work in Fleet 4, so you can do this before you upgrade.
3. Update anything that reads the hosts CSV or filters activities, like SIEM rules, to use the new names.

Issue: [#52795](https://github.com/fleetdm/fleet/issues/52795)


### SSO: Fleet role SAML attribute renamed

#### Identity provider (IdP)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `FLEET_JIT_USER_ROLE_TEAM_<ID>` SAML attribute | `FLEET_JIT_USER_ROLE_FLEET_<ID>` ([docs](https://fleetdm.com/docs/deploy/single-sign-on-sso#just-in-time-jit-user-provisioning)) |

What to do: Rename the attribute in your IdP before you upgrade. Fleet 4 already accepts the new name. Fleet 5 ignores the old one, so new users who log in with SSO get the global Observer role, and existing users keep the roles they have now.

Issue: [#53925](https://github.com/fleetdm/fleet/issues/53925)


### Policy endpoints merged

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/global/policies` and `GET /api/v1/fleet/fleets/:id/policies` | [`GET /api/v1/fleet/policies`](https://fleetdm.com/docs/rest-api/rest-api#list-policies), with `fleet_id` for a fleet's policies |
| `GET /api/v1/fleet/global/policies/:id` and `GET /api/v1/fleet/fleets/:fleet_id/policies/:policy_id` | [`GET /api/v1/fleet/policies/:id`](https://fleetdm.com/docs/rest-api/rest-api#get-policy) |
| `POST /api/v1/fleet/global/policies` and `POST /api/v1/fleet/fleets/:id/policies` | [`POST /api/v1/fleet/policies`](https://fleetdm.com/docs/rest-api/rest-api#create-policy), with `fleet_id` in the request body for a fleet policy |
| `PATCH /api/v1/fleet/global/policies/:id` and `PATCH /api/v1/fleet/fleets/:fleet_id/policies/:policy_id` | [`PATCH /api/v1/fleet/policies/:id`](https://fleetdm.com/docs/rest-api/rest-api#update-policy) |
| `POST /api/v1/fleet/global/policies/delete` and `POST /api/v1/fleet/fleets/:fleet_id/policies/delete` | [`POST /api/v1/fleet/policies/delete`](https://fleetdm.com/docs/rest-api/rest-api#delete-policies) |
| `failing_policies_webhook.policy_ids` in [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | `webhooks_and_tickets_enabled` on each policy |

The `/team/...` and `/teams/...` policy paths are also removed.

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `policy_ids` under `failing_policies_webhook` | `webhooks_and_tickets_enabled: true` on each policy ([docs](https://fleetdm.com/docs/configuration/yaml-files#automations)) |

What to do:

1. Search for the old paths: `grep -rnE '/fleet/global/policies|/(teams?|fleets)/[^/]+/policies' .`
2. Use `/api/v1/fleet/policies`. To work with a fleet's policies, pass `fleet_id`. Without it, you get global policies.
3. Move the IDs in `failing_policies_webhook.policy_ids` to `webhooks_and_tickets_enabled: true` on each policy.

Issue: [#52900](https://github.com/fleetdm/fleet/issues/52900)


### One automation per policy

#### REST API and YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| A policy can have more than one of `install_software`, `run_script`, `resend_configuration_profile`, `calendar_events_enabled`, `conditional_access_enabled`, and `webhooks_and_tickets_enabled` | One automation per policy ([docs](https://fleetdm.com/docs/configuration/yaml-files#automations)). `fleetctl gitops` fails if a policy has more than one. |

What to do: Before you upgrade, find policies with more than one automation and move the extra automations to new policies.

Issue: [#52900](https://github.com/fleetdm/fleet/issues/52900)


### Schedule endpoints replaced by reports

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/global/schedule` and `GET /api/v1/fleet/fleets/:id/schedule` | [`GET /api/v1/fleet/reports`](https://fleetdm.com/docs/rest-api/rest-api#list-reports), with `fleet_id` for a fleet. Scheduled reports have an `interval` greater than `0`. |
| `POST /api/v1/fleet/global/schedule` and `POST /api/v1/fleet/fleets/:id/schedule`, which copied the report and scheduled the copy | [`POST /api/v1/fleet/reports`](https://fleetdm.com/docs/rest-api/rest-api#create-report) with `interval`, `platform`, `min_osquery_version`, and `logging` (and `fleet_id` for a fleet). No copy. |
| `PATCH /api/v1/fleet/global/schedule/:id` and `PATCH /api/v1/fleet/fleets/:id/schedule/:report_id` | [`PATCH /api/v1/fleet/reports/:id`](https://fleetdm.com/docs/rest-api/rest-api#update-report) |
| `DELETE /api/v1/fleet/global/schedule/:id` and `DELETE /api/v1/fleet/fleets/:id/schedule/:report_id` | [`DELETE /api/v1/fleet/reports/id/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-report-by-id) |

The `/api/latest/fleet/schedule`, `/team/:id/schedule`, and `/teams/:id/schedule` paths are also removed.

What to do:

1. Search for the old paths: `grep -rnE '/schedule\b' .`
2. Create or update reports with an `interval` instead. To unschedule a report and keep it, [update the report](https://fleetdm.com/docs/rest-api/rest-api#update-report) with `interval` set to `0`.

Issue: [#54461](https://github.com/fleetdm/fleet/issues/54461)


### Built-in labels can't target reports

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| [`POST /api/v1/fleet/reports`](https://fleetdm.com/docs/rest-api/rest-api#create-report) accepts built-in labels in `labels_include_any` or `labels_include_all` | Returns an error |

What to do: Target the report by `platform` instead.

Issue: [#53800](https://github.com/fleetdm/fleet/issues/53800)


### MDM endpoints moved out of /mdm

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/mdm/commands` and `GET /api/v1/fleet/mdm/apple/commands` | [`GET /api/v1/fleet/commands`](https://fleetdm.com/docs/rest-api/rest-api#list-mdm-commands) |
| `GET /api/v1/fleet/mdm/commandresults` | [`GET /api/v1/fleet/commands/results`](https://fleetdm.com/docs/rest-api/rest-api#get-mdm-command-results) |
| `POST /api/v1/fleet/mdm/commands/run` | [`POST /api/v1/fleet/commands/run`](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command) |
| `PATCH /api/v1/fleet/mdm/hosts/:id/unenroll` | [`DELETE /api/v1/fleet/hosts/:id/mdm`](https://fleetdm.com/docs/rest-api/rest-api#turn-off-hosts-mdm) |
| `POST /api/v1/fleet/mdm/hosts/:id/lock` | [`POST /api/v1/fleet/hosts/:id/lock`](https://fleetdm.com/docs/rest-api/rest-api#lock-host) |
| `POST /api/v1/fleet/mdm/hosts/:id/wipe` | [`POST /api/v1/fleet/hosts/:id/wipe`](https://fleetdm.com/docs/rest-api/rest-api#wipe-host) |
| `GET /api/v1/fleet/mdm/hosts/:id/profiles` | [`GET /api/v1/fleet/hosts/:id/configuration_profiles`](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-os-settings-configuration-profile) |
| `GET /api/v1/fleet/mdm/hosts/:id/encryption_key` | [`GET /api/v1/fleet/hosts/:id/encryption_key`](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-disk-encryption-key) |
| `GET /api/v1/fleet/mdm/disk_encryption/summary` (and `/mdm/apple/filevault/summary`) | [`GET /api/v1/fleet/disk_encryption`](https://fleetdm.com/docs/rest-api/rest-api#get-disk-encryption-status) |
| `GET /api/v1/fleet/mdm/profiles` (and `/mdm/apple/profiles`) | [`GET /api/v1/fleet/configuration_profiles`](https://fleetdm.com/docs/rest-api/rest-api#list-configuration-profiles) |
| `POST /api/v1/fleet/mdm/profiles` (and `/mdm/apple/profiles`) | [`POST /api/v1/fleet/configuration_profiles`](https://fleetdm.com/docs/rest-api/rest-api#create-configuration-profile) |
| `GET /api/v1/fleet/mdm/profiles/:profile_uuid` (and `/mdm/apple/profiles/:profile_id`) | [`GET /api/v1/fleet/configuration_profiles/:profile_uuid`](https://fleetdm.com/docs/rest-api/rest-api#get-or-download-configuration-profile) |
| `DELETE /api/v1/fleet/mdm/profiles/:profile_uuid` (and `/mdm/apple/profiles/:profile_id`) | [`DELETE /api/v1/fleet/configuration_profiles/:profile_uuid`](https://fleetdm.com/docs/rest-api/rest-api#delete-configuration-profile) |
| `GET /api/v1/fleet/mdm/profiles/summary` (and `/mdm/apple/profiles/summary`) | [`GET /api/v1/fleet/configuration_profiles/summary`](https://fleetdm.com/docs/rest-api/rest-api#get-os-settings-configuration-profiles-status) |
| `PATCH /api/v1/fleet/mdm/apple/setup` | [`PATCH /api/v1/fleet/setup_experience`](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience) |
| `GET /api/v1/fleet/mdm/apple` | [`GET /api/v1/fleet/apns`](https://fleetdm.com/docs/rest-api/rest-api#get-apple-push-notification-service-apns) |
| `POST /api/v1/fleet/mdm/apple/enrollment_profile` | [`POST /api/v1/fleet/enrollment_profiles/automatic`](https://fleetdm.com/docs/rest-api/rest-api#update-custom-mdm-setup-enrollment-profile) |
| `GET /api/v1/fleet/mdm/apple/enrollment_profile` | [`GET /api/v1/fleet/enrollment_profiles/automatic`](https://fleetdm.com/docs/rest-api/rest-api#get-custom-mdm-setup-enrollment-profile) |
| `DELETE /api/v1/fleet/mdm/apple/enrollment_profile` | [`DELETE /api/v1/fleet/enrollment_profiles/automatic`](https://fleetdm.com/docs/rest-api/rest-api#delete-custom-mdm-setup-enrollment-profile) |
| `GET /api/v1/fleet/mdm/manual_enrollment_profile` | [`GET /api/v1/fleet/enrollment_profiles/manual`](https://fleetdm.com/docs/rest-api/rest-api#get-manual-enrollment-profile) |
| `POST /api/v1/fleet/mdm/bootstrap` (and `/mdm/apple/bootstrap`) | [`POST /api/v1/fleet/bootstrap`](https://fleetdm.com/docs/rest-api/rest-api#create-bootstrap-package) |
| `GET /api/v1/fleet/mdm/bootstrap/:fleet_id/metadata` (and `/mdm/apple/bootstrap/:fleet_id/metadata`) | [`GET /api/v1/fleet/bootstrap/:fleet_id/metadata`](https://fleetdm.com/docs/rest-api/rest-api#get-bootstrap-package-metadata) |
| `DELETE /api/v1/fleet/mdm/bootstrap/:fleet_id` (and `/mdm/apple/bootstrap/:fleet_id`) | [`DELETE /api/v1/fleet/bootstrap/:fleet_id`](https://fleetdm.com/docs/rest-api/rest-api#delete-bootstrap-package) |
| `GET /api/v1/fleet/mdm/bootstrap/summary` (and `/mdm/apple/bootstrap/summary`) | [`GET /api/v1/fleet/bootstrap/summary`](https://fleetdm.com/docs/rest-api/rest-api#get-bootstrap-package-status) |
| `POST /api/v1/fleet/mdm/setup/eula` (and `/mdm/apple/setup/eula`) | [`POST /api/v1/fleet/setup_experience/eula`](https://fleetdm.com/docs/rest-api/rest-api#create-eula) |
| `GET /api/v1/fleet/mdm/setup/eula/metadata` (and `/mdm/apple/setup/eula/metadata`) | [`GET /api/v1/fleet/setup_experience/eula/metadata`](https://fleetdm.com/docs/rest-api/rest-api#get-eula-metadata) |
| `DELETE /api/v1/fleet/mdm/setup/eula/:token` (and `/mdm/apple/setup/eula/:token`) | [`DELETE /api/v1/fleet/setup_experience/eula/:token`](https://fleetdm.com/docs/rest-api/rest-api#delete-eula) |

What to do: Search for `/fleet/mdm/` in your scripts and integrations, and replace each path with its Fleet 5 path.

Issue: [#53563](https://github.com/fleetdm/fleet/issues/53563)


### List MDM commands requires a host

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/commands` without `host_identifier` | [`GET /api/v1/fleet/commands?host_identifier=<host>`](https://fleetdm.com/docs/rest-api/rest-api#list-mdm-commands). `host_identifier` is required. |

#### fleetctl

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get mdm-commands` | `fleetctl get mdm-commands --host <host>`. `--host` is required. |

What to do: Pass `host_identifier` in the API, or `--host` in `fleetctl`, on every call.

Issue: [#53146](https://github.com/fleetdm/fleet/issues/53146)


### Other endpoints moved

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/hosts/identifier/:identifier` | [`GET /api/v1/fleet/hosts_identifier/:identifier`](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier) |
| `GET /api/v1/fleet/scripts/batch/summary/:batch_execution_id` | [`GET /api/v1/fleet/scripts/batch/:batch_execution_id`](https://fleetdm.com/docs/rest-api/rest-api#get-batch-script) |
| `POST /api/v1/fleet/hosts/:host_id/configuration_profiles/resend/:profile_uuid` | [`POST /api/v1/fleet/hosts/:id/configuration_profiles/:profile_uuid/resend`](https://fleetdm.com/docs/rest-api/rest-api#resend-configuration-profile) |

What to do: Search for `hosts/identifier/`, `scripts/batch/summary/`, and `configuration_profiles/resend/`, and replace each path.

Issue: [#53080](https://github.com/fleetdm/fleet/issues/53080)


### Apple Business Manager renamed to Apple Business

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/abm_tokens`, `GET /api/v1/fleet/abm`, and `GET /api/v1/fleet/mdm/apple_bm` | [`GET /api/v1/fleet/ab_tokens`](https://fleetdm.com/docs/rest-api/rest-api#list-apple-business-ab-tokens) |
| `POST /api/v1/fleet/abm_tokens` | `POST /api/v1/fleet/ab_tokens` |
| `DELETE /api/v1/fleet/abm_tokens/:id` | `DELETE /api/v1/fleet/ab_tokens/:id` |
| `GET /api/v1/fleet/abm_tokens/count` | `GET /api/v1/fleet/ab_tokens/count` |
| `PATCH /api/v1/fleet/abm_tokens/:id/renew` | `PATCH /api/v1/fleet/ab_tokens/:id/renew` |
| `GET /api/v1/fleet/mdm/apple/abm_public_key` | `GET /api/v1/fleet/mdm/apple/ab_public_key` |
| `abm_token`, `abm_tokens`, and `host_dep_assignment.abm_token_id` in responses | `ab_token`, `ab_tokens`, and `ab_token_id` |
| `mdm.apple_business_manager`, `mdm.apple_bm_enabled_and_configured`, and `mdm.apple_bm_terms_expired` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) | `mdm.apple_business`, `mdm.apple_business_enabled_and_configured`, and `mdm.apple_business_terms_expired` |
| `mdm.apple_bm_default_team` in [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | Removed |

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `org_settings.mdm.apple_business_manager` | [`org_settings.mdm.apple_business`](https://fleetdm.com/docs/configuration/yaml-files#apple-business) |

#### fleetctl

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get mdm-apple-bm` (and the `mdm_apple_bm` alias) | `fleetctl get mdm-ab` |
| `fleetctl generate mdm-apple-bm` (and the `mdm_apple_bm` alias) | `fleetctl generate mdm-ab` |

#### Fleet server configuration

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.apple_bm_server_token` and `mdm.apple_bm_server_token_bytes` | `mdm.apple_business_server_token` and `mdm.apple_business_server_token_bytes` |
| `mdm.apple_bm_cert` and `mdm.apple_bm_cert_bytes` | `mdm.apple_business_cert` and `mdm.apple_business_cert_bytes` |
| `mdm.apple_bm_key` and `mdm.apple_bm_key_bytes` | `mdm.apple_business_key` and `mdm.apple_business_key_bytes` |
| `FLEET_MDM_APPLE_BM_*` environment variables | `FLEET_MDM_APPLE_BUSINESS_*` |

What to do:

1. Search your scripts, GitOps repo, and server configuration: `grep -rnE 'abm|apple_bm|APPLE_BM|apple_business_manager|mdm-apple-bm' .`
2. Replace each match with its Fleet 5 name. `mdm-ab` already works in Fleet 4.
3. Rename the server options and environment variables in the same deploy that upgrades to Fleet 5. Fleet 5 doesn't start if an old one is still set.

Issue: [#53272](https://github.com/fleetdm/fleet/issues/53272)


### macos_setup renamed to setup_experience

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.macos_setup` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration), [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration), and fleet settings | `mdm.setup_experience` |

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `controls.macos_setup` | [`controls.setup_experience`](https://fleetdm.com/docs/configuration/yaml-files#setup-experience) |

#### fleetctl

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get config` prints `macos_setup` | `fleetctl get config` prints `setup_experience` |

What to do: Search for `macos_setup` and rename it to `setup_experience`.

Issue: [#53894](https://github.com/fleetdm/fleet/issues/53894)


### Setup experience script removed

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `POST`, `GET`, and `DELETE /api/v1/fleet/setup_experience/script` | Removed. Add the script as a script-only package and include it in [setup experience software](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience-software). |

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `controls.setup_experience.macos_script` | A script-only package under [`software.packages`](https://fleetdm.com/docs/configuration/yaml-files#packages) with `setup_experience` or `setup_experience_platform` |

#### Fleet UI

| Fleet 4 | Fleet 5 |
|:--|:--|
| **Run script** tab in **Controls > Setup experience** | Removed |

What to do: Add your setup script as a script-only package that runs during setup experience, then remove `macos_script` and any calls to `/setup_experience/script`.

Issue: [#53274](https://github.com/fleetdm/fleet/issues/53274)


### Setup experience and Recovery Lock settings renamed

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `require_all_software_macos` and `require_all_software_windows` in [Update setup experience](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience) | `macos_require_all_software` and `windows_require_all_software` |
| `mdm.enable_recovery_lock_password` in [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | `mdm.macos_enable_recovery_lock_password` |

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `require_all_software_macos` and `require_all_software_windows` under `controls.setup_experience` | `macos_require_all_software` and `windows_require_all_software` ([docs](https://fleetdm.com/docs/configuration/yaml-files#setup-experience)) |
| `controls.enable_recovery_lock_password` | `controls.macos_enable_recovery_lock_password` ([docs](https://fleetdm.com/docs/configuration/yaml-files#controls)) |

What to do: Search for these keys and rename them.

Issue: [#53872](https://github.com/fleetdm/fleet/issues/53872)


### Organization logo settings renamed

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `org_logo_url` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) and [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | `org_logo_url_dark_mode` |
| `org_logo_url_light_background` | `org_logo_url_light_mode` |

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `org_info.org_logo_url` | [`org_info.org_logo_url_dark_mode`](https://fleetdm.com/docs/configuration/yaml-files#org-info) |
| `org_info.org_logo_url_light_background` | [`org_info.org_logo_url_light_mode`](https://fleetdm.com/docs/configuration/yaml-files#org-info) |

What to do: Rename both keys. Fleet 4 already accepts the new names.

Issue: [#52800](https://github.com/fleetdm/fleet/issues/52800)


### all_linux_count removed

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `all_linux_count` in [Get hosts summary](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-summary) | [`GET /api/v1/fleet/hosts/count`](https://fleetdm.com/docs/rest-api/rest-api#count-hosts) with `label_id` set to the ID of the built-in "All Linux" label |

What to do: Get the "All Linux" label's ID from `builtin_labels` in the Get hosts summary response, then call Count hosts with it.

Issue: [#53872](https://github.com/fleetdm/fleet/issues/53872)


### os_name and os_version are separate

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `os_name` and `os_version` combined in `GET /api/v1/fleet/hosts` and `GET /api/v1/fleet/hosts/report` | Separate `os_name` and `os_version` keys in [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) and [Get hosts report in CSV](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-report-in-csv) |

What to do: Update code that parses `os_name` to read the version from `os_version`.

Issue: [#53877](https://github.com/fleetdm/fleet/issues/53877)


### Windows wipe options moved

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `windows` object with `wipe_type` in `POST /api/v1/fleet/hosts/:id/wipe` | `POST /api/v1/fleet/hosts/:id/reset`. [Wipe host](https://fleetdm.com/docs/rest-api/rest-api#wipe-host) no longer takes a `windows` object. |

What to do: Use `POST /api/v1/fleet/hosts/:id/reset` instead.

Issue: [#55003](https://github.com/fleetdm/fleet/issues/55003)


### YAML-only lists removed from configuration responses

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.apple_settings.configuration_profiles`, `mdm.windows_settings.configuration_profiles`, `mdm.setup_experience`, `mdm.volume_purchasing_program`, and `scripts` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) and [Get fleet](https://fleetdm.com/docs/rest-api/rest-api#get-fleet) | Removed. Use each feature's list endpoint, for example [List configuration profiles](https://fleetdm.com/docs/rest-api/rest-api#list-configuration-profiles). |

What to do: These lists only included settings applied with YAML. Call the list endpoint for each feature instead.

Issue: [#53932](https://github.com/fleetdm/fleet/issues/53932)


### detail_query_overrides removed

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `features.detail_query_overrides` in [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | Removed |

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `features.detail_query_overrides` | Removed |

What to do: Remove `detail_query_overrides` from your YAML and API calls.

Issue: [#33769](https://github.com/fleetdm/fleet/issues/33769)


### Software API changes

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/software` and `GET /api/v1/fleet/software/count` | [`GET /api/v1/fleet/software/versions`](https://fleetdm.com/docs/rest-api/rest-api#list-software-versions) |
| `GET /api/v1/fleet/software/:id` | [`GET /api/v1/fleet/software/versions/:id`](https://fleetdm.com/docs/rest-api/rest-api#get-software-version) |
| `software_id` filter on `GET /api/v1/fleet/hosts` | `software_version_id` or `software_title_id` on [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) |
| `software_package` object in software title responses | `packages` array in [Get software](https://fleetdm.com/docs/rest-api/rest-api#get-software) |
| `app_store_app` object in software title responses | `app_store_apps` array in [Get software](https://fleetdm.com/docs/rest-api/rest-api#get-software) |
| `browser` in host, software title, and software version responses | `extension_for` |

#### fleetctl

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get software --json` and `--yaml` include `software_package`, `app_store_app`, and `browser` | They include `packages`, `app_store_apps`, and `extension_for` |

What to do: Search for `/fleet/software`, `software_id`, `software_package`, `app_store_app`, and `browser`, and update each request and the code that reads responses.

Issues: [#54408](https://github.com/fleetdm/fleet/issues/54408), [#54410](https://github.com/fleetdm/fleet/issues/54410), [#54409](https://github.com/fleetdm/fleet/issues/54409), [#52687](https://github.com/fleetdm/fleet/issues/52687), [#53395](https://github.com/fleetdm/fleet/issues/53395)


### Label platforms: centos and ubuntu replaced by linux

#### REST API and YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `centos` and `ubuntu` as a label's `platform` | `linux` ([API docs](https://fleetdm.com/docs/rest-api/rest-api#add-label), [YAML docs](https://fleetdm.com/docs/configuration/yaml-files#labels)) |

What to do: Change `centos` and `ubuntu` label platforms to `linux`.

Issue: [#52905](https://github.com/fleetdm/fleet/issues/52905)


### Host identifiers for personal and Android hosts

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `uuid` and `hardware_serial` for personal (BYOD) iOS, iPadOS, and Android hosts | Empty strings. `enrollment_id` has the enrollment ID ([docs](https://fleetdm.com/docs/rest-api/rest-api#get-host)). |
| `uuid` for company-owned Android hosts (a hash of the brand and serial) | Empty string. `hardware_serial` doesn't change, and `enrollment_id` is `null`. |

What to do: Identify personal hosts by `enrollment_id`, and company-owned Android hosts by `hardware_serial`.

Issue: [#53797](https://github.com/fleetdm/fleet/issues/53797)


### Android Lock resets the PIN

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `POST /api/v1/fleet/hosts/:id/lock` locks the screen of an Android host | [Lock host](https://fleetdm.com/docs/rest-api/rest-api#lock-host) resets the PIN to a random six-digit PIN and locks the host |

#### fleetctl

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl mdm unlock` doesn't support Android hosts | `fleetctl mdm unlock --host=<host>` returns the Android PIN |

What to do: To get the PIN, use [Unlock host](https://fleetdm.com/docs/rest-api/rest-api#unlock-host) or `fleetctl mdm unlock --host=<host>`. Update any end user instructions.

Issue: [#49909](https://github.com/fleetdm/fleet/issues/49909)


### SCIM user delete requires an admin

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `DELETE /api/v1/fleet/scim/Users/:id` works with a maintainer's API token | [Delete SCIM user](https://fleetdm.com/docs/rest-api/rest-api#delete-scim-user) requires an admin's API token |

What to do: Use an admin's API token for your SCIM integration.

Issue: [#53472](https://github.com/fleetdm/fleet/issues/53472)


### GitOps users can't run custom MDM commands

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| GitOps users can call `POST /api/v1/fleet/commands/run` | GitOps users get a 403 from [Run MDM command](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command) |

What to do: Run custom MDM commands with a different [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user).

Issue: [#53145](https://github.com/fleetdm/fleet/issues/53145)


### Omitted GitOps settings reset to defaults

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| If your YAML leaves out `gitops`, `host_expiry_settings`, or `smtp_settings`, they keep their current values | They reset to their defaults ([`gitops`](https://fleetdm.com/docs/configuration/yaml-files#gitops), [`host_expiry_settings`](https://fleetdm.com/docs/configuration/yaml-files#host-expiry-settings), [`smtp_settings`](https://fleetdm.com/docs/configuration/yaml-files#smtp-settings)) |

What to do: Before you upgrade, add these settings to your YAML with the values you use today. To get the current values, run `fleetctl generate-gitops`.

Issue: [#53899](https://github.com/fleetdm/fleet/issues/53899)


### Manual labels don't accept hostname

#### REST API and YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `hostname` as a host identifier in the `hosts` key of manual labels | Not accepted ([API docs](https://fleetdm.com/docs/rest-api/rest-api#add-label), [YAML docs](https://fleetdm.com/docs/configuration/yaml-files#labels)) |

What to do: Identify hosts in manual labels by something other than `hostname`.

Issue: [#53924](https://github.com/fleetdm/fleet/issues/53924)


### scripts_disabled removed

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| `server_settings.scripts_disabled` | Removed ([docs](https://fleetdm.com/docs/configuration/yaml-files#server-settings)). `fleetctl gitops` fails if it's set. Fleet 5 turns script features back on for instances that had `scripts_disabled: true`. |

#### Fleet UI

| Fleet 4 | Fleet 5 |
|:--|:--|
| **Disable scripts** checkbox in **Settings > Organization settings > Advanced options** | Removed |

What to do: Remove `scripts_disabled` from your YAML. To keep scripts off, deploy fleetd without `--enable-scripts`.

Issue: [#53897](https://github.com/fleetdm/fleet/issues/53897)


### Package YAML files must be lists

#### YAML (GitOps)

| Fleet 4 | Fleet 5 |
|:--|:--|
| A package YAML file can hold one package that isn't in a list | Packages must be in a list ([docs](https://fleetdm.com/docs/configuration/yaml-files#packages)). `fleetctl gitops` fails otherwise. |

What to do: In each package YAML file, put packages in a list, even if there's only one.

Issue: [#53378](https://github.com/fleetdm/fleet/issues/53378)


### Certificate authority names must be unique

#### REST API, YAML (GitOps), and Fleet UI

| Fleet 4 | Fleet 5 |
|:--|:--|
| Certificate authorities (CAs) of different types can share a name | Every CA needs a unique name ([API docs](https://fleetdm.com/docs/rest-api/rest-api#connect-certificate-authority-ca), [YAML docs](https://fleetdm.com/docs/configuration/yaml-files#certificate-authorities)) |

What to do: Before you upgrade, call [List certificate authorities](https://fleetdm.com/docs/rest-api/rest-api#list-certificate-authorities-cas), find CAs that share a `name`, and rename them. Update the configuration profiles that reference them.

Issue: [#56271](https://github.com/fleetdm/fleet/issues/56271)


### Fleet names must be unique

#### REST API and Fleet UI

| Fleet 4 | Fleet 5 |
|:--|:--|
| Fleets can have names that differ only by case or special characters | Names must be unique after Fleet trims whitespace, lowercases the name, and removes special characters ([docs](https://fleetdm.com/docs/rest-api/rest-api#create-fleet)). GitOps already blocks this. |

What to do: Run `fleetctl get fleets`, find names that match after you ignore case, whitespace, and special characters (for example `Workstations` and `workstations!`), and rename one in each pair.

Issue: [#53790](https://github.com/fleetdm/fleet/issues/53790)


### Packs removed

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `/api/v1/fleet/packs` endpoints, including the packs spec endpoints | Removed. Use [reports](https://fleetdm.com/docs/rest-api/rest-api#create-report). |

#### fleetctl

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get packs` and `fleetctl apply` with `kind: pack` | Removed |
| `fleetctl upgrade-packs` | Removed |

#### Activities

| Fleet 4 | Fleet 5 |
|:--|:--|
| `created_pack`, `edited_pack`, `deleted_pack`, and `applied_spec_pack` | Fleet 5 keeps existing activities with these types but doesn't create new ones |

What to do: Before you upgrade, run `fleetctl upgrade-packs -o packs-to-reports.yml` on Fleet 4, review the file, and apply it with `fleetctl apply -f packs-to-reports.yml`. 2017 packs stop running on hosts after the upgrade.

Issue: [#53470](https://github.com/fleetdm/fleet/issues/53470)


### Disk encryption activity types merged

#### Activities

| Fleet 4 | Fleet 5 |
|:--|:--|
| `enabled_macos_disk_encryption` and `disabled_macos_disk_encryption` | `edited_disk_encryption_settings` ([docs](https://fleetdm.com/docs/api/global-audit-logs)) |

What to do: Update anything that filters or parses activities, like SIEM rules, to use the new type.

Issue: [#53481](https://github.com/fleetdm/fleet/issues/53481)


### fleetd process name is Fleet

#### fleetd

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleet-osquery` (Linux), `Fleet osquery` (Windows), and `orbit` (macOS) | `Fleet` on all platforms |

What to do: Add `Fleet` to EDR, monitoring, and firewall rules that match the old names. Keep the old names until every host runs the new fleetd.

Issue: [#56803](https://github.com/fleetdm/fleet/issues/56803)


### fleetctl package file names changed

#### fleetctl

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl package` file names like `fleet-osquery_1.61.0_amd64.deb` | `fleet-{platform}{-arch}-{fleet-name}.ext`, for example `fleet-macos-workstations.pkg`, or `fleet-windows-amd64.msi` when there's no fleet name. Global enroll secrets use `unassigned`. ([docs](https://fleetdm.com/guides/enroll-hosts)) |

What to do: Update scripts and CI that use the file names. `fleetctl package` leaves out the fleet name if it can't find it, for example when you're not logged in to `fleetctl`.

Issue: [#53365](https://github.com/fleetdm/fleet/issues/53365)


### Static SCEP challenge off by default

#### Fleet server configuration

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.apple_scep_static_challenge_enabled` defaults to `true` | Defaults to `false` ([docs](https://fleetdm.com/docs/configuration/fleet-server-configuration#mobile-device-management-mdm)) |

#### REST API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/enrollment_profiles/manual` works | [Get manual enrollment profile](https://fleetdm.com/docs/rest-api/rest-api#get-manual-enrollment-profile) returns a `400` unless `mdm.apple_scep_static_challenge_enabled` is `true` |

What to do: If you use the manual enrollment profile endpoint, set `mdm.apple_scep_static_challenge_enabled` (`FLEET_MDM_APPLE_SCEP_STATIC_CHALLENGE_ENABLED`) to `true`. That turns the static SCEP challenge back on for all Apple MDM enrollments, so anyone with the challenge can turn on MDM for any host. Previously downloaded profiles stop working. Hosts that already have MDM on aren't affected.

Issue: [#54190](https://github.com/fleetdm/fleet/issues/54190)


### Async host processing removed

#### Fleet server configuration

| Fleet 4 | Fleet 5 |
|:--|:--|
| `osquery_enable_async_host_processing` (`FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING`) | Removed. Fleet always processes label membership, policy membership, last seen, and scheduled report stats synchronously. |
| `osquery_async_host_collect_interval`, `osquery_async_host_collect_max_jitter_percent`, `osquery_async_host_collect_lock_timeout`, `osquery_async_host_collect_log_stats_interval`, `osquery_async_host_insert_batch`, `osquery_async_host_delete_batch`, `osquery_async_host_update_batch`, `osquery_async_host_redis_pop_count`, and `osquery_async_host_redis_scan_keys_count` (and their `FLEET_OSQUERY_*` environment variables) | Removed |

What to do: Remove these options from your server configuration. Data still waiting in Redis when you upgrade is dropped and corrected on the next check-in.

Issue: [#54009](https://github.com/fleetdm/fleet/issues/54009)


### osquery_min_software_last_opened_at_diff removed

#### Fleet server configuration

| Fleet 4 | Fleet 5 |
|:--|:--|
| `osquery_min_software_last_opened_at_diff` (`FLEET_OSQUERY_MIN_SOFTWARE_LAST_OPENED_AT_DIFF`) | Removed |

What to do: Remove it from your server configuration.

Issue: [#53787](https://github.com/fleetdm/fleet/issues/53787)


<meta name="articleTitle" value="Migrate from Fleet 4 to Fleet 5">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="publishedOn" value="2026-10-30">
<meta name="category" value="guides">
<meta name="description" value="Prepare for the upgrade to Fleet 5.">

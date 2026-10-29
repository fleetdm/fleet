# Migrate from Fleet 4 to Fleet 5

Fleet 5 removes deprecated features such as API endpoints, API parameters, and YAML (GitOps) keys. This guide lists each breaking change and the steps to update your setup, so you can upgrade without issues.

Fleet will support Fleet 4.95 with security fixes through June 30, 2027.


## Prerequisites

Check these before you start:

- You're running the latest version of Fleet 4. Update to it first, then follow the steps below.
- For SSO changes, you have access to your identity provider (IdP).


## Farewell, teams

You'll be missed, and we'll keep saying "teams" out of muscle memory.

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| `POST /api/v1/fleet/spec/teams` | `POST /api/v1/fleet/spec/fleets` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `POST /api/v1/fleet/teams` | `POST /api/v1/fleet/fleets` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `GET /api/v1/fleet/teams` | `GET /api/v1/fleet/fleets` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `GET /api/v1/fleet/teams/:id` | `GET /api/v1/fleet/fleets/:id` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `PATCH /api/v1/fleet/teams/:id` | `PATCH /api/v1/fleet/fleets/:id` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `DELETE /api/v1/fleet/teams/:id` | `DELETE /api/v1/fleet/fleets/:id` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `PATCH /api/v1/fleet/teams/:id/secrets` | `PATCH /api/v1/fleet/fleets/:id/secrets` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `GET /api/v1/fleet/teams/:id/secrets` | `GET /api/v1/fleet/fleets/:id/secrets` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `POST /api/v1/fleet/teams/:id/agent_options` | `POST /api/v1/fleet/fleets/:id/agent_options` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `GET /api/v1/fleet/teams/:id/users` | `GET /api/v1/fleet/fleets/:id/users` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `PATCH /api/v1/fleet/teams/:id/users` | `PATCH /api/v1/fleet/fleets/:id/users` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `DELETE /api/v1/fleet/teams/:id/users` | `DELETE /api/v1/fleet/fleets/:id/users` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `PATCH /api/v1/fleet/ab_tokens/:id/teams` | `PATCH /api/v1/fleet/ab_tokens/:id/fleets` | `/api/v1/fleet/abm_tokens/:id/teams` and `/api/v1/fleet/abm_tokens/:id/fleets` are also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `PATCH /api/v1/fleet/vpp_tokens/:id/teams` | `PATCH /api/v1/fleet/vpp_tokens/:id/fleets` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `team_id` | `fleet_id` | Applies to REST API requests, responses, and query parameters. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `team_name` | `fleet_name` | Applies to REST API requests, responses, and query parameters. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `team` | `fleet` | Applies to REST API requests and responses. Includes the `team` target type. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `teams` | `fleets` | Applies to REST API requests and responses. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `team_ids` | `fleet_ids` | Applies to REST API requests and responses. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `team_ids_by_name` | `fleet_ids_by_name` | Applies to REST API requests and responses. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `available_teams` | `available_fleets` | Applies to REST API requests and responses. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `default_team` | `default_fleet` | Applies to Apple Business (AB) settings. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `macos_team`, `ios_team`, `ipados_team`, and `byod_team` | `macos_fleet`, `ios_fleet`, `ipados_fleet`, and `byod_fleet` | Applies to AB settings. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `macos_team_id`, `ios_team_id`, `ipados_team_id`, and `byod_team_id` | `macos_fleet_id`, `ios_fleet_id`, `ipados_fleet_id`, and `byod_fleet_id` | Used when you assign fleets to an AB token. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `team_id` and `team_name` as `order_key` values | `fleet_id` and `fleet_name` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `team_id` and `team_name` columns in the hosts CSV | `fleet_id` and `fleet_name` columns |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `fleetctl get teams` | `fleetctl get fleets` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `fleetctl gitops --delete-other-teams` | `fleetctl gitops --delete-other-fleets` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `fleetctl apply --policies-team` | `fleetctl apply --policies-fleet` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `--team` on `fleetctl report`, `user create`, `hosts transfer`, `run-script`, `get reports`, `get labels`, `get hosts`, and `get software` | `--fleet` on the same commands |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `fleetctl generate-gitops --team no-team` | `fleetctl generate-gitops --fleet unassigned` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| Activity types `created_team`, `deleted_team`, `applied_spec_team`, `changed_user_team_role`, and `deleted_user_team_role` | `created_fleet`, `deleted_fleet`, `applied_spec_fleet`, `changed_user_fleet_role`, and `deleted_user_fleet_role` | Update anything that filters or parses activities by type. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `FLEET_JIT_USER_ROLE_TEAM_<ID>` SAML attribute | `FLEET_JIT_USER_ROLE_FLEET_<ID>` | Rename it in your IdP. | [#53925](https://github.com/fleetdm/fleet/issues/53925) |

> **Warning:** Fleet 5 ignores `FLEET_JIT_USER_ROLE_TEAM_<ID>`. If you don't rename it, new users who log in with SSO get the global observer role, and existing users keep the roles they have now.


## REST API

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| `GET /api/v1/fleet/hosts/identifier/:identifier` | `GET /api/v1/fleet/hosts_identifier/:identifier` |  | [#53080](https://github.com/fleetdm/fleet/issues/53080) |
| `GET /api/v1/fleet/scripts/batch/summary/:batch_execution_id` | `GET /api/v1/fleet/scripts/batch/:batch_execution_id` |  | [#53080](https://github.com/fleetdm/fleet/issues/53080) |
| `POST /api/v1/fleet/hosts/:host_id/configuration_profiles/resend/:profile_uuid` | `POST /api/v1/fleet/hosts/:id/configuration_profiles/:profile_uuid/resend` |  | [#53080](https://github.com/fleetdm/fleet/issues/53080) |
| `os_name` and `os_version` combined in responses | `os_name` and `os_version` as separate keys | Applies to [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts), `GET /api/v1/fleet/hosts`, and [Get hosts report in CSV](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-report-in-csv), `GET /api/v1/fleet/hosts/report`. | [#53877](https://github.com/fleetdm/fleet/issues/53877) |
| `require_all_software_macos` | `macos_require_all_software` | Applies to the request body of [Update setup experience](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience), `PATCH /api/v1/fleet/setup_experience`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `require_all_software_windows` | `windows_require_all_software` | Applies to the request body of [Update setup experience](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience), `PATCH /api/v1/fleet/setup_experience`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `enable_recovery_lock_password` | `macos_enable_recovery_lock_password` | Applies to the `mdm` settings in the request and response of [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration), `PATCH /api/v1/fleet/config`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `all_linux_count` in the [Get hosts summary](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-summary) response | The [Count hosts](https://fleetdm.com/docs/rest-api/rest-api#count-hosts) endpoint with `label_id` set to the ID of the built-in "All Linux" label | You can find the ID in `builtin_labels` in the [Get hosts summary](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-summary) response. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `org_logo_url` | `org_logo_url_dark_mode` | Applies to API requests and responses. | [#52800](https://github.com/fleetdm/fleet/issues/52800) |
| `org_logo_url_light_background` | `org_logo_url_light_mode` | Applies to API requests and responses. | [#52800](https://github.com/fleetdm/fleet/issues/52800) |
| `windows` object with `wipe_type` in the request body of [Wipe host](https://fleetdm.com/docs/rest-api/rest-api#wipe-host), `POST /api/v1/fleet/hosts/:id/wipe` | Removed. Use `POST /api/v1/fleet/hosts/:id/reset` instead. |  | [#55003](https://github.com/fleetdm/fleet/issues/55003) |
| `mdm.apple_settings.configuration_profiles`, `mdm.windows_settings.configuration_profiles`, `mdm.setup_experience`, `mdm.volume_purchasing_program`, and `scripts` in the responses of [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) and [Get fleet](https://fleetdm.com/docs/rest-api/rest-api#get-fleet) | Removed | These lists only included settings applied with YAML. Use the list endpoints for each feature instead. | [#53932](https://github.com/fleetdm/fleet/issues/53932) |
| `features.detail_query_overrides` | Removed | Applies to the REST API and YAML. | [#33769](https://github.com/fleetdm/fleet/issues/33769) |
| `GET /api/v1/fleet/global/policies` | `GET /api/v1/fleet/policies` | Returns global policies when `fleet_id` is omitted. | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `GET /api/v1/fleet/fleets/:id/policies` | `GET /api/v1/fleet/policies?fleet_id=:id` | The `/team/:id/policies` and `/teams/:id/policies` paths are also removed. | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `GET /api/v1/fleet/global/policies/:id` | `GET /api/v1/fleet/policies/:id` |  | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `GET /api/v1/fleet/fleets/:fleet_id/policies/:policy_id` | `GET /api/v1/fleet/policies/:id` | The `/team/...` and `/teams/...` paths are also removed. | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `POST /api/v1/fleet/global/policies` | `POST /api/v1/fleet/policies` |  | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `POST /api/v1/fleet/fleets/:id/policies` | `POST /api/v1/fleet/policies` | Pass `fleet_id` in the request body. The `/team/:id/policies` and `/teams/:id/policies` paths are also removed. | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `PATCH /api/v1/fleet/global/policies/:id` | `PATCH /api/v1/fleet/policies/:id` |  | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `PATCH /api/v1/fleet/fleets/:fleet_id/policies/:policy_id` | `PATCH /api/v1/fleet/policies/:id` | The `/teams/...` path is also removed. | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `POST /api/v1/fleet/global/policies/delete` | `POST /api/v1/fleet/policies/delete` |  | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `POST /api/v1/fleet/fleets/:fleet_id/policies/delete` | `POST /api/v1/fleet/policies/delete` | The `/team/...` and `/teams/...` paths are also removed. | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `failing_policies_webhook.policy_ids` | `webhooks_and_tickets_enabled` on each policy | Applies to the REST API and YAML. | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| A policy with more than one automation (`install_software`, `run_script`, `resend_configuration_profile`, `calendar_events_enabled`, `conditional_access_enabled`, or `webhooks_and_tickets_enabled`) | One automation per policy | `fleetctl gitops` fails if a policy has more than one. Update these policies before you upgrade. | [#52900](https://github.com/fleetdm/fleet/issues/52900) |
| `GET /api/v1/fleet/global/schedule` | `GET /api/v1/fleet/reports` | Scheduled reports have an `interval` greater than `0`. The `/api/latest/fleet/schedule` path is also removed. | [#54461](https://github.com/fleetdm/fleet/issues/54461) |
| `GET /api/v1/fleet/fleets/:id/schedule` | `GET /api/v1/fleet/reports?fleet_id=:id` | Scheduled reports have an `interval` greater than `0`. The `/team/:id/schedule` and `/teams/:id/schedule` paths are also removed. | [#54461](https://github.com/fleetdm/fleet/issues/54461) |
| `POST /api/v1/fleet/global/schedule` | `POST /api/v1/fleet/reports` | Set `interval`, `platform`, `min_osquery_version`, and `logging` on the report. Fleet 4 copied the report and scheduled the copy. Fleet 5 doesn't. | [#54461](https://github.com/fleetdm/fleet/issues/54461) |
| `POST /api/v1/fleet/fleets/:id/schedule` | `POST /api/v1/fleet/reports` | Set `fleet_id`, `interval`, `platform`, `min_osquery_version`, and `logging` on the report. Fleet 4 copied the report and scheduled the copy. Fleet 5 doesn't. | [#54461](https://github.com/fleetdm/fleet/issues/54461) |
| `PATCH /api/v1/fleet/global/schedule/:id` | `PATCH /api/v1/fleet/reports/:id` |  | [#54461](https://github.com/fleetdm/fleet/issues/54461) |
| `PATCH /api/v1/fleet/fleets/:id/schedule/:report_id` | `PATCH /api/v1/fleet/reports/:id` |  | [#54461](https://github.com/fleetdm/fleet/issues/54461) |
| `DELETE /api/v1/fleet/global/schedule/:id` | `DELETE /api/v1/fleet/reports/:id` | To unschedule a report and keep it, send `PATCH` with `interval` set to `0`. | [#54461](https://github.com/fleetdm/fleet/issues/54461) |
| `DELETE /api/v1/fleet/fleets/:id/schedule/:report_id` | `DELETE /api/v1/fleet/reports/:id` | To unschedule a report and keep it, send `PATCH` with `interval` set to `0`. | [#54461](https://github.com/fleetdm/fleet/issues/54461) |
| `POST /api/v1/fleet/reports` with built-in labels in `labels_include_any` or `labels_include_all` | Returns an error | Use platform targeting instead. | [#53800](https://github.com/fleetdm/fleet/issues/53800) |
| `GET /api/v1/fleet/mdm/commands` | `GET /api/v1/fleet/commands` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/commandresults` | `GET /api/v1/fleet/commands/results` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `POST /api/v1/fleet/mdm/commands/run` | `POST /api/v1/fleet/commands/run` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/commands` without `host_identifier` | `host_identifier` is required | `fleetctl get mdm-commands` requires `--host`. | [#53146](https://github.com/fleetdm/fleet/issues/53146) |
| `PATCH /api/v1/fleet/mdm/hosts/:id/unenroll` | `PATCH /api/v1/fleet/hosts/:id/mdm` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `POST /api/v1/fleet/mdm/hosts/:id/lock` | `POST /api/v1/fleet/hosts/:id/lock` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `POST /api/v1/fleet/mdm/hosts/:id/wipe` | `POST /api/v1/fleet/hosts/:id/wipe` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/hosts/:id/profiles` | `GET /api/v1/fleet/hosts/:id/configuration_profiles` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/hosts/:id/encryption_key` | `GET /api/v1/fleet/hosts/:id/encryption_key` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/disk_encryption/summary` | `GET /api/v1/fleet/disk_encryption` | The `/mdm/apple/filevault/summary` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/profiles` | `GET /api/v1/fleet/configuration_profiles` | The `/mdm/apple/profiles` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `POST /api/v1/fleet/mdm/profiles` | `POST /api/v1/fleet/configuration_profiles` | The `/mdm/apple/profiles` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/profiles/:profile_uuid` | `GET /api/v1/fleet/configuration_profiles/:profile_uuid` | The `/mdm/apple/profiles/:profile_id` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `DELETE /api/v1/fleet/mdm/profiles/:profile_uuid` | `DELETE /api/v1/fleet/configuration_profiles/:profile_uuid` | The `/mdm/apple/profiles/:profile_id` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/profiles/summary` | `GET /api/v1/fleet/configuration_profiles/summary` | The `/mdm/apple/profiles/summary` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/apple/commands` | `GET /api/v1/fleet/commands` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `PATCH /api/v1/fleet/mdm/apple/setup` | `PATCH /api/v1/fleet/setup_experience` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/apple` | `GET /api/v1/fleet/apns` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `POST /api/v1/fleet/mdm/apple/enrollment_profile` | `POST /api/v1/fleet/enrollment_profiles/automatic` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/apple/enrollment_profile` | `GET /api/v1/fleet/enrollment_profiles/automatic` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `DELETE /api/v1/fleet/mdm/apple/enrollment_profile` | `DELETE /api/v1/fleet/enrollment_profiles/automatic` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/manual_enrollment_profile` | `GET /api/v1/fleet/enrollment_profiles/manual` |  | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `POST /api/v1/fleet/mdm/bootstrap` | `POST /api/v1/fleet/bootstrap` | The `/mdm/apple/bootstrap` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/bootstrap/:fleet_id/metadata` | `GET /api/v1/fleet/bootstrap/:fleet_id/metadata` | The `/mdm/apple/bootstrap/:fleet_id/metadata` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `DELETE /api/v1/fleet/mdm/bootstrap/:fleet_id` | `DELETE /api/v1/fleet/bootstrap/:fleet_id` | The `/mdm/apple/bootstrap/:fleet_id` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/bootstrap/summary` | `GET /api/v1/fleet/bootstrap/summary` | The `/mdm/apple/bootstrap/summary` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `POST /api/v1/fleet/mdm/setup/eula` | `POST /api/v1/fleet/setup_experience/eula` | The `/mdm/apple/setup/eula` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `GET /api/v1/fleet/mdm/setup/eula/metadata` | `GET /api/v1/fleet/setup_experience/eula/metadata` | The `/mdm/apple/setup/eula/metadata` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `DELETE /api/v1/fleet/mdm/setup/eula/:token` | `DELETE /api/v1/fleet/setup_experience/eula/:token` | The `/mdm/apple/setup/eula/:token` path is also removed. | [#53563](https://github.com/fleetdm/fleet/issues/53563) |
| `mdm.macos_setup` in `GET` and `PATCH` `/api/v1/fleet/config` | `mdm.setup_experience` | Applies to organization and fleet settings. In Fleet 5, `macos_setup` is an unknown field. | [#53894](https://github.com/fleetdm/fleet/issues/53894) |
| `POST`, `GET`, and `DELETE /api/v1/fleet/setup_experience/script` | Removed | Use a script-only package instead. | [#53274](https://github.com/fleetdm/fleet/issues/53274) |
| `GET /api/v1/fleet/abm_tokens` | `GET /api/v1/fleet/ab_tokens` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `POST /api/v1/fleet/abm_tokens` | `POST /api/v1/fleet/ab_tokens` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `DELETE /api/v1/fleet/abm_tokens/:id` | `DELETE /api/v1/fleet/ab_tokens/:id` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `GET /api/v1/fleet/abm_tokens/count` | `GET /api/v1/fleet/ab_tokens/count` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `PATCH /api/v1/fleet/abm_tokens/:id/renew` | `PATCH /api/v1/fleet/ab_tokens/:id/renew` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `GET /api/v1/fleet/mdm/apple/abm_public_key` | `GET /api/v1/fleet/mdm/apple/ab_public_key` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `GET /api/v1/fleet/abm` and `GET /api/v1/fleet/mdm/apple_bm` | `GET /api/v1/fleet/ab_tokens` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `abm_token` and `abm_tokens` in responses | `ab_token` and `ab_tokens` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `host_dep_assignment.abm_token_id` | `ab_token_id` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `mdm.apple_business_manager` in responses | `mdm.apple_business` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `mdm.apple_bm_default_team` in requests | Removed |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `mdm.apple_bm_enabled_and_configured` in Get configuration | `mdm.apple_business_enabled_and_configured` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `mdm.apple_bm_terms_expired` in Get configuration | `mdm.apple_business_terms_expired` |  | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| Lock host (`POST /api/v1/fleet/hosts/:id/lock`) locks the screen of an Android host | Lock host resets the PIN to a random six-digit PIN and locks the host | To get the PIN, use Unlock host (`POST /api/v1/fleet/hosts/:id/unlock`) or `fleetctl mdm unlock --host=<host>`. Both now work for Android hosts. | [#49909](https://github.com/fleetdm/fleet/issues/49909) |
| `DELETE /api/v1/fleet/scim/Users/:id` with a maintainer's API token succeeds | Returns an error | Use an admin's API token instead. | [#53472](https://github.com/fleetdm/fleet/issues/53472) |
| `software_id` filter on `GET /api/v1/fleet/hosts` | `software_version_id` or `software_title_id` |  | [#54410](https://github.com/fleetdm/fleet/issues/54410) |
| `software_package` object in software title responses | `packages` array | `fleetctl get software --json` and `--yaml` output also no longer includes `software_package`. | [#54409](https://github.com/fleetdm/fleet/issues/54409) |
| `app_store_app` object in software title responses | `app_store_apps` array | `fleetctl get software --json` and `--yaml` output also no longer includes `app_store_app`. | [#52687](https://github.com/fleetdm/fleet/issues/52687) |
| `browser` | `extension_for` | Applies to host, software title, and software version responses. `fleetctl get software --json` and `--yaml` output also no longer includes `browser`. | [#53395](https://github.com/fleetdm/fleet/issues/53395) |
| `GET /api/v1/fleet/software` | `GET /api/v1/fleet/software/versions` |  | [#54408](https://github.com/fleetdm/fleet/issues/54408) |
| `GET /api/v1/fleet/software/:id` | `GET /api/v1/fleet/software/versions/:id` |  | [#54408](https://github.com/fleetdm/fleet/issues/54408) |
| `GET /api/v1/fleet/software/count` | `GET /api/v1/fleet/software/versions` | Fleet's dashboard now uses this endpoint for counts. | [#54408](https://github.com/fleetdm/fleet/issues/54408) |
| `centos` and `ubuntu` for label platform targeting | `linux` |  | [#52905](https://github.com/fleetdm/fleet/issues/52905) |
| `uuid` and `hardware_serial` for personal (BYOD) hosts | `uuid` and `hardware_serial` are empty strings. `enrollment_id` has the enrollment ID | Applies to iOS and iPadOS User Enrollment and Android work profile hosts. | [#53797](https://github.com/fleetdm/fleet/issues/53797) |
| `uuid` for company-owned Android hosts | `uuid` is an empty string | Fleet 4 returned a hash Fleet generates from the brand and serial. `hardware_serial` doesn't change, and `enrollment_id` is `null`. | [#53797](https://github.com/fleetdm/fleet/issues/53797) |

## YAML

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| `require_all_software_macos` | `macos_require_all_software` | Under `controls.setup_experience`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `require_all_software_windows` | `windows_require_all_software` | Under `controls.setup_experience`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `enable_recovery_lock_password` | `macos_enable_recovery_lock_password` | Under `controls`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `org_logo_url` | `org_logo_url_dark_mode` | Under `org_info`. | [#52800](https://github.com/fleetdm/fleet/issues/52800) |
| `org_logo_url_light_background` | `org_logo_url_light_mode` | Under `org_info`. | [#52800](https://github.com/fleetdm/fleet/issues/52800) |
| `controls.macos_setup` in YAML | `controls.setup_experience` | `fleetctl apply` fails with an unknown-field error. `fleetctl get config` no longer prints `macos_setup`. | [#53894](https://github.com/fleetdm/fleet/issues/53894) |
| `controls.setup_experience.macos_script` in YAML | A script-only package under `software.packages` with `setup_experience` or `setup_experience_platform` | `fleetctl gitops` no longer accepts `macos_script`. The **Run script** tab in **Controls > Setup experience** is hidden. | [#53274](https://github.com/fleetdm/fleet/issues/53274) |
| `org_settings.mdm.apple_business_manager` in YAML | `org_settings.mdm.apple_business` | `fleetctl gitops` fails with an error that points to `apple_business`. | [#53272](https://github.com/fleetdm/fleet/issues/53272) |


## GitOps

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| GitOps users can run custom MDM commands | GitOps users get a 403 | Applies to the [Run MDM command](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command) endpoint, `POST /api/v1/fleet/commands/run`. | [#53145](https://github.com/fleetdm/fleet/issues/53145) |
| Omitted `gitops`, `host_expiry_settings`, and `smtp_settings` keep their current values | Omitted settings reset to their defaults | Add these settings to your YAML to keep their current values. | [#53899](https://github.com/fleetdm/fleet/issues/53899) |
| `hostname` as a host identifier for manual labels | Not accepted | Applies to the `hosts` key of labels in GitOps and the REST API. | [#53924](https://github.com/fleetdm/fleet/issues/53924) |
| `server_settings.scripts_disabled` | `fleetctl gitops` fails if it's set | Fleet 5 turns script features back on for instances that had `scripts_disabled: true`. To keep scripts off, deploy fleetd without scripts enabled. The **Disable scripts** checkbox in **Settings > Organization settings > Advanced options** is also removed. | [#53897](https://github.com/fleetdm/fleet/issues/53897) |
| A package YAML file with one package that isn't in a list | A list of packages | `fleetctl gitops` fails if the package isn't in a list. | [#53378](https://github.com/fleetdm/fleet/issues/53378) |


## Certificate authorities (CAs)

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| CAs of different types can share a name | Every CA needs a unique name | Rename any CAs that share a name before you upgrade. | [#56271](https://github.com/fleetdm/fleet/issues/56271) |


## Fleet agent

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| Process name `fleet-osquery` (Linux), `Fleet osquery` (Windows), and `orbit` (macOS) | `Fleet` on all platforms | Update any rules that match the old names. | [#56803](https://github.com/fleetdm/fleet/issues/56803) |


## fleetctl

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| `fleetctl package` file names like `fleet-osquery_1.61.0_amd64.deb` | `fleet-{platform}{-arch}-{fleet-name}.ext`, for example `fleet-macos-workstations.pkg`, or `fleet-windows-amd64.msi` when there's no fleet name | Update scripts that use the file names. `fleetctl package` leaves out the fleet name if it can't find it, for example when you're not logged in to `fleetctl`. Global enroll secrets use `unassigned`. | [#53365](https://github.com/fleetdm/fleet/issues/53365) |
| `fleetctl get mdm-apple-bm` | `fleetctl get mdm-ab` | The `mdm_apple_bm` alias is also removed. | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `fleetctl generate mdm-apple-bm` | `fleetctl generate mdm-ab` | The `mdm_apple_bm` alias is also removed. | [#53272](https://github.com/fleetdm/fleet/issues/53272) |


## Fleet names

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| Fleets can have names that differ only by case or special characters | Fleet names must be unique after Fleet trims whitespace, lowercases the name, and removes special characters | Applies when you create or edit a fleet in the UI or API. GitOps already blocks this. | [#53790](https://github.com/fleetdm/fleet/issues/53790) |


## Packs

Packs are removed in Fleet 5. Use reports instead.

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| `/packs` endpoints, including the packs spec endpoints | Removed | They return 404. Before you upgrade, run `fleetctl upgrade-packs` on Fleet 4 to convert your 2017 packs into reports, then apply the output. 2017 packs stop running on hosts after the upgrade. | [#53470](https://github.com/fleetdm/fleet/issues/53470) |
| `fleetctl get packs` | Removed |  | [#53470](https://github.com/fleetdm/fleet/issues/53470) |
| `fleetctl upgrade-packs` | Removed | Run it on Fleet 4 before you upgrade. | [#53470](https://github.com/fleetdm/fleet/issues/53470) |
| `fleetctl apply` with `kind: pack` | Removed |  | [#53470](https://github.com/fleetdm/fleet/issues/53470) |


## Activities

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| `enabled_macos_disk_encryption` and `disabled_macos_disk_encryption` | `edited_disk_encryption_settings` |  | [#53481](https://github.com/fleetdm/fleet/issues/53481) |
| `created_setup_experience_script` and `deleted_setup_experience_script` | Removed |  | [#53274](https://github.com/fleetdm/fleet/issues/53274) |
| `created_pack`, `edited_pack`, `deleted_pack`, and `applied_spec_pack` | Removed | Fleet 5 keeps existing activities with these types but doesn't create new ones. | [#53470](https://github.com/fleetdm/fleet/issues/53470) |


## Fleet server configuration

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| `mdm.apple_scep_static_challenge_enabled` defaults to `true`, and the manual enrollment profile endpoint works | `mdm.apple_scep_static_challenge_enabled` defaults to `false`, and the endpoint returns a 400 | To keep the endpoint working, set `mdm.apple_scep_static_challenge_enabled` (`FLEET_MDM_APPLE_SCEP_STATIC_CHALLENGE_ENABLED`) to `true`. That turns the static SCEP challenge back on for all Apple MDM enrollments, so anyone with the challenge can turn on MDM for any host. Previously downloaded profiles stop working. Hosts that already have MDM on aren't affected. | [#54190](https://github.com/fleetdm/fleet/issues/54190) |
| `osquery_enable_async_host_processing` (`FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING`) | Removed | Fleet always processes label membership, policy membership, last seen, and scheduled report stats synchronously. Data still waiting in Redis when you upgrade is dropped and corrected on the next check-in. | [#54009](https://github.com/fleetdm/fleet/issues/54009) |
| `osquery_async_host_collect_interval`, `osquery_async_host_collect_max_jitter_percent`, `osquery_async_host_collect_lock_timeout`, `osquery_async_host_collect_log_stats_interval`, `osquery_async_host_insert_batch`, `osquery_async_host_delete_batch`, `osquery_async_host_update_batch`, `osquery_async_host_redis_pop_count`, and `osquery_async_host_redis_scan_keys_count` | Removed | The matching `FLEET_OSQUERY_*` environment variables are also removed. | [#54009](https://github.com/fleetdm/fleet/issues/54009) |
| `osquery_min_software_last_opened_at_diff` (`FLEET_OSQUERY_MIN_SOFTWARE_LAST_OPENED_AT_DIFF`) | Removed |  | [#53787](https://github.com/fleetdm/fleet/issues/53787) |
| `mdm.apple_bm_server_token` and `mdm.apple_bm_server_token_bytes` | `mdm.apple_business_server_token` and `mdm.apple_business_server_token_bytes` | Fleet fails at startup if an old key is still set. | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `mdm.apple_bm_cert` and `mdm.apple_bm_cert_bytes` | `mdm.apple_business_cert` and `mdm.apple_business_cert_bytes` | Fleet fails at startup if an old key is still set. | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `mdm.apple_bm_key` and `mdm.apple_bm_key_bytes` | `mdm.apple_business_key` and `mdm.apple_business_key_bytes` | Fleet fails at startup if an old key is still set. | [#53272](https://github.com/fleetdm/fleet/issues/53272) |
| `FLEET_MDM_APPLE_BM_*` environment variables | `FLEET_MDM_APPLE_BUSINESS_*` | Fleet fails at startup if an old variable is still set. | [#53272](https://github.com/fleetdm/fleet/issues/53272) |


<meta name="articleTitle" value="Migrate from Fleet 4 to Fleet 5">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="publishedOn" value="2026-10-30">
<meta name="category" value="guides">
<meta name="description" value="Prepare for the upgrade to Fleet 5.">

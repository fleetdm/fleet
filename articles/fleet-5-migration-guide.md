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
| `POST /api/v1/fleet/teams/:id/policies` | `POST /api/v1/fleet/fleets/:id/policies` | `/api/v1/fleet/team/:id/policies` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `GET /api/v1/fleet/teams/:id/policies` | `GET /api/v1/fleet/fleets/:id/policies` | `/api/v1/fleet/team/:id/policies` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `GET /api/v1/fleet/teams/:id/policies/count` | `GET /api/v1/fleet/fleets/:id/policies/count` | `/api/v1/fleet/team/:id/policies/count` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `GET /api/v1/fleet/teams/:id/policies/:policy_id` | `GET /api/v1/fleet/fleets/:id/policies/:policy_id` | `/api/v1/fleet/team/:id/policies/:policy_id` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `POST /api/v1/fleet/teams/:id/policies/delete` | `POST /api/v1/fleet/fleets/:id/policies/delete` | `/api/v1/fleet/team/:id/policies/delete` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `PATCH /api/v1/fleet/teams/:id/policies/:policy_id` | `PATCH /api/v1/fleet/fleets/:id/policies/:policy_id` |  | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `GET /api/v1/fleet/teams/:id/schedule` | `GET /api/v1/fleet/fleets/:id/schedule` | `/api/v1/fleet/team/:id/schedule` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `POST /api/v1/fleet/teams/:id/schedule` | `POST /api/v1/fleet/fleets/:id/schedule` | `/api/v1/fleet/team/:id/schedule` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `PATCH /api/v1/fleet/teams/:id/schedule/:report_id` | `PATCH /api/v1/fleet/fleets/:id/schedule/:report_id` | `/api/v1/fleet/team/:id/schedule/:report_id` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
| `DELETE /api/v1/fleet/teams/:id/schedule/:report_id` | `DELETE /api/v1/fleet/fleets/:id/schedule/:report_id` | `/api/v1/fleet/team/:id/schedule/:report_id` is also removed. | [#52795](https://github.com/fleetdm/fleet/issues/52795) |
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


## YAML

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| `require_all_software_macos` | `macos_require_all_software` | Under `controls.setup_experience`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `require_all_software_windows` | `windows_require_all_software` | Under `controls.setup_experience`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `enable_recovery_lock_password` | `macos_enable_recovery_lock_password` | Under `controls`. | [#53872](https://github.com/fleetdm/fleet/issues/53872) |
| `org_logo_url` | `org_logo_url_dark_mode` | Under `org_info`. | [#52800](https://github.com/fleetdm/fleet/issues/52800) |
| `org_logo_url_light_background` | `org_logo_url_light_mode` | Under `org_info`. | [#52800](https://github.com/fleetdm/fleet/issues/52800) |


## GitOps

| Fleet 4 | Fleet 5 | Notes | Issue |
|:--|:--|:--|:--|
| GitOps users can run custom MDM commands | GitOps users get a 403 | Applies to the [Run MDM command](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command) endpoint, `POST /api/v1/fleet/commands/run`. | [#53145](https://github.com/fleetdm/fleet/issues/53145) |


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


<meta name="articleTitle" value="Migrate from Fleet 4 to Fleet 5">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="publishedOn" value="2026-10-30">
<meta name="category" value="guides">
<meta name="description" value="Prepare for the upgrade to Fleet 5.">

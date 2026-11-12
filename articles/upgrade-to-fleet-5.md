# Upgrade to Fleet 5

Fleet 5 introduces a cleaner, more consistent API and GitOps (YAML) by removing deprecated API endpoints and parameters, YAML (GitOps) keys, and more. This guide lists [every breaking change](#breaking-changes) and what you need to do.

> **Note:** Fleet 5 is coming in [early December](https://github.com/fleetdm/fleet/milestone/295). This guide is published early so you can plan your upgrade.

Fleet will support Fleet 4.96 with security fixes through June 30, 2027.

#### How to upgrade

1. [Upgrade](https://fleetdm.com/docs/deploy/upgrading-fleet#upgrading-fleet) to the **latest Fleet 4 release**.
2. Work through the list of [breaking changes](#breaking-changes).
3. Back up your Fleet database. If you need to go back to Fleet 4 after upgrading, restore this backup. Fleet 4 can't read the settings Fleet 5 renames, so it would turn off features like Windows MDM.
5. Upgrade to the **latest Fleet 5 release**. Why? If you upgrade to an earlier version of Fleet 5, you'll miss database migrations included in later Fleet 4 releases (e.g. 4.96.0 or 4.96.1).

#### Why are we doing this?

Fleet follows [semantic versioning (SemVer)](https://semver.org/), which means breaking changes are only made in major version releases except in [rare cases](https://fleetdm.com/docs/deploy/upgrading-fleet#compatibility).

The last major version (Fleet 4) was released over 5 years ago (2021-06-29). Since then, the API, GitOps (YAML), and other interfaces have collected blemishes and inconsistencies. Fleet 5, among [other new features](https://fleetdm.com/announcements/roadmap-preview-october-2026), brings clear and consistent interfaces to Fleet.


## Breaking changes

Breaking changes are grouped by where you use Fleet. Read the sections that apply to you:

- [REST API](#rest-api): scripts and integrations that call Fleet's API
- [YAML (GitOps)](#yaml-gitops): your GitOps repo
- [fleetctl](#fleetctl): scripts and CI that run `fleetctl`
- [Fleet server configuration](#fleet-server-configuration): your Fleet server's configuration file and environment variables
- [Single sign-on (SSO)](#single-sign-on-sso): your identity provider (IdP)
- [Activities](#activities): SIEM rules and log pipelines that read Fleet activities
- [fleetd](#fleetd): EDR, monitoring, and firewall rules that match fleetd
- [Fleet UI](#fleet-ui)
- [Deprecated](#deprecated): features that keep working in Fleet 5 and are removed in Fleet 6


## REST API

When you switch your scripts and integrations to the new endpoints, don't forget to update what's tied to the old ones, so your automation keeps running without interruption:

- API-only user permissions: Give your [API-only users](https://fleetdm.com/guides/fleetctl#create-api-only-user) permission for the new endpoints in **Settings > Users**. Permissions for an old endpoint don't automatically cover the new endpoint.
- Request size overrides: If you set [`server.endpoint_request_size_overrides`](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-endpoint-request-size-overrides) for an old endpoint, add the same entry for its replacement.

### Farewell, teams

You'll be missed, and we'll keep saying "teams" out of muscle memory.

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /teams` | [`GET /fleets`](https://fleetdm.com/docs/rest-api/rest-api#list-fleets) |
| `GET /teams/:id` | [`GET /fleets/:id`](https://fleetdm.com/docs/rest-api/rest-api#get-fleet) |
| `POST /teams` | [`POST /fleets`](https://fleetdm.com/docs/rest-api/rest-api#create-fleet) |
| `PATCH /teams/:id` | [`PATCH /fleets/:id`](https://fleetdm.com/docs/rest-api/rest-api#update-fleet) |
| `DELETE /teams/:id` | [`DELETE /fleets/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-fleet) |
| `GET /teams/:id/secrets` | [`GET /fleets/:id/secrets`](https://fleetdm.com/docs/rest-api/rest-api#get-fleet-level-enroll-secrets) |
| `PATCH /teams/:id/secrets` | [`PATCH /fleets/:id/secrets`](https://fleetdm.com/docs/rest-api/rest-api#update-fleet-level-enroll-secrets) |
| `POST /teams/:id/agent_options` | [`POST /fleets/:id/agent_options`](https://fleetdm.com/docs/rest-api/rest-api#update-fleet-level-agent-options) |
| `PATCH /teams/:id/users` | [`PATCH /fleets/:id/users`](https://fleetdm.com/docs/rest-api/rest-api#add-users-to-fleet) |
| `GET` and `DELETE /teams/:id/users` | `GET` and `DELETE /fleets/:id/users` |
| `POST /spec/teams` | `POST /spec/fleets` |
| `PATCH /vpp_tokens/:id/teams` | `PATCH /vpp_tokens/:id/fleets` |
| `PATCH /ab_tokens/:id/teams` (and `/abm_tokens/:id/teams` and `/abm_tokens/:id/fleets`) | `PATCH /ab_tokens/:id/fleets` |
| `team_id`, `team_name`, `team`, `teams`, `team_ids`, `team_ids_by_name`, and `available_teams` in requests, responses, and query parameters | `fleet_id`, `fleet_name`, `fleet`, `fleets`, `fleet_ids`, `fleet_ids_by_name`, and `available_fleets` |
| `team_id` and `team_name` as `order_key` values | `fleet_id` and `fleet_name` |
| `team` as a target type | `fleet` |
| `team_id` and `team_name` columns in the hosts CSV | `fleet_id` and `fleet_name` columns in [Get hosts report in CSV](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-report-in-csv) |
| `macos_team_id`, `ios_team_id`, `ipados_team_id`, and `byod_team_id` when you assign fleets to an Apple Business token | `macos_fleet_id`, `ios_fleet_id`, `ipados_fleet_id`, and `byod_fleet_id` |

#### What to do

Fleet 4 already supports the `/fleets` endpoints and `fleet_id`, so change them before you upgrade.

1. Search your scripts and integrations for the old names:
   ```sh
   grep -rnE '/fleet/teams|/teams\b|team_ids?\b|team_name|available_teams|(macos|ios|ipados|byod)_team_id' .
   ```
2. Replace each match with its Fleet 5 name.

GitHub issue: [#52795](https://github.com/fleetdm/fleet/issues/52795)

### Policy endpoints merged

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /global/policies` and `GET /fleets/:id/policies` | [`GET /policies`](https://fleetdm.com/docs/rest-api/rest-api#list-policies), with `fleet_id` for a fleet's policies |
| `GET /global/policies/:id` and `GET /fleets/:fleet_id/policies/:policy_id` | [`GET /policies/:id`](https://fleetdm.com/docs/rest-api/rest-api#get-policy) |
| `POST /global/policies` and `POST /fleets/:id/policies` | [`POST /policies`](https://fleetdm.com/docs/rest-api/rest-api#create-policy), with `fleet_id` in the request body for a fleet policy |
| `PATCH /global/policies/:id` and `PATCH /fleets/:fleet_id/policies/:policy_id` | [`PATCH /policies/:id`](https://fleetdm.com/docs/rest-api/rest-api#update-policy) |
| `POST /global/policies/delete` and `POST /fleets/:fleet_id/policies/delete` | [`POST /policies/delete`](https://fleetdm.com/docs/rest-api/rest-api#delete-policies) |
| `failing_policies_webhook.policy_ids` in [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | `webhooks_and_tickets_enabled` on each policy |

The `/team/...` and `/teams/...` policy paths are also removed.

#### What to do

Fleet 4 already supports `/policies`, so change them before you upgrade.

1. Search for the old paths: `grep -rnE '/fleet/global/policies|/(teams?|fleets)/[^/]+/policies' .`
2. Use `/policies`. To work with a fleet's policies, pass `fleet_id`. Without it, you get global policies.
3. Move the IDs in `failing_policies_webhook.policy_ids` to `webhooks_and_tickets_enabled: true` on each policy.

GitHub issue: [#52900](https://github.com/fleetdm/fleet/issues/52900)

### One automation per policy

| Fleet 4 | Fleet 5 |
|:--|:--|
| A policy can have more than one of `install_software`, `run_script`, `resend_configuration_profile`, `calendar_events_enabled`, `conditional_access_enabled`, and `webhooks_and_tickets_enabled` | One automation per policy in [Create policy](https://fleetdm.com/docs/rest-api/rest-api#create-policy) and [Update policy](https://fleetdm.com/docs/rest-api/rest-api#update-policy) |

#### What to do

Change these before you upgrade.

Find policies with more than one automation and move the extra automations to new policies.

GitHub issue: [#52900](https://github.com/fleetdm/fleet/issues/52900)

### Schedule endpoints replaced by reports

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /global/schedule` and `GET /fleets/:id/schedule` | [`GET /reports`](https://fleetdm.com/docs/rest-api/rest-api#list-reports), with `fleet_id` for a fleet. Scheduled reports have an `interval` greater than `0`. |
| `POST /global/schedule` and `POST /fleets/:id/schedule`, which copied the report and scheduled the copy | [`POST /reports`](https://fleetdm.com/docs/rest-api/rest-api#create-report) with `interval`, `platform`, `min_osquery_version`, and `logging` (and `fleet_id` for a fleet). No copy. |
| `PATCH /global/schedule/:id` and `PATCH /fleets/:id/schedule/:report_id` | [`PATCH /reports/:id`](https://fleetdm.com/docs/rest-api/rest-api#update-report) |
| `DELETE /global/schedule/:id` and `DELETE /fleets/:id/schedule/:report_id` | [`DELETE /reports/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-report-by-id) |

The `/schedule`, `/team/:id/schedule`, and `/teams/:id/schedule` paths are also removed.

#### What to do

Fleet 4 already supports reports with an `interval`, so change them before you upgrade.

1. Search for the old paths: `grep -rnE '/schedule\b' .`
2. Create or update reports with an `interval` instead. To unschedule a report and keep it, [update the report](https://fleetdm.com/docs/rest-api/rest-api#update-report) with `interval` set to `0`.

GitHub issue: [#54461](https://github.com/fleetdm/fleet/issues/54461)

### Reports can't target built-in labels

| Fleet 4 | Fleet 5 |
|:--|:--|
| [`POST /reports`](https://fleetdm.com/docs/rest-api/rest-api#create-report) accepts built-in labels in `labels_include_any` or `labels_include_all` | Returns an error |

#### What to do

Change these before you upgrade.

Target the report by `platform` instead.

GitHub issue: [#53800](https://github.com/fleetdm/fleet/issues/53800)

### MDM endpoints moved out of /mdm

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /mdm/commands` and `GET /mdm/apple/commands` | [`GET /commands`](https://fleetdm.com/docs/rest-api/rest-api#list-mdm-commands). For `/mdm/apple/commands`, results have `host_uuid` instead of `device_id`. |
| `GET /mdm/commandresults` | [`GET /commands/results`](https://fleetdm.com/docs/rest-api/rest-api#get-mdm-command-results) |
| `POST /mdm/commands/run` | [`POST /commands/run`](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command) |
| `POST /mdm/apple/enqueue` | [`POST /commands/run`](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command). Send the host UUIDs in `host_uuids` instead of `device_ids`. |
| `PATCH /mdm/hosts/:id/unenroll` | [`DELETE /hosts/:id/mdm`](https://fleetdm.com/docs/rest-api/rest-api#turn-off-hosts-mdm) |
| `POST /mdm/hosts/:id/lock` | [`POST /hosts/:id/lock`](https://fleetdm.com/docs/rest-api/rest-api#lock-host). Returns a `200` with a JSON body instead of an empty `204`. |
| `POST /mdm/hosts/:id/wipe` | [`POST /hosts/:id/wipe`](https://fleetdm.com/docs/rest-api/rest-api#wipe-host). Returns a `200` with a JSON body instead of an empty `204`. |
| `GET /mdm/hosts/:id/profiles` | [`GET /hosts/:id/configuration_profiles`](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-os-settings-configuration-profile) |
| `GET /mdm/hosts/:id/encryption_key` | [`GET /hosts/:id/encryption_key`](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-disk-encryption-key) |
| `GET /mdm/disk_encryption/summary` (and `/mdm/apple/filevault/summary`) | [`GET /disk_encryption`](https://fleetdm.com/docs/rest-api/rest-api#get-disk-encryption-status). For `/mdm/apple/filevault/summary`, each status is an object with `macos`, `windows`, and `linux` counts instead of a number. |
| `PATCH /mdm/apple/settings` | [`POST /disk_encryption`](https://fleetdm.com/docs/rest-api/rest-api#update-disk-encryption) |
| `GET /mdm/profiles` (and `/mdm/apple/profiles`) | [`GET /configuration_profiles`](https://fleetdm.com/docs/rest-api/rest-api#list-configuration-profiles). For `/mdm/apple/profiles`, profiles don't have `profile_id`, and the list also includes profiles for other platforms and declaration (DDM) profiles. |
| `POST /mdm/profiles` (and `/mdm/apple/profiles`) | [`POST /configuration_profiles`](https://fleetdm.com/docs/rest-api/rest-api#create-configuration-profile). For `/mdm/apple/profiles`, the file name must end in `.mobileconfig`, and the response has `profile_uuid` instead of `profile_id`. |
| `GET /mdm/profiles/:profile_uuid` (and `/mdm/apple/profiles/:profile_id`) | [`GET /configuration_profiles/:profile_uuid`](https://fleetdm.com/docs/rest-api/rest-api#get-or-download-configuration-profile). For `/mdm/apple/profiles/:profile_id`, use the profile's UUID instead of its ID. The new endpoint returns JSON. To download the profile, add `?alt=media`. |
| `DELETE /mdm/profiles/:profile_uuid` (and `/mdm/apple/profiles/:profile_id`) | [`DELETE /configuration_profiles/:profile_uuid`](https://fleetdm.com/docs/rest-api/rest-api#delete-configuration-profile). For `/mdm/apple/profiles/:profile_id`, use the profile's UUID instead of its ID. |
| `GET /mdm/profiles/summary` (and `/mdm/apple/profiles/summary`) | [`GET /configuration_profiles/summary`](https://fleetdm.com/docs/rest-api/rest-api#get-os-settings-configuration-profiles-status). For `/mdm/apple/profiles/summary`, the counts include hosts on other platforms. |
| `POST /mdm/apple/profiles/batch` | [`POST /configuration_profiles/batch`](https://fleetdm.com/docs/rest-api/rest-api#batch-update-configuration-profiles). Send each profile as an object in `configuration_profiles` instead of a string in `profiles`. The new endpoint replaces all of the fleet's profiles, including Windows, Android, and declaration (DDM) profiles, so include them in the request. |
| `PATCH /mdm/apple/setup` | [`PATCH /setup_experience`](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience) |
| `GET /mdm/apple` | [`GET /apns`](https://fleetdm.com/docs/rest-api/rest-api#get-apple-push-notification-service-apns) |
| `POST /mdm/apple/request_csr` | Removed. To turn on Apple MDM, use `GET /mdm/apple/request_csr` and then `POST /mdm/apple/apns_certificate`, `fleetctl generate mdm-apple`, or the Fleet UI ([guide](https://fleetdm.com/guides/apple-mdm-setup#turn-on-apple-mdm)). |
| `POST /mdm/apple/dep/key_pair` | Removed. To connect Apple Business, use `GET /mdm/apple/ab_public_key` and then `POST /ab_tokens`, `fleetctl generate mdm-ab`, or the Fleet UI ([guide](https://fleetdm.com/guides/apple-mdm-setup#apple-business-ab)). |
| `POST /mdm/apple/enrollment_profile` | [`POST /enrollment_profiles/automatic`](https://fleetdm.com/docs/rest-api/rest-api#update-custom-mdm-setup-enrollment-profile) |
| `GET /mdm/apple/enrollment_profile` | [`GET /enrollment_profiles/automatic`](https://fleetdm.com/docs/rest-api/rest-api#get-custom-mdm-setup-enrollment-profile) |
| `DELETE /mdm/apple/enrollment_profile` | [`DELETE /enrollment_profiles/automatic`](https://fleetdm.com/docs/rest-api/rest-api#delete-custom-mdm-setup-enrollment-profile) |
| `GET /mdm/manual_enrollment_profile` | [`GET /enrollment_profiles/manual`](https://fleetdm.com/docs/rest-api/rest-api#get-manual-enrollment-profile) |
| `POST /mdm/bootstrap` (and `/mdm/apple/bootstrap`) | [`POST /bootstrap`](https://fleetdm.com/docs/rest-api/rest-api#create-bootstrap-package) |
| `GET /mdm/bootstrap/:fleet_id/metadata` (and `/mdm/apple/bootstrap/:fleet_id/metadata`) | [`GET /bootstrap/:fleet_id/metadata`](https://fleetdm.com/docs/rest-api/rest-api#get-bootstrap-package-metadata) |
| `DELETE /mdm/bootstrap/:fleet_id` (and `/mdm/apple/bootstrap/:fleet_id`) | [`DELETE /bootstrap/:fleet_id`](https://fleetdm.com/docs/rest-api/rest-api#delete-bootstrap-package) |
| `GET /mdm/bootstrap/summary` (and `/mdm/apple/bootstrap/summary`) | [`GET /bootstrap/summary`](https://fleetdm.com/docs/rest-api/rest-api#get-bootstrap-package-status) |
| `GET /mdm/apple/bootstrap?token=` | [`GET /bootstrap?token=`](https://fleetdm.com/docs/rest-api/rest-api#download-bootstrap-package) |
| `POST /mdm/setup/eula` (and `/mdm/apple/setup/eula`) | [`POST /setup_experience/eula`](https://fleetdm.com/docs/rest-api/rest-api#create-eula) |
| `GET /mdm/setup/eula/metadata` (and `/mdm/apple/setup/eula/metadata`) | [`GET /setup_experience/eula/metadata`](https://fleetdm.com/docs/rest-api/rest-api#get-eula-metadata) |
| `GET /mdm/setup/eula/:token` (and `/mdm/apple/setup/eula/:token`) | [`GET /setup_experience/eula/:token`](https://fleetdm.com/docs/rest-api/rest-api#download-eula) |
| `DELETE /mdm/setup/eula/:token` (and `/mdm/apple/setup/eula/:token`) | [`DELETE /setup_experience/eula/:token`](https://fleetdm.com/docs/rest-api/rest-api#delete-eula) |

#### What to do

Fleet 4 already supports the new paths, so change them before you upgrade.

1. Search for `/fleet/mdm/` in your scripts and integrations, and replace each path with its Fleet 5 path. Where a row has a note, also update the request and the code that reads the response.
2. If a proxy or firewall in front of Fleet only allows some paths, also allow `/api/*/fleet/bootstrap` and `/api/*/fleet/setup_experience/eula/*`. Hosts download the bootstrap package and end users see the EULA from these paths during setup. Keep `/api/*/fleet/mdm/bootstrap` allowed for hosts that started setup before the upgrade. Learn more about [which API endpoints to expose](https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet).

GitHub issue: [#53563](https://github.com/fleetdm/fleet/issues/53563)

### List MDM commands requires a host

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /commands` without `host_identifier` | [`GET /commands?host_identifier=<host>`](https://fleetdm.com/docs/rest-api/rest-api#list-mdm-commands). `host_identifier` is required. |

#### What to do

Fleet 4 already supports `host_identifier`, so add it before you upgrade.

Pass `host_identifier` on every call.

GitHub issue: [#53146](https://github.com/fleetdm/fleet/issues/53146)

### Other endpoints moved

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /hosts/identifier/:identifier` | [`GET /hosts/:id`](https://fleetdm.com/docs/rest-api/rest-api#get-host). `:id` also accepts a host identifier (`hostname`, `uuid`, or `hardware_serial`). |
| `POST /hosts/identifier/:identifier/query` | [`POST /hosts/:id/query`](https://fleetdm.com/docs/rest-api/rest-api#run-live-report-on-host-ad-hoc). `:id` also accepts a host identifier. |
| `GET /scripts/batch/summary/:batch_execution_id` | [`GET /scripts/batch/:batch_execution_id`](https://fleetdm.com/docs/rest-api/rest-api#get-batch-script) |
| `POST /hosts/:host_id/configuration_profiles/resend/:profile_uuid` | [`POST /hosts/:id/configuration_profiles/:profile_uuid/resend`](https://fleetdm.com/docs/rest-api/rest-api#resend-configuration-profile) |

#### What to do

Fleet 4 supports the new `scripts/batch` and `configuration_profiles/resend` paths, so change them before you upgrade. It doesn't accept an identifier in `GET /hosts/:id` or `POST /hosts/:id/query`, so you can't switch `hosts/identifier` early. Instead:

1. Before you upgrade, search for `scripts/batch/summary/` and `configuration_profiles/resend/`, and replace each path.
2. After you upgrade, search for `hosts/identifier/` and put the identifier where `:id` goes in `GET /hosts/:id` or `POST /hosts/:id/query`.
   - Until you do, `GET /hosts/identifier/:identifier` and `POST /hosts/identifier/:identifier/query` fail with a `410` error.
   - Fleet looks for a host ID first, then a serial number, UUID, and hostname. An all-digit identifier is used as a host ID if one exists, even if another host has that serial number. If no host has that ID, Fleet looks it up as a serial number, UUID, or hostname.
   - If more than one host matches an identifier, Fleet now returns the newest host instead of the oldest (the one with the lowest ID). This applies to `GET /hosts/:id`, the other `GET /hosts/:id/…` endpoints, `POST /hosts/:id/query`, and [Translate IDs](https://fleetdm.com/docs/rest-api/rest-api#translate-ids).
   - Every `GET /hosts/:id/…` endpoint accepts an identifier in `:id` too, except [Get host's device page URL](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-device-page-url).
3. After you upgrade, update conditional access setups that followed the Google or PingFederate guides to call `GET /hosts/:id`.

GitHub issues: [#53080](https://github.com/fleetdm/fleet/issues/53080), [#58203](https://github.com/fleetdm/fleet/issues/58203)

### Apple Business Manager endpoints and keys renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /abm_tokens`, `GET /abm`, and `GET /mdm/apple_bm` | [`GET /ab_tokens`](https://fleetdm.com/docs/rest-api/rest-api#list-apple-business-ab-tokens) |
| `POST /abm_tokens` | `POST /ab_tokens` |
| `DELETE /abm_tokens/:id` | `DELETE /ab_tokens/:id` |
| `GET /abm_tokens/count` | `GET /ab_tokens/count` |
| `PATCH /abm_tokens/:id/renew` | `PATCH /ab_tokens/:id/renew` |
| `GET /mdm/apple/abm_public_key` | `GET /mdm/apple/ab_public_key` |
| `abm_token`, `abm_tokens`, and `host_dep_assignment.abm_token_id` in responses | `ab_token`, `ab_tokens`, and `ab_token_id` |
| `mdm.apple_business_manager`, `mdm.apple_bm_enabled_and_configured`, and `mdm.apple_bm_terms_expired` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) | `mdm.apple_business`, `mdm.apple_business_enabled_and_configured`, and `mdm.apple_business_terms_expired` |
| `mdm.apple_bm_default_team` in [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | Removed |

#### What to do

Fleet 4 already supports the new endpoints and keys, so change them before you upgrade.

1. Search for `abm`, `apple_bm`, and `apple_business_manager` in your scripts and integrations, and replace each match with its Fleet 5 name.
   - The exceptions are `mdm.apple_business_enabled_and_configured` and `mdm.apple_business_terms_expired`, which Fleet 4 doesn't return.
2. Have your code read the new key when it's there and fall back to the old one.

GitHub issue: [#53272](https://github.com/fleetdm/fleet/issues/53272)

### mdm.macos_setup renamed to mdm.setup_experience

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.macos_setup` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration), [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration), and fleet settings | `mdm.setup_experience` |

#### What to do

Fleet 4 already supports `setup_experience`, so change it before you upgrade.

Search for `macos_setup` and rename it to `setup_experience`.

GitHub issue: [#53894](https://github.com/fleetdm/fleet/issues/53894)

### Setup experience and Recovery Lock keys renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `require_all_software_macos` and `require_all_software_windows` in [Update setup experience](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience), and under `mdm.setup_experience` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration), [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration), [Get fleet](https://fleetdm.com/docs/rest-api/rest-api#get-fleet), and [Update fleet](https://fleetdm.com/docs/rest-api/rest-api#update-fleet) | `macos_require_all_software` and `windows_require_all_software` |
| `mdm.enable_recovery_lock_password` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration), [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration), [Get fleet](https://fleetdm.com/docs/rest-api/rest-api#get-fleet), and [Update fleet](https://fleetdm.com/docs/rest-api/rest-api#update-fleet) | `mdm.macos_recovery_lock_password_enabled` |

#### What to do

Fleet 4 doesn't support the new names, so you can't change them before you upgrade.

1. Right after you upgrade, search for these keys in your scripts and integrations, and rename them.
   - Until you do, calls that use the old names fail with a `422` error like: `The "enable_recovery_lock_password" parameter is not available anymore. What to use instead: https://fleetdm.com/learn-more-about/fleet-5-upgrade`

GitHub issue: [#53872](https://github.com/fleetdm/fleet/issues/53872)

### Windows MDM keys renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.windows_enabled_and_configured` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) and [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | `mdm.windows_mdm_enabled` |
| `mdm.windows_migration_enabled` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) and [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | `mdm.windows_mdm_migration_enabled` |

#### What to do

Fleet 4 doesn't support the new names, so you can't change them before you upgrade.

1. Right after you upgrade, search for these keys in your scripts and integrations, and rename them.
   - Until you do, calls that use the old names fail with a `422` error like: `The "windows_enabled_and_configured" parameter is not available anymore. What to use instead: https://fleetdm.com/learn-more-about/fleet-5-upgrade`

GitHub issue: [#53873](https://github.com/fleetdm/fleet/issues/53873)

### Managed local account keys renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `enable_managed_local_account` and `enable_create_local_admin_account` in [Update setup experience](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience), and under `mdm.macos_setup` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration), [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration), and fleet settings | `macos_managed_local_account_enabled` in the same endpoints, under `mdm.setup_experience` |
| `end_user_local_account_type` in the same endpoints | `macos_end_user_local_account_type` |
| `mdm.windows_settings.enable_managed_local_account` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration), [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration), and fleet settings | `windows_managed_local_account_enabled` in [Update setup experience](https://fleetdm.com/docs/rest-api/rest-api#update-setup-experience), and under `mdm.setup_experience` in the same endpoints |

#### What to do

1. Search for these keys and rename them.
2. Move the Windows key from `windows_settings` to setup experience.

GitHub issue: [#50084](https://github.com/fleetdm/fleet/issues/50084)

### Organization logo keys renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `org_logo_url` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration), [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration), and [Get host by Fleet Desktop token](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-fleet-desktop-token) | `org_logo_url_dark_mode` |
| `org_logo_url_light_background` in the same endpoints | `org_logo_url_light_mode` |

#### What to do

Fleet 4 already supports the new names, so change them before you upgrade.

Rename both keys.

GitHub issue: [#52800](https://github.com/fleetdm/fleet/issues/52800)

### all_linux_count removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `all_linux_count` in [Get hosts summary](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-summary) | [`GET /hosts/count`](https://fleetdm.com/docs/rest-api/rest-api#count-hosts) with `label_id` set to the ID of the built-in "All Linux" label |

#### What to do

Fleet 4 already supports this, so change it before you upgrade.

Get the "All Linux" label's ID from `builtin_labels` in the Get hosts summary response, then call Count hosts with it.

GitHub issue: [#53872](https://github.com/fleetdm/fleet/issues/53872)

### os_name and os_version are separate

| Fleet 4 | Fleet 5 |
|:--|:--|
| `os_version` includes the OS name (for example `macOS 15.2`), and there's no `os_name` in host responses | Separate `os_name` and `os_version` keys in the hosts returned by [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts), [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host), [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier), [Get host by Fleet Desktop token](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-fleet-desktop-token), [Get host's device health report](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-device-health-report), [List label's hosts](https://fleetdm.com/docs/rest-api/rest-api#list-labels-hosts), and [Search targets](https://fleetdm.com/docs/rest-api/rest-api#search-targets), and as columns in [Get hosts report in CSV](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-report-in-csv). For example, `"os_name": "macOS"` and `"os_version": "15.2"`, or `"os_name": "Windows 11 Home Single Language 23H2"` and `"os_version": "10.0.22631.5335"`. |
| Windows OS names start with "Microsoft" (for example `Microsoft Windows 11 Pro 23H2`) in the `os_name` filter, and in `name` and `name_only` in [List operating systems](https://fleetdm.com/docs/rest-api/rest-api#list-operating-systems) and [Get operating system version](https://fleetdm.com/docs/rest-api/rest-api#get-operating-system-version) | Windows OS names don't include "Microsoft" (for example `Windows 11 Pro 23H2`), including in the `os_name` filter |

#### What to do

Fleet 4 doesn't return `os_name`, so you can't switch to it early. Instead:

1. Before you upgrade, update code that splits the OS name out of `os_version`. When a host has `os_name`, read the name from `os_name` and the version from `os_version`. When it doesn't, keep splitting `os_version`. This works on both versions.
2. After you upgrade, remove "Microsoft " from any Windows OS name you filter by or compare against.

GitHub issue: [#53877](https://github.com/fleetdm/fleet/issues/53877)

### Windows wipe options moved

| Fleet 4 | Fleet 5 |
|:--|:--|
| `windows` object with `wipe_type` in `POST /hosts/:id/wipe` | Removed. [Wipe host](https://fleetdm.com/docs/rest-api/rest-api#wipe-host) always uses `doWipeProtected` for Windows hosts |
| `wipe_type` set to `"doWipe"` | [`POST /hosts/:id/reset`](https://fleetdm.com/docs/rest-api/rest-api#reset-host) to repurpose a Windows host for another end user |

#### What to do

Fleet 4 doesn't support `POST /hosts/:id/reset`, so you can't switch to it early. Instead:

1. Before you upgrade, remove the `windows` object from any request that sends `"doWipeProtected"`.
   - Wiping works the same.
2. After you upgrade, use `POST /hosts/:id/reset` instead of `"doWipe"` to repurpose hosts.
   - Until you do, Windows wipes that pass a `windows` object fail with a `422` error like: `The "windows" parameter is not available anymore. What to use instead: https://fleetdm.com/learn-more-about/fleet-5-upgrade`

GitHub issue: [#55003](https://github.com/fleetdm/fleet/issues/55003)

### YAML-only lists removed from configuration responses

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.apple_settings.configuration_profiles`, `mdm.windows_settings.configuration_profiles`, `mdm.setup_experience`, `mdm.volume_purchasing_program`, and `scripts` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) and [Get fleet](https://fleetdm.com/docs/rest-api/rest-api#get-fleet) | Removed. Use each feature's list endpoint, for example [List configuration profiles](https://fleetdm.com/docs/rest-api/rest-api#list-configuration-profiles). |

#### What to do

Change these before you upgrade.

These lists only included settings applied with YAML, so call the list endpoint for each feature instead.

GitHub issue: [#53932](https://github.com/fleetdm/fleet/issues/53932)

### features.detail_query_overrides removed from the API

| Fleet 4 | Fleet 5 |
|:--|:--|
| `features.detail_query_overrides` in [Update configuration](https://fleetdm.com/docs/rest-api/rest-api#update-configuration) | Removed |

Fleet 5 ignores overrides that are already saved, so built-in queries you turned off or replaced run as built in.

#### What to do

Change these before you upgrade.

Remove `detail_query_overrides` from your API calls. To stop collecting users, set `features.enable_host_users` to `false`. To stop collecting software on all platforms, set `features.enable_software_inventory` to `false`. Other built-in queries can't be turned off.

GitHub issue: [#33769](https://github.com/fleetdm/fleet/issues/33769)

### Software endpoints and keys changed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /software` and `GET /software/count` | [`GET /software/versions`](https://fleetdm.com/docs/rest-api/rest-api#list-software-versions) |
| `GET /software/:id` | [`GET /software/versions/:id`](https://fleetdm.com/docs/rest-api/rest-api#get-software-version) |
| `software_id` filter on `GET /hosts`, `GET /hosts/count`, `GET /hosts/report`, and `GET /labels/:id/hosts` | `software_version_id` or `software_title_id` on [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) |
| `software_package` object in software title responses | `packages` array in [Get software](https://fleetdm.com/docs/rest-api/rest-api#get-software) |
| `app_store_app` object in software title responses | `app_store_apps` array in [Get software](https://fleetdm.com/docs/rest-api/rest-api#get-software) |
| `browser` in host, software title, and software version responses | `extension_for` |

#### What to do

Fleet 4 already supports the new endpoints, filters, and keys, so change them before you upgrade.

Search for `/fleet/software`, `software_id`, `software_package`, `app_store_app`, and `browser`, and update each request and the code that reads responses.

GitHub issues: [#54408](https://github.com/fleetdm/fleet/issues/54408), [#54410](https://github.com/fleetdm/fleet/issues/54410), [#54409](https://github.com/fleetdm/fleet/issues/54409), [#52687](https://github.com/fleetdm/fleet/issues/52687), [#53395](https://github.com/fleetdm/fleet/issues/53395)

### Labels: platform and manual label changes

| Fleet 4 | Fleet 5 |
|:--|:--|
| `centos` and `ubuntu` as a label's `platform` | `linux` ([docs](https://fleetdm.com/docs/rest-api/rest-api#add-label)) |
| `hostname`, `node_key`, or `osquery_host_id` as a host identifier in a manual label's `hosts` | Not accepted ([docs](https://fleetdm.com/docs/rest-api/rest-api#add-label)) |

#### What to do

Make these changes before you upgrade.

- Change `centos` and `ubuntu` label platforms to `linux`.
  - If you don't, Fleet 5 changes them to `linux` for you.
- Identify hosts in manual labels by `hardware_serial` or `uuid`, or pass their IDs in `host_ids`.
  - Fleet 5 ignores a `hostname`, `node_key`, or `osquery_host_id` in `hosts`.

A label with the `linux` platform runs its query on every Linux host. Its current hosts stay, but it can add hosts running other Linux distributions if its query matches them.

To keep a label limited to Ubuntu or CentOS, wrap its query in an `os_version` check. For example, an Ubuntu label with this query:

```sql
SELECT 1 FROM deb_packages WHERE name = 'nginx';
```

becomes:

```sql
SELECT 1 FROM os_version WHERE platform = 'ubuntu' AND EXISTS (SELECT 1 FROM deb_packages WHERE name = 'nginx');
```

For a CentOS label, use `name LIKE '%centos%'` instead of `platform = 'ubuntu'`. Some CentOS hosts report `rhel` as their platform, so check the name.

[Update label](https://fleetdm.com/docs/rest-api/rest-api#update-label) can't change a label's query or platform. With the API, delete the label with [Delete label by ID](https://fleetdm.com/docs/rest-api/rest-api#delete-label-by-id) and create a new one with [Add label](https://fleetdm.com/docs/rest-api/rest-api#add-label). With [GitOps](https://fleetdm.com/docs/configuration/yaml-files#labels), change the label's `query` or `platform` in your YAML.

GitHub issues: [#52905](https://github.com/fleetdm/fleet/issues/52905), [#53924](https://github.com/fleetdm/fleet/issues/53924)

### Delete label by name removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `DELETE /labels/:name` | Removed, since fleets can now have labels with the same name. Use [`DELETE /labels/id/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-label-by-id). |

#### What to do

Make these changes before you upgrade.

1. Delete labels by ID. Get a label's ID from [List labels](https://fleetdm.com/docs/rest-api/rest-api#list-labels), with `fleet_id` for a fleet's labels.
2. If an API-only user can only call "Delete label by name", give it "Delete label by ID" instead.

GitHub issue: [#53801](https://github.com/fleetdm/fleet/issues/53801)

### Delete report by name removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `DELETE /reports/:name` | Removed. Use [`DELETE /reports/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-report-by-id). |
| `DELETE /reports/id/:id` | Moved to [`DELETE /reports/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-report-by-id). `DELETE /reports/id/:id` still works, but is deprecated. |

`DELETE /reports/:name` returns a `410` error, except when the name is only digits (for example `42`). Then Fleet 5 deletes the report with that ID. This includes `fleetctl delete` from Fleet 4.

#### What to do

Make these changes before you upgrade.

1. Delete reports with `DELETE /reports/id/:id`, which works in Fleet 4 and Fleet 5. Get a report's ID from [List reports](https://fleetdm.com/docs/rest-api/rest-api#list-reports), with `fleet_id` for a fleet's reports. After you upgrade, you can switch to `DELETE /reports/:id`.
2. If an API-only user can only call "Delete report by name", give it "Delete report by ID" instead. If you don't, Fleet does this when you upgrade.

GitHub issue: [#53799](https://github.com/fleetdm/fleet/issues/53799)

### Android Lock resets the PIN

| Fleet 4 | Fleet 5 |
|:--|:--|
| `POST /hosts/:id/lock` locks the screen of an Android host | [Lock host](https://fleetdm.com/docs/rest-api/rest-api#lock-host) resets the PIN to a random six-digit PIN and locks the host |

#### What to do

After you upgrade, use [Unlock host](https://fleetdm.com/docs/rest-api/rest-api#unlock-host) to get the PIN after you lock an Android host.

GitHub issue: [#49909](https://github.com/fleetdm/fleet/issues/49909)

### SCIM user delete requires an admin

| Fleet 4 | Fleet 5 |
|:--|:--|
| `DELETE /scim/Users/:id` works with a maintainer's API token | [Delete SCIM user](https://fleetdm.com/docs/rest-api/rest-api#delete-scim-user) requires an admin's API token |

#### What to do

Change these before you upgrade.

Use an admin's API token for your SCIM integration.

GitHub issue: [#53472](https://github.com/fleetdm/fleet/issues/53472)

### GitOps users can't run MDM commands

| Fleet 4 | Fleet 5 |
|:--|:--|
| GitOps users can run custom MDM commands (including lock, unlock, and wipe hosts) | GitOps users get a 403 from [Run MDM command](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command), [Lock host](https://fleetdm.com/docs/rest-api/rest-api#lock-host), [Unlock host](https://fleetdm.com/docs/rest-api/rest-api#unlock-host), [Wipe host](https://fleetdm.com/docs/rest-api/rest-api#wipe-host), [Turn off host's MDM](https://fleetdm.com/docs/rest-api/rest-api#turn-off-hosts-mdm), [Cancel host's pending MDM command](https://fleetdm.com/docs/rest-api/rest-api#cancel-hosts-pending-mdm-command), [Rotate host's Recovery Lock password](https://fleetdm.com/docs/rest-api/rest-api#rotate-hosts-recovery-lock-password), and [Rotate host's managed local account password](https://fleetdm.com/docs/rest-api/rest-api#rotate-hosts-managed-local-account-password) |
| `fleetctl mdm run-command`, `fleetctl mdm lock`, `unlock`, and `wipe` work with a GitOps user's API token | Return a 403 |

#### What to do

Change these before you upgrade.

Call these endpoints, and run these `fleetctl mdm` commands, with a different [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user).

GitHub issue: [#53145](https://github.com/fleetdm/fleet/issues/53145)

### GitOps users can't view hosts by identifier

| Fleet 4 | Fleet 5 |
|:--|:--|
| GitOps users can get a host's `id` from `GET /hosts/identifier/:identifier` | GitOps users get a 403 from [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) when `:id` is an identifier, and can't look up hosts by identifier on any other endpoint |

#### What to do

Change these before you upgrade.

If an integration uses a GitOps user's API token to look up hosts by identifier, switch it to a different [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) that can view hosts.

GitHub issue: [#57406](https://github.com/fleetdm/fleet/issues/57406)

### GitOps users can't delete hosts

| Fleet 4 | Fleet 5 |
|:--|:--|
| GitOps users can delete hosts | GitOps users get a 403 from [Delete host](https://fleetdm.com/docs/rest-api/rest-api#delete-host) and [Batch-delete hosts](https://fleetdm.com/docs/rest-api/rest-api#batch-delete-hosts) |

#### What to do

Change these before you upgrade.

Delete hosts with a different [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user).

GitHub issue: [#57406](https://github.com/fleetdm/fleet/issues/57406)

### GitOps users can't transfer hosts between fleets

| Fleet 4 | Fleet 5 |
|:--|:--|
| GitOps users can transfer hosts between fleets | GitOps users get a 403 from [Update hosts' fleet](https://fleetdm.com/docs/rest-api/rest-api#update-hosts-fleet) and [Update hosts' fleet by filter](https://fleetdm.com/docs/rest-api/rest-api#update-hosts-fleet-by-filter) |

#### What to do

Change these before you upgrade.

Transfer hosts between fleets with a different [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user).

GitHub issue: [#57406](https://github.com/fleetdm/fleet/issues/57406)

### Certificate authority names must be unique

| Fleet 4 | Fleet 5 |
|:--|:--|
| Certificate authorities (CAs) of different types can share a name | Every CA needs a unique name, ignoring case, in [Connect certificate authority](https://fleetdm.com/docs/rest-api/rest-api#connect-certificate-authority-ca) and [Update certificate authority](https://fleetdm.com/docs/rest-api/rest-api#update-certificate-authority-ca) |
| A name that's already used returns `400` in Connect certificate authority and `409` in Update certificate authority | Both return `422` |

#### What to do

Change these before you upgrade.

1. Call [List certificate authorities](https://fleetdm.com/docs/rest-api/rest-api#list-certificate-authorities-cas), find CAs that share a `name` (ignoring case), and rename them.
2. Update the configuration profiles that reference them.
3. If your scripts or integrations check for a `400` or `409` when a CA name is taken, check for `422` instead.

GitHub issue: [#56271](https://github.com/fleetdm/fleet/issues/56271)

### Fleet names must be unique

| Fleet 4 | Fleet 5 |
|:--|:--|
| Fleets can have names that differ only by punctuation, whitespace, symbols, or emoji, or names with no letters or numbers | [Create fleet](https://fleetdm.com/docs/rest-api/rest-api#create-fleet), [Update fleet](https://fleetdm.com/docs/rest-api/rest-api#update-fleet), and GitOps require names that include at least one letter or number and are unique when Fleet ignores letter case, accents, whitespace, punctuation, symbols, and emoji |

When you upgrade, Fleet renames existing fleets that break these rules:

- If two or more fleets' names match (for example, `Workstations` and `workstations!`), the oldest fleet keeps its name. Fleet adds a number to the others, like `workstations! 2`.
- If a fleet's name has no letters or numbers (for example, `🚀`), Fleet adds a number, like `🚀 1`.

Fleet only changes the name. Hosts, settings, and everything else in the fleet stay the same. Apple Business and Volume Purchasing Program (VPP) settings show the fleet's new name.

#### What to do

Change these before you upgrade, so Fleet doesn't rename your fleets.

1. Call [List fleets](https://fleetdm.com/docs/rest-api/rest-api#list-fleets) and find names that match when you ignore letter case, accents, whitespace, punctuation, symbols, and emoji, plus names with no letters or numbers.
2. Rename those fleets. If you use GitOps, change `name` in each fleet's YAML file and run GitOps. Also update any other YAML that refers to the fleet by name.

If you upgrade first, update your YAML to use the new names: `name` in each renamed fleet's file, and any other YAML that refers to those fleets by name, like Apple Business, VPP, and Windows automatic enrollment settings in `default.yml`. Otherwise, your GitOps runs fail.

GitHub issue: [#53790](https://github.com/fleetdm/fleet/issues/53790)

### $FLEET_VAR_HOST_END_USER_EMAIL_IDP removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` works in configuration profiles, managed app configurations, and the DigiCert and EJBCA CA fields in [Connect certificate authority](https://fleetdm.com/docs/rest-api/rest-api#connect-certificate-authority-ca) and [Update certificate authority](https://fleetdm.com/docs/rest-api/rest-api#update-certificate-authority-ca) | Removed. Use `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` |

#### What to do

1. Before you upgrade, replace `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` with `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` everywhere you use it, including your configuration profiles, managed app configurations, the DigiCert CA fields `certificate_common_name`, `certificate_user_principal_names`, and `certificate_seat_id`, and the EJBCA CA fields `certificate_common_name`, `username_template`, and `certificate_user_principal_names`.
   - Profiles, managed app configurations, and CA fields that still use the old variable fail after you upgrade, the next time Fleet sends them to a host.
   - Certificates whose profiles or certificate authorities still use it stop renewing and expire.
   - On Android, one profile that uses it stops all of the host's profiles from being applied.
   - In most setups, the new variable has the same value. If your IdP usernames aren't your end users' emails, hosts get the IdP username instead.
   - Editing the DigiCert or EJBCA CA fields reissues certificates to every host that uses the CA.
   - See [Fleet variables](https://fleetdm.com/guides/fleet-variables).

GitHub issue: [#57603](https://github.com/fleetdm/fleet/issues/57603)

### Packs endpoints removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `/packs` endpoints, including the packs spec endpoints | Removed. Use [reports](https://fleetdm.com/docs/rest-api/rest-api#create-report). |

#### What to do

Change these before you upgrade.

1. Convert 2017 packs into reports.
   - See [fleetctl upgrade-packs](#packs-commands-removed).
   - 2017 packs stop running on hosts after the upgrade.

GitHub issue: [#53470](https://github.com/fleetdm/fleet/issues/53470)

### User spec endpoints removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `POST /users/roles/spec`, which `fleetctl apply` calls for `kind: user_roles` files | Removed. Use [Update user](https://fleetdm.com/docs/rest-api/rest-api#update-user). |

#### What to do

Change these before you upgrade.

1. Replace calls to `POST /users/roles/spec` (and `fleetctl apply` with `kind: user_roles`) with [`PATCH /users/:id`](https://fleetdm.com/docs/rest-api/rest-api#update-user), once for each user.
   - Set `global_role` for a global role, or `fleets` for fleet roles.
   - To find each user's `id`, call [List users](https://fleetdm.com/docs/rest-api/rest-api#list-users).

GitHub issue: [#58158](https://github.com/fleetdm/fleet/issues/58158)

### Puppet module removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| Fleet's Puppet module (`fleetdm-fleetdm` on the Puppet Forge), and the `POST /mdm/apple/profiles/preassign` and `POST /mdm/apple/profiles/match` endpoints it calls | Removed. |

#### What to do

Change these before you upgrade.

1. If you use the Puppet module, assign configuration profiles to fleets in the UI, API, or GitOps instead, and move hosts between fleets the same way.
   - Fleets the module created stay as regular fleets.

GitHub issue: [#28334](https://github.com/fleetdm/fleet/issues/28334)

### Manual enrollment profile off by default

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /enrollment_profiles/manual` works | [Get manual enrollment profile](https://fleetdm.com/docs/rest-api/rest-api#get-manual-enrollment-profile) returns a `410` unless `mdm.apple_scep_static_challenge_enabled` is `true` |

#### What to do

Change these before you upgrade.

Follow the steps in [Static SCEP challenge off by default](#static-scep-challenge-off-by-default).

GitHub issue: [#54190](https://github.com/fleetdm/fleet/issues/54190)


## YAML (GitOps)

### Team keys renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `team_settings` in a fleet's YAML file | `settings` |
| `teams` under `org_settings.mdm.volume_purchasing_program` | `fleets` |

#### What to do

Fleet 4 already supports the new names, so change them before you upgrade.

Rename these keys in your YAML files.

GitHub issue: [#52795](https://github.com/fleetdm/fleet/issues/52795)

### Apple Business keys renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `org_settings.mdm.apple_business_manager` | [`org_settings.mdm.apple_business`](https://fleetdm.com/docs/configuration/yaml-files#apple-business) |
| `macos_team`, `ios_team`, `ipados_team`, and `byod_team` under `apple_business` | `macos_fleet`, `ios_fleet`, `ipados_fleet`, and `byod_fleet` |
| `org_settings.mdm.apple_bm_default_team` | Removed |

#### What to do

Fleet 4 already supports the new names, so change them before you upgrade.

Rename these keys in your YAML files.

GitHub issues: [#53272](https://github.com/fleetdm/fleet/issues/53272), [#52795](https://github.com/fleetdm/fleet/issues/52795)

### controls.macos_setup renamed to controls.setup_experience

| Fleet 4 | Fleet 5 |
|:--|:--|
| `controls.macos_setup` | [`controls.setup_experience`](https://fleetdm.com/docs/configuration/yaml-files#setup-experience) |

#### What to do

Fleet 4 already supports `setup_experience`, so change it before you upgrade.

Rename `macos_setup` to `setup_experience` in your YAML files.

GitHub issue: [#53894](https://github.com/fleetdm/fleet/issues/53894)

### Setup experience and Recovery Lock keys renamed in YAML

| Fleet 4 | Fleet 5 |
|:--|:--|
| `require_all_software_macos` and `require_all_software_windows` under `controls.setup_experience` | `macos_require_all_software` and `windows_require_all_software` ([docs](https://fleetdm.com/docs/configuration/yaml-files#setup-experience)) |
| `controls.enable_recovery_lock_password` | `controls.macos_recovery_lock_password_enabled` ([docs](https://fleetdm.com/docs/configuration/yaml-files#controls)) |

#### What to do

Fleet 4 doesn't support the new names, so you can't change them before you upgrade.

1. After you upgrade, rename these keys before your next GitOps run.
   - Until you do, your GitOps run fails with an error like: `The "controls.enable_recovery_lock_password" key is not available anymore. What to use instead: https://fleetdm.com/learn-more-about/fleet-5-upgrade`

GitHub issue: [#53872](https://github.com/fleetdm/fleet/issues/53872)

### Setup experience script key removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `script` under `controls.setup_experience` (or `controls.macos_setup`) | Removed. `fleetctl gitops` fails if it's set. |

#### What to do

Fleet 4 already supports script-only packages in setup experience and `macos_script`, so change this before you upgrade.

Move the script to a script-only package. Add the .sh file to `software.packages` with `setup_experience_platform: darwin` ([docs](https://fleetdm.com/docs/configuration/yaml-files#packages)), then remove `script`. Setup experience software installs in alphabetical order, and the setup script runs after all software. To run the package last, give it a name that sorts after your other setup experience software. If you can't move the script yet, rename `script` to `macos_script`. `macos_script` keeps working in Fleet 5 and is removed in Fleet 6. See [Setup experience script deprecated](#setup-experience-script-deprecated).

GitHub issues: [#53274](https://github.com/fleetdm/fleet/issues/53274), [#53894](https://github.com/fleetdm/fleet/issues/53894)

### Windows MDM keys renamed in YAML

| Fleet 4 | Fleet 5 |
|:--|:--|
| `controls.windows_enabled_and_configured` | `controls.windows_mdm_enabled` ([docs](https://fleetdm.com/docs/configuration/yaml-files#controls)) |
| `controls.windows_migration_enabled` | `controls.windows_mdm_migration_enabled` ([docs](https://fleetdm.com/docs/configuration/yaml-files#controls)) |

#### What to do

Fleet 4 doesn't support the new names, so you can't change them before you upgrade.

1. After you upgrade, rename these keys before your next GitOps run.
   - `fleetctl generate-gitops` writes the new names.
   - Until you do, your GitOps run fails with an error like: `The "controls.windows_enabled_and_configured" key is not available anymore. What to use instead: https://fleetdm.com/learn-more-about/fleet-5-upgrade`

GitHub issue: [#53873](https://github.com/fleetdm/fleet/issues/53873)

### Managed local account keys renamed in YAML

| Fleet 4 | Fleet 5 |
|:--|:--|
| `enable_managed_local_account` and `enable_create_local_admin_account` under `controls.setup_experience` (or `controls.macos_setup`) | `macos_managed_local_account_enabled` ([docs](https://fleetdm.com/docs/configuration/yaml-files#setup-experience)) |
| `end_user_local_account_type` under `controls.setup_experience` (or `controls.macos_setup`) | `macos_end_user_local_account_type` ([docs](https://fleetdm.com/docs/configuration/yaml-files#setup-experience)) |
| `controls.windows_settings.enable_managed_local_account` | `controls.setup_experience.windows_managed_local_account_enabled` ([docs](https://fleetdm.com/docs/configuration/yaml-files#setup-experience)) |

#### What to do

1. Rename these keys.
2. Move the Windows key from `windows_settings` to `setup_experience`.

GitHub issue: [#50084](https://github.com/fleetdm/fleet/issues/50084)

### Organization logo keys renamed in YAML

| Fleet 4 | Fleet 5 |
|:--|:--|
| `org_info.org_logo_url` | [`org_info.org_logo_url_dark_mode`](https://fleetdm.com/docs/configuration/yaml-files#org-info) |
| `org_info.org_logo_url_light_background` | [`org_info.org_logo_url_light_mode`](https://fleetdm.com/docs/configuration/yaml-files#org-info) |

#### What to do

Fleet 4 already supports the new names, so change them before you upgrade.

Rename both keys in your YAML files.

GitHub issue: [#52800](https://github.com/fleetdm/fleet/issues/52800)

### Failing policies webhook moved to each policy

| Fleet 4 | Fleet 5 |
|:--|:--|
| `policy_ids` under `failing_policies_webhook` | `webhooks_and_tickets_enabled: true` on each policy ([docs](https://fleetdm.com/docs/configuration/yaml-files#automations)) |

#### What to do

Fleet 4 already supports `webhooks_and_tickets_enabled`, so change this before you upgrade.

Move the IDs in `policy_ids` to `webhooks_and_tickets_enabled: true` on each policy.

GitHub issue: [#52900](https://github.com/fleetdm/fleet/issues/52900)

### One automation per policy in YAML

| Fleet 4 | Fleet 5 |
|:--|:--|
| A policy can have more than one of `install_software`, `run_script`, `resend_configuration_profile`, `calendar_events_enabled`, `conditional_access_enabled`, and `webhooks_and_tickets_enabled` | One automation per policy ([docs](https://fleetdm.com/docs/configuration/yaml-files#automations)). `fleetctl gitops` fails if a policy has more than one. |

#### What to do

Change these before you upgrade.

Move extra automations to new policies.

GitHub issue: [#52900](https://github.com/fleetdm/fleet/issues/52900)

### features.detail_query_overrides removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `features.detail_query_overrides` | Removed |

Fleet 5 ignores overrides that are already saved, so built-in queries you turned off or replaced run as built in.

#### What to do

Change these before you upgrade.

Remove `detail_query_overrides` from your YAML. To stop collecting users, set `features.enable_host_users` to `false`. To stop collecting software on all platforms, set `features.enable_software_inventory` to `false`. Other built-in queries can't be turned off.

GitHub issue: [#33769](https://github.com/fleetdm/fleet/issues/33769)

### Labels: platform and manual label changes in YAML

| Fleet 4 | Fleet 5 |
|:--|:--|
| `centos` and `ubuntu` as a label's `platform` | `linux` ([docs](https://fleetdm.com/docs/configuration/yaml-files#labels)) |
| `hostname` as a host identifier in a manual label's `hosts` | Not accepted ([docs](https://fleetdm.com/docs/configuration/yaml-files#labels)) |

#### What to do

Make these changes before you upgrade.

- Change `centos` and `ubuntu` label platforms to `linux`.
  - Fleet 5 GitOps runs fail on `centos` or `ubuntu`.
- Identify hosts in manual labels by something other than `hostname`.

A label with the `linux` platform runs its query on every Linux host. Its current hosts stay, but it can add hosts running other Linux distributions if its query matches them.

To keep a label limited to Ubuntu or CentOS, wrap its query in an `os_version` check. For example, an Ubuntu label with this query:

```sql
SELECT 1 FROM deb_packages WHERE name = 'nginx';
```

becomes:

```sql
SELECT 1 FROM os_version WHERE platform = 'ubuntu' AND EXISTS (SELECT 1 FROM deb_packages WHERE name = 'nginx');
```

For a CentOS label, use `name LIKE '%centos%'` instead of `platform = 'ubuntu'`. Some CentOS hosts report `rhel` as their platform, so check the name.

GitHub issues: [#52905](https://github.com/fleetdm/fleet/issues/52905), [#53924](https://github.com/fleetdm/fleet/issues/53924)

### Omitted settings reset to defaults

| Fleet 4 | Fleet 5 |
|:--|:--|
| If your YAML leaves out `gitops`, `host_expiry_settings`, or `smtp_settings`, they keep their current values | They reset to their defaults ([`gitops`](https://fleetdm.com/docs/configuration/yaml-files#gitops), [`host_expiry_settings`](https://fleetdm.com/docs/configuration/yaml-files#host-expiry-settings), [`smtp_settings`](https://fleetdm.com/docs/configuration/yaml-files#smtp-settings)) |

#### What to do

Change these before you upgrade.

1. To get the current values, run `fleetctl generate-gitops`.
2. Add these settings to your YAML with the values you use today.

GitHub issue: [#53899](https://github.com/fleetdm/fleet/issues/53899)

### server_settings.scripts_disabled removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `server_settings.scripts_disabled` | Removed ([docs](https://fleetdm.com/docs/configuration/yaml-files#server-settings)). `fleetctl gitops` fails if it's `true` and shows a warning if it's `false`. Fleet 5 turns script features back on for instances that had `scripts_disabled: true`. |
| `server_settings.scripts_disabled` in `GET` and `PATCH /api/v1/fleet/config` | Removed. `GET /config` doesn't return it, and `PATCH /config` returns an error if it's set. |

#### What to do

Change these before you upgrade.

1. Remove `scripts_disabled` from your YAML, and from scripts that call `PATCH /api/v1/fleet/config`.
2. To keep scripts off, deploy fleetd without `--enable-scripts`.

GitHub issue: [#53897](https://github.com/fleetdm/fleet/issues/53897)

### Package YAML files must be lists

| Fleet 4 | Fleet 5 |
|:--|:--|
| A package YAML file can hold one package that isn't in a list | Packages must be in a list, even if there's only one: start the package with `- ` and indent its other keys ([docs](https://fleetdm.com/docs/configuration/yaml-files#packages)). `fleetctl gitops` fails otherwise. |

#### What to do

Fleet 4 already supports lists in package YAML files, so change them before you upgrade.

In each package YAML file, put packages in a list, even if there's only one.

GitHub issue: [#53378](https://github.com/fleetdm/fleet/issues/53378)

### Certificate authority names must be unique in YAML

| Fleet 4 | Fleet 5 |
|:--|:--|
| Certificate authorities (CAs) of different types can share a name | Every CA needs a unique name, ignoring case ([docs](https://fleetdm.com/docs/configuration/yaml-files#certificate-authorities)) |

#### What to do

Change these before you upgrade.

1. Rename CAs that share a name (ignoring case).
2. Update the configuration profiles that reference them.

GitHub issue: [#56271](https://github.com/fleetdm/fleet/issues/56271)


### $FLEET_VAR_HOST_END_USER_EMAIL_IDP removed in YAML

| Fleet 4 | Fleet 5 |
|:--|:--|
| `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` works in configuration profiles, managed app configurations, and in the `certificate_common_name`, `certificate_user_principal_names`, and `certificate_seat_id` keys of `org_settings.certificate_authorities.digicert`, and the `certificate_common_name`, `username_template`, and `certificate_user_principal_names` keys of `org_settings.certificate_authorities.ejbca` ([docs](https://fleetdm.com/docs/configuration/yaml-files#certificate-authorities)) | Removed. Use `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` |

#### What to do

1. Before you upgrade, replace `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` with `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` everywhere you use it, including your configuration profiles, managed app configurations, and the DigiCert and EJBCA keys.
   - Profiles, managed app configurations, and CA keys that still use the old variable fail after you upgrade, the next time Fleet sends them to a host.
   - Certificates whose profiles or certificate authorities still use it stop renewing and expire.
   - On Android, one profile that uses it stops all of the host's profiles from being applied.
   - In most setups, the new variable has the same value. If your IdP usernames aren't your end users' emails, hosts get the IdP username instead.
   - Editing the DigiCert or EJBCA keys reissues certificates to every host that uses the CA.
   - See [Fleet variables](https://fleetdm.com/guides/fleet-variables).

GitHub issue: [#57603](https://github.com/fleetdm/fleet/issues/57603)

## fleetctl

### --team is now --fleet

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get teams`, `fleetctl get team`, and `fleetctl get t` | `fleetctl get fleets` |
| `fleetctl gitops --delete-other-teams` and the `DELETE_OTHER_TEAMS` environment variable | `fleetctl gitops --delete-other-fleets` and `DELETE_OTHER_FLEETS` |
| `fleetctl apply --policies-team` | `fleetctl apply --policies-fleet` |
| `--team` on `fleetctl report`, `user create`, `hosts transfer`, `run-script`, `get reports`, `get labels`, `get hosts`, and `get software` | `--fleet` on the same commands |
| `-t` on `fleetctl user create` | `--fleet` |
| `fleetctl generate-gitops --team no-team`, and `no-team` as the value of `--fleet` or `--key` | `--fleet unassigned` and `--key unassigned` |

#### What to do

1. Search your scripts and CI for `--team`, `-t`, `--policies-team`, `--delete-other-teams`, `DELETE_OTHER_TEAMS`, `no-team`, and `get teams`, and replace each match.
   - The new flags already work in Fleet 4.

GitHub issue: [#52795](https://github.com/fleetdm/fleet/issues/52795)

### get mdm-commands requires --host

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get mdm-commands` | `fleetctl get mdm-commands --host <host>`. `--host` is required. |

#### What to do

Pass `--host` on every call.

GitHub issue: [#53146](https://github.com/fleetdm/fleet/issues/53146)

### mdm-apple-bm is now mdm-ab

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get mdm-apple-bm` (and the `mdm_apple_bm` alias) | `fleetctl get mdm-ab` |
| `fleetctl generate mdm-apple-bm` (and the `mdm_apple_bm` alias) | `fleetctl generate mdm-ab` |

#### What to do

1. Replace `mdm-apple-bm` with `mdm-ab`.
   - It already works in Fleet 4.

GitHub issue: [#53272](https://github.com/fleetdm/fleet/issues/53272)

### get config prints setup_experience

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get config` prints `macos_setup` | `fleetctl get config` prints `setup_experience` |

#### What to do

Update scripts that read `macos_setup` from `fleetctl get config` output.

GitHub issue: [#53894](https://github.com/fleetdm/fleet/issues/53894)

### get config prints renamed managed local account keys

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get config` and `fleetctl get fleets` print `enable_create_local_admin_account` and `end_user_local_account_type` under `macos_setup`, and `enable_managed_local_account` under `windows_settings` | They print `macos_managed_local_account_enabled`, `macos_end_user_local_account_type`, and `windows_managed_local_account_enabled` under `setup_experience` |

#### What to do

Update scripts that read these keys from `fleetctl get config` or `fleetctl get fleets` output.

GitHub issue: [#50084](https://github.com/fleetdm/fleet/issues/50084)

### get software output changed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get software --json` and `--yaml` include `software_package`, `app_store_app`, and `browser` | They include `packages`, `app_store_apps`, and `extension_for` |

#### What to do

Update scripts that read these keys.

GitHub issues: [#54409](https://github.com/fleetdm/fleet/issues/54409), [#52687](https://github.com/fleetdm/fleet/issues/52687), [#53395](https://github.com/fleetdm/fleet/issues/53395)

### mdm unlock returns the Android PIN

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl mdm unlock` doesn't support Android hosts | `fleetctl mdm unlock --host=<host>` returns the PIN that Lock set on an Android host |

#### What to do

Use `fleetctl mdm unlock` to get the PIN after you lock an Android host.

GitHub issue: [#49909](https://github.com/fleetdm/fleet/issues/49909)

### Packs commands removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl get packs` and `fleetctl apply` with `kind: pack` | Removed |
| `fleetctl upgrade-packs` | Removed |

#### What to do

1. Before you upgrade, run `fleetctl upgrade-packs -o packs-to-reports.yml` on Fleet 4 and review the file.
   - 2017 packs stop running on hosts after the upgrade.
2. Add the reports to Fleet with [GitOps (YAML)](https://fleetdm.com/docs/configuration/yaml-files#reports) or [Create report](https://fleetdm.com/docs/rest-api/rest-api#create-report) (API).
   - With GitOps, copy each report's settings from under `spec:` into `reports` in `default.yml` or the fleet's YAML file.
   - With the API, call [`POST /reports`](https://fleetdm.com/docs/rest-api/rest-api#create-report) once for each report.

GitHub issue: [#53470](https://github.com/fleetdm/fleet/issues/53470)

### fleetctl delete removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl delete -f` deletes the reports and labels listed in a file | Removed |

#### What to do

1. Delete reports and labels in the UI, with the API, or with GitOps.
   - With the API, use [`DELETE /reports/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-report-by-id) and [`DELETE /labels/id/:id`](https://fleetdm.com/docs/rest-api/rest-api#delete-label-by-id).

GitHub issue: [#53799](https://github.com/fleetdm/fleet/issues/53799)

### fleetctl user create-users and delete-users removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl user create-users --csv` and `fleetctl user delete-users --csv` create and delete users listed in a CSV file | Removed. Use `fleetctl user create` and `fleetctl user delete`. |

#### What to do

1. Replace each `fleetctl user create-users` run with one `fleetctl user create` run for each row in the CSV file.
   - Pass the CSV columns as flags: `--email`, `--name`, `--sso`, `--api-only`, `--global-role`, and `--fleet` (for example, `--fleet 1:observer`).
   - `create-users` set a random password. With `fleetctl user create`, pass `--password` for users that don't use SSO.
2. Replace each `fleetctl user delete-users` run with one `fleetctl user delete --email <email>` run for each email in the CSV file.
3. You can also use the [Create user](https://fleetdm.com/docs/rest-api/rest-api#create-user) and [Delete user](https://fleetdm.com/docs/rest-api/rest-api#delete-user) API endpoints.

GitHub issue: [#58158](https://github.com/fleetdm/fleet/issues/58158)

### fleetctl package file names changed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl package` file names like `fleet-osquery_1.61.0_amd64.deb` | `fleet-{platform}{-arch}-{fleet-name}.ext`, for example `fleet-macos-workstations.pkg`, or `fleet-windows-amd64.msi` when there's no fleet name. If two fleets' names would give the same file name, the fleet ID is added, like `fleet-macos-workstations-3.pkg`. ([docs](https://fleetdm.com/guides/enroll-hosts)) |

#### What to do

1. Update scripts and CI that use the file names, or add `--outfile` to keep a fixed name.
   - `fleetctl package` leaves out the fleet name if it can't find it, for example when you're not logged in to `fleetctl` or the enroll secret is global (hosts in Unassigned).

GitHub issue: [#53365](https://github.com/fleetdm/fleet/issues/53365)

### fleetctl 5 required for GitOps

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleetctl gitops` with fleetctl 4 | Fails when it deletes a label, or moves a label between `default.yml` and a fleet's YAML file |

#### What to do

1. Upgrade `fleetctl` to 5 wherever you run `fleetctl gitops`, including CI, when you upgrade your Fleet server.

GitHub issue: [#53801](https://github.com/fleetdm/fleet/issues/53801)


## Fleet server configuration

### Apple Business options renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.apple_bm_server_token` and `mdm.apple_bm_server_token_bytes` | `mdm.apple_business_server_token` and `mdm.apple_business_server_token_bytes` |
| `mdm.apple_bm_cert` and `mdm.apple_bm_cert_bytes` | `mdm.apple_business_cert` and `mdm.apple_business_cert_bytes` |
| `mdm.apple_bm_key` and `mdm.apple_bm_key_bytes` | `mdm.apple_business_key` and `mdm.apple_business_key_bytes` |
| `FLEET_MDM_APPLE_BM_*` environment variables | `FLEET_MDM_APPLE_BUSINESS_*` |

#### What to do

1. Rename the options and environment variables in the same deploy that upgrades to Fleet 5.
   - Fleet 5 doesn't start if an old one is still set.

GitHub issue: [#53272](https://github.com/fleetdm/fleet/issues/53272)

### Static SCEP challenge off by default

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.apple_scep_static_challenge_enabled` defaults to `true` | Defaults to `false` ([docs](https://fleetdm.com/docs/configuration/fleet-server-configuration#mobile-device-management-mdm)) |

#### What to do

1. If you use the [manual enrollment profile endpoint](https://fleetdm.com/docs/rest-api/rest-api#get-manual-enrollment-profile), set `mdm.apple_scep_static_challenge_enabled` (`FLEET_MDM_APPLE_SCEP_STATIC_CHALLENGE_ENABLED`) to `true`.
   - That turns the static SCEP challenge back on for all Apple MDM enrollments, so anyone with the challenge can turn on MDM for any host.
   - Previously downloaded profiles stop working.
   - Hosts that already have MDM on aren't affected.

GitHub issue: [#54190](https://github.com/fleetdm/fleet/issues/54190)

### Async host processing removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `osquery_enable_async_host_processing` (`FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING`) | Removed. Fleet always processes label membership, policy membership, last seen, and scheduled report stats synchronously. |
| `osquery_async_host_collect_interval`, `osquery_async_host_collect_max_jitter_percent`, `osquery_async_host_collect_lock_timeout`, `osquery_async_host_collect_log_stats_interval`, `osquery_async_host_insert_batch`, `osquery_async_host_delete_batch`, `osquery_async_host_update_batch`, `osquery_async_host_redis_pop_count`, and `osquery_async_host_redis_scan_keys_count` (and their `FLEET_OSQUERY_*` environment variables) | Removed |

#### What to do

1. Remove these options from your server configuration.
   - Data still waiting in Redis when you upgrade is dropped and corrected on the next check-in.

GitHub issue: [#54009](https://github.com/fleetdm/fleet/issues/54009)

### osquery_min_software_last_opened_at_diff removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `osquery_min_software_last_opened_at_diff` (`FLEET_OSQUERY_MIN_SOFTWARE_LAST_OPENED_AT_DIFF`) | Removed |

#### What to do

Remove it from your server configuration.

GitHub issue: [#53787](https://github.com/fleetdm/fleet/issues/53787)

### Performance tracing changes

This only applies if you send Fleet server performance data (traces) to a monitoring tool like Elastic, Datadog, or Grafana. You'd know because you set `logging.tracing_enabled` (`FLEET_LOGGING_TRACING_ENABLED`) to `true`. It's off by default, so most deployments can skip this.

| Fleet 4 | Fleet 5 |
|:--|:--|
| Setting `logging.tracing_type` to `elasticapm` sends traces to Elastic APM | Removed. Fleet stops sending traces and logs a warning at startup. [OpenTelemetry](https://fleetdm.com/docs/configuration/fleet-server-configuration#logging-tracing-enabled) is the only option. |
| Trace names for API requests look like `GET /api/{fleetversion:(?:v1\|2022-04\|latest)}/fleet/hosts/{id:[0-9]+}` | Trace names are simpler, like `GET /api/_version_/fleet/hosts/{id}` |

#### What to do

1. If you use Elastic APM, remove `logging.tracing_type` and set `OTEL_EXPORTER_OTLP_ENDPOINT` to your OpenTelemetry collector.
   - Elastic accepts OpenTelemetry data, so you can keep using Elastic.
2. Update any dashboards or alerts in your monitoring tool that match on the old trace names.

GitHub issue: [#58257](https://github.com/fleetdm/fleet/issues/58257)


## Single sign-on (SSO)

### Fleet role SAML attribute renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `FLEET_JIT_USER_ROLE_TEAM_<ID>` SAML attribute | `FLEET_JIT_USER_ROLE_FLEET_<ID>` ([docs](https://fleetdm.com/docs/deploy/single-sign-on-sso#just-in-time-jit-user-provisioning)) |

#### What to do

1. Rename the attribute in your IdP before you upgrade.
   - Fleet 4 already accepts the new name.
   - Fleet 5 ignores the old one. A new user whose fleet roles come only from the old attribute gets the global Observer role.
   - An existing user whose login has only old attributes keeps their roles. If the login mixes old and new attributes, they lose the fleets sent only through the old one.

GitHub issue: [#53925](https://github.com/fleetdm/fleet/issues/53925)


## Activities

### Activity types renamed or removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `created_team`, `deleted_team`, `applied_spec_team`, `changed_user_team_role`, and `deleted_user_team_role` | `created_fleet`, `deleted_fleet`, `applied_spec_fleet`, `changed_user_fleet_role`, and `deleted_user_fleet_role` |
| `team_id`, `team_name`, and `teams` in an activity's `details` | `fleet_id`, `fleet_name`, and `fleets` |
| `enabled_macos_disk_encryption` and `disabled_macos_disk_encryption` | `edited_disk_encryption_settings` |
| `created_pack`, `edited_pack`, `deleted_pack`, and `applied_spec_pack` | Removed. Fleet 5 keeps existing activities with these types but doesn't create new ones. |

See the [activity types reference](https://fleetdm.com/docs/api/global-audit-logs) for the full list.

#### What to do

Update anything that filters or parses activities, like SIEM rules and log pipelines, to use the new types.

GitHub issues: [#52795](https://github.com/fleetdm/fleet/issues/52795), [#53481](https://github.com/fleetdm/fleet/issues/53481), [#53470](https://github.com/fleetdm/fleet/issues/53470)


## fleetd

### Process and app names are Fleet

| Fleet 4 | Fleet 5 |
|:--|:--|
| Windows Task Manager shows "Orbit osquery runtime and autoupdater" | `Fleet`. This changes when fleetd updates, whatever the Fleet server version. |
| Windows **Settings > Apps > Installed apps** shows "Fleet osquery" | `Fleet`. This changes when fleetd connects to a Fleet 5 server. |
| Linux `ps`, `top`, and System Monitor show the process name `orbit` | `Fleet`. This changes when fleetd talks to a Fleet 5 server. |
| macOS Activity Monitor and `ps` show the process name `orbit`, and fleetd starts from `/opt/orbit/bin/orbit/orbit`, a link to `/opt/orbit/bin/orbit/macos/<channel>/orbit` | `Fleet`, and fleetd starts from `/opt/orbit/bin/orbit/Fleet`, a link to `/opt/orbit/bin/orbit/macos/<channel>/Fleet`. |

#### What to do

1. Add the new names to EDR, monitoring, and firewall rules. On macOS, this includes the new path `/opt/orbit/bin/orbit/Fleet`.
   - Keep the old names until every host runs the new fleetd.
2. On macOS, add the new paths to your Full Disk Access profile. Fleet's [sample profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/allow-fleetd-full-disk-access.mobileconfig) lists both the old and new paths, so you can deploy it before you upgrade.
   - Without the new path, fleetd and osquery lose Full Disk Access on hosts that run the new fleetd.
3. Update policies and scripts that match "Fleet osquery" in the `programs` table so they match "Fleet" too, including copies of Fleet's Windows uninstall script.
   - Repairing fleetd from the installer brings "Fleet osquery" back until fleetd renames it again.
   - On upgraded Fleet servers, the Software page keeps listing it as "Fleet osquery". New Fleet 5 servers list "Fleet".

GitHub issue: [#56803](https://github.com/fleetdm/fleet/issues/56803)

### Older fleetd can't find the organization logo

| Fleet 4 | Fleet 5 |
|:--|:--|
| fleetd earlier than 1.56.2 reads `org_logo_url` and `org_logo_url_light_background` to show your organization's logo in the macOS MDM migration window | Fleet no longer returns those keys. fleetd 1.56.2 and later read `org_logo_url_dark_mode` and `org_logo_url_light_mode`. Earlier versions show the default Fleet logo. |

#### What to do

Update fleetd to 1.56.2 or later before you upgrade to Fleet 5.

GitHub issue: [#52800](https://github.com/fleetdm/fleet/issues/52800)


## Fleet UI

### Disable scripts setting removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| **Disable scripts** checkbox in **Settings > Organization settings > Advanced options** | Removed. To keep scripts off, deploy fleetd without `--enable-scripts`. |

#### What to do

To keep scripts off, deploy fleetd without `--enable-scripts`.

GitHub issue: [#53897](https://github.com/fleetdm/fleet/issues/53897)


## Deprecated

These features keep working in Fleet 5 and are removed in Fleet 6.

### Setup experience script deprecated

| Fleet 4 | Fleet 5 |
|:--|:--|
| **Run script** in **Controls > Setup experience** | Hidden. The page still works at `https://<your-fleet-url>/controls/setup-experience/run-script`. |
| `controls.setup_experience.macos_script` | Works, but isn't documented. `fleetctl gitops` prints a deprecation warning when it's set. |
| `POST`, `GET`, and `DELETE /api/v1/fleet/setup_experience/script` | Work, but aren't documented. Fleet logs a deprecation warning when you add or delete a setup script. |

#### What to do

Move the script to a [script-only package](https://fleetdm.com/guides/deploy-software-packages#script-only-packages):

1. Upload the .sh file as a [custom package](https://fleetdm.com/guides/deploy-software-packages), or add it to `software.packages` with `setup_experience_platform: darwin` ([docs](https://fleetdm.com/docs/configuration/yaml-files#packages)).
2. In the Fleet UI, add the package in **Controls > Setup experience > Install software**. With GitOps, `setup_experience_platform: darwin` does this.
3. Delete the setup script, or remove `macos_script` from your YAML.

Setup experience software installs in alphabetical order, and the setup script runs after all software. To run the package last, give it a name that sorts after your other setup experience software.

GitHub issue: [#53274](https://github.com/fleetdm/fleet/issues/53274)

### Software keys deprecated in host software and setup experience responses

| Fleet 4 | Fleet 5 |
|:--|:--|
| `software_package` in [Get host's software](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-software) and [List setup experience software](https://fleetdm.com/docs/rest-api/rest-api#list-setup-experience-software) | Use `package` (host software) or the `packages` array (setup experience). `software_package` still works, but isn't documented. |
| `app_store_app` in [List setup experience software](https://fleetdm.com/docs/rest-api/rest-api#list-setup-experience-software) | Use the `app_store_apps` array. `app_store_app` still works, but isn't documented. |

#### What to do

Update scripts that read these keys from host software (including `GET /device/:token/software`) and setup experience software responses.

GitHub issues: [#54409](https://github.com/fleetdm/fleet/issues/54409), [#52687](https://github.com/fleetdm/fleet/issues/52687)


<meta name="articleTitle" value="Upgrade to Fleet 5">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="publishedOn" value="2026-11-09">
<meta name="category" value="guides">
<meta name="description" value="Prepare for the upgrade to Fleet 5.">

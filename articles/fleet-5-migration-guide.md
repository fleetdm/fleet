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

| Fleet 4 | Fleet 5 |
|:--|:--|
| `/api/v1/fleet/teams` endpoints, including `/spec/teams`, `/teams/:id`, `/teams/:id/secrets`, `/teams/:id/agent_options`, and `/teams/:id/users` | The same paths under `/api/v1/fleet/fleets` |
| `PATCH /api/v1/fleet/ab_tokens/:id/teams` and `PATCH /api/v1/fleet/vpp_tokens/:id/teams` | `PATCH /api/v1/fleet/ab_tokens/:id/fleets` and `PATCH /api/v1/fleet/vpp_tokens/:id/fleets` |
| `team_id`, `team_name`, `team`, `teams`, `team_ids`, `team_ids_by_name`, and `available_teams` in API requests, responses, and query parameters | `fleet_id`, `fleet_name`, `fleet`, `fleets`, `fleet_ids`, `fleet_ids_by_name`, and `available_fleets` |
| `team_id` and `team_name` as `order_key` values and hosts CSV columns | `fleet_id` and `fleet_name` |
| `default_team`, `macos_team`, `ios_team`, `ipados_team`, and `byod_team` in Apple Business settings, and `macos_team_id`, `ios_team_id`, `ipados_team_id`, and `byod_team_id` | `default_fleet`, `macos_fleet`, `ios_fleet`, `ipados_fleet`, and `byod_fleet`, and `macos_fleet_id`, `ios_fleet_id`, `ipados_fleet_id`, and `byod_fleet_id` |
| `fleetctl get teams`, `--team`, `--policies-team`, `--delete-other-teams`, and `generate-gitops --team no-team` | `fleetctl get fleets`, `--fleet`, `--policies-fleet`, `--delete-other-fleets`, and `generate-gitops --fleet unassigned` |
| Activity types `created_team`, `deleted_team`, `applied_spec_team`, `changed_user_team_role`, and `deleted_user_team_role` | `created_fleet`, `deleted_fleet`, `applied_spec_fleet`, `changed_user_fleet_role`, and `deleted_user_fleet_role` |

What to do:

1. Search your scripts, integrations, and GitOps repo for the old names:
   ```sh
   grep -rnE '/fleet/teams|/teams\b|team_ids?\b|team_name|available_teams|default_team|(macos|ios|ipados|byod)_team|--team\b|--policies-team|--delete-other-teams|no-team|get teams' .
   ```
2. Replace each match with its Fleet 5 name. The `/fleets` endpoints and `--fleet` flags already work in Fleet 4, so you can do this before you upgrade.
3. Update anything that reads the hosts CSV or filters activities, like SIEM rules, to use the new names.

Issue: [#52795](https://github.com/fleetdm/fleet/issues/52795)


### SSO: Fleet role SAML attribute renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `FLEET_JIT_USER_ROLE_TEAM_<ID>` | `FLEET_JIT_USER_ROLE_FLEET_<ID>` |

What to do: Rename the attribute in your identity provider (IdP) before you upgrade. Fleet 4 already accepts the new name. Fleet 5 ignores the old one, so new users who log in with SSO get the global Observer role, and existing users keep the roles they have now.

Issue: [#53925](https://github.com/fleetdm/fleet/issues/53925)


### Policy endpoints merged

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET`, `POST`, and `PATCH /api/v1/fleet/global/policies` and `/global/policies/:id` | `GET`, `POST`, and `PATCH /api/v1/fleet/policies` and `/policies/:id` |
| `/api/v1/fleet/fleets/:id/policies` and `/fleets/:fleet_id/policies/:policy_id` (and the `/team/...` and `/teams/...` paths) | `/api/v1/fleet/policies` with `fleet_id`, and `/api/v1/fleet/policies/:id` |
| `POST /api/v1/fleet/global/policies/delete` and `/fleets/:fleet_id/policies/delete` | `POST /api/v1/fleet/policies/delete` |
| `failing_policies_webhook.policy_ids` (API and YAML) | `webhooks_and_tickets_enabled` on each policy |

What to do:

1. Search for the old paths: `grep -rnE '/fleet/global/policies|/(teams?|fleets)/[^/]+/policies' .`
2. Use `/api/v1/fleet/policies`. To work with a fleet's policies, pass `fleet_id`. Without it, you get global policies.
3. Move the IDs in `failing_policies_webhook.policy_ids` to `webhooks_and_tickets_enabled: true` on each policy.

Issue: [#52900](https://github.com/fleetdm/fleet/issues/52900)


### One automation per policy

| Fleet 4 | Fleet 5 |
|:--|:--|
| A policy can have more than one of `install_software`, `run_script`, `resend_configuration_profile`, `calendar_events_enabled`, `conditional_access_enabled`, and `webhooks_and_tickets_enabled` | One automation per policy. `fleetctl gitops` fails if a policy has more than one. |

What to do: Before you upgrade, find policies with more than one automation and move the extra automations to new policies.

Issue: [#52900](https://github.com/fleetdm/fleet/issues/52900)


### Schedule endpoints replaced by reports

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/global/schedule` and `/fleets/:id/schedule` | `GET /api/v1/fleet/reports` (with `fleet_id` for a fleet). Scheduled reports have an `interval` greater than `0`. |
| `POST /api/v1/fleet/global/schedule` and `/fleets/:id/schedule`, which copied the report and scheduled the copy | `POST /api/v1/fleet/reports` with `interval`, `platform`, `min_osquery_version`, and `logging` (and `fleet_id` for a fleet). No copy. |
| `PATCH` and `DELETE /api/v1/fleet/global/schedule/:id` and `/fleets/:id/schedule/:report_id` | `PATCH` and `DELETE /api/v1/fleet/reports/:id` |

The `/api/latest/fleet/schedule`, `/team/:id/schedule`, and `/teams/:id/schedule` paths are also removed.

What to do:

1. Search for the old paths: `grep -rnE '/schedule\b' .`
2. Create or update reports with an `interval` instead. To unschedule a report and keep it, send `PATCH` with `interval` set to `0`.

Issue: [#54461](https://github.com/fleetdm/fleet/issues/54461)


### Built-in labels can't target reports

| Fleet 4 | Fleet 5 |
|:--|:--|
| `POST /api/v1/fleet/reports` accepts built-in labels in `labels_include_any` or `labels_include_all` | Returns an error |

What to do: Target the report by platform instead.

Issue: [#53800](https://github.com/fleetdm/fleet/issues/53800)


### MDM endpoints moved out of /mdm

| Fleet 4 | Fleet 5 |
|:--|:--|
| `/api/v1/fleet/mdm/commands`, `/mdm/commandresults`, `/mdm/commands/run`, and `/mdm/apple/commands` | `/api/v1/fleet/commands`, `/commands/results`, and `/commands/run` |
| `PATCH /api/v1/fleet/mdm/hosts/:id/unenroll` | `PATCH /api/v1/fleet/hosts/:id/mdm` |
| `/api/v1/fleet/mdm/hosts/:id/lock`, `/wipe`, `/profiles`, and `/encryption_key` | `/api/v1/fleet/hosts/:id/lock`, `/wipe`, `/configuration_profiles`, and `/encryption_key` |
| `/api/v1/fleet/mdm/disk_encryption/summary` (and `/mdm/apple/filevault/summary`) | `/api/v1/fleet/disk_encryption` |
| `/api/v1/fleet/mdm/profiles`, `/mdm/profiles/:profile_uuid`, and `/mdm/profiles/summary` (and the `/mdm/apple/profiles` paths) | `/api/v1/fleet/configuration_profiles`, `/configuration_profiles/:profile_uuid`, and `/configuration_profiles/summary` |
| `PATCH /api/v1/fleet/mdm/apple/setup` | `PATCH /api/v1/fleet/setup_experience` |
| `GET /api/v1/fleet/mdm/apple` | `GET /api/v1/fleet/apns` |
| `/api/v1/fleet/mdm/apple/enrollment_profile` and `/mdm/manual_enrollment_profile` | `/api/v1/fleet/enrollment_profiles/automatic` and `/enrollment_profiles/manual` |
| `/api/v1/fleet/mdm/bootstrap` paths (and `/mdm/apple/bootstrap`) | The same paths under `/api/v1/fleet/bootstrap` |
| `/api/v1/fleet/mdm/setup/eula` paths (and `/mdm/apple/setup/eula`) | The same paths under `/api/v1/fleet/setup_experience/eula` |

What to do: Search for `/fleet/mdm/` in your scripts and integrations, and replace each path with its Fleet 5 path.

Issue: [#53563](https://github.com/fleetdm/fleet/issues/53563)


### List MDM commands requires a host

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/commands` without `host_identifier` | `host_identifier` is required. `fleetctl get mdm-commands` requires `--host`. |

What to do: Pass `host_identifier` (or `--host`) on every call.

Issue: [#53146](https://github.com/fleetdm/fleet/issues/53146)


### Other endpoints moved

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/hosts/identifier/:identifier` | `GET /api/v1/fleet/hosts_identifier/:identifier` |
| `GET /api/v1/fleet/scripts/batch/summary/:batch_execution_id` | `GET /api/v1/fleet/scripts/batch/:batch_execution_id` |
| `POST /api/v1/fleet/hosts/:host_id/configuration_profiles/resend/:profile_uuid` | `POST /api/v1/fleet/hosts/:id/configuration_profiles/:profile_uuid/resend` |

What to do: Search for `hosts/identifier/`, `scripts/batch/summary/`, and `configuration_profiles/resend/`, and replace each path.

Issue: [#53080](https://github.com/fleetdm/fleet/issues/53080)


### Apple Business Manager renamed to Apple Business

| Fleet 4 | Fleet 5 |
|:--|:--|
| `/api/v1/fleet/abm_tokens` endpoints, `GET /api/v1/fleet/abm`, and `GET /api/v1/fleet/mdm/apple_bm` | `/api/v1/fleet/ab_tokens` endpoints |
| `GET /api/v1/fleet/mdm/apple/abm_public_key` | `GET /api/v1/fleet/mdm/apple/ab_public_key` |
| `abm_token`, `abm_tokens`, and `host_dep_assignment.abm_token_id` in responses | `ab_token`, `ab_tokens`, and `ab_token_id` |
| `mdm.apple_business_manager`, `mdm.apple_bm_enabled_and_configured`, and `mdm.apple_bm_terms_expired` | `mdm.apple_business`, `mdm.apple_business_enabled_and_configured`, and `mdm.apple_business_terms_expired` |
| `mdm.apple_bm_default_team` in requests | Removed |
| `org_settings.mdm.apple_business_manager` in YAML | `org_settings.mdm.apple_business` |
| `fleetctl get mdm-apple-bm` and `fleetctl generate mdm-apple-bm` (and the `mdm_apple_bm` alias) | `fleetctl get mdm-ab` and `fleetctl generate mdm-ab` |
| Server options `mdm.apple_bm_server_token`, `mdm.apple_bm_cert`, and `mdm.apple_bm_key` (and their `_bytes` versions), and `FLEET_MDM_APPLE_BM_*` environment variables | `mdm.apple_business_server_token`, `mdm.apple_business_cert`, and `mdm.apple_business_key` (and their `_bytes` versions), and `FLEET_MDM_APPLE_BUSINESS_*` |

What to do:

1. Search your scripts, GitOps repo, and server configuration: `grep -rnE 'abm|apple_bm|APPLE_BM|apple_business_manager|mdm-apple-bm' .`
2. Replace each match with its Fleet 5 name. `mdm-ab` already works in Fleet 4.
3. Rename the server options and environment variables in the same deploy that upgrades to Fleet 5. Fleet 5 doesn't start if an old one is still set.

Issue: [#53272](https://github.com/fleetdm/fleet/issues/53272)


### macos_setup renamed to setup_experience

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.macos_setup` in `GET` and `PATCH /api/v1/fleet/config` and fleet settings | `mdm.setup_experience` |
| `controls.macos_setup` in YAML | `controls.setup_experience`. `fleetctl get config` no longer prints `macos_setup`. |

What to do: Search for `macos_setup` and rename it to `setup_experience`.

Issue: [#53894](https://github.com/fleetdm/fleet/issues/53894)


### Setup experience script removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `POST`, `GET`, and `DELETE /api/v1/fleet/setup_experience/script` | Removed |
| `controls.setup_experience.macos_script` in YAML | A script-only package under `software.packages` with `setup_experience` or `setup_experience_platform` |
| The **Run script** tab in **Controls > Setup experience** | Removed |

What to do: Add your setup script as a script-only package that runs during setup experience, then remove `macos_script` and any calls to `/setup_experience/script`.

Issue: [#53274](https://github.com/fleetdm/fleet/issues/53274)


### Setup experience and Recovery Lock settings renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `require_all_software_macos` and `require_all_software_windows` (API and `controls.setup_experience` in YAML) | `macos_require_all_software` and `windows_require_all_software` |
| `enable_recovery_lock_password` (`mdm` in the API and `controls` in YAML) | `macos_enable_recovery_lock_password` |

What to do: Search for these keys and rename them.

Issue: [#53872](https://github.com/fleetdm/fleet/issues/53872)


### Organization logo settings renamed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `org_logo_url` (API and `org_info` in YAML) | `org_logo_url_dark_mode` |
| `org_logo_url_light_background` | `org_logo_url_light_mode` |

What to do: Rename both keys. Fleet 4 already accepts the new names.

Issue: [#52800](https://github.com/fleetdm/fleet/issues/52800)


### all_linux_count removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `all_linux_count` in the [Get hosts summary](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-summary) response | Removed |

What to do: Call [Count hosts](https://fleetdm.com/docs/rest-api/rest-api#count-hosts) with `label_id` set to the ID of the built-in "All Linux" label. You can find the ID in `builtin_labels` in the Get hosts summary response.

Issue: [#53872](https://github.com/fleetdm/fleet/issues/53872)


### os_name and os_version are separate

| Fleet 4 | Fleet 5 |
|:--|:--|
| `os_name` and `os_version` combined in [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) and [Get hosts report in CSV](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-report-in-csv) | Separate `os_name` and `os_version` keys |

What to do: Update code that parses `os_name` to read the version from `os_version`.

Issue: [#53877](https://github.com/fleetdm/fleet/issues/53877)


### Windows wipe options moved

| Fleet 4 | Fleet 5 |
|:--|:--|
| `windows` object with `wipe_type` in [Wipe host](https://fleetdm.com/docs/rest-api/rest-api#wipe-host) | Removed |

What to do: Use `POST /api/v1/fleet/hosts/:id/reset` instead.

Issue: [#55003](https://github.com/fleetdm/fleet/issues/55003)


### YAML-only lists removed from configuration responses

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.apple_settings.configuration_profiles`, `mdm.windows_settings.configuration_profiles`, `mdm.setup_experience`, `mdm.volume_purchasing_program`, and `scripts` in [Get configuration](https://fleetdm.com/docs/rest-api/rest-api#get-configuration) and [Get fleet](https://fleetdm.com/docs/rest-api/rest-api#get-fleet) | Removed |

What to do: These lists only included settings applied with YAML. Call the list endpoint for each feature instead.

Issue: [#53932](https://github.com/fleetdm/fleet/issues/53932)


### detail_query_overrides removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `features.detail_query_overrides` (API and YAML) | Removed |

What to do: Remove `detail_query_overrides` from your YAML and API calls.

Issue: [#33769](https://github.com/fleetdm/fleet/issues/33769)


### Software API changes

| Fleet 4 | Fleet 5 |
|:--|:--|
| `GET /api/v1/fleet/software`, `/software/:id`, and `/software/count` | `GET /api/v1/fleet/software/versions` and `/software/versions/:id` |
| `software_id` filter on `GET /api/v1/fleet/hosts` | `software_version_id` or `software_title_id` |
| `software_package` object in software title responses | `packages` array |
| `app_store_app` object in software title responses | `app_store_apps` array |
| `browser` in host, software title, and software version responses | `extension_for` |

`fleetctl get software --json` and `--yaml` output changes the same way.

What to do: Search for `/fleet/software`, `software_id`, `software_package`, `app_store_app`, and `browser`, and update each request and the code that reads responses.

Issues: [#54408](https://github.com/fleetdm/fleet/issues/54408), [#54410](https://github.com/fleetdm/fleet/issues/54410), [#54409](https://github.com/fleetdm/fleet/issues/54409), [#52687](https://github.com/fleetdm/fleet/issues/52687), [#53395](https://github.com/fleetdm/fleet/issues/53395)


### Label platforms: centos and ubuntu replaced by linux

| Fleet 4 | Fleet 5 |
|:--|:--|
| `centos` and `ubuntu` for label platform targeting | `linux` |

What to do: Change `centos` and `ubuntu` label platforms to `linux`.

Issue: [#52905](https://github.com/fleetdm/fleet/issues/52905)


### Host identifiers for personal and Android hosts

| Fleet 4 | Fleet 5 |
|:--|:--|
| `uuid` and `hardware_serial` for personal (BYOD) iOS, iPadOS, and Android hosts | Empty strings. `enrollment_id` has the enrollment ID. |
| `uuid` for company-owned Android hosts (a hash of the brand and serial) | Empty string. `hardware_serial` doesn't change, and `enrollment_id` is `null`. |

What to do: Identify personal hosts by `enrollment_id`, and company-owned Android hosts by `hardware_serial`.

Issue: [#53797](https://github.com/fleetdm/fleet/issues/53797)


### Android Lock resets the PIN

| Fleet 4 | Fleet 5 |
|:--|:--|
| Lock host (`POST /api/v1/fleet/hosts/:id/lock`) locks the screen of an Android host | Lock host resets the PIN to a random six-digit PIN and locks the host |

What to do: To get the PIN, use Unlock host (`POST /api/v1/fleet/hosts/:id/unlock`) or `fleetctl mdm unlock --host=<host>`. Both now work for Android hosts. Update any end user instructions.

Issue: [#49909](https://github.com/fleetdm/fleet/issues/49909)


### SCIM user delete requires an admin

| Fleet 4 | Fleet 5 |
|:--|:--|
| `DELETE /api/v1/fleet/scim/Users/:id` works with a maintainer's API token | Returns an error |

What to do: Use an admin's API token for your SCIM integration.

Issue: [#53472](https://github.com/fleetdm/fleet/issues/53472)


### GitOps users can't run custom MDM commands

| Fleet 4 | Fleet 5 |
|:--|:--|
| GitOps users can call [Run MDM command](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command) | GitOps users get a 403 |

What to do: Run custom MDM commands with a different API-only user.

Issue: [#53145](https://github.com/fleetdm/fleet/issues/53145)


### Omitted GitOps settings reset to defaults

| Fleet 4 | Fleet 5 |
|:--|:--|
| If your YAML leaves out `gitops`, `host_expiry_settings`, or `smtp_settings`, they keep their current values | They reset to their defaults |

What to do: Before you upgrade, add these settings to your YAML with the values you use today. To get the current values, run `fleetctl generate-gitops`.

Issue: [#53899](https://github.com/fleetdm/fleet/issues/53899)


### Manual labels don't accept hostname

| Fleet 4 | Fleet 5 |
|:--|:--|
| `hostname` as a host identifier in the `hosts` key of manual labels (GitOps and API) | Not accepted |

What to do: Identify hosts in manual labels by something other than `hostname`.

Issue: [#53924](https://github.com/fleetdm/fleet/issues/53924)


### scripts_disabled removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `server_settings.scripts_disabled` and the **Disable scripts** checkbox in **Settings > Organization settings > Advanced options** | `fleetctl gitops` fails if it's set. Fleet 5 turns script features back on for instances that had `scripts_disabled: true`. |

What to do: Remove `scripts_disabled` from your YAML. To keep scripts off, deploy fleetd without `--enable-scripts`.

Issue: [#53897](https://github.com/fleetdm/fleet/issues/53897)


### Package YAML files must be lists

| Fleet 4 | Fleet 5 |
|:--|:--|
| A package YAML file can hold one package that isn't in a list | Packages must be in a list. `fleetctl gitops` fails otherwise. |

What to do: In each package YAML file, put packages in a list, even if there's only one.

Issue: [#53378](https://github.com/fleetdm/fleet/issues/53378)


### Certificate authority names must be unique

| Fleet 4 | Fleet 5 |
|:--|:--|
| Certificate authorities (CAs) of different types can share a name | Every CA needs a unique name |

What to do: Before you upgrade, call `GET /api/v1/fleet/certificate_authorities`, find CAs that share a `name`, and rename them. Update the configuration profiles that reference them.

Issue: [#56271](https://github.com/fleetdm/fleet/issues/56271)


### Fleet names must be unique

| Fleet 4 | Fleet 5 |
|:--|:--|
| Fleets can have names that differ only by case or special characters | Names must be unique after Fleet trims whitespace, lowercases the name, and removes special characters. Applies when you create or edit a fleet in the UI or API. GitOps already blocks this. |

What to do: Run `fleetctl get fleets`, find names that match after you ignore case, whitespace, and special characters (for example `Workstations` and `workstations!`), and rename one in each pair.

Issue: [#53790](https://github.com/fleetdm/fleet/issues/53790)


### Packs removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `/packs` endpoints, including the packs spec endpoints | Removed. They return `410 Gone`. |
| `fleetctl get packs`, `fleetctl upgrade-packs`, and `fleetctl apply` with `kind: pack` | Removed |
| Activity types `created_pack`, `edited_pack`, `deleted_pack`, and `applied_spec_pack` | Fleet 5 keeps existing activities with these types but doesn't create new ones |

What to do: Before you upgrade, run `fleetctl upgrade-packs -o packs-to-reports.yml` on Fleet 4, review the file, and apply it with `fleetctl apply -f packs-to-reports.yml`. 2017 packs stop running on hosts after the upgrade.

Issue: [#53470](https://github.com/fleetdm/fleet/issues/53470)


### Disk encryption activity types merged

| Fleet 4 | Fleet 5 |
|:--|:--|
| `enabled_macos_disk_encryption` and `disabled_macos_disk_encryption` | `edited_disk_encryption_settings` |

What to do: Update anything that filters or parses activities, like SIEM rules, to use the new type.

Issue: [#53481](https://github.com/fleetdm/fleet/issues/53481)


### fleetd process name is Fleet

| Fleet 4 | Fleet 5 |
|:--|:--|
| `fleet-osquery` (Linux), `Fleet osquery` (Windows), and `orbit` (macOS) | `Fleet` on all platforms |

What to do: Add `Fleet` to EDR, monitoring, and firewall rules that match the old names. Keep the old names until every host runs the new fleetd.

Issue: [#56803](https://github.com/fleetdm/fleet/issues/56803)


### fleetctl package file names changed

| Fleet 4 | Fleet 5 |
|:--|:--|
| File names like `fleet-osquery_1.61.0_amd64.deb` | `fleet-{platform}{-arch}-{fleet-name}.ext`, for example `fleet-macos-workstations.pkg`, or `fleet-windows-amd64.msi` when there's no fleet name. Global enroll secrets use `unassigned`. |

What to do: Update scripts and CI that use the file names. `fleetctl package` leaves out the fleet name if it can't find it, for example when you're not logged in to `fleetctl`.

Issue: [#53365](https://github.com/fleetdm/fleet/issues/53365)


### Static SCEP challenge off by default

| Fleet 4 | Fleet 5 |
|:--|:--|
| `mdm.apple_scep_static_challenge_enabled` defaults to `true`, and the manual enrollment profile endpoint works | Defaults to `false`, and the endpoint returns a `400` |

What to do: If you use the manual enrollment profile endpoint, set `mdm.apple_scep_static_challenge_enabled` (`FLEET_MDM_APPLE_SCEP_STATIC_CHALLENGE_ENABLED`) to `true`. That turns the static SCEP challenge back on for all Apple MDM enrollments, so anyone with the challenge can turn on MDM for any host. Previously downloaded profiles stop working. Hosts that already have MDM on aren't affected.

Issue: [#54190](https://github.com/fleetdm/fleet/issues/54190)


### Async host processing removed

| Fleet 4 | Fleet 5 |
|:--|:--|
| `osquery_enable_async_host_processing` and the `osquery_async_host_*` options (and their `FLEET_OSQUERY_*` environment variables) | Removed. Fleet always processes label membership, policy membership, last seen, and scheduled report stats synchronously. |

What to do: Remove these options from your server configuration. Data still waiting in Redis when you upgrade is dropped and corrected on the next check-in.

Issue: [#54009](https://github.com/fleetdm/fleet/issues/54009)


### osquery_min_software_last_opened_at_diff removed

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

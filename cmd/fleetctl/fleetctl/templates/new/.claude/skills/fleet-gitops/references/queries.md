# Policies, reports, labels, and scripts

Three YAML shapes share one engine: an osquery SQL statement that runs on the host. What differs is what Fleet does with the rows.

| Shape | Rows mean | Key fields | Where |
|---|---|---|---|
| **Policy** | At least one row = pass, zero rows = fail | `name`, `query`, `description`, `resolution`, `platform`, `critical`, labels, automations | `policies:` in `default.yml` (global, no automations) or a fleet file |
| **Report** | Rows are collected every `interval` seconds and sent to the log destination | `name`, `query`, `description`, `platform`, `interval`, `logging`, `automations_enabled`, `observer_can_run`, `min_osquery_version`, `discard_data`, `labels_include_any` | `reports:` in `default.yml` or a fleet file (not `unassigned.yml`) |
| **Label (dynamic)** | A host is in the label while the query returns rows | `name`, `description`, `query`, `platform`, `label_membership_type: dynamic` | `labels:` in `default.yml` (global) or a fleet file |

Items can be inline or in a separate file referenced with `path:` (a list of items) or `paths:` (a glob of such files). Match whatever the repo does; most repos keep one file per topic under a `policies/`, `reports/`, or `labels/` folder per platform.

## Use only tables that exist on the target platform

A query against a table that doesn't exist on a platform errors on the host: the policy fails everywhere, the report collects nothing, the label stays empty. The source of truth is Fleet's osquery schema (osquery tables plus the tables fleetd adds), which is what https://fleetdm.com/tables renders.

Look a table up before using it:

- `python3 <skill>/scripts/validate.py --table <name>` prints its platforms and columns (from the same schema the validator uses).
- https://fleetdm.com/tables/<name> for the prose, examples, and notes.
- `jq '.[] | select(.name=="<name>") | {platforms, columns: [.columns[].name]}' schema/osquery_fleet_schema.json` when you're inside the Fleet source repo.
- `contour osquery table <name>` or `contour osquery search <keyword>` if contour is installed.

Idiomatic sources for common checks:

| Question | macOS | Windows | Linux |
|---|---|---|---|
| Is an app installed, and which version? | `apps` by `bundle_identifier` (never by display name); versions in `bundle_short_version` | `programs` by `name` (the Add/Remove Programs DisplayName, which is often versioned, so prefer `LIKE 'Name%'`) | `deb_packages` / `rpm_packages` by `name` |
| OS version | `os_version.version` | `os_version.version`, `build` | `os_version` plus `platform` / `platform_like` for the distro |
| A setting enforced by MDM | `managed_policies` (domain, name, value) or `plist` | `registry` under `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\PolicyManager\current\device\<Area>` | config files via `augeas` or `file_lines` |
| Is a profile or declaration installed | `macos_profiles.identifier` | n/a | n/a |
| Disk encryption | `filevault_status` (prefer Fleet's `enable_disk_encryption` for enforcement) | `bitlocker_info` | `disk_encryption` joined to `mounts` |
| Firewall | `alf.global_state` | `windows_security_products`, `registry` | `iptables` |
| Per-user configuration | `users` joined to `file_lines` / `plist` on `directory` | `users` joined to `file_lines` | same, `uid >= 1000` |

Compare versions with `version_compare(a, b)` (fleetd provides it); don't string-compare dotted versions. The fail-if-outdated shape is `SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'x' AND version_compare(bundle_short_version, '1.2.3') < 0);`, which also passes on hosts that don't have the app, so pair it with a label when you only want hosts that have it installed.

Write one policy per platform rather than one query that tries to run everywhere: the table names differ, the resolution text differs, and `platform` accepts `darwin`, `windows`, `linux`, `chrome` (comma-separated is allowed but rarely right). A report that legitimately spans platforms (for example `system_info`) can list several.

## Policies

Required: `name` (unique within its fleet; mirror the repo's naming, such as a platform suffix only if existing names carry one), `query`, `platform`. Strongly expected: `description` (what and why, for admins) and `resolution` (what the end user should do, in the repo's voice; Fleet Desktop shows it to them). `critical: true` marks it for the critical-policy views. `calendar_events_enabled` needs the Google Calendar integration and only works in a fleet file.

Automations, all fleet-file only, each pointing at something defined in the same fleet:

- `run_script: {path: ../scripts/fix.sh}` runs the script on hosts that fail. The script must also be listed under that fleet's `controls.scripts`. The path is relative to the policy file.
- `install_software:` with exactly one of `package_path` (a package YAML in that fleet's `software.packages`), `fleet_maintained_app_slug` (a slug in that fleet's `fleet_maintained_apps`), `app_store_id`, or `hash_sha256`. Pair it with a "software installed" check such as `SELECT 1 FROM apps WHERE bundle_identifier = '...'`.
- `resend_configuration_profile: "<profile name>"` re-pushes a profile listed in that fleet's `controls`.
- `continuous_automations_enabled: true` keeps re-running the automation on every policy failure instead of once; patch policies set it automatically.

Patch policies (`type: patch`) keep a Fleet-maintained app current and generate their own query; see [software.md](software.md).

Path-constrained tables (`file`, `augeas`, `hash`, `plist`, `file_lines`, …) take their paths from the `WHERE` clause, and osquery's wildcards aren't SQL's: in `path LIKE`, a single `%` matches one directory level and `%%` matches recursively. `path LIKE '/etc/ssh/sshd_config.d/%'` on `augeas` returns only the drop-in file nodes, never the directives inside them, so a check that looks correct silently passes. Use `%%` when you mean "everything under here".

Before shipping a policy, check the SQL as cheaply as the situation allows and stop there: `osqueryi` on a host of the right platform if one is at hand, otherwise a quick logic check of the SQL against a few fixture rows in `sqlite3`, or ask the user to run it as a live query in Fleet. Don't build Docker or VM harnesses unless the user asks; a reviewer with a real host can confirm in a minute what a harness only approximates. Say in your report exactly what ran and what didn't; a query that parses isn't a query that's right.

## Reports

`interval` is seconds (3600 = hourly; use the repo's existing cadence for similar data). `logging: snapshot` sends every row each run; `differential` sends only changes. `automations_enabled: true` forwards results to the log destination; without it the latest results are only kept in Fleet's report view (`discard_data: true` keeps nothing). `observer_can_run: true` lets observers run it on demand. `min_osquery_version` gates hosts whose osquery is too old for a table. A query that errors on a host produces no rows and no error you'll see in the YAML, so the table rules above apply here too.

## Labels

- **Dynamic** (`label_membership_type: dynamic`, the default): `query` plus optional `platform` (`darwin`, `windows`, `linux`, `ubuntu`, `centos`). Name them for what they select, the way the repo does (for example "Macs with Docker Desktop installed", "x86-based Windows hosts").
- **Manual**: `hosts:` is a list of host ids, serials, or UUIDs (quoted). Omitting `hosts` preserves the current membership; an empty list clears it.
- **Host vitals**: `criteria: {vital: <name>, value: <value>}`, where the vital is an IdP attribute (`end_user_idp_department`, `end_user_idp_group`, …) or a custom host vital (`vital: custom_host_vital` plus `custom_host_vital_id`).

Exactly one of `query`, `hosts`, or `criteria`. Names are unique across the whole instance, `description` is capped at 255 characters, and a label used anywhere in the config must be declared in a `labels:` section (see the rules in SKILL.md). Scope things with existing labels before adding new ones; when you add one, register it where the repo registers labels (`default.yml` lists them individually or with a `paths:` glob).

## Scripts

Scripts live in a scripts folder per platform and are listed in a fleet's `controls.scripts` (individually or with a `paths:` glob), which makes them runnable from the host page and usable by `run_script` automations.

- `.sh` runs as root on macOS and Linux. macOS ships bash 3.2 and zsh; `/bin/sh` is safest. Linux hosts may lack bash entirely.
- `.ps1` runs as SYSTEM on Windows, 64-bit PowerShell. Nothing in the user's profile is visible (no mapped drives, no per-user registry hive loaded), and there's no interactive desktop. For per-user work, enumerate `HKEY_USERS` or schedule a task in the user's context.
- `.py` runs with the host's Python where one exists.
- Exit non-zero to signal failure; the exit code and output appear in Fleet. Keep scripts idempotent, because a `run_script` automation reruns them every time the policy fails again. The default execution timeout is 300 seconds (`agent_options.script_execution_timeout` raises it).
- `$FLEET_SECRET_NAME` inside a script is replaced by Fleet at delivery; `$FLEET_VAR_HOST_…` likewise for host attributes. Don't put real values in the file.

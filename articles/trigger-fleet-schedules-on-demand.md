# Trigger Fleet server schedules on demand

The Fleet server does much of its work on a timer. It checks for vulnerabilities every hour, sends policy webhooks once a day, and syncs the Fleet-maintained app catalog hourly. Usually that's fine, but sometimes you can't wait. Maybe you just set up a failing-policy webhook and want to test it now, not tomorrow. `fleetctl trigger` starts a schedule's next run right away. This guide covers how to use it, every schedule name you can trigger, how often each one runs, and when triggering helps.

The names and intervals below match Fleet 4.95. Schedules come and go between releases, so ask your own server for its list (see [List the schedules your server runs](#list-the-schedules-your-server-runs)).

## Prerequisites

- `fleetctl` installed and logged in to your Fleet server with `fleetctl login`.
- The global admin role. Other roles get a permission error.

## Why trigger a schedule early

Triggering adds an extra run now. It doesn't reset the timer, so the next scheduled run still happens on time. Common reasons to trigger early:

- Test automations without waiting a day: failing-policy and host status webhooks and tickets run on the `automations` schedule, which runs every 24 hours by default. Trigger it to confirm a new webhook URL or Jira/Zendesk integration works.
- See new vulnerability results: after you turn on software inventory, change vulnerability settings, or hear about a new CVE, trigger `vulnerabilities` to rescan right away instead of waiting up to an hour.
- Pick up a new Fleet-maintained app or version: Fleet publishes new apps and versions to the catalog often. Trigger `maintained_apps` to pull the latest catalog without waiting for the hourly sync.
- Check host expiry and certificate renewal: host expiry, SCEP certificate renewal, and most data cleanup run on `cleanups_then_aggregation`. Trigger it to confirm a host expiry change or a renewal without waiting an hour.
- Sync an identity or device source after a change: after you change groups in Google Workspace or register devices in Windows Autopilot, trigger `google_workspace_sync` or `microsoft_autopilot_sync`.
- Refresh dashboard numbers: software host counts update with `vulnerabilities`, and dashboard charts update with `chart_data_collection`.

Schedules that already run every few seconds or minutes, like the MDM profile managers and workers, rarely need a trigger.

> **Warning:** `vulnerabilities` and `cleanups_then_aggregation` do heavy database work. On large deployments, trigger them outside peak hours.

## Trigger a schedule

Run `fleetctl trigger` with the schedule name:

```bash
fleetctl trigger --name vulnerabilities
```

If the request goes through, `fleetctl` prints:

```
[+] Sent request to trigger vulnerabilities schedule
```

This means the server got the request, not that the run is finished. Most schedules finish in seconds. `vulnerabilities` can take several minutes or more.

## List the schedules your server runs

Run `fleetctl trigger` without `--name`:

```bash
fleetctl trigger
```

The server replies with every schedule it runs:

```
[!] Name must be specified; supported trigger names are activities_streaming, apple_mdm_apns_sweep, apple_mdm_dep_profile_assigner, ...
```

The list depends on your Fleet version, license, and server configuration, so it's the most reliable reference for your deployment. A name that's not on the list returns `Invalid name` with the same list.

## Understand "conflicts with current status"

Only one run of a schedule happens at a time. If a run is already going, the trigger is rejected:

```
[!] Conflicts with current status of vulnerabilities schedule: scheduled run started 2m14.512s ago
```

Fleet doesn't queue a second run. Wait for the current run to finish, then trigger again if you need a fresh one. The message tells you whether the current run is `scheduled` (from the timer) or `triggered` (by someone else).

If a Fleet server crashes in the middle of a run, the conflict can stick around for up to two hours. After that, Fleet marks the stuck run as expired and accepts triggers again.

## Schedule reference

**Free** schedules run on every Fleet server. **Premium** schedules only run with a Fleet Premium license. "Configurable" means you can change the interval with the setting listed.

### Core

| Name | What it does | Default interval | Availability |
|---|---|---|---|
| `cleanups_then_aggregation` | Expires hosts based on host expiry settings, renews SCEP and other MDM-managed certificates, updates policy and report stats, cleans up old activities, scripts, software installers, and MDM commands, and more. | 1 hour | Free |
| `frequent_cleanups` | Cleans up expired live report data in Redis. | 15 minutes | Free. Only runs when `server.frequent_cleanups_enabled` is on (off by default). |
| `query_results_cleanup` | Deletes report results that go over the per-report row limit and removes stale results. | 1 minute | Free |
| `usage_statistics` | Sends anonymous usage statistics, if enabled. | 1 hour | Free |
| `vulnerabilities` | Matches host software against vulnerability data, sends vulnerability webhooks and tickets, updates OS versions, and recalculates software host counts. | Configurable ([`vulnerabilities.periodicity`](https://fleetdm.com/docs/configuration/fleet-server-configuration#periodicity), default 1 hour) | Free |
| `automations` | Sends failing-policy and host status webhooks, and creates Jira and Zendesk tickets for failing policies. | Configurable ([`webhook_settings.interval`](https://fleetdm.com/docs/configuration/yaml-files#webhook-settings), default 24 hours) | Free |
| `integrations` | Processes queued background jobs, like Jira and Zendesk tickets and macOS Setup Assistant profile updates. | Every minute (each run can last up to 10 minutes) | Free |
| `host_vitals_label_membership` | Updates host membership for labels based on host vitals. | 5 minutes | Free |
| `chart_data_collection` | Collects data for dashboard charts. | 1 hour | Free |
| `upcoming_activities_maintenance` | Unblocks host activity queues and clears stuck App Store app installs. | 10 minutes | Free |
| `fleet_initiated_activities_release` | Releases queued policy automation installs and scripts to hosts at the rate set by [`activity.fleet_initiated_release_per_minute`](https://fleetdm.com/docs/configuration/fleet-server-configuration#activity-fleet-initiated-release-per-minute). | 1 minute | Free. Doesn't run when that setting is `0`. |
| `scheduled_batch_activities` | Starts scripts scheduled to run on many hosts. | 2 minutes | Free |
| `batch_activity_completion_checker` | Marks batch script runs as complete. | 5 minutes | Free |
| `activities_streaming` | Streams activities to your audit log destination. | 5 minutes | Premium. Only runs when [`activity.enable_audit_log`](https://fleetdm.com/docs/configuration/fleet-server-configuration#activity-enable-audit-log) is on. |
| `calendar` | Creates and updates calendar events for hosts failing policies. | 5 minutes | Premium |
| `google_workspace_sync` | Syncs users and groups from Google Workspace. | 5 minutes | Premium |
| `end_user_notifications` | Sends due end user notifications and expires old ones. | 1 minute | Premium |

> **Note:** If vulnerability processing runs on a dedicated server (with [`vulnerabilities.disable_schedule`](https://fleetdm.com/docs/configuration/fleet-server-configuration#disable-schedule) set to `true` on your other instances), you can still trigger `vulnerabilities` from any instance. The dedicated server picks up the request within 60 seconds. Learn more in [Vulnerability processing](https://fleetdm.com/guides/vulnerability-processing).

### Software

| Name | What it does | Default interval | Availability |
|---|---|---|---|
| `maintained_apps` | Syncs the Fleet-maintained app catalog. | 1 hour | Premium |
| `maintained_apps_auto_update` | Moves each Fleet-maintained app to the newest version its pin allows. | 1 hour | Premium |
| `windows_maintained_app_titles` | Merges Windows software titles that include the version in their name into the matching Fleet-maintained app. | 1 hour | Premium |
| `refresh_vpp_app_versions` | Updates App Store (VPP) app versions to the latest. | 1 hour | Premium |
| `app_store_app_configuration_reconciler` | Reinstalls iOS and iPadOS App Store apps on hosts that didn't get the latest app configuration. | 1 minute | Free. Fleet 4.95 and later. |

### Apple MDM

| Name | What it does | Default interval | Availability |
|---|---|---|---|
| `apple_mdm_dep_profile_assigner` | Syncs devices from Apple Business and assigns enrollment profiles. | Configurable ([`mdm.apple_dep_sync_periodicity`](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-apple-dep-sync-periodicity), default 1 minute) | Free |
| `mdm_apple_profile_manager` | Sends configuration profiles, declarations, and device name changes to Apple hosts. | 30 seconds | Free |
| `apple_mdm_worker` | Processes queued Apple MDM jobs, like post-enrollment steps. | 10 seconds (each run can last up to 10 minutes) | Free |
| `apple_mdm_apns_sweep` | Sends push notifications to Apple hosts that haven't checked in for more than a day. | 1 minute | Free |
| `apple_mdm_iphone_ipad_refetcher` | Asks iPhones and iPads for updated host vitals and software. | 10 minutes | Free |
| `apple_mdm_iphone_ipad_reviver` | Sends push notifications to iPhones and iPads that stopped checking in. | 1 hour | Free |
| `apple_mdm_os_updates` | Updates the list of available macOS and iOS versions from Apple. | 1 hour | Free |
| `mdm_service_discovery` | Keeps Apple account-driven enrollment service discovery up to date. | 1 hour | Free |
| `send_recovery_lock_commands` | Sends Recovery Lock password commands to Macs. | 30 seconds | Premium |
| `send_managed_local_account_rotation_commands` | Rotates managed local account passwords that are due. | 5 minutes | Premium |
| `cleanup_expired_adue_challenges` | Deletes expired enrollment challenges. | 24 hours | Premium |

> **Note:** `apple_mdm_os_updates` only downloads a new list from Apple once every 24 hours. Triggering it sooner recalculates targets from the list Fleet already has.

### Windows MDM

| Name | What it does | Default interval | Availability |
|---|---|---|---|
| `mdm_windows_profile_manager` | Sends configuration profiles to Windows hosts. | 30 seconds | Free |
| `microsoft_autopilot_sync` | Syncs devices from Windows Autopilot. | 5 minutes | Premium |
| `reencrypt_windows_disk_encryption_keys` | Re-encrypts BitLocker recovery keys stored by older Fleet versions, and asks hosts with unreadable keys to escrow a new one. | 1 hour (first run 10 minutes after the server starts) | Free. Fleet 4.95 and later. |

### Android MDM

| Name | What it does | Default interval | Availability |
|---|---|---|---|
| `mdm_android_profile_manager` | Sends configuration profiles to Android hosts. | 30 seconds | Free |
| `mdm_android_device_reconciler` | Syncs Android host status with Google. | 1 hour | Free |
| `mdm_android_command_reconciler` | Checks Google for results of Android commands that never reported back. | 24 hours | Free |

### One-time migrations

These do their work once, after you upgrade to the version that added them.

| Name | What it does |
|---|---|
| `uninstall_software_migration` | Updates uninstall scripts for software packages. |
| `upgrade_code_software_migration` | Adds upgrade codes to existing MSI packages. |
| `enable_android_app_reports_on_default_policy` | Turns on app reports for Android hosts (Fleet 4.76.0). |
| `migrate_to_per_host_policy` | Moves Android hosts to their own MDM policy (Fleet 4.77.0). |

> **Note:** You don't need to trigger migrations. Once a migration has finished, triggering it may do nothing, even though `fleetctl` still prints `Sent request`.

<meta name="articleTitle" value="Trigger Fleet server schedules on demand">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-11-12">
<meta name="category" value="guides">
<meta name="description" value="Use fleetctl trigger to run Fleet server schedules like vulnerabilities and automations now. Includes every schedule name and its default interval.">

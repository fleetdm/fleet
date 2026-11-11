# Connect Fleet to ServiceNow

_Available in Fleet Premium._

> **Coming soon:** The ServiceNow integration is planned for [Fleet 5.2.0](https://github.com/fleetdm/fleet/issues/38864). Some fields in this guide depend on other features that are also coming soon. See [What's available and what's coming soon](#whats-available-and-whats-coming-soon).

Fleet can sync host vitals (hardware, end user, and software inventory) to ServiceNow so your configuration management database (CMDB) and software asset management (SAM) records stay current without a custom integration.

This guide covers:

- [What's available and what's coming soon](#whats-available-and-whats-coming-soon)
- [How the integration works](#how-the-integration-works)
- [Prepare ServiceNow](#prepare-servicenow)
- [Run alongside an existing integration](#run-alongside-an-existing-integration)
- [Computers](#computers)
- [End user (Assigned to)](#end-user-assigned-to)
- [Hardware models](#hardware-models)
- [Software installations](#software-installations)
- [Sync schedule and host lifecycle](#sync-schedule-and-host-lifecycle)
- [Field mapping reference](#field-mapping-reference)

## What's available and what's coming soon

| Feature | Status | Tracking issue |
| --- | --- | --- |
| ServiceNow integration (computers, end users, hardware models, software installations) | Coming soon in Fleet 5.2.0 | [#38864](https://github.com/fleetdm/fleet/issues/38864) |
| Software publisher on every platform, shown in Fleet's UI, API, and CSV export | Coming soon in Fleet 5.2.0 | [#54914](https://github.com/fleetdm/fleet/issues/54914) |
| Software install date (or "first seen" date) for each software item on each host | Coming soon in Fleet 5.2.0 | [#54912](https://github.com/fleetdm/fleet/issues/54912) |
| Collect the end user's IdP information when they sign in to Fleet Desktop | Coming soon in Fleet 5.2.0 | [#51472](https://github.com/fleetdm/fleet/issues/51472) |
| IdP user ID, custom emails, and Google Chrome profile emails used to match end users | Available now | |
| Apple marketing names for hardware models (`hardware_marketing_name`) | Available now | |

Until publisher and install date ship, the integration leaves those ServiceNow fields blank (or partially filled) instead of guessing. Learn more in [Software installations](#software-installations).

## How the integration works

In Fleet, head to **Settings > Integrations > Asset management**, select **Add tool**, choose **ServiceNow**, and enter your ServiceNow instance URL and API key. Fleet syncs host vitals every hour.

Fleet writes every record through ServiceNow's [Identification and Reconciliation Engine (IRE)](https://www.servicenow.com/docs/csh?topicname=identification-reconciliation.html) with **Fleet** as the discovery source. IRE matches Fleet's data to existing configuration items (CIs) instead of creating duplicates. A CI is ServiceNow's record for a computer, software install, or other asset. IRE also lets Fleet run alongside other discovery sources such as SCCM, Intune, or Jamf.

## Prepare ServiceNow

Before you connect Fleet, a ServiceNow admin should complete these steps. Field and table names in this guide are ServiceNow defaults. If your instance uses different names, map Fleet's data to your fields instead.

1. **Create a discovery source** named `Fleet` (the `discovery_source` choice list on `cmdb_ci`).
2. **Confirm the identification rule for computers** includes `serial_number`. Fleet uses serial number as the primary match and hardware UUID as the fallback.
3. **Add a custom field** on `cmdb_ci_computer`: `u_fleet_host_id` (Fleet's host ID, used to link back to the host in Fleet).
4. **Add custom fields** on the software installation table (see [Software installations](#software-installations)): `u_bundle_identifier` and `u_apple_team_id`.
5. **Store your IdP's user ID on `sys_user`**, for example the Okta user ID, if it isn't there already. Fleet matches end users by this ID first.
6. **Create a model alias table** (for example, `u_fleet_model_alias`) with three columns: manufacturer, Fleet model string, and a reference to `cmdb_hardware_product_model`. Learn more in [Hardware models](#hardware-models).
7. **Create a placeholder hardware model** named `Unmapped (Fleet)`.
8. **Create a ServiceNow integration user** with permission to write to the tables above, and generate the API key you'll enter in Fleet.

## Run alongside an existing integration

If you already sync Fleet data to ServiceNow with your own integration, you can move to Fleet's integration without losing data.

- **Nothing changes until you turn it on.** Fleet doesn't write to ServiceNow until you add ServiceNow under **Settings > Integrations > Asset management**.
- **Existing CIs are updated, not duplicated.** IRE matches Fleet's records to your existing computers by serial number, so the CIs your integration created are updated in place.
- **Both integrations can run at once.** Each writes with its own discovery source. Use IRE [reconciliation rules](https://www.servicenow.com/docs/csh?topicname=reconciliation-rules.html) to decide which source owns each field while you compare results.

To switch over:

1. Complete [Prepare ServiceNow](#prepare-servicenow).
2. Add ServiceNow in Fleet and let it sync for at least one cycle (one hour).
3. Compare a sample of computers and software installation records written by both sources.
4. Set reconciliation rules so the `Fleet` discovery source is authoritative.
5. Turn off your own integration.
6. Clean up hardware models your integration created. Learn more in [Clean up existing models](#clean-up-existing-models).

## Computers

Fleet writes one `cmdb_ci_computer` record per host, using data from the [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) API endpoint. Fleet matches each host to an existing computer by serial number first, then by hardware UUID. Blank or placeholder serial numbers (common on virtual machines) aren't used for matching.

See every field in the [field mapping reference](#field-mapping-reference).

## End user (Assigned to)

Fleet sets **Assigned to** by matching the host's end user to a `sys_user` record. Fleet tries each source in order and stops at the first match:

1. **IdP user ID**, for example an Okta user ID (`00u...`), matched to your IdP user ID field on `sys_user`. Requires [IdP host vitals](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts).
2. **Custom email**, matched to `sys_user.email`. Set with the [Update human-device mapping](https://fleetdm.com/docs/rest-api/rest-api#update-human-device-mapping) API endpoint, or at install time with Fleet's agent (fleetd) `--end-user-email` flag on Windows and Linux.
3. **Google Chrome profile email**, matched to `sys_user.email`. Fleet uses the first email in alphanumeric order, not the most recently used. For more reliable matches, use the IdP user ID (step 1).
4. **No match:** **Assigned to** is left unchanged. Fleet never clears it.

To improve match rates:

- **Use IdP host vitals (best practice).** Fleet collects the IdP user ID when an end user authenticates during enrollment, and on Windows hosts joined to Microsoft Entra ID. Coming soon in Fleet 5.2.0, Fleet will also collect it when an end user signs in to Fleet Desktop ([#51472](https://github.com/fleetdm/fleet/issues/51472)). Fleet only fills in IdP information when none exists, and never collects it from an admin previewing an end user's **My device** page.
- **Backfill hosts that enrolled before you required IdP authentication.** On the **Host details** page, add the end user's IdP username. Fleet links the host to the IdP user and fills in the IdP user ID.
- **Use one source for custom emails.** Emails set by the API and by fleetd both appear as `custom`, so if both set an email, Fleet can't tell which is more recent.

## Hardware models

ServiceNow keeps a library of hardware models, and the strings Fleet reports (for example, `MacBookPro18,3`) often don't match model names in that library. To avoid creating a new model for every string, Fleet never creates models. It sets `model_id` in this order:

1. **Alias table:** looks up the manufacturer and Fleet model string in your alias table and uses the model it points to.
2. **Exact match:** matches `hardware_marketing_name` or `hardware_model` to a model's name or model number. On Apple hosts, `hardware_marketing_name` is the Apple marketing name (for example, "MacBook Pro (16-inch, Nov 2023)"), which usually matches ServiceNow's model library.
3. **Placeholder:** sets the `Unmapped (Fleet)` model. Review these computers and add the missing strings to your alias table.

To seed the alias table, export the hosts list from Fleet (**Hosts > Export hosts**), list the unique `hardware_vendor` and `hardware_model` pairs, and map each pair to a model in ServiceNow. After that, only new hardware shows up as unmapped.

### Clean up existing models

If your current integration created a model for each Fleet string, you can consolidate them:

1. Add each auto-created model's string to the alias table, pointing to the correct ServiceNow model.
2. After Fleet's next sync, confirm no computers still reference the auto-created models.
3. Retire the auto-created models.

## Software installations

Fleet writes one software installation record per software version installed on each host, using data from the [Get host's software](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-software) API endpoint. If you use ServiceNow SAM, Fleet writes to SAM software installations (`cmdb_sam_sw_install`). Otherwise, it writes to software instances (`cmdb_software_instance`). ServiceNow SAM normalization matches or creates the discovery model from the name, version, and publisher.

<!-- TODO(product): Confirm target tables and whether Fleet detects SAM automatically or the admin chooses. -->

Two fields are limited until other features ship:

- **Publisher:** Today, Fleet collects the publisher for Windows apps, RPM packages, and VS Code extensions. On macOS, Fleet collects the Apple Developer Team ID (written to `u_apple_team_id`), not the publisher name. Publisher isn't available for DEB packages, ChromeOS, iOS, iPadOS, or Android. Publisher on more platforms is coming soon in Fleet 5.2.0 ([#54914](https://github.com/fleetdm/fleet/issues/54914)).
- **Install date:** Today, Fleet doesn't collect install dates for software inventory, so `install_date` is left blank instead of guessed. Install date (or the date Fleet first saw the software) is coming soon in Fleet 5.2.0 ([#54912](https://github.com/fleetdm/fleet/issues/54912)).

Learn which software Fleet collects on each platform in the [software inventory reference](https://fleetdm.com/guides/software-inventory-reference).

## Sync schedule and host lifecycle

- **Schedule:** Fleet syncs to ServiceNow every hour. Fleet refreshes host vitals on the same interval by default (`FLEET_OSQUERY_DETAIL_UPDATE_INTERVAL`, default `1h`), so ServiceNow is at most about two hours behind the host.
- **Offline hosts:** **Most recent discovery** (`last_discovered`) stops updating. Use it to find stale CIs.
- **Deleted hosts:** When a host is deleted from Fleet, Fleet stops updating its CI but doesn't delete it, so your audit history stays intact. Use your normal CMDB process to retire it.
- **Removed software:** When software is no longer installed, Fleet sets the installation record's `active` field to `false` instead of deleting it.

<!-- TODO(product): Confirm deleted-host behavior (leave as-is vs. set install_status to Retired). -->
<!-- TODO(product): Decide whether iOS, iPadOS, and Android hosts are in scope, and which CI class they use (e.g. cmdb_ci_comm / mobile devices). Android BYOD hosts have no serial number to match on. -->

## Field mapping reference

| Section | Fleet field (API) | ServiceNow field | Role or match | Notes | Availability |
| --- | --- | --- | --- | --- | --- |
| Computers | `hardware_serial` | `serial_number` | Primary identifier | Blank or placeholder serials (virtual machines, some Linux hosts) aren't used for matching. | Coming soon (5.2.0) |
| Computers | `uuid` | `correlation_id` | Secondary identifier | Used when the serial number is missing. | Coming soon (5.2.0) |
| Computers | `id` | `u_fleet_host_id` (custom) | Reference | Fleet host ID, used to link back to the host in Fleet. | Coming soon (5.2.0) |
| Computers | `computer_name` | `name` | Attribute | | Coming soon (5.2.0) |
| Computers | `hostname` | `host_name` / `fqdn` | Attribute | | Coming soon (5.2.0) |
| Computers | `hardware_vendor` | `manufacturer` | Reference lookup | Matched to a `core_company` record. | Coming soon (5.2.0) |
| Computers | `hardware_model`, `hardware_marketing_name` | `model_id` | Reference lookup | Matched as described in [Hardware models](#hardware-models). Fleet never creates a model. Marketing name is Apple only. | Coming soon (5.2.0) |
| Computers | `platform`, `os_version` | `os`, `os_version` | Attribute | | Coming soon (5.2.0) |
| Computers | `primary_ip` | `ip_address` | Attribute | | Coming soon (5.2.0) |
| Computers | `primary_mac` | `mac_address` | Attribute | | Coming soon (5.2.0) |
| Computers | `cpu_brand` | `cpu_type` | Attribute | | Coming soon (5.2.0) |
| Computers | `memory` | `ram` | Attribute | Fleet reports bytes. Converted to MB (divided by 1,048,576). | Coming soon (5.2.0) |
| Computers | `gigs_total_disk_space` | `disk_space` | Attribute | GB. | Coming soon (5.2.0) |
| Computers | `seen_time` | `last_discovered` | Attribute | Use to find stale CIs instead of deleting them. | Coming soon (5.2.0) |
| Computers | `end_users[0].idp_department` | `department` | Attribute | | Coming soon (5.2.0) |
| Computers | `end_users` | `assigned_to` | Reference lookup | Matched as described in [End user (Assigned to)](#end-user-assigned-to). | Coming soon (5.2.0) |
| Software installations | `name`, `display_name` | `display_name` | Attribute | | Coming soon (5.2.0) |
| Software installations | `installed_versions[].version` | `version` | Attribute | One record per installed version. | Coming soon (5.2.0) |
| Software installations | The host's computer record | `installed_on` | Reference | | Coming soon (5.2.0) |
| Software installations | `name`, `version`, and publisher | `discovery_model` | Reference lookup | ServiceNow SAM normalization matches or creates the discovery model. | Coming soon (5.2.0) |
| Software installations | `vendor` | `publisher` | Attribute | Windows apps, RPM packages, and VS Code extensions only. More platforms in [#54914](https://github.com/fleetdm/fleet/issues/54914). | Coming soon (5.2.0). Partial until #54914 ships. |
| Software installations | `installed_versions[].signature_information[].team_identifier` | `u_apple_team_id` (custom) | Attribute | macOS Apple Developer Team ID. | Coming soon (5.2.0) |
| Software installations | `installed_versions[].installed_paths` | `install_location` | Attribute | | Coming soon (5.2.0) |
| Software installations | Not collected yet | `install_date` | Attribute | Left blank until [#54912](https://github.com/fleetdm/fleet/issues/54912) ships. Fleet doesn't guess a date. | Coming soon (5.2.0) |
| Software installations | `installed_versions[].last_opened_at` | `last_used` | Attribute | Reliable on macOS. Approximate on Windows and Linux. | Coming soon (5.2.0) |
| Software installations | Present in Fleet's latest inventory | `active` | Attribute | Set to `false` when the software is no longer installed. The record isn't deleted. | Coming soon (5.2.0) |
| Software installations | `Fleet` | `discovery_source` | Attribute | | Coming soon (5.2.0) |
| Software installations | `detail_updated_at` (host) | `last_scanned` | Attribute | When Fleet last refreshed the host's vitals. | Coming soon (5.2.0) |
| Software installations | `bundle_identifier` | `u_bundle_identifier` (custom) | Attribute | macOS only. | Coming soon (5.2.0) |
| End user, step 1 | `end_users[0].idp_id` | IdP user ID field on `sys_user` | First match | For example, an Okta user ID (`00u...`). Also collected at Fleet Desktop sign-in in Fleet 5.2.0 ([#51472](https://github.com/fleetdm/fleet/issues/51472)). | Coming soon (5.2.0) |
| End user, step 2 | `other_emails[]` with `source` set to `custom` | `sys_user.email` | If step 1 has no match | Set by the API or fleetd `--end-user-email`. Both appear as `custom`, so use one source. | Coming soon (5.2.0) |
| End user, step 3 | `other_emails[]` with `source` set to `google_chrome_profiles` | `sys_user.email` | If step 2 has no match | First email in alphanumeric order. | Coming soon (5.2.0) |
| End user, step 4 | None | | No match | **Assigned to** is left unchanged. Fleet never clears it. | Coming soon (5.2.0) |
| Hardware models, step 1 | `hardware_vendor` and `hardware_model` | Alias table (for example, `u_fleet_model_alias`) | First lookup | Maps each Fleet string to a `cmdb_hardware_product_model` record. | Coming soon (5.2.0) |
| Hardware models, step 2 | `hardware_marketing_name` or `hardware_model` | `cmdb_hardware_product_model` name or model number | Exact match | | Coming soon (5.2.0) |
| Hardware models, step 3 | No match | `Unmapped (Fleet)` placeholder model | Fallback | Review and add the string to your alias table. Fleet doesn't create a new model. | Coming soon (5.2.0) |

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2026-11-11">
<meta name="articleTitle" value="Connect Fleet to ServiceNow">
<meta name="description" value="Sync host vitals, end users, and software inventory from Fleet to ServiceNow, including how to switch from an existing integration.">

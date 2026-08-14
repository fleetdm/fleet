# Labels

In Fleet, labels organize hosts into groups you can target with [software](https://fleetdm.com/guides/deploy-software-packages), [policies](https://fleetdm.com/securing/what-are-fleet-policies), [reports](https://fleetdm.com/guides/queries), and [configuration profiles](https://fleetdm.com/guides/custom-os-settings). You can also use labels to filter the hosts view.

> We recommend labels, rather than separate fleets, as your primary way to target these features.

## Label types

- **Dynamic:** Query-based; auto-applied to any host returning a result for the label's SQL query. Optionally restrict to a platform (`darwin`, `windows`, `ubuntu`, `centos`).
- **Manual:** Applied to an explicit list of hosts, specified by `hardware_serial`, `uuid`, or Fleet host ID. Useful for one-off groupings (e.g., a pilot group).
- **Host vitals:** Auto-applied to hosts matching a single host vital's value (exact match only). Supported criteria: any of the vitals listed in [Supported built-in host vitals](#supported-built-in-host-vitals), or any [custom host vital](https://fleetdm.com/guides/custom-host-vitals) you've defined.

> To change a dynamic label's query/platform or a host vitals label's criteria in the UI, you must delete and re-create it.

## Targeting with labels

Labels can target or exclude hosts using one scoping mode per item. Configuration Profiles support custom targeting via "Include any" and "Exclude any":

| Scope | Behavior | Available for |
| --- | --- | --- |
| **Include any** | Targets hosts with **any** of the labels | Software, policies, reports, configuration profiles |
| **Include all** | Targets hosts with **all** of the labels | Software, policies, reports, configuration profiles |
| **Exclude any** | Excludes hosts with **any** of the labels | Software, policies, configuration profiles |

## Label scope: global vs. fleet

A label's scope is set based on where it's created, not by its name:

- **Global:** Available across all fleets. Created by a global user in the UI, or defined in `default.yml`.
- **Fleet:** (Fleet Premium) Scoped to a single fleet and visible only alongside global labels for that fleet. Defined in that fleet's `fleets/fleet-name.yml`. Defining a label here scopes it to the fleet; it does **not** become global.

> **Tip:** Label names share one namespace, so creating a label whose name already exists (global or fleet) will fail. If multiple teams manage labels independently, prefix them to avoid collisions—either **by owner/fleet** (e.g. `[Workstations] Kiosk`, `ws-kiosk`) or by **centralizing all labels** in one place (e.g. a `labels/` directory referenced from `default.yml`) as the single source of truth, so collisions surface in a single PR.

## Managing labels

To add or edit a label in Fleet, select the avatar on the right side of the top navigation and select **Labels**.

You can also manage labels via [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#labels) or [best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#labels).

## Target configuration profiles with labels

_Available in Fleet Premium._

You can use labels to control which hosts receive a [configuration profile](https://fleetdm.com/guides/custom-os-settings). Fleet supports three targeting options:

- **Include all**: Only hosts that have **all** specified labels receive the profile (`labels_include_all`).
- **Include any**: Hosts that have **any** of the specified labels receive the profile (`labels_include_any`).
- **Exclude any**: Hosts that have **any** of the specified labels are excluded from receiving the profile (`labels_exclude_any`).

### Combining include and exclude

You can combine `labels_exclude_any` with either `labels_include_all` or `labels_include_any` on the same profile. This lets you include a broad set of hosts and then carve out exceptions without writing a complex label query.

> `labels_include_all` and `labels_include_any` cannot be combined with each other on the same profile.

For example, to deliver a profile to all hosts in the "Engineering" or "Product" labels but skip hosts in the "Macs on Sequoia" label:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos-profile.mobileconfig
        labels_include_any:
          - Engineering
          - Product
        labels_exclude_any:
          - Macs on Sequoia
```

Or, to deliver a profile only to hosts that have **both** the "Sonoma" and "Managed" labels while excluding hosts labeled "Contractors":

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos-profile.mobileconfig
        labels_include_all:
          - Sonoma
          - Managed
        labels_exclude_any:
          - Contractors
```

If no label targeting is specified, the profile is delivered to all hosts on the specified platform.

You can also set label targets through the Fleet UI when adding or editing a configuration profile under **Controls > OS settings > Configuration profiles**, or via the [REST API](https://fleetdm.com/docs/rest-api/rest-api#create-configuration-profile).

## Supported built-in host vitals

The following built-in host vitals are availlable to use in host vitals labels:

| Name | Host vital | Criteria name | Type | Description |
|---|---|---|---|---|
| IdP group | `end_user_idp_group` | `end_user_idp_group` | string | The SCIM group the host's end user belongs to. Requires a connected IdP. |
| IdP department | `end_user_idp_department` | `end_user_idp_department` | string | The SCIM department of the host's end user. Requires a connected IdP. |
| Platform | `platform` | `platform` | string | The host's OS platform, as reported by osquery. Can be one of: `darwin`, `windows`, `chrome`, `ios`, `ipados`, `android`, or a Linux distribution identifier (e.g. `ubuntu`, `rhel`, `debian`). If using this criteria, it's recommended not to specify a platform for your label. |
| OS build | `build` | `build` | string | The precise OS build number (e.g. `24C101`), more granular than `os_version` — useful for targeting or excluding a specific problematic build. |
| Hardware model | `hardware_model` | `hardware_model` | string | The device's hardware model identifier (e.g. `"MacBookPro17,1"`, `"Latitude 5420"`). |
| Hardware vendor | `hardware_vendor` | `hardware_vendor` | string | The manufacturer of the device (e.g. `"Apple Inc."`, `"Dell Inc."`, `"Lenovo"`). |
| CPU architecture | `cpu_type` | `cpu_type` | string | The device's CPU architecture, as reported by osquery (e.g. `arm64e`, `x86_64`). Useful for targeting Apple Silicon vs. Intel Macs for architecture-specific software. |
| Scripts enabled | `scripts_enabled` | `scripts_enabled` | boolean | Whether the fleetd agent on this host has script execution enabled. |
| Osquery version | `osquery_version` | `osquery_version` | string | The version of osquery running on the host. |
| Agent version | `orbit_version` | `orbit_version` | string | The version of Fleet's osquery launcher (Orbit) running on the host, if fleetd is installed. |
| Fleet Desktop version | `fleet_desktop_version` | `fleet_desktop_version` | string | The version of Fleet Desktop installed on the host, if any. |
| MDM enrollment status | `mdm.enrollment_status` | `mdm_enrollment_status` | string | The host's MDM enrollment state. Can be one of: `"On (company-owned)"`, `"On (manual)"`, `"On (manual - personal)"`, `"Pending"`, `"Off"`. |
| MDM solution name | `mdm.name` | `mdm_name` | string | The name of the MDM solution managing the host (e.g. `"Fleet"`, `"Jamf Pro"`, `"Microsoft Intune"`), empty if unmanaged. |
| ADE assigned | `dep_assigned_to_fleet` | `dep_assigned_to_fleet` | boolean | Whether the host is assigned to Fleet in Apple Business (AB). |
| ADE assignment error | `mdm.dep_profile_error` | `mdm_dep_profile_error` | boolean | Whether Fleet received a failed response when attempting to assign an Apple Business Manager profile to the host. |
| MDM hardware attestation | `mdm_enrollment_hardware_attested` | `mdm_enrollment_hardware_attested` | boolean | Whether the host's MDM enrollment was verified via Apple's hardware attestation. |
| Device lock/wipe status | `mdm.device_status` | `mdm_device_status` | string | The host's current lock/wipe state. Can be one of: `"unlocked"`, `"locked"`, `"wiped"`. |
| Pending device action | `mdm.pending_action` | `mdm_pending_action` | string | A queued device action awaiting execution. Can be one of: `"lock"`, `"unlock"`, `"wipe"`, `"clear_passcode"`, `"location"`, `""` (none pending). |
| Disk encryption status (MDM) | `mdm.os_settings.disk_encryption.status` | `mdm_os_settings_disk_encryption_status` | string | The MDM-reported disk encryption enforcement status. Can be one of: `"verified"`, `"verifying"`, `"action_required"`, `"enforcing"`, `"failed"`, `"removing_enforcement"`. |
| Recovery lock password status | `mdm.os_settings.recovery_lock_password.status` | `mdm_os_settings_recovery_lock_password_status` | string | The enforcement status of the recovery lock password. Can be one of: `"verified"`, `"pending"`, `"failed"`, `"removing_enforcement"`. |
| Hostname enforcement status | `mdm.os_settings.host_name.status` | `mdm_os_settings_host_name_status` | string | The enforcement status of Fleet's hostname template on this host. Can be one of: `pending`, `"verifying"`, `"verified"`, `"failed"`. |
| Managed local account status | `mdm.os_settings.managed_local_account.status` | `mdm_os_settings_managed_local_account_status` | string | The provisioning status of the managed local admin account. Can be one of: `pending`, `verified`, `failed`. |
| Battery health | `batteries.health` | `batteries_health` | string | The reported health of the host's battery. Can be one of: `"Normal"`, `"Service recommended"`. |
| Cellular technology | `cellular_technology` | `cellular_technology` | string | The cellular radio technology the device's modem supports. Can be one of: `"None"`, `"GSM"`, `"CDMA"`, `"GSM and CDMA"`, `"unknown"`. |
| Personal hotspot enabled | `personal_hotspot_enabled` | `personal_hotspot_enabled` | boolean | Whether Personal Hotspot is enabled on the device (iOS/iPadOS). |
| Data roaming enabled | `data_roaming_enabled` | `data_roaming_enabled` | boolean | Whether cellular data roaming is enabled on the device (iOS/iPadOS). |
| Diagnostics submission enabled | `diagnostic_submission_enabled` | `diagnostic_submission_enabled` | boolean | Whether the device is configured to submit diagnostic and usage data to Apple. |
| iCloud backup enabled | `is_cloud_backup_enabled` | `is_cloud_backup_enabled` | boolean | Whether iCloud Backup is enabled on the device. |
| Lost Mode enabled | `is_mdm_lost_mode_enabled` | `is_mdm_lost_mode_enabled` | boolean | Whether the device currently has MDM Lost Mode activated. |
| Model number | `model_number` | `model_number` | string | The Apple-reported model number for the device (distinct from `hardware_model`), sourced via MDM device information. |
| iTunes Store account active | `itunes_store_account_is_active` | `itunes_store_account_is_active` | boolean | Whether an active iTunes Store account is signed in on the device. |
| App analytics enabled | `app_analytics_enabled` | `app_analytics_enabled` | boolean | Whether sharing of app analytics with developers is enabled on the device. |
| Awaiting configuration | `awaiting_configuration` | `awaiting_configuration` | boolean | Whether the device is awaiting Setup Assistant configuration. |
| Find My enabled | `is_device_locator_service_enabled` | `is_device_locator_service_enabled` | boolean | Whether Apple's device locator service ("Find My") is enabled on the device. |
| Do Not Disturb in effect | `is_do_not_disturb_in_effect` | `is_do_not_disturb_in_effect` | boolean | Whether Do Not Disturb is currently active on the device. |
| Network tethered | `is_network_tethered` | `is_network_tethered` | boolean | Whether the device is currently network-tethered to another device. |
| Carrier network | `service_subscriptions.current_carrier_network` | `service_subscriptions_current_carrier_network` | string | The cellular carrier the device is currently registered on (e.g. `"AT&T"`). Array-wrapped (dual-SIM devices may report two subscriptions), but Jamf has a direct equivalent Smart Group criterion. |
| Carrier settings version | `service_subscriptions.carrier_settings_version` | `service_subscriptions_carrier_settings_version` | string | The version of the carrier settings bundle installed on the device. |
| Country | `geolocation.country_iso` | `geolocation_country_iso` | string | ISO 3166-1 alpha-2 country code derived from a GeoIP lookup of the host's public IP address. |
| City | `geolocation.city_name` | `geolocation_city_name` | string | City name derived from a GeoIP lookup of the host's public IP address. |
| Timezone | `timezone` | `timezone` | string | The host's configured IANA timezone (currently only ingested for iOS/iPadOS hosts via MDM). Useful for maintenance-window scheduling or region-based targeting. |
| Public IP | `public_ip` | `public_ip` | string | The host's public IP address. |







<meta name="articleTitle" value="Labels in Fleet">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-10-24">
<meta name="articleImageUrl" value="../website/assets/images/articles/managing-labels-in-fleet-1600x900@2x.png">
<meta name="description" value="Using labels in the Fleet">

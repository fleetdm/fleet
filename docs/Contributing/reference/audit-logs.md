# Audit logs

Fleet logs activities.

To see activities in Fleet, select the Fleet icon in the top navigation and see the **Activity** section.

This page includes a list of activities.

## created_pack

Generated when creating scheduled query packs.

This activity contains the following fields:
- "pack_id": the id of the created pack.
- "pack_name": the name of the created pack.

#### Example

```json
{
	"pack_id": 123,
	"pack_name": "foo"
}
```

## edited_pack

Generated when editing scheduled query packs.

This activity contains the following fields:
- "pack_id": the id of the edited pack.
- "pack_name": the name of the edited pack.

#### Example

```json
{
	"pack_id": 123,
	"pack_name": "foo"
}
```

## deleted_pack

Generated when deleting scheduled query packs.

This activity contains the following fields:
- "pack_name": the name of the created pack.

#### Example

```json
{
	"pack_name": "foo"
}
```

## applied_spec_pack

Generated when applying a scheduled query pack spec.

This activity does not contain any detail fields.

## created_policy

Generated when creating policies.

This activity contains the following fields:
- "policy_id": the ID of the created policy.
- "policy_name": the name of the created policy.
- "fleet_id": the ID of the fleet the policy belongs to. Use -1 for global policies, 0 for "No Fleet" policies.
- "fleet_name": the name of the fleet the policy belongs to. null for global policies and "No Fleet" policies.

#### Example

```json
{
	"policy_id": 123,
	"policy_name": "foo",
	"fleet_id": 1,
	"fleet_name": "Workstations"
}
```

## edited_policy

Generated when editing policies.

This activity contains the following fields:
- "policy_id": the ID of the edited policy.
- "policy_name": the name of the edited policy.
- "fleet_id": the ID of the fleet the policy belongs to. Use -1 for global policies, 0 for "No Fleet" policies.
- "fleet_name": the name of the fleet the policy belongs to. null for global policies and "No Fleet" policies.

#### Example

```json
{
	"policy_id": 123,
	"policy_name": "foo",
	"fleet_id": 1,
	"fleet_name": "Workstations"
}
```

## deleted_policy

Generated when deleting policies.

This activity contains the following fields:
- "policy_id": the ID of the deleted policy.
- "policy_name": the name of the deleted policy.
- "fleet_id": the ID of the fleet the policy belonged to. Use -1 for global policies, 0 for "No Fleet" policies.
- "fleet_name": the name of the fleet the policy belonged to. null for global policies and "No Fleet" policies.

#### Example

```json
{
	"policy_id": 123,
	"policy_name": "foo",
	"fleet_id": 1,
	"fleet_name": "Workstations"
}
```

## applied_spec_policy

Generated when applying policy specs.

This activity contains a field "policies" where each item is a policy spec with the following fields:
- "name": Name of the applied policy.
- "query": SQL query of the policy.
- "description": Description of the policy.
- "critical": Marks the policy as high impact.
- "resolution": Describes how to solve a failing policy.
- "fleet": Name of the fleet this policy belongs to.
- "platform": Comma-separated string to indicate the target platforms.


#### Example

```json
{
	"policies": [
		{
			"name":"Gatekeeper enabled (macOS)",
			"query":"SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
			"critical":false,
			"platform":"darwin",
			"resolution":"To enable Gatekeeper, on the failing device [...]",
			"description":"Checks to make sure that the Gatekeeper feature is [...]"
		},
		{
			"name":"Full disk encryption enabled (Windows)",
			"query":"SELECT 1 FROM bitlocker_info WHERE drive_letter='C:' AND protection_status=1;",
			"critical":false,
			"platform":"windows",
			"resolution":"To get additional information, run the following osquery [...]",
			"description":"Checks to make sure that full disk encryption is enabled on Windows devices."
		}
	]
}
```

## created_saved_report

Generated when creating a new report.

This activity contains the following fields:
- "report_id": the ID of the created report.
- "report_name": the name of the created report.
- "fleet_id": the ID of the fleet the report belongs to.
- "fleet_name": the name of the fleet the report belongs to.

#### Example

```json
{
	"report_id": 123,
	"report_name": "foo",
	"fleet_id": 1,
	"fleet_name": "Workstations"
}
```

## edited_saved_report

Generated when editing a saved report.

This activity contains the following fields:
- "report_id": the ID of the report being edited.
- "report_name": the name of the report being edited.
- "fleet_id": the ID of the fleet the report belongs to.
- "fleet_name": the name of the fleet the report belongs to.

#### Example

```json
{
	"report_id": 123,
	"report_name": "foo",
	"fleet_id": 1,
	"fleet_name": "Workstations"
}
```

## deleted_saved_report

Generated when deleting a saved report.

This activity contains the following fields:
- "report_name": the name of the report being deleted.
- "fleet_id": the ID of the fleet the report belongs to.
- "fleet_name": the name of the fleet the report belongs to.

#### Example

```json
{
	"report_name": "foo",
	"fleet_id": 1,
	"fleet_name": "Workstations"
}
```

## deleted_multiple_saved_report

Generated when deleting multiple saved reports.

This activity contains the following fields:
- "report_ids": list of IDs of the deleted saved reports.
- "fleet_id": the ID of the fleet the reports belonged to. -1 for global reports, null for no fleet.
- "fleet_name": the name of the fleet the reports belonged to. null for global or no fleet reports.

#### Example

```json
{
	"report_ids": [1, 42, 100],
	"fleet_id": 123,
	"fleet_name": "Workstations"
}
```

## applied_spec_saved_report

Generated when applying a report spec.

This activity contains a field "specs" where each item is a report spec with the following fields:
- "name": Name of the report.
- "description": Description of the report.
- "query": SQL query.

#### Example

```json
{
	"specs": [
		{
			"name":"Get OpenSSL versions",
			"query":"SELECT name AS name, version AS version, 'deb_packages' AS source FROM [...]",
			"description":"Retrieves the OpenSSL version."
		}
	]
}
```

## created_fleet

Generated when creating fleets.

This activity contains the following fields:
- "fleet_id": unique ID of the created fleet.
- "fleet_name": the name of the created fleet.

#### Example

```json
{
	"fleet_id": 123,
	"fleet_name": "Workstations"
}
```

## deleted_fleet

Generated when deleting fleets.

This activity contains the following fields:
- "fleet_id": unique ID of the deleted fleet.
- "fleet_name": the name of the deleted fleet.

#### Example

```json
{
	"fleet_id": 123,
	"fleet_name": "Workstations"
}
```

## applied_spec_fleet

Generated when applying fleet specs.

This activity contains a field "fleets" where each item contains the fleet details with the following fields:
- "id": Unique ID of the fleet.
- "name": Name of the fleet.

#### Example

```json
{
	"fleets": [
		{
			"id": 123,
			"name": "foo"
		}
	]
}
```

## transferred_hosts

Generated when a user transfers a host (or multiple hosts) to a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the hosts were transferred to, `null` if transferred to no fleet.
- "fleet_name": The name of the fleet that the hosts were transferred to, `null` if transferred to no fleet.
- "host_ids": The list of identifiers of the hosts that were transferred.
- "host_display_names": The list of display names of the hosts that were transferred (in the same order as the "host_ids").

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "host_ids": [1, 2, 3],
  "host_display_names": ["alice-macbook-air", "bob-macbook-pro", "linux-server"]
}
```

## edited_agent_options

Generated when agent options are edited (either globally or for a fleet).

This activity contains the following fields:
- "global": "true" if the user updated the global agent options, "false" if the agent options of a fleet were updated.
- "fleet_id": unique ID of the fleet for which the agent options were updated (`null` if global is true).
- "fleet_name": the name of the fleet for which the agent options were updated (`null` if global is true).

#### Example

```json
{
	"fleet_id": 123,
	"fleet_name": "Workstations",
	"global": false
}
```

## live_report

Generated when running live reports.

This activity contains the following fields:
- "targets_count": Number of hosts where the live report was targeted to run.
- "report_sql": The SQL query to run on hosts.
- "report_name": Name of the report (this field is not set if this was not a saved report).

#### Example

```json
{
	"targets_count": 5000,
	"report_sql": "SELECT * from osquery_info;",
	"report_name": "foo"
}
```

## user_added_by_sso

Generated when new users are added via SSO JIT provisioning

This activity does not contain any detail fields.

## user_logged_in

Generated when users successfully log in to Fleet.

This activity contains the following fields:
- "public_ip": Public IP of the login request.

#### Example

```json
{
	"public_ip": "168.226.215.82"
}
```

## user_failed_login

Generated when users try to log in to Fleet and fail.

This activity contains the following fields:
- "email": The email used in the login request.
- "public_ip": Public IP of the login request.

#### Example

```json
{
	"email": "foo@example.com",
	"public_ip": "168.226.215.82"
}
```

## created_user

Generated when a user is created.

This activity contains the following fields:
- "user_id": Unique ID of the created user in Fleet.
- "user_name": Name of the created user.
- "user_email": E-mail of the created user.

#### Example

```json
{
	"user_id": 42,
	"user_name": "Foo",
	"user_email": "foo@example.com"
}
```

## deleted_user

Generated when a user is deleted.

This activity contains the following fields:
- "user_id": Unique ID of the deleted user in Fleet.
- "user_name": Name of the deleted user.
- "user_email": E-mail of the deleted user.

#### Example

```json
{
	"user_id": 42,
	"user_name": "Foo",
	"user_email": "foo@example.com"
}
```

## deleted_host

Generated when a host is deleted.

This activity contains the following fields:
- "host_id": Unique ID of the deleted host in Fleet.
- "host_display_name": Display name of the deleted host.
- "host_serial": Hardware serial number of the deleted host.
- "triggered_by": How the deletion was triggered. Can be "manual" for manual deletions or "expiration" for automatic deletions due to host expiry settings.
- "host_expiry_window": (Optional) The number of days configured for host expiry. Only present when "triggered_by" is "expiration".

#### Example

```json
{
	"host_id": 42,
	"host_display_name": "USER-WINDOWS",
	"host_serial": "ABC123",
	"triggered_by": "expiration",
	"host_expiry_window": 30
}
```

## changed_user_global_role

Generated when user global roles are changed.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": New global role of the edited user.

#### Example

```json
{
	"user_id": 42,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Observer"
}
```

## deleted_user_global_role

Generated when user global roles are deleted.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Deleted global role of the edited user.

#### Example

```json
{
	"user_id": 43,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Maintainer"
}
```

## changed_user_fleet_role

Generated when user fleet roles are changed.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Fleet role set to the edited user.
- "fleet_id": Unique ID of the fleet of the changed role.
- "fleet_name": Name of the fleet of the changed role.

#### Example

```json
{
	"user_id": 43,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Maintainer",
	"fleet_id": 5,
	"fleet_name": "Bar"
}
```

## deleted_user_fleet_role

Generated when user fleet roles are deleted.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Fleet role deleted from the edited user.
- "fleet_id": Unique ID of the fleet of the deleted role.
- "fleet_name": Name of the fleet of the deleted role.

#### Example

```json
{
	"user_id": 44,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Observer",
	"fleet_id": 2,
	"fleet_name": "Zoo"
}
```

## fleet_enrolled

Generated when a host is enrolled to Fleet (Fleet's agent fleetd is installed).

This activity contains the following fields:
- "host_id": ID of the host.
- "host_serial": Serial number of the host.
- "host_display_name": Display name of the host.

#### Example

```json
{
	"host_id": "123",
	"host_serial": "B04FL3ALPT21",
	"host_display_name": "WIN-DESKTOP-JGS78KJ7C"
}
```

## mdm_enrolled

Generated when a host is enrolled in Fleet's MDM.

This activity contains the following fields:
- "host_serial": Serial number of the host (Apple enrollments only, always empty for Microsoft).
- "host_display_name": Display name of the host.
- "installed_from_dep": Whether the host was enrolled via DEP (Apple enrollments only, always false for Microsoft).
- "mdm_platform": Used to distinguish between Apple and Microsoft enrollments. Can be "apple", "microsoft" or not present. If missing, this value is treated as "apple" for backwards compatibility.
- "enrollment_id": The unique identifier for MDM BYOD enrollments; null for other enrollments.
- "platform": The enrolled host's platform

#### Example

```json
{
  "host_serial": "C08VQ2AXHT96",
  "host_display_name": "MacBookPro16,1 (C08VQ2AXHT96)",
  "installed_from_dep": true,
  "mdm_platform": "apple",
  "enrollment_id": null,
  "platform": "darwin"
}
```

## mdm_unenrolled

Generated when a host is unenrolled from Fleet's MDM.

This activity contains the following fields:
- "host_serial": Serial number of the host.
- "enrollment_id": Unique identifier for personal (BYOD) hosts.
- "host_display_name": Display name of the host.
- "installed_from_dep": Whether the host was enrolled via DEP.
- "platform": The unenrolled host's platform

#### Example

```json
{
  "host_serial": "C08VQ2AXHT96",
  "enrollment_id": null,
  "host_display_name": "MacBookPro16,1 (C08VQ2AXHT96)",
  "installed_from_dep": true,
  "platform": "darwin"
}
```

## edited_macos_min_version

Generated when the minimum required macOS version or deadline is modified.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the minimum macOS version applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the minimum macOS version applies to, `null` if it applies to devices that are not in a fleet.
- "minimum_version": The minimum macOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.

#### Example

```json
{
  "fleet_id": 3,
  "fleet_name": "Workstations",
  "minimum_version": "13.0.1",
  "deadline": "2023-06-01"
}
```

## edited_ios_min_version

Generated when the minimum required iOS version or deadline is modified.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the minimum iOS version applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the minimum iOS version applies to, `null` if it applies to devices that are not in a fleet.
- "minimum_version": The minimum iOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.

#### Example

```json
{
  "fleet_id": 3,
  "fleet_name": "iPhones",
  "minimum_version": "17.5.1",
  "deadline": "2023-06-01"
}
```

## edited_ipados_min_version

Generated when the minimum required iPadOS version or deadline is modified.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the minimum iPadOS version applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the minimum iPadOS version applies to, `null` if it applies to devices that are not in a fleet.
- "minimum_version": The minimum iPadOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.

#### Example

```json
{
  "fleet_id": 3,
  "fleet_name": "iPads",
  "minimum_version": "17.5.1",
  "deadline": "2023-06-01"
}
```

## edited_windows_updates

Generated when the Windows OS updates deadline or grace period is modified.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the Windows OS updates settings applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the Windows OS updates settings applies to, `null` if it applies to devices that are not in a fleet.
- "deadline_days": The number of days before updates are installed, `null` if the requirement was removed.
- "grace_period_days": The number of days after the deadline before the host is forced to restart, `null` if the requirement was removed.

#### Example

```json
{
  "fleet_id": 3,
  "fleet_name": "Workstations",
  "deadline_days": 5,
  "grace_period_days": 2
}
```

## enabled_macos_update_new_hosts

Generated when a user turns on updates during macOS Setup Assistant for hosts that automatically enroll (ADE).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the setting applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the setting applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## disabled_macos_update_new_hosts

Generated when a user turns off updates during macOS Setup Assistant for hosts that automatically enroll (ADE).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the setting applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the setting applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## read_host_disk_encryption_key

Generated when a user reads the disk encryption key for a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro"
}
```

## created_macos_profile

Generated when a user adds a new macOS profile to a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the profile.
- "profile_identifier": Identifier of the profile.
- "fleet_id": The ID of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## deleted_macos_profile

Generated when a user deletes a macOS profile from a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "profile_identifier": Identifier of deleted the profile.
- "fleet_id": The ID of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## edited_macos_profile

Generated when a user edits the macOS profiles of a fleet (or no fleet) via the fleetctl CLI.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## changed_macos_setup_assistant

Generated when a user sets the macOS setup assistant for a fleet (or no fleet).

This activity contains the following fields:
- "name": Name of the macOS setup assistant file.
- "fleet_id": The ID of the fleet that the setup assistant applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the setup assistant applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "name": "dep_profile.json",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## deleted_macos_setup_assistant

Generated when a user deletes the macOS setup assistant for a fleet (or no fleet).

This activity contains the following fields:
- "name": Name of the deleted macOS setup assistant file.
- "fleet_id": The ID of the fleet that the setup assistant applied to, `null` if it applied to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the setup assistant applied to, `null` if it applied to devices that are not in a fleet.

#### Example

```json
{
  "name": "dep_profile.json",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## enabled_macos_disk_encryption

Generated when a user turns on macOS disk encryption for a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that disk encryption applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that disk encryption applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## disabled_macos_disk_encryption

Generated when a user turns off macOS disk encryption for a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that disk encryption applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that disk encryption applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## enabled_gitops_mode

Generated when a user enables GitOps mode.

This activity does not contain any detail fields.

## disabled_gitops_mode

Generated when a user disables GitOps mode.

This activity does not contain any detail fields.

## added_bootstrap_package

Generated when a user adds a new bootstrap package to a fleet (or no fleet).

This activity contains the following fields:
- "package_name": Name of the package.
- "fleet_id": The ID of the fleet that the package applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the package applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "bootstrap_package_name": "bootstrap-package.pkg",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## deleted_bootstrap_package

Generated when a user deletes a bootstrap package from a fleet (or no fleet).

This activity contains the following fields:
- "package_name": Name of the package.
- "fleet_id": The ID of the fleet that the package applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the package applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "package_name": "bootstrap-package.pkg",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## enabled_macos_setup_end_user_auth

Generated when a user turns on end user authentication for macOS hosts that automatically enroll to a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that end user authentication applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that end user authentication applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## disabled_macos_setup_end_user_auth

Generated when a user turns off end user authentication for macOS hosts that automatically enroll to a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that end user authentication applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that end user authentication applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## enabled_windows_mdm

Generated when a user turns on MDM features for all Windows hosts (servers excluded).

This activity does not contain any detail fields.

## disabled_windows_mdm

Generated when a user turns off MDM features for all Windows hosts.

This activity does not contain any detail fields.

## enabled_android_mdm

Generated when a user turns on MDM features for all Android hosts.

This activity does not contain any detail fields.

## disabled_android_mdm

Generated when a user turns off MDM features for all Android hosts.

This activity does not contain any detail fields.

## enabled_windows_mdm_migration

Generated when a user enables automatic MDM migration for Windows hosts, if Windows MDM is turned on.

This activity does not contain any detail fields.

## disabled_windows_mdm_migration

Generated when a user disables automatic MDM migration for Windows hosts, if Windows MDM is turned on.

This activity does not contain any detail fields.

## ran_script

Generated when a script is sent to be run for a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "script_execution_id": Execution ID of the script run.
- "batch_execution_id": Batch execution ID of the script run.
- "script_name": Name of the script (empty if it was an anonymous script).
- "async": Whether the script was executed asynchronously.
- "policy_id": ID of the policy whose failure triggered the script run. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered the script run. Null if no associated policy.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "script_name": "set-timezones.sh",
  "script_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "batch_execution_id": "3274d95a-c140-4b17-b185-fb33c93b84e3",
  "async": false,
  "policy_id": 123,
  "policy_name": "Ensure photon torpedoes are primed"
}
```

## added_script

Generated when a script is added to a fleet (or no fleet).

This activity contains the following fields:
- "script_name": Name of the script.
- "fleet_id": The ID of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## deleted_script

Generated when a script is deleted from a fleet (or no fleet).

This activity contains the following fields:
- "script_name": Name of the script.
- "fleet_id": The ID of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## edited_script

Generated when a user edits the scripts of a fleet (or no fleet) via the fleetctl CLI.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the scripts apply to, `null` if they apply to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the scripts apply to, `null` if they apply to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## updated_script

Generated when a script is updated.

This activity contains the following fields:
- "script_name": Name of the script.
- "fleet_id": The ID of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## created_windows_profile

Generated when a user adds a new Windows profile to a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the profile.
- "fleet_id": The ID of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## deleted_windows_profile

Generated when a user deletes a Windows profile from a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "fleet_id": The ID of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## edited_windows_profile

Generated when a user edits the Windows profiles of a fleet (or no fleet) via the fleetctl CLI.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## locked_host

Generated when a user sends a request to lock a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "view_pin": Whether lock PIN was viewed (for Apple devices).

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "view_pin": true
}
```

## unlocked_host

Generated when a user sends a request to unlock a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "host_platform": Platform of the host.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "host_platform": "darwin"
}
```

## wiped_host

Generated when a user sends a request to wipe a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro"
}
```

## created_declaration_profile

Generated when a user adds a new macOS declaration to a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the declaration.
- "identifier": Identifier of the declaration.
- "fleet_id": The ID of the fleet that the declaration applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the declaration applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## deleted_declaration_profile

Generated when a user removes a macOS declaration from a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the declaration.
- "identifier": Identifier of the declaration.
- "fleet_id": The ID of the fleet that the declaration applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the declaration applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## edited_declaration_profile

Generated when a user edits the macOS declarations of a fleet (or no fleet) via the fleetctl CLI.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the declarations apply to, `null` if they apply to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the declarations apply to, `null` if they apply to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## created_android_profile

Generated when a user adds a new Android profile to a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the profile.
- "fleet_id": The ID of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## deleted_android_profile

Generated when a user deletes an Android profile from a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "fleet_id": The ID of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## edited_android_profile

Generated when a user edits the Android profiles of a fleet (or no fleet) via the fleetctl CLI.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## edited_android_certificate

Generated when a user adds or removes Android certificate templates of a fleet (or no fleet) via the fleetctl CLI.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the certificate templates apply to, `null` if they apply to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the certificate templates apply to, `null` if they apply to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## resent_configuration_profile

Generated when a user resends a configuration profile to a host.

This activity contains the following fields:
- "host_id": The ID of the host.
- "host_display_name": The display name of the host.
- "profile_name": The name of the configuration profile.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "profile_name": "Passcode requirements"
}
```

## resent_configuration_profile_batch

Generated when a user resends a configuration profile to a batch of hosts.

This activity contains the following fields:
- "profile_name": The name of the configuration profile.
- "host_count": Number of hosts in the batch.

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "host_count": 3
}
```

## installed_software

Generated when a Fleet-maintained app or custom package is installed on a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "install_uuid": ID of the software installation.
- "self_service": Whether the installation was initiated by the end user.
- "software_title": Name of the software.
- "software_package": Filename of the installer.
- "status": Status of the software installation.
- "source": Software source type (e.g., "pkg_packages", "sh_packages", "ps1_packages").
- "policy_id": ID of the policy whose failure triggered the installation. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered installation. Null if no associated policy.
- "command_uuid": ID of the in-house app installation.


#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "self_service": true,
  "install_uuid": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "status": "pending",
  "source": "pkg_packages",
  "policy_id": 1337,
  "policy_name": "Ensure 1Password is installed and up to date"
}
```

## uninstalled_software

Generated when a Fleet-maintained app or custom package is uninstalled on a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "script_execution_id": ID of the software uninstall script.
- "self_service": Whether the uninstallation was initiated by the end user from the My device UI.
- "status": Status of the software uninstallation.
- "source": Software source type (e.g., "pkg_packages", "sh_packages", "ps1_packages").

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Falcon.app",
  "script_execution_id": "ece8d99d-4313-446a-9af2-e152cd1bad1e",
  "self_service": false,
  "status": "uninstalled",
  "source": "pkg_packages"
}
```

## added_software

Generated when a Fleet-maintained app or custom package is added to Fleet.

This activity contains the following fields:
- "software_title": Name of the software.
- "software_package": Filename of the installer.
- "fleet_name": Name of the fleet to which this software was added. `null` if it was added to no fleet." +
- "fleet_id": The ID of the fleet to which this software was added. `null` if it was added to no fleet.
- "self_service": Whether the software is available for installation by the end user.
- "software_title_id": ID of the added software title.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array.

#### Example

```json
{
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "fleet_name": "Workstations",
  "fleet_id": 123,
  "self_service": true,
  "software_title_id": 2234,
  "labels_include_any": [
    {
      "name": "Engineering",
      "id": 12
    },
    {
      "name": "Product",
      "id": 17
    }
  ]
}
```

## edited_software

Generated when a Fleet-maintained app or custom package is edited in Fleet.

This activity contains the following fields:
- "software_title": Name of the software.
- "software_package": Filename of the installer as of this update (including if unchanged).
- "fleet_name": Name of the fleet on which this software was updated. `null` if it was updated on no fleet.
- "fleet_id": The ID of the fleet on which this software was updated. `null` if it was updated on no fleet.
- "self_service": Whether the software is available for installation by the end user.
- "software_title_id": ID of the added software title.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array.
- "software_display_name": Display name of the software title.

#### Example

```json
{
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "fleet_name": "Workstations",
  "fleet_id": 123,
  "self_service": true,
  "software_title_id": 2234,
  "software_icon_url": "/api/latest/fleet/software/titles/2234/icon?fleet_id=123",
  "software_display_name": "Crowdstrike Falcon",
  "labels_include_any": [
    {
      "name": "Engineering",
      "id": 12
    },
    {
      "name": "Product",
      "id": 17
    }
  ]
}
```

## deleted_software

Generated when a Fleet maintained app or custom package is deleted from Fleet.

This activity contains the following fields:
- "software_title": Name of the software.
- "software_package": Filename of the installer.
- "fleet_name": Name of the fleet to which this software was added. `null` if it was added to no fleet.
- "fleet_id": The ID of the fleet to which this software was added. `null` if it was added to no fleet.
- "self_service": Whether the software was available for installation by the end user.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array.

#### Example

```json
{
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "fleet_name": "Workstations",
  "fleet_id": 123,
  "self_service": true,
  "software_icon_url": "",
  "labels_include_any": [
    {
      "name": "Engineering",
      "id": 12
    },
    {
      "name": "Product",
      "id": 17
    }
  ]
}
```

## enabled_vpp

Generated when VPP features are enabled in Fleet.

This activity contains the following fields:
- "location": Location associated with the VPP content token for the enabled VPP features.

#### Example

```json
{
  "location": "Acme Inc."
}
```

## disabled_vpp

Generated when VPP features are disabled in Fleet.

This activity contains the following fields:
- "location": Location associated with the VPP content token for the disabled VPP features.

#### Example

```json
{
  "location": "Acme Inc."
}
```

## added_app_store_app

Generated when an App Store app is added to Fleet.

This activity contains the following fields:
- "software_title": Name of the App Store app.
- "software_title_id": ID of the added software title.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "platform": Platform of the app (`android`, `darwin`, `ios`, or `ipados`).
- "self_service": App installation can be initiated by device owner.
- "fleet_name": Name of the fleet to which this App Store app was added, or `null` if it was added to no fleet.
- "fleet_id": ID of the fleet to which this App Store app was added, or `null`if it was added to no fleet.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array.

#### Example

```json
{
  "software_title": "Logic Pro",
  "software_title_id": 123,
  "app_store_id": "1234567",
  "platform": "darwin",
  "self_service": false,
  "fleet_name": "Workstations",
  "fleet_id": 1,
  "labels_include_any": [
    {
      "name": "Engineering",
      "id": 12
    },
    {
      "name": "Product",
      "id": 17
    }
  ]
}
```

## deleted_app_store_app

Generated when an App Store app is deleted from Fleet.

This activity contains the following fields:
- "software_title": Name of the App Store app.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "platform": Platform of the app (`android`, `darwin`, `ios`, or `ipados`).
- "fleet_name": Name of the fleet from which this App Store app was deleted, or `null` if it was deleted from no fleet.
- "fleet_id": ID of the fleet from which this App Store app was deleted, or `null`if it was deleted from no fleet.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array

#### Example

```json
{
  "software_title": "Logic Pro",
  "app_store_id": "1234567",
  "platform": "darwin",
  "fleet_name": "Workstations",
  "fleet_id": 1,
  "software_icon_url": "",
  "labels_include_any": [
    {
      "name": "Engineering",
      "id": 12
    },
    {
      "name": "Product",
      "id": 17
    }
  ]
}
```

## installed_app_store_app

Generated when an App Store app is installed on a device.

This activity contains the following fields:
- "host_id": ID of the host on which the app was installed.
- "self_service": App installation was initiated by device owner.
- "host_display_name": Display name of the host.
- "software_title": Name of the App Store app.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "status": Status of the App Store app installation.
- "command_uuid": UUID of the MDM command used to install the app.
- "policy_id": ID of the policy whose failure triggered the install. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered the install. Null if no associated policy.

#### Example

```json
{
  "host_id": 42,
  "self_service": true,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Logic Pro",
  "app_store_id": "1234567",
  "command_uuid": "98765432-1234-1234-1234-1234567890ab",
  "policy_id": 123,
  "policy_name": "[Install Software] Logic Pro"
}
```

## edited_app_store_app

Generated when an App Store app is updated in Fleet.

This activity contains the following fields:
- "software_title": Name of the App Store app.
- "software_title_id": ID of the updated app's software title.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "platform": Platform of the app (`android`, `darwin`, `ios`, or `ipados`).
- "self_service": App installation can be initiated by device owner.
- "fleet_name": Name of the fleet on which this App Store app was updated, or `null` if it was updated on no fleet.
- "fleet_id": ID of the fleet on which this App Store app was updated, or `null`if it was updated on no fleet.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array.
- "software_display_name": Display name of the software title.
- "auto_update_enabled": Whether automatic updates are enabled for iOS/iPadOS App Store (VPP) apps.
- "auto_update_window_start": Update window start time (local time of the device) when automatic updates will take place for iOS/iPadOS App Store (VPP) apps, formatted as HH:MM.
- "auto_update_window_end": Update window end time (local time of the device) when automatic updates will take place for iOS/iPadOS App Store (VPP) apps, formatted as HH:MM.


#### Example

```json
{
  "software_title": "Logic Pro",
  "software_title_id": 123,
  "app_store_id": "1234567",
  "platform": "darwin",
  "self_service": true,
  "fleet_name": "Workstations",
  "fleet_id": 1,
  "software_icon_url": "/api/latest/fleet/software/titles/123/icon?fleet_id=1",
  "labels_include_any": [
    {
      "name": "Engineering",
      "id": 12
    },
    {
      "name": "Product",
      "id": 17
    }
  ]
  "software_display_name": "Logic Pro DAW"
  "auto_update_enabled": true
  "auto_update_window_start": "22:00"
  "auto_update_window_end": "02:00"
}
```

## added_ndes_scep_proxy

Generated when NDES SCEP proxy is configured in Fleet.

This activity does not contain any detail fields.

## deleted_ndes_scep_proxy

Generated when NDES SCEP proxy configuration is deleted in Fleet.

This activity does not contain any detail fields.

## edited_ndes_scep_proxy

Generated when NDES SCEP proxy configuration is edited in Fleet.

This activity does not contain any detail fields.

## added_custom_scep_proxy

Generated when SCEP certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SCEP_WIFI"
}
```

## deleted_custom_scep_proxy

Generated when SCEP certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SCEP_WIFI"
}
```

## edited_custom_scep_proxy

Generated when SCEP certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SCEP_WIFI"
}
```

## added_digicert

Generated when DigiCert certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "DIGICERT_WIFI"
}
```

## deleted_digicert

Generated when DigiCert certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "DIGICERT_WIFI"
}
```

## edited_digicert

Generated when DigiCert certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "DIGICERT_WIFI"
}
```

## added_hydrant

Generated when Hydrant certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "HYDRANT_WIFI"
}
```

## deleted_hydrant

Generated when Hydrant certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "HYDRANT_WIFI"
}
```

## edited_hydrant

Generated when Hydrant certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "HYDRANT_WIFI"
}
```

## added_custom_est_proxy

Generated when a custom EST certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "EST_WIFI"
}
```

## deleted_custom_est_proxy

Generated when a custom EST certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "EST_WIFI"
}
```

## edited_custom_est_proxy

Generated when a custom EST certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "EST_WIFI"
}
```

## added_smallstep

Generated when Smallstep certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SMALLSTEP_WIFI"
}
```

## deleted_smallstep

Generated when Smallstep certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SMALLSTEP_WIFI"
}
```

## edited_smallstep

Generated when Smallstep certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SMALLSTEP_WIFI"
}
```

## enabled_activity_automations

Generated when activity automations are enabled

This activity contains the following field:
- "webhook_url": the URL to broadcast activities to.

#### Example

```json
{
	"webhook_url": "https://example.com/notify"
}
```

## edited_activity_automations

Generated when activity automations are edited while enabled

This activity contains the following field:
- "webhook_url": the URL to broadcast activities to, post-edit.

#### Example

```json
{
	"webhook_url": "https://example.com/notify"
}
```

## disabled_activity_automations

Generated when activity automations are disabled

This activity does not contain any detail fields.

## canceled_run_script

Generated when upcoming activity `ran_script` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "script_name": Name of the script (empty if it was an anonymous script).

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "script_name": "set-timezones.sh"
}
```

## canceled_install_software

Generated when upcoming activity `installed_software` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_title_id": ID of the software title.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_title_id": 12334
}
```

## canceled_uninstall_software

Generated when upcoming activity `uninstalled_software` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_title_id": ID of the software title.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_title_id": 12334
}
```

## canceled_install_app_store_app

Generated when upcoming activity `installed_app_store_app` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_title_id": ID of the software title.

#### Example

```json
{
  "host_id": 123,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_title_id": 12334
}
```

## ran_script_batch

Generated when a script is run on a batch of hosts.

This activity contains the following fields:
- "script_name": Name of the script.
- "batch_execution_id": Execution ID of the batch script run.
- "host_count": Number of hosts in the batch.

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "batch_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "host_count": 12
}
```

## scheduled_script_batch

Generated when a batch script is scheduled.

This activity contains the following fields:
- "batch_execution_id": Execution ID of the batch script run.
- "script_name": Name of the script.
- "host_count": Number of hosts in the batch.
- "not_before": Time that the batch activity is scheduled to launch.

#### Example

```json
{
  "batch_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "script_name": "set-timezones.sh",
  "host_count": 12,
  "not_before": "2025-08-06T17:49:21.810204Z"
}
```

## canceled_script_batch

Generated when a batch script is canceled.

This activity contains the following fields:
- "batch_execution_id": Execution ID of the batch script run.
- "script_name": Name of the script.
- "host_count": Number of hosts in the batch.
- "canceled_count": Number of hosts the job was canceled for.

#### Example

```json
{
  "batch_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "script_name": "set-timezones.sh",
  "host_count": 12,
  "canceled_count": 5
}
```

## added_conditional_access_integration_microsoft

Generated when Microsoft Entra is connected for conditional access.

This activity does not contain any detail fields.

## deleted_conditional_access_integration_microsoft

Generated when Microsoft Entra is integration is disconnected.

This activity does not contain any detail fields.

## added_conditional_access_okta

Generated when Okta is configured or edited for conditional access.

This activity does not contain any detail fields.

## deleted_conditional_access_okta

Generated when Okta conditional access configuration is removed.

This activity does not contain any detail fields.

## enabled_conditional_access_automations

Generated when conditional access automations are enabled for a fleet.

This activity contains the following field:
- "fleet_id": The ID of the fleet  ("null" for "No fleet").
- "fleet_name": The name of the fleet (empty for "No fleet").

#### Example

```json
{
  "fleet_id": 5,
  "fleet_name": "Workstations"
}
```

## disabled_conditional_access_automations

Generated when conditional access automations are disabled for a fleet.

This activity contains the following field:
- "fleet_id": The ID of the fleet (`null` for "No fleet").
- "fleet_name": The name of the fleet (empty for "No fleet").

#### Example

```json
{
  "fleet_id": 5,
  "fleet_name": "Workstations"
}
```

## escrowed_disk_encryption_key

Generated when a disk encryption key is escrowed.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.

#### Example

```json
{
	"host_id": 123,
	"host_display_name": "PWNED-VM-123"
}
```

## created_custom_variable

Generated when custom variable is added.

This activity contains the following fields:
- "custom_variable_id": the id of the new custom variable.
- "custom_variable_name": the name of the new custom variable.

#### Example

```json
{
	"custom_variable_id": 123,
	"custom_variable_name": "SOME_API_KEY"
}
```

## deleted_custom_variable

Generated when custom variable is deleted.

This activity contains the following fields:
- "custom_variable_id": the id of the custom variable.
- "custom_variable_name": the name of the custom variable.

#### Example

```json
{
	"custom_variable_id": 123,
	"custom_variable_name": "SOME_API_KEY"
}
```

## edited_setup_experience_software

Generated when a user edits setup experience software.

This activity contains the following fields:
- "platform": the platform of the host ("darwin", "android", "windows", or "linux").
- "fleet_id": the ID of the fleet associated with the setup experience (0 for "No fleet").
- "fleet_name": the name of the fleet associated with the setup experience (empty for "No fleet").

#### Example

```json
{
	"platform": "darwin",
	"fleet_id": 1,
	"fleet_name": "Workstations"
}
```

## edited_host_idp_data

Generated when a user updates a host's IdP data. Currently IdP username can be edited.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "host_idp_username": The updated IdP username for this host.

#### Example

```json
{
	"host_id": 1,
	"host_display_name": "Anna's MacBook Pro",
	"host_idp_username": "anna.chao@example.com"
}
```

## edited_enroll_secrets

Generated when global or fleet enroll secrets are edited.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the enroll secret applies to, `null` if it applies to devices that are not in a fleet.
- "fleet_name": The name of the fleet that the enroll secret applies to, `null` if it applies to devices that are not in a fleet.

#### Example

```json
{
  "fleet_id": 1,
  "fleet_name": "Workstations",
}
```


<meta name="title" value="Audit logs">
<meta name="pageOrderInSection" value="1400">
<meta name="description" value="Learn how Fleet logs administrative actions in JSON format.">
<meta name="navSection" value="Dig deeper">

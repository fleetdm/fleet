<!-- DO NOT EDIT. This document is automatically generated. -->
# Audit logs

Fleet logs the following information for administrative activities (in JSON):

- `created_at`: Timestamp of the event.
- `id`: Unique ID of the generated event in Fleet.
- `actor_full_name`: Author user name (missing if the user was deleted).
- `actor_id`: Unique ID of the author in Fleet (missing if the user was deleted).
- `actor_gravatar`: Gravatar URL of the author (missing if the user was deleted).
- `actor_email`: E-mail of the author (missing if the user was deleted).
- `type`: Type of the activity (see all types below).
- `details`: Specific details depending on the type of activity (see details for each activity type below).

Example:
```json
{
	"created_at": "2022-12-20T14:54:17Z",
	"id": 6,
	"actor_full_name": "Gandalf",
	"actor_id": 2,
	"actor_gravatar": "foo@example.com",
	"actor_email": "foo@example.com",
	"type": "edited_saved_query",
	"details":{
		"query_id": 42,
		"query_name": "Some query name"
	}
}
```

You can automatically send these logs to your log destination. Learn how to configure this [here](https://fleetdm.com/docs/configuration/fleet-server-configuration#external-activity-audit-logging).

To view activities in the UI, click the Fleet icon in the top navigation bar and locate the **Activity** section.

Following is a summary of the types of administrative activities logged by Fleet:

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

#### Example

```json
{
	"policy_id": 123,
	"policy_name": "foo"
}
```

## edited_policy

Generated when editing policies.

This activity contains the following fields:
- "policy_id": the ID of the edited policy.
- "policy_name": the name of the edited policy.

#### Example

```json
{
	"policy_id": 123,
	"policy_name": "foo"
}
```

## deleted_policy

Generated when deleting policies.

This activity contains the following fields:
- "policy_id": the ID of the deleted policy.
- "policy_name": the name of the deleted policy.

#### Example

```json
{
	"policy_id": 123,
	"policy_name": "foo"
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
- "team": Name of the team this policy belongs to.
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

## created_saved_query

Generated when creating a new query.

This activity contains the following fields:
- "query_id": the ID of the created query.
- "query_name": the name of the created query.

#### Example

```json
{
	"query_id": 123,
	"query_name": "foo"
}
```

## edited_saved_query

Generated when editing a saved query.

This activity contains the following fields:
- "query_id": the ID of the query being edited.
- "query_name": the name of the query being edited.

#### Example

```json
{
	"query_id": 123,
	"query_name": "foo"
}
```

## deleted_saved_query

Generated when deleting a saved query.

This activity contains the following fields:
- "query_name": the name of the query being deleted.

#### Example

```json
{
	"query_name": "foo"
}
```

## deleted_multiple_saved_query

Generated when deleting multiple saved queries.

This activity contains the following fields:
- "query_ids": list of IDs of the deleted saved queries.

#### Example

```json
{
	"query_ids": [1, 42, 100]
}
```

## applied_spec_saved_query

Generated when applying a query spec.

This activity contains a field "specs" where each item is a query spec with the following fields:
- "name": Name of the query.
- "description": Description of the query.
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

## created_team

Generated when creating teams.

This activity contains the following fields:
- "team_id": unique ID of the created team.
- "team_name": the name of the created team.

#### Example

```json
{
	"team_id": 123,
	"team_name": "foo"
}
```

## deleted_team

Generated when deleting teams.

This activity contains the following fields:
- "team_id": unique ID of the deleted team.
- "team_name": the name of the deleted team.

#### Example

```json
{
	"team_id": 123,
	"team_name": "foo"
}
```

## applied_spec_team

Generated when applying team specs.

This activity contains a field "teams" where each item contains the team details with the following fields:
- "id": Unique ID of the team.
- "name": Name of the team.

#### Example

```json
{
	"teams": [
		{
			"id": 123,
			"name": "foo"
		}
	]
}
```

## transferred_hosts

Generated when a user transfers a host (or multiple hosts) to a team (or no team).

This activity contains the following fields:
- "team_id": The ID of the team that the hosts were transferred to, `null` if transferred to no team.
- "team_name": The name of the team that the hosts were transferred to, `null` if transferred to no team.
- "host_ids": The list of identifiers of the hosts that were transferred.
- "host_display_names": The list of display names of the hosts that were transferred (in the same order as the "host_ids").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "host_ids": [1, 2, 3],
  "host_display_names": ["alice-macbook-air", "bob-macbook-pro", "linux-server"]
}
```

## edited_agent_options

Generated when agent options are edited (either globally or for a team).

This activity contains the following fields:
- "global": "true" if the user updated the global agent options, "false" if the agent options of a team were updated.
- "team_id": unique ID of the team for which the agent options were updated (`null` if global is true).
- "team_name": the name of the team for which the agent options were updated (`null` if global is true).

#### Example

```json
{
	"team_id": 123,
	"team_name": "foo",
	"global": false
}
```

## live_query

Generated when running live queries.

This activity contains the following fields:
- "targets_count": Number of hosts where the live query was targeted to run.
- "query_sql": The SQL query to run on hosts.
- "query_name": Name of the query (this field is not set if this was not a saved query).

#### Example

```json
{
	"targets_count": 5000,
	"query_sql": "SELECT * from osquery_info;",
	"query_name": "foo"
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

## changed_user_team_role

Generated when user team roles are changed.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Team role set to the edited user.
- "team_id": Unique ID of the team of the changed role.
- "team_name": Name of the team of the changed role.

#### Example

```json
{
	"user_id": 43,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Maintainer",
	"team_id": 5,
	"team_name": "Bar"
}
```

## deleted_user_team_role

Generated when user team roles are deleted.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Team role deleted from the edited user.
- "team_id": Unique ID of the team of the deleted role.
- "team_name": Name of the team of the deleted role.

#### Example

```json
{
	"user_id": 44,
	"user_name": "Foo",
	"user_email": "foo@example.com",
	"role": "Observer",
	"team_id": 2,
	"team_name": "Zoo"
}
```

## mdm_enrolled

Generated when a host is enrolled in Fleet's MDM.

This activity contains the following fields:
- "host_serial": Serial number of the host.
- "host_display_name": Display name of the host.
- "installed_from_dep": Whether the host was enrolled via DEP.
- "mdm_platform": Used to distinguish between Apple and Microsoft enrollments. Can be "apple", "microsoft" or not present. If missing, this value is treated as "apple" for backwards compatibility.

#### Example

```json
{
  "host_serial": "C08VQ2AXHT96",
  "host_display_name": "MacBookPro16,1 (C08VQ2AXHT96)",
  "installed_from_dep": true,
  "mdm_platform": "apple"
}
```

## mdm_unenrolled

Generated when a host is unenrolled from Fleet's MDM.

This activity contains the following fields:
- "host_serial": Serial number of the host.
- "host_display_name": Display name of the host.
- "installed_from_dep": Whether the host was enrolled via DEP.

#### Example

```json
{
  "host_serial": "C08VQ2AXHT96",
  "host_display_name": "MacBookPro16,1 (C08VQ2AXHT96)",
  "installed_from_dep": true
}
```

## edited_macos_min_version

Generated when the minimum required macOS version or deadline is modified.

This activity contains the following fields:
- "team_id": The ID of the team that the minimum macOS version applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the minimum macOS version applies to, `null` if it applies to devices that are not in a team.
- "minimum_version": The minimum macOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.

#### Example

```json
{
  "team_id": 3,
  "team_name": "Workstations",
  "minimum_version": "13.0.1",
  "deadline": "2023-06-01"
}
```

## edited_windows_updates

Generated when the Windows OS updates deadline or grace period is modified.

This activity contains the following fields:
- "team_id": The ID of the team that the Windows OS updates settings applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the Windows OS updates settings applies to, `null` if it applies to devices that are not in a team.
- "deadline_days": The number of days before updates are installed, `null` if the requirement was removed.
- "grace_period_days": The number of days after the deadline before the host is forced to restart, `null` if the requirement was removed.

#### Example

```json
{
  "team_id": 3,
  "team_name": "Workstations",
  "deadline_days": 5,
  "grace_period_days": 2
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
  "host_display_name": "Anna's MacBook Pro",
}
```

## created_macos_profile

Generated when a user adds a new macOS profile to a team (or no team).

This activity contains the following fields:
- "profile_name": Name of the profile.
- "profile_identifier": Identifier of the profile.
- "team_id": The ID of the team that the profile applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the profile applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## deleted_macos_profile

Generated when a user deletes a macOS profile from a team (or no team).

This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "profile_identifier": Identifier of deleted the profile.
- "team_id": The ID of the team that the profile applied to, `null` if it applied to devices that are not in a team.
- "team_name": The name of the team that the profile applied to, `null` if it applied to devices that are not in a team.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## edited_macos_profile

Generated when a user edits the macOS profiles of a team (or no team) via the fleetctl CLI.

This activity contains the following fields:
- "team_id": The ID of the team that the profiles apply to, `null` if they apply to devices that are not in a team.
- "team_name": The name of the team that the profiles apply to, `null` if they apply to devices that are not in a team.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations"
}
```

## changed_macos_setup_assistant

Generated when a user sets the macOS setup assistant for a team (or no team).

This activity contains the following fields:
- "name": Name of the macOS setup assistant file.
- "team_id": The ID of the team that the setup assistant applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the setup assistant applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "name": "dep_profile.json",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## deleted_macos_setup_assistant

Generated when a user deletes the macOS setup assistant for a team (or no team).

This activity contains the following fields:
- "name": Name of the deleted macOS setup assistant file.
- "team_id": The ID of the team that the setup assistant applied to, `null` if it applied to devices that are not in a team.
- "team_name": The name of the team that the setup assistant applied to, `null` if it applied to devices that are not in a team.

#### Example

```json
{
  "name": "dep_profile.json",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## enabled_macos_disk_encryption

Generated when a user turns on macOS disk encryption for a team (or no team).

This activity contains the following fields:
- "team_id": The ID of the team that disk encryption applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that disk encryption applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations"
}
```

## disabled_macos_disk_encryption

Generated when a user turns off macOS disk encryption for a team (or no team).

This activity contains the following fields:
- "team_id": The ID of the team that disk encryption applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that disk encryption applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations"
}
```

## added_bootstrap_package

Generated when a user adds a new bootstrap package to a team (or no team).

This activity contains the following fields:
- "package_name": Name of the package.
- "team_id": The ID of the team that the package applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the package applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "bootstrap_package_name": "bootstrap-package.pkg",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## deleted_bootstrap_package

Generated when a user deletes a bootstrap package from a team (or no team).

This activity contains the following fields:
- "package_name": Name of the package.
- "team_id": The ID of the team that the package applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the package applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "package_name": "bootstrap-package.pkg",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## enabled_macos_setup_end_user_auth

Generated when a user turns on end user authentication for macOS hosts that automatically enroll to a team (or no team).

This activity contains the following fields:
- "team_id": The ID of the team that end user authentication applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that end user authentication applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations"
}
```

## disabled_macos_setup_end_user_auth

Generated when a user turns off end user authentication for macOS hosts that automatically enroll to a team (or no team).

This activity contains the following fields:
- "team_id": The ID of the team that end user authentication applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that end user authentication applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations"
}
```

## enabled_windows_mdm

Generated when a user turns on MDM features for all Windows hosts (servers excluded).

This activity does not contain any detail fields.

## disabled_windows_mdm

Generated when a user turns off MDM features for all Windows hosts.

This activity does not contain any detail fields.

## ran_script

Generated when a script is sent to be run for a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "script_execution_id": Execution ID of the script run.
- "script_name": Name of the script (empty if it was an anonymous script).
- "async": Whether the script was executed asynchronously.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "script_name": "set-timezones.sh",
  "script_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "async": false
}
```

## added_script

Generated when a script is added to a team (or no team).

This activity contains the following fields:
- "script_name": Name of the script.
- "team_id": The ID of the team that the script applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the script applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## deleted_script

Generated when a script is deleted from a team (or no team).

This activity contains the following fields:
- "script_name": Name of the script.
- "team_id": The ID of the team that the script applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the script applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## edited_script

Generated when a user edits the scripts of a team (or no team) via the fleetctl CLI.

This activity contains the following fields:
- "team_id": The ID of the team that the scripts apply to, `null` if they apply to devices that are not in a team.
- "team_name": The name of the team that the scripts apply to, `null` if they apply to devices that are not in a team.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations"
}
```

## created_windows_profile

Generated when a user adds a new Windows profile to a team (or no team).

This activity contains the following fields:
- "profile_name": Name of the profile.
- "team_id": The ID of the team that the profile applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the profile applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## deleted_windows_profile

Generated when a user deletes a Windows profile from a team (or no team).

This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "team_id": The ID of the team that the profile applied to, `null` if it applied to devices that are not in a team.
- "team_name": The name of the team that the profile applied to, `null` if it applied to devices that are not in a team.

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## edited_windows_profile

Generated when a user edits the Windows profiles of a team (or no team) via the fleetctl CLI.

This activity contains the following fields:
- "team_id": The ID of the team that the profiles apply to, `null` if they apply to devices that are not in a team.
- "team_name": The name of the team that the profiles apply to, `null` if they apply to devices that are not in a team.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations"
}
```

## locked_host

Generated when a user sends a request to lock a host.

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

Generated when a user adds a new macOS declaration to a team (or no team).

This activity contains the following fields:
- "profile_name": Name of the declaration.
- "identifier": Identifier of the declaration.
- "team_id": The ID of the team that the declaration applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the declaration applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## deleted_declaration_profile

Generated when a user removes a macOS declaration from a team (or no team).

This activity contains the following fields:
- "profile_name": Name of the declaration.
- "identifier": Identifier of the declaration.
- "team_id": The ID of the team that the declaration applies to, `null` if it applies to devices that are not in a team.
- "team_name": The name of the team that the declaration applies to, `null` if it applies to devices that are not in a team.

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "team_id": 123,
  "team_name": "Workstations"
}
```

## edited_declaration_profile

Generated when a user edits the macOS declarations of a team (or no team) via the fleetctl CLI.

This activity contains the following fields:
- "team_id": The ID of the team that the declarations apply to, `null` if they apply to devices that are not in a team.
- "team_name": The name of the team that the declarations apply to, `null` if they apply to devices that are not in a team.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations"
}
```


<meta name="title" value="Audit logs">
<meta name="pageOrderInSection" value="1400">
<meta name="description" value="Learn how Fleet logs administrative actions in JSON format.">
<meta name="navSection" value="Dig deeper">

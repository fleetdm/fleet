# Host audit logs

_Available in Fleet Premium._

Fleet can send a webhook request every time an activity linked to one of a fleet's hosts is created. Use this to send a fleet's host activities to your SIEM or trigger automations, without receiving activities for every fleet.

To see a host's activities in the Fleet UI, go to the host's details page and select **Activity > Past**.

This webhook sends the same payload format as [Global audit logs](https://fleetdm.com/docs/api/global-audit-logs), filtered to activities linked to the fleet's hosts. This page lists those activity types and their fields.

## Configure

Configure host audit logs per fleet, using the `host_activities_webhook` object under `webhook_settings`. Set it from the Hosts page in the Fleet UI, the [fleets API](./rest-api.md#update-fleet), or [Fleet's GitOps YAML](https://fleetdm.com/docs/configuration/yaml-files#host-activities-webhook):

```yaml
name: Workstations
team_settings:
  webhook_settings:
    host_activities_webhook:
      enable_host_activities_webhook: true
      destination_url: https://example.org/webhook_handler
```

You can also configure this for "Unassigned" hosts in `unassigned.yml`.

## Activity types

Each activity below includes its fields and an example payload.

### reset_policy

Generated when a user resets a policy's results for a single host. Resets for every host a policy applies to aren't linked to specific hosts, so this webhook doesn't send them.

This activity contains the following fields:
- "policy_id": ID of the policy.
- "policy_name": Name of the policy.
- "fleet_id": ID of the fleet the policy belongs to. `-1` for global policies and `0` for Unassigned.
- "fleet_name": Name of the fleet the policy belongs to. Omitted for global policies and Unassigned.
- "host_id": ID of the host.
- "host_display_name": Display name of the host.

#### Example

```json
{
  "policy_id": 123,
  "policy_name": "Disk encryption enabled",
  "fleet_id": 1,
  "fleet_name": "Workstations",
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro"
}
```

### host_enrollment_rejected

Generated when Fleet refuses an orbit or osquery enrollment because of the one-time enroll secret rules. Recorded by Fleet, rate-limited per host and reason.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "host_serial": Serial number of the host.
- "host_uuid": UUID of the host.
- "platform": Platform of the host.
- "enrollment_plane": Component that attempted to enroll: "orbit" or "osquery".
- "reason": Why the enrollment was rejected: "one_time_secret_spent", "one_time_secret_identifier_mismatch", "shared_secret_for_mdm_managed_host", or "host_identity_cert_required" (the host has an identity certificate and the enrollment wasn't signed with it).

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "host_serial": "C02XXXXXXXXX",
  "host_uuid": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "platform": "darwin",
  "enrollment_plane": "orbit",
  "reason": "one_time_secret_spent"
}
```

### mdm_enrolled

Generated when a host is enrolled in Fleet's MDM.

This activity contains the following fields:
- "host_id": ID of the host. Omitted from activities generated before Fleet added this field.
- "host_serial": Serial number of the host. For Apple BYOD (account-driven user) enrollments, which have no serial number, this is the enrollment ID instead. `null` if the serial number is unknown.
- "host_display_name": Display name of the host.
- "installed_from_dep": Whether the host was enrolled automatically. `true` for Apple hosts enrolled via DEP, and for Windows hosts enrolled through Microsoft Entra ID during the out-of-box experience (OOBE), such as Windows Autopilot.
- "mdm_platform": Used to distinguish between Apple and Microsoft enrollments. Can be "apple", "microsoft" or not present. If missing, this value is treated as "apple" for backwards compatibility.
- "enrollment_id": The unique identifier for MDM BYOD enrollments; null for other enrollments.
- "platform": The enrolled host's platform

#### Example

```json
{
  "host_id": 42,
  "host_serial": "C08VQ2AXHT96",
  "host_display_name": "MacBookPro16,1 (C08VQ2AXHT96)",
  "installed_from_dep": true,
  "mdm_platform": "apple",
  "enrollment_id": null,
  "platform": "darwin"
}
```

### mdm_unenrolled

Generated when a host is unenrolled from Fleet's MDM.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_serial": Serial number of the host.
- "enrollment_id": Unique identifier for personal (BYOD) hosts.
- "host_display_name": Display name of the host.
- "installed_from_dep": Whether the host was enrolled via DEP.
- "platform": The unenrolled host's platform

#### Example

```json
{
  "host_id": 42,
  "host_serial": "C08VQ2AXHT96",
  "enrollment_id": null,
  "host_display_name": "MacBookPro16,1 (C08VQ2AXHT96)",
  "installed_from_dep": true,
  "platform": "darwin"
}
```

### read_host_disk_encryption_key

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

### viewed_host_recovery_lock_password

Generated when a user views the Recovery Lock password for a host.

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

### retrieved_host_my_device_url

Generated when a global admin retrieves a host's "My device" page URL (a credential-bearing link that opens the end user's device page). Fleet logs this for every retrieval, including reuse of an existing token.

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

### set_host_recovery_lock_password

Generated when Fleet sets the Recovery Lock password on a host.

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

### created_managed_local_account

Generated when a local managed account and password is created for a host.

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

### created_disk_encryption_pin

Generated when a BitLocker PIN is created.

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

### read_managed_local_account

Generated when a user reads the information for the local managed account for a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.

#### Example

```json
{
	"host_id": 123,
	"host_display_name": "Anna's MacBook Pro"
}
```

### ran_script

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
- "from_setup_experience": Whether the script was run as part of the setup experience.

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
  "policy_name": "Ensure photon torpedoes are primed",
  "from_setup_experience": false
}
```

### ran_custom_mdm_command

Generated when a user runs a custom MDM command via API or the fleetctl CLI.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "host_uuid": UUID of the host.
- "command_uuid": UUID of the MDM command that was run.
- "request_type": the type of custom MDM command.
- "platform": the platform of the host ("darwin", "windows", or "android").

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "host_uuid": "1b3d5e7f-9a2c-4e6d-8b0a-1c3d5e7f9a2b",
  "command_uuid": "98765432-1234-1234-1234-1234567890ab",
  "request_type": "EraseDevice",
  "platform": "darwin"
}
```

Android example:

```json
{
  "host_id": 42,
  "host_display_name": "Samsung SM-A176U1",
  "host_uuid": "0a22e1b2-51b7-fe74-41b9-381f5a785317",
  "command_uuid": "fe64941b-f7b8-4275-be57-5eb3535e87da",
  "request_type": "REBOOT",
  "platform": "android"
}
```

### locked_host

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

### unlocked_host

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

### wiped_host

Generated when a user sends a request to wipe a host.

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

### failed_wipe

Generated when a Windows host reports that a wipe MDM command failed.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "host_platform": Platform of the host.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "DESKTOP-1C3ARC1",
  "host_platform": "windows"
}
```

### rotated_host_recovery_lock_password

Generated when the Recovery Lock password for a host is rotated, either by a user or automatically by Fleet after the password is viewed.

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

### rotated_disk_encryption_key

Generated when a user requests a rotation of a host's disk encryption key. Recorded when the command is enqueued.

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

### failed_to_rotate_disk_encryption_key

Generated when a host reports it failed to rotate its disk encryption key.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "detail": Reason the device reported for the failure. Omitted when the device doesn't send one.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "detail": "Rotation failed"
}
```

### rotated_managed_local_account_password

Generated when a managed local account password is rotated.

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

### failed_to_rotate_managed_local_account_password

Generated when a host reports it failed to rotate the managed local account password.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "detail": Reason the device reported for the failure. Only Windows hosts report one; omitted otherwise.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "detail": "Rotation failed"
}
```

### resent_configuration_profile

Generated when a user resends a configuration profile to a host.

This activity contains the following fields:
- "host_id": The ID of the host.
- "host_display_name": The display name of the host.
- "profile_name": The name of the configuration profile.
- "profile_uuid": The UUID of the configuration profile.
- "policy_id": The ID of the policy whose failure triggered the resend. `null` if no associated policy.
- "policy_name": The name of the policy whose failure triggered the resend. `null` if no associated policy.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "profile_name": "Passcode requirements",
  "profile_uuid": "a1234567-1234-1234-1234-1234567890ab",
  "policy_id": 123,
  "policy_name": "Fix Wi-Fi"
}
```

### installed_software

Generated when a Fleet-maintained app or custom package is installed on a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "install_uuid": ID of the software installation.
- "self_service": Whether the installation was initiated by the end user.
- "software_title": Name of the software.
- "software_display_name": Custom display name of the software, if one is set. Omitted otherwise.
- "software_package": Filename of the installer.
- "hash_sha256": SHA-256 hash of the installer. Omitted when not available.
- "status": Status of the software installation.
- "source": Software source type (e.g., "pkg_packages", "sh_packages", "ps1_packages").
- "policy_id": ID of the policy whose failure triggered the installation. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered installation. Null if no associated policy.
- "command_uuid": ID of the in-house app installation.
- "from_setup_experience": Whether the installation was triggered as part of the setup experience.
- "failure_reason": Reason the installation failed before reaching the device (e.g. an unresolvable Fleet variable in the managed app configuration). Only present when "status" is "failed_install" and Fleet failed the install pre-flight; omitted otherwise.
- "skipped_install": Whether the install was skipped because the app was open. This is `true` when the Fleet-maintained app is installed by the patch policy's automation, when `patch_when_closed` is set. Only present when "status" is "failed_install" and the install was skipped for this reason, omitted otherwise.
- "patch_when_closed": Whether the installation was triggered by a patch policy configured to patch only when the app is closed.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "self_service": true,
  "install_uuid": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "status": "pending_install",
  "source": "pkg_packages",
  "policy_id": 1337,
  "policy_name": "Ensure 1Password is installed and up to date",
  "from_setup_experience": false
}
```

### notified_end_user_before_patching

Generated when Fleet notifies the end user that software will be patched on their host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "patch_notification_uuid": ID of the patch notification.
- "software_titles": Names of the software that will be patched.
- "policy_ids": IDs of the patch policies that triggered the notification.
- "time_before": Seconds between the notification and the scheduled install.
- "install_at": When the install is scheduled to start. `null` if the notification wasn't displayed.
- "status": Whether the notification script succeeded: "success" or "failed".
- "script_execution_id": ID of the notification script execution. Omitted when no script ran.
- "exit_code": Exit code of the notification script. Omitted when no script ran.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "patch_notification_uuid": "ece8d99d-4313-446a-9af2-e152cd1bad1e",
  "software_titles": ["Firefox.app"],
  "policy_ids": [1337],
  "time_before": 3600,
  "install_at": "2026-10-02T09:00:00Z",
  "status": "success",
  "script_execution_id": "98765432-1234-1234-1234-1234567890ab",
  "exit_code": 0
}
```

### uninstalled_software

Generated when a Fleet-maintained app or custom package is uninstalled on a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom display name of the software, if one is set. Omitted otherwise.
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

### installed_all_self_service_software

Generated when an end user clicks **Install all** on the **My device > Self-service** page. A separate [`installed_software`](#installed_software) activity is also generated for each queued title.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "self_service_category_id": ID of the self-service category the install was scoped to, or `null` if the end user installed across all categories.
- "self_service_category_name": Name of the self-service category the install was scoped to, or `null` if the end user installed across all categories.
- "software_titles_count": Number of software titles queued for install.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "self_service_category_id": 12,
  "self_service_category_name": "🌎 Browsers",
  "software_titles_count": 3
}
```

### installed_app_store_app

Generated when an App Store app is installed on a device.

This activity contains the following fields:
- "host_id": ID of the host on which the app was installed.
- "self_service": App installation was initiated by device owner.
- "host_display_name": Display name of the host.
- "software_title": Name of the App Store app.
- "software_display_name": Custom display name of the app, if one is set. Omitted otherwise.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "status": Status of the App Store app installation.
- "command_uuid": UUID of the MDM command used to install the app.
- "policy_id": ID of the policy whose failure triggered the install. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered the install. Null if no associated policy.
- "host_platform": Platform of the host (e.g., "darwin", "ios", "ipados", "android").
- "from_setup_experience": Whether the app was installed as part of the setup experience.
- "from_auto_update": Whether the app was installed by an automatic update.
- "failure_reason": Reason the installation failed before reaching the device (e.g. an unresolvable Fleet variable in the managed app configuration). Only present when "status" is "failed_install" and Fleet failed the install pre-flight; omitted otherwise.

#### Example

```json
{
  "host_id": 42,
  "self_service": true,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Logic Pro",
  "app_store_id": "1234567",
  "command_uuid": "98765432-1234-1234-1234-1234567890ab",
  "status": "installed",
  "policy_id": 123,
  "policy_name": "[Install Software] Logic Pro",
  "host_platform": "darwin",
  "from_setup_experience": false,
  "from_auto_update": false
}
```

### canceled_run_script

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

### canceled_mdm_command

Generated when a user cancels an upcoming MDM command.

This activity contains the following fields:
- "host_id": The ID of the host.
- "host_display_name": The display name of the host.
- "command_type": The type of MDM command.

#### Example

```json
{
  "host_id": 123,
  "host_display_name": "Anna's MacBook Pro",
  "command_type": "lock"
}
```

### canceled_install_software

Generated when upcoming activity `installed_software` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom display name of the software, if one is set. Omitted otherwise.
- "software_title_id": ID of the software title.
- "from_setup_experience": Whether the install was part of the setup experience.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_title_id": 12334
}
```

### canceled_uninstall_software

Generated when upcoming activity `uninstalled_software` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom display name of the software, if one is set. Omitted otherwise.
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

### canceled_install_app_store_app

Generated when upcoming activity `installed_app_store_app` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom display name of the software, if one is set. Omitted otherwise.
- "software_title_id": ID of the software title.
- "from_setup_experience": Whether the install was part of the setup experience.

#### Example

```json
{
  "host_id": 123,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_title_id": 12334
}
```

### edited_custom_host_vital_value

Generated when a user edits the value of a custom host vital on a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "custom_host_vital_id": ID of the custom host vital.
- "custom_host_vital_name": Name of the custom host vital.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "custom_host_vital_id": 12,
  "custom_host_vital_name": "Asset tag"
}
```

### resent_certificate

Generated when a user resends a certificate to a host.

This activity contains the following fields:
- "host_id": The ID of the host.
- "host_display_name": The display name of the host.
- "certificate_template_id": The ID of the certificate template
- "certificate_name": The name of the certificate

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "certificate_template_id": 123,
  "certificate_name": "Zero trust certificate"
}
```

### installed_certificate

Generated when a certificate is installed on a host or fails to install.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "certificate_template_id": ID of the certificate template.
- "certificate_name": Name of the certificate.
- "status": Status of the certificate installation ("installed" or "failed_install").
- "detail": Details of the failure. Only present when status is "failed_install".

#### Example (success)

```json
{
  "host_id": 42,
  "host_display_name": "Samsung SM-F946U",
  "certificate_template_id": 19,
  "certificate_name": "cert-6",
  "status": "installed"
}
```

#### Example (failure)

```json
{
  "host_id": 42,
  "host_display_name": "Samsung SM-F946U",
  "certificate_template_id": 19,
  "certificate_name": "cert-6",
  "status": "failed_install",
  "detail": "Network error during SCEP enrollment: Failed to communicate with SCEP server"
}
```

### cleared_passcode

Generated when a user clears the passcode on a host.

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

### canceled_setup_experience

Generated when macOS setup experience is canceled due to software install failure.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom display name of the software, if one is set. Omitted otherwise.
- "software_title_id": ID of the software title.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_title_id": 1234
}
```

### failed_enrollment_profile_renewal

Generated when an enrollment profile renewal (SCEP or ACME) has failed.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "command_uuid": Command ID we display details for.

#### Example

```json
{
  "host_id": 123,
  "host_display_name": "PWNED-VM-123",
  "command_uuid": "98765432-1234-1234-1234-1234567890ab"
}
```

### ran_automation_ticket

Generated when a failing-policy ticket automation (Jira or Zendesk) creates a ticket. One activity is recorded per created ticket and is associated with every host in that batch.

This activity contains the following fields:
- "policy_id": ID of the failing policy.
- "host_ids": IDs of the hosts in the batch. Included in [host activities webhook](https://fleetdm.com/docs/rest-api/rest-api#webhook-settings-host-activities-webhook) payloads only; not stored in the activity, so it's not returned by the activities API.
- "type": Ticket destination ("jira" or "zendesk").
- "ticket_key": (Optional) Key of the created ticket.
- "ticket_id": (Optional) ID of the created ticket.

#### Example

```json
{
  "policy_id": 123,
  "host_ids": [1, 2, 3],
  "type": "jira",
  "ticket_key": "ABC-123"
}
```

### failed_automation_ticket

Generated when a failing-policy ticket automation (Jira or Zendesk) fails to create a ticket. One activity is recorded per failed attempt and is associated with every host in that batch.

This activity contains the following fields:
- "policy_id": ID of the failing policy.
- "host_ids": IDs of the hosts in the batch. Included in [host activities webhook](https://fleetdm.com/docs/rest-api/rest-api#webhook-settings-host-activities-webhook) payloads only; not stored in the activity, so it's not returned by the activities API.
- "type": Ticket destination ("jira" or "zendesk").
- "error_response": Error returned by the destination.

#### Example

```json
{
  "policy_id": 123,
  "host_ids": [1, 2, 3],
  "type": "jira",
  "error_response": "401 Unauthorized"
}
```

### failed_automation_calendar_event

Generated when a failing-calendar-policy automation fails. The activity is associated with the affected host.

This activity contains the following fields:
- "policy_id": ID of the failing policy.
- "host_ids": IDs of the affected hosts. Included in [host activities webhook](https://fleetdm.com/docs/rest-api/rest-api#webhook-settings-host-activities-webhook) payloads only; not stored in the activity, so it's not returned by the activities API.
- "status_code": (Optional) HTTP status code returned by the calendar provider.
- "error_response": Error details.

#### Example

```json
{
  "policy_id": 123,
  "host_ids": [1],
  "error_response": "calendar API error"
}
```

### ran_automation_calendar_event

Generated when a failing calendar policy results in a calendar event. The activity is associated with the affected host.

This activity contains the following fields:
- "policy_id": ID of the failing policy.
- "host_ids": IDs of the affected hosts. Included in [host activities webhook](https://fleetdm.com/docs/rest-api/rest-api#webhook-settings-host-activities-webhook) payloads only; not stored in the activity, so it's not returned by the activities API.

#### Example

```json
{
  "policy_id": 123,
  "host_ids": [1]
}
```

### failed_automation_webhook

Generated when a failing-policy webhook automation batch is rejected by the destination server. One activity is recorded per failed batch POST and is associated with every host in that batch.

This activity contains the following fields:
- "policy_id": ID of the failing policy.
- "host_ids": IDs of the hosts in the batch. Included in [host activities webhook](https://fleetdm.com/docs/rest-api/rest-api#webhook-settings-host-activities-webhook) payloads only; not stored in the activity, so it's not returned by the activities API.
- "status_code": (Optional) HTTP status code returned by the destination.
- "error_response": Error returned by the destination.

#### Example

```json
{
  "policy_id": 123,
  "host_ids": [1, 2, 3],
  "status_code": 500,
  "error_response": "Internal Server Error"
}
```

### ran_automation_webhook

Generated when a failing-policy webhook automation batch is accepted by the destination server. One activity is recorded per successful batch POST and is associated with every host in that batch.

This activity contains the following fields:
- "policy_id": ID of the failing policy.
- "host_ids": IDs of the hosts in the batch. Included in [host activities webhook](https://fleetdm.com/docs/rest-api/rest-api#webhook-settings-host-activities-webhook) payloads only; not stored in the activity, so it's not returned by the activities API.
- "status_code": (Optional) HTTP status code returned by the destination.

#### Example

```json
{
  "policy_id": 123,
  "host_ids": [1, 2, 3],
  "status_code": 200
}
```

### failed_automation_conditional_access

Generated when a failing-policy conditional access automation fails to push a host's compliance status to the provider. The activity is associated with the affected host.

This activity contains the following fields:
- "policy_id": ID of the failing policy.
- "host_ids": IDs of the affected hosts. Included in [host activities webhook](https://fleetdm.com/docs/rest-api/rest-api#webhook-settings-host-activities-webhook) payloads only; not stored in the activity, so it's not returned by the activities API.
- "status_code": (Optional) HTTP status code returned by the provider.
- "error_response": Error returned by the provider.

#### Example

```json
{
  "policy_id": 123,
  "host_ids": [1],
  "status_code": 403,
  "error_response": "Forbidden"
}
```

### ran_automation_conditional_access

Generated when a failing-policy conditional access automation pushes a host's compliance status to the provider as non-compliant. The activity is associated with the affected host.

This activity contains the following fields:
- "policy_id": ID of the failing policy.
- "host_ids": IDs of the affected hosts. Included in [host activities webhook](https://fleetdm.com/docs/rest-api/rest-api#webhook-settings-host-activities-webhook) payloads only; not stored in the activity, so it's not returned by the activities API.

#### Example

```json
{
  "policy_id": 123,
  "host_ids": [1]
}
```

### released_from_ab

Generated when a host has been released from Apple Business (AB).

This activity contains the following fields:
- "host_id": ID of the host being released from AB.
- "host_display_name": Display name of the host being released from AB.
- "host_serial": Hardware serial number of the host being released from AB.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "host_serial": "ABC123"
}
```

### installed_opt_in_configuration_profile

Generated when an opt-in configuration profile is installed on a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "self_service": Whether the end user (rather than an IT admin) opted in to the profile.
- "profile_name": Name of the configuration profile.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "self_service": true,
  "profile_name": "Developer tools"
}
```

### uninstalled_opt_in_configuration_profile

Generated when an opt-in configuration profile is removed from a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "self_service": Whether the end user (rather than an IT admin) opted out of the profile.
- "profile_name": Name of the configuration profile.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "self_service": true,
  "profile_name": "Developer tools"
}
```

## Limitations

MDM command results, shown via **Show MDM commands** on the host details page, are not activities and don't trigger this webhook.

<meta name="title" value="Host audit logs">
<meta name="pageOrderInSection" value="90">
<meta name="description" value="Learn how to receive a webhook for activities linked to a fleet's hosts.">

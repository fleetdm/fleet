# Global audit logs

Use audit logs to automate based on what happens in Fleet. For example, send every activity to your [SIEM](https://en.wikipedia.org/wiki/Security_information_and_event_management), or trigger a workflow when a user runs a script.

This page includes a list of activity types and their fields. To learn how to see activities and send them to a webhook or log destination, see the [Automations guide](https://fleetdm.com/guides/automations#activity-automations). To receive activities from hosts, see [Host audit logs](https://fleetdm.com/docs/api/host-audit-logs).

Each webhook request is a JSON object with the following fields:
- "timestamp": When the activity happened.
- "actor_full_name": Name of the user who performed the activity. "Fleet" if Fleet performed it automatically. Null if there's no user.
- "actor_id": ID of the user who performed the activity. Null if there's no user or the user was deleted.
- "actor_email": Email of the user who performed the activity. Null if there's no user.
- "type": The activity type. See the list below.
- "details": Fields for the activity type. See the list below.

Activities sent to a [log destination](https://fleetdm.com/docs/configuration/fleet-server-configuration#external-activity-audit-logging) use the same format as the [List activities](https://fleetdm.com/docs/api/rest-api#list-activities) API response instead of the above webhook payload.

## Authentication

- [user_added_by_sso](#user-added-by-sso)
- [user_logged_in](#user-logged-in)
- [user_failed_login](#user-failed-login)
- [user_mfa_requested](#user-mfa-requested)

### user_added_by_sso

Generated when new users are added via SSO JIT provisioning

This activity does not contain any detail fields.

### user_logged_in

Generated when users successfully log in to Fleet.

This activity contains the following fields:
- "public_ip": Public IP of the login request.

#### Example

```json
{
  "public_ip": "168.226.215.82"
}
```

### user_failed_login

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

### user_mfa_requested

Generated when a user with multi-factor authentication (MFA) enabled submits valid credentials and Fleet sends a verification email.

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

## Activities

- [enabled_activity_automations](#enabled-activity-automations)
- [edited_activity_automations](#edited-activity-automations)
- [disabled_activity_automations](#disabled-activity-automations)

### enabled_activity_automations

Generated when activity automations are enabled

This activity contains the following field:
- "webhook_url": the URL to broadcast activities to.

#### Example

```json
{
  "webhook_url": "https://example.com/notify"
}
```

### edited_activity_automations

Generated when activity automations are edited while enabled

This activity contains the following field:
- "webhook_url": the URL to broadcast activities to, post-edit.

#### Example

```json
{
  "webhook_url": "https://example.com/notify"
}
```

### disabled_activity_automations

Generated when activity automations are disabled

This activity does not contain any detail fields.

## Certificates

- [edited_android_certificate](#edited-android-certificate)
- [installed_certificate](#installed-certificate)
- [added_ndes_scep_proxy](#added-ndes-scep-proxy)
- [deleted_ndes_scep_proxy](#deleted-ndes-scep-proxy)
- [edited_ndes_scep_proxy](#edited-ndes-scep-proxy)
- [added_custom_scep_proxy](#added-custom-scep-proxy)
- [deleted_custom_scep_proxy](#deleted-custom-scep-proxy)
- [edited_custom_scep_proxy](#edited-custom-scep-proxy)
- [added_digicert](#added-digicert)
- [deleted_digicert](#deleted-digicert)
- [edited_digicert](#edited-digicert)
- [added_hydrant](#added-hydrant)
- [deleted_hydrant](#deleted-hydrant)
- [edited_hydrant](#edited-hydrant)
- [added_custom_est_proxy](#added-custom-est-proxy)
- [deleted_custom_est_proxy](#deleted-custom-est-proxy)
- [edited_custom_est_proxy](#edited-custom-est-proxy)
- [added_smallstep](#added-smallstep)
- [deleted_smallstep](#deleted-smallstep)
- [edited_smallstep](#edited-smallstep)
- [added_google_cas](#added-google-cas)
- [edited_google_cas](#edited-google-cas)
- [deleted_google_cas](#deleted-google-cas)
- [added_ejbca](#added-ejbca)
- [edited_ejbca](#edited-ejbca)
- [deleted_ejbca](#deleted-ejbca)
- [resent_certificate](#resent-certificate)

### edited_android_certificate

Generated when a user adds or removes Android certificate templates of a fleet (or no fleet) via the fleetctl CLI.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the certificate templates apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the certificate templates apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
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

### added_ndes_scep_proxy

Generated when NDES SCEP proxy is configured in Fleet.

This activity does not contain any detail fields.

### deleted_ndes_scep_proxy

Generated when NDES SCEP proxy configuration is deleted in Fleet.

This activity does not contain any detail fields.

### edited_ndes_scep_proxy

Generated when NDES SCEP proxy configuration is edited in Fleet.

This activity does not contain any detail fields.

### added_custom_scep_proxy

Generated when SCEP certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SCEP_WIFI"
}
```

### deleted_custom_scep_proxy

Generated when SCEP certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SCEP_WIFI"
}
```

### edited_custom_scep_proxy

Generated when SCEP certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SCEP_WIFI"
}
```

### added_digicert

Generated when DigiCert certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "DIGICERT_WIFI"
}
```

### deleted_digicert

Generated when DigiCert certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "DIGICERT_WIFI"
}
```

### edited_digicert

Generated when DigiCert certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "DIGICERT_WIFI"
}
```

### added_hydrant

Generated when Hydrant certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "HYDRANT_WIFI"
}
```

### deleted_hydrant

Generated when Hydrant certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "HYDRANT_WIFI"
}
```

### edited_hydrant

Generated when Hydrant certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "HYDRANT_WIFI"
}
```

### added_custom_est_proxy

Generated when a custom EST certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "EST_WIFI"
}
```

### deleted_custom_est_proxy

Generated when a custom EST certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "EST_WIFI"
}
```

### edited_custom_est_proxy

Generated when a custom EST certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "EST_WIFI"
}
```

### added_smallstep

Generated when Smallstep certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SMALLSTEP_WIFI"
}
```

### deleted_smallstep

Generated when Smallstep certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SMALLSTEP_WIFI"
}
```

### edited_smallstep

Generated when Smallstep certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "SMALLSTEP_WIFI"
}
```

## added_google_cas

Generated when Google CAS certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "WIFI_CERTIFICATE"
}
```

## edited_google_cas

Generated when Google CAS certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "WIFI_CERTIFICATE"
}
```

## deleted_google_cas

Generated when Google CAS certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "WIFI_CERTIFICATE"
}
```

## added_ejbca

Generated when EJBCA certificate authority configuration is added in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "WIFI_CERTIFICATE"
}
```

## edited_ejbca

Generated when EJBCA certificate authority configuration is edited in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "WIFI_CERTIFICATE"
}
```

## deleted_ejbca

Generated when EJBCA certificate authority configuration is deleted in Fleet.

This activity contains the following fields:
- "name": Name of the certificate authority.

#### Example

```json
{
  "name": "WIFI_CERTIFICATE"
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

## Charts

- [enabled_historical_dataset](#enabled-historical-dataset)
- [disabled_historical_dataset](#disabled-historical-dataset)

### enabled_historical_dataset

Generated when collection of a chart historical dataset is enabled, either globally or for a specific fleet.

This activity contains the following fields:
- "dataset": The public config sub-key of the dataset. One of `"uptime"`, `"vulnerabilities"`. 
- "fleet_id": The ID of the fleet the toggle applies to, `null` if applied globally.
- "fleet_name": The name of the fleet the toggle applies to, `null` if applied globally.

#### Example

```json
{
  "dataset": "uptime",
  "fleet_id": 2,
  "fleet_name": "EMEA"
}
```

### disabled_historical_dataset

Generated when collection of a chart historical dataset is disabled, either globally or for a specific fleet.

This activity contains the following fields:
- "dataset": The public config sub-key of the dataset. One of `"uptime"`, `"vulnerabilities"`.
- "fleet_id": The ID of the fleet the toggle applies to, `null` if applied globally.
- "fleet_name": The name of the fleet the toggle applies to, `null` if applied globally.

#### Example

```json
{
  "dataset": "vulnerabilities",
  "fleet_id": 2,
  "fleet_name": "EMEA"
}
```

## Conditional access

- [added_conditional_access_integration_microsoft](#added-conditional-access-integration-microsoft)
- [deleted_conditional_access_integration_microsoft](#deleted-conditional-access-integration-microsoft)
- [added_conditional_access_okta](#added-conditional-access-okta)
- [deleted_conditional_access_okta](#deleted-conditional-access-okta)
- [enabled_conditional_access_automations](#enabled-conditional-access-automations)
- [disabled_conditional_access_automations](#disabled-conditional-access-automations)

### added_conditional_access_integration_microsoft

Generated when Microsoft Entra is connected for conditional access.

This activity does not contain any detail fields.

### deleted_conditional_access_integration_microsoft

Generated when Microsoft Entra is integration is disconnected.

This activity does not contain any detail fields.

### added_conditional_access_okta

Generated when Okta is configured or edited for conditional access.

This activity does not contain any detail fields.

### deleted_conditional_access_okta

Generated when Okta conditional access configuration is removed.

This activity does not contain any detail fields.

### enabled_conditional_access_automations

Generated when conditional access automations are enabled for a fleet.

This activity contains the following field:
- "fleet_id": The ID of the fleet (`null` for no fleet).
- "fleet_name": The name of the fleet (empty for no fleet).

#### Example

```json
{
  "team_id": 5,
  "team_name": "Workstations",
  "fleet_id": 5,
  "fleet_name": "Workstations"
}
```

### disabled_conditional_access_automations

Generated when conditional access automations are disabled for a fleet.

This activity contains the following field:
- "fleet_id": The ID of the fleet (`null` for no fleet).
- "fleet_name": The name of the fleet (empty for no fleet).

#### Example

```json
{
  "team_id": 5,
  "team_name": "Workstations",
  "fleet_id": 5,
  "fleet_name": "Workstations"
}
```

## Fleet configuration

- [edited_agent_options](#edited-agent-options)
- [enabled_gitops_mode](#enabled-gitops-mode)
- [disabled_gitops_mode](#disabled-gitops-mode)
- [enabled_gitops_exception](#enabled-gitops-exception)
- [disabled_gitops_exception](#disabled-gitops-exception)
- [enabled_sso_fleet_desktop](#enabled-sso-fleet-desktop)
- [disabled_sso_fleet_desktop](#disabled-sso-fleet-desktop)
- [enabled_windows_mdm](#enabled-windows-mdm)
- [disabled_windows_mdm](#disabled-windows-mdm)
- [enabled_android_mdm](#enabled-android-mdm)
- [disabled_android_mdm](#disabled-android-mdm)
- [enabled_windows_mdm_migration](#enabled-windows-mdm-migration)
- [disabled_windows_mdm_migration](#disabled-windows-mdm-migration)
- [edited_enroll_secrets](#edited-enroll-secrets)
- [edited_windows_enrollment_default_fleet](#edited-windows-enrollment-default-fleet)

### edited_agent_options

Generated when agent options are edited (either globally or for a fleet).

This activity contains the following fields:
- "global": "true" if the user updated the global agent options, "false" if the agent options of a fleet were updated.
- "fleet_id": unique ID of the fleet for which the agent options were updated (`null` if global is true).
- "fleet_name": the name of the fleet for which the agent options were updated (`null` if global is true).

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "global": false
}
```

### enabled_gitops_mode

Generated when a user enables GitOps mode.

This activity does not contain any detail fields.

### disabled_gitops_mode

Generated when a user disables GitOps mode.

This activity does not contain any detail fields.

### enabled_gitops_exception

Generated when a user enables a GitOps exception.

This activity contains the following fields:
- "exception": Name of the exception that was enabled. One of `"labels"`, `"software"`, `"secrets"`.

#### Example

```json
{
  "exception": "labels"
}
```

### disabled_gitops_exception

Generated when a user disables a GitOps exception.

This activity contains the following fields:
- "exception": Name of the exception that was disabled. One of `"labels"`, `"software"`, `"secrets"`.

#### Example

```json
{
  "exception": "software"
}
```

### enabled_sso_fleet_desktop

Generated when a user enables SSO in front of Fleet Desktop.

This activity does not contain any detail fields.

### disabled_sso_fleet_desktop

Generated when a user disables SSO in front of Fleet Desktop.

This activity does not contain any detail fields.

### enabled_windows_mdm

Generated when a user turns on MDM features for all Windows hosts (servers excluded).

This activity does not contain any detail fields.

### disabled_windows_mdm

Generated when a user turns off MDM features for all Windows hosts.

This activity does not contain any detail fields.

### enabled_android_mdm

Generated when a user turns on MDM features for all Android hosts.

This activity does not contain any detail fields.

### disabled_android_mdm

Generated when a user turns off MDM features for all Android hosts.

This activity does not contain any detail fields.

### enabled_windows_mdm_migration

Generated when a user enables automatic MDM migration for Windows hosts, if Windows MDM is turned on.

This activity does not contain any detail fields.

### disabled_windows_mdm_migration

Generated when a user disables automatic MDM migration for Windows hosts, if Windows MDM is turned on.

This activity does not contain any detail fields.

### edited_enroll_secrets

Generated when global or fleet enroll secrets are edited.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the enroll secret applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the enroll secret applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### edited_windows_enrollment_default_fleet

Generated when the default fleet for new MDM enrolled Windows hosts is edited.

This activity contains the following fields:
- "fleet_id": The ID of the default fleet for new MDM enrolled Windows hosts, `null` if set to no fleet.
- "fleet_name": The name of the default fleet for new MDM enrolled Windows hosts, `null` if set to no fleet.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## Hosts

- [transferred_hosts](#transferred-hosts)
- [deleted_host](#deleted-host)
- [fleet_enrolled](#fleet-enrolled)
- [host_enrollment_rejected](#host-enrollment-rejected)
- [mdm_enrolled](#mdm-enrolled)
- [mdm_unenrolled](#mdm-unenrolled)
- [read_host_disk_encryption_key](#read-host-disk-encryption-key)
- [retrieved_host_my_device_url](#retrieved-host-my-device-url)
- [locked_host](#locked-host)
- [unlocked_host](#unlocked-host)
- [wiped_host](#wiped-host)
- [failed_wipe](#failed-wipe)
- [escrowed_disk_encryption_key](#escrowed-disk-encryption-key)
- [rotated_disk_encryption_key](#rotated-disk-encryption-key)
- [failed_to_rotate_disk_encryption_key](#failed-to-rotate-disk-encryption-key)
- [bound_host_to_idp_account](#bound-host-to-idp-account)
- [refused_host_idp_account_change](#refused-host-idp-account-change)
- [set_host_google_device_id](#set-host-google-device-id)
- [deleted_host_google_device_id](#deleted-host-google-device-id)
- [edited_host_idp_data](#edited-host-idp-data)
- [read_host_recovery_lock_password](#read-host-recovery-lock-password)
- [escrowed_recovery_lock_password](#escrowed-recovery-lock-password)
- [rotated_recovery_lock_password](#rotated-recovery-lock-password)
- [rotated_managed_local_account_password](#rotated-managed-local-account-password)
- [failed_to_rotate_managed_local_account_password](#failed-to-rotate-managed-local-account-password)
- [read_managed_local_account](#read-managed-local-account)
- [failed_enrollment_profile_renewal](#failed-enrollment-profile-renewal)
- [cleared_passcode](#cleared-passcode)
- [created_managed_local_account](#created-managed-local-account)
- [added_label_to_host](#added-label-to-host)
- [removed_label_from_host](#removed-label-from-host)
- [created_disk_encryption_pin](#created-disk-encryption-pin)
- [released_from_ab](#released-from-ab)

### transferred_hosts

Generated when a user transfers a host (or multiple hosts) to a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the hosts were transferred to, `null` if transferred to no fleet.
- "fleet_name": The name of the fleet that the hosts were transferred to, `null` if transferred to no fleet.
- "host_ids": The list of identifiers of the hosts that were transferred.
- "host_display_names": The list of display names of the hosts that were transferred (in the same order as the "host_ids").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "host_ids": [1, 2, 3],
  "host_display_names": ["alice-macbook-air", "bob-macbook-pro", "linux-server"]
}
```

### deleted_host

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

### fleet_enrolled

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

### host_enrollment_rejected

Generated when Fleet refuses an enrollment in either of these cases:
- An Orbit or osquery enrollment is refused under the one-time enroll secret rules. See the [`mdm.apple_one_time_enroll_secrets`](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-apple-one-time-enroll-secrets) server configuration for macOS, and [`auth_mdm_windows_one_time_enroll_secrets`](https://fleetdm.com/docs/configuration/fleet-server-configuration#auth-mdm-windows-one-time-enroll-secrets) for Windows.
- An automatic enrollment (ADE) is refused because the host is in a fleet that requires end user authentication and the device didn't complete it.

Fleet records at most one of these per host and reason per 12 hours, so a host that keeps retrying doesn't flood the activity feed.

This activity contains the following fields:
- "host_id": ID of the host the attempt targeted, or null if the host is unknown.
- "host_display_name": Display name of the host, if known.
- "host_serial": Serial number the enrolling device presented.
- "host_uuid": Hardware UUID the enrolling device presented. For "apple_mdm", this is the device's UDID.
- "platform": Platform of the enrolling device.
- "enrollment_plane": What attempted to enroll. One of:
  - "orbit" or "osquery": a fleetd component.
  - "apple_mdm": an Apple device during automatic enrollment (ADE).
- "reason": Why the attempt was refused. One of:
  - "one_time_secret_spent": the host's one-time enroll secret was already used. Resend the "Fleetd configuration" profile (macOS) or the "Fleetd enroll secret" profile (Windows) to issue a new one.
  - "one_time_secret_identifier_mismatch": a one-time enroll secret was presented with a different serial number or hardware UUID than it was issued for.
  - "shared_secret_for_mdm_managed_host": a global or fleet-level enroll secret was used for a host that is enrolled in Fleet MDM, assigned to Fleet in Apple Business, or registered in Windows Autopilot. On Windows, this also covers a deleted host whose device checked in with Fleet MDM after it was deleted.
  - "end_user_authentication_required": the host is in a fleet that requires end user authentication, but the device tried to enroll without completing it. This happens when the device got its automatic enrollment configuration before its fleet required end user authentication, or before it moved to that fleet. Erase and reactivate the device so it goes through end user authentication. Fleet only records this reason when every other enrollment check passed.

#### Example

```json
{
  "host_id": 123,
  "host_display_name": "Anna's MacBook Pro",
  "host_serial": "C02ABC123DEF",
  "host_uuid": "5F0F24C3-1F58-4C2B-9E7B-1F0E6C2B4D6A",
  "platform": "darwin",
  "enrollment_plane": "orbit",
  "reason": "one_time_secret_spent"
}
```

#### Example: automatic enrollment

```json
{
  "host_id": 456,
  "host_display_name": "Anna's iPad",
  "host_serial": "DMPXK2ABCDEF",
  "host_uuid": "00008103-001A2B3C4D5E6F70",
  "platform": "ipados",
  "enrollment_plane": "apple_mdm",
  "reason": "end_user_authentication_required"
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

Windows hosts that enroll automatically through Microsoft Entra ID get this activity once it matches the enrollment to a host. This is usually within a minute of the device enrolling, but can be later if the host has not run Fleet's agent (fleetd) yet.

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
- "host_id": ID of the host. `0` when the host record was already deleted at unenroll time.
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

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro"
}
```

### failed_wipe

Generated when a Windows host reports that a wipe MDM command failed.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "DESKTOP-1C3ARC1"
}
```

### escrowed_disk_encryption_key

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

### rotated_disk_encryption_key

Generated when a user triggers rotation of a host's disk encryption key.

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

### failed_to_rotate_disk_encryption_key

Generated when Fleet can't rotate a host's disk encryption key after a rotation was triggered. Fleet then prompts the end user for their password at their next login to escrow a new key.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "detail": Why the rotation failed, as reported by the host or by Fleet.

#### Example

```json
{
  "host_id": 123,
  "host_display_name": "Anna's MacBook Pro",
  "detail": "The host reported an error: NSTaskExitCode 34"
}
```

### bound_host_to_idp_account

Generated when an end user signs in with the identity provider (IdP) during the Linux or Windows setup experience and Fleet links the host to that IdP account. Fleet records this activity, so it does not include a user.

This activity contains the following fields:
- "host_uuid": Hardware UUID of the host. The host may not exist in Fleet yet, because the link is made before enrollment completes.
- "idp_email": Email of the IdP account the host is now linked to.
- "replaced_idp_email": Email of the IdP account the host was linked to before. Only present when the host was already linked to a different account, for example when the end user signed in again with another account before the host finished enrolling; omitted otherwise.

#### Example

```json
{
  "host_uuid": "7d3f1a2c-9b4e-4f60-a1c8-2e5d6b7f8a90",
  "idp_email": "ana.torres@example.com",
  "replaced_idp_email": "ben.ito@example.com"
}
```

## unbound_host_from_idp_account

Generated when a host's link to an identity provider (IdP) account is removed, for example when the host re-enrolls without end user authentication. Fleet records this activity, so it does not include a user.

This activity contains the following fields:
- "host_uuid": Hardware UUID of the host.
- "idp_email": Email of the IdP account the host was linked to.

#### Example

```json
{
  "host_uuid": "C8D90CC1-0C2A-52D4-A6F4-DF55522A740F",
  "idp_email": "anna@example.com"
}
```

### refused_host_idp_account_change

Generated when an IdP sign-in from the Linux or Windows setup experience completes after the host has already enrolled and would have replaced the IdP account the host is linked to. Fleet keeps the existing link and records this activity, so it does not include a user.

This activity contains the following fields:
- "host_uuid": Hardware UUID of the host.
- "idp_email": Email of the IdP account that signed in and was not linked.
- "existing_idp_email": Email of the IdP account the host stays linked to.

#### Example

```json
{
  "host_uuid": "7d3f1a2c-9b4e-4f60-a1c8-2e5d6b7f8a90",
  "idp_email": "mallory@example.com",
  "existing_idp_email": "ana.torres@example.com"
}
```

### set_host_google_device_id

Generated when a host is marked as company-owned in Google by saving its Google device ID, for example by the registration page from the Google conditional access guide. Not generated when the same ID is set again.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "google_device_id": The host's device ID in Google.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's iPhone",
  "google_device_id": "c3d1a9e2-7b4f-4e8a-9f1d-2a6b8c0e4f12"
}
```

### deleted_host_google_device_id

Generated when a user unmarks a host as company-owned in Google by removing its Google device ID. Not generated when the host has no Google device ID.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "google_device_id": The Google device ID that was removed.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's iPhone",
  "google_device_id": "c3d1a9e2-7b4f-4e8a-9f1d-2a6b8c0e4f12"
}
```

### edited_host_idp_data

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

### read_host_recovery_lock_password

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

### escrowed_recovery_lock_password

Generated when a Recovery Lock password is escrowed.

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

### rotated_recovery_lock_password

Generated when a Recovery Lock password is rotated.

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

### rotated_managed_local_account_password

Generated when a managed local account password is rotated.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that create managed local account applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that create managed local account applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### failed_to_rotate_managed_local_account_password

Generated when a host reports that it could not rotate its managed local account password. Attributed to Fleet, because the failure arrives from the device outside any user's request.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "detail": The reason the host reported, if it sent one. Only Windows hosts report a reason today, so the field is omitted for macOS.

#### Example

```json
{
  "host_id": 123,
  "host_display_name": "DESKTOP-ABC123",
  "detail": "Resetting password for _fleetadmin failed: The password does not meet the password policy requirements."
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

### added_label_to_host

Generated when a label is added to a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "label_id": ID of the label.
- "label_name": Name of the label.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "label_id": 42,
  "label_name": "Engineering"
}
```

### removed_label_from_host

Generated when a label is removed from a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "label_id": ID of the label.
- "label_name": Name of the label.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "label_id": 42,
  "label_name": "Engineering"
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

## Labels

- [created_label](#created-label)
- [edited_label](#edited-label)
- [deleted_label](#deleted-label)

### created_label

Generated when creating labels.

This activity contains the following fields:
- "label_id": the ID of the new label.
- "label_name": the name of the new label.
- "fleet_id": the ID of the fleet the label belongs to.
- "fleet_name": the name of the fleet the label belongs to.

#### Example

```json
{
  "label_id": 123,
  "label_name": "foo",
  "fleet_id": 1,
  "fleet_name": "💻 Workstations"
}
```

### edited_label

Generated when editing labels.

This activity contains the following fields:
- "label_id": the ID of the edited label.
- "label_name": the name of the edited label.
- "fleet_id": the ID of the fleet the label belongs to.
- "fleet_name": the name of the fleet the label belongs to.

#### Example

```json
{
  "label_id": 123,
  "label_name": "foo",
  "fleet_id": 1,
  "fleet_name": "💻 Workstations"
}
```

### deleted_label

Generated when deleting labels.

This activity contains the following fields:
- "label_id": the ID of the deleted label.
- "label_name": the name of the deleted label.
- "fleet_id": the ID of the fleet the label belonged to.
- "fleet_name": the name of the fleet the label belonged to.

#### Example

```json
{
  "label_id": 123,
  "label_name": "foo",
  "fleet_id": 1,
  "fleet_name": "💻 Workstations"
}
```

## OS settings

- [edited_macos_min_version](#edited-macos-min-version)
- [edited_ios_min_version](#edited-ios-min-version)
- [edited_ipados_min_version](#edited-ipados-min-version)
- [edited_windows_updates](#edited-windows-updates)
- [enabled_macos_update_new_hosts](#enabled-macos-update-new-hosts)
- [disabled_macos_update_new_hosts](#disabled-macos-update-new-hosts)
- [created_macos_profile](#created-macos-profile)
- [deleted_macos_profile](#deleted-macos-profile)
- [edited_macos_profile](#edited-macos-profile)
- [enabled_macos_disk_encryption](#enabled-macos-disk-encryption)
- [disabled_macos_disk_encryption](#disabled-macos-disk-encryption)
- [created_windows_profile](#created-windows-profile)
- [deleted_windows_profile](#deleted-windows-profile)
- [edited_windows_profile](#edited-windows-profile)
- [created_declaration_profile](#created-declaration-profile)
- [deleted_declaration_profile](#deleted-declaration-profile)
- [edited_declaration_profile](#edited-declaration-profile)
- [created_android_profile](#created-android-profile)
- [deleted_android_profile](#deleted-android-profile)
- [edited_android_profile](#edited-android-profile)
- [resent_configuration_profile](#resent-configuration-profile)
- [resent_configuration_profile_batch](#resent-configuration-profile-batch)
- [installed_opt_in_configuration_profile](#installed-opt-in-configuration-profile)
- [uninstalled_opt_in_configuration_profile](#uninstalled-opt-in-configuration-profile)
- [enabled_recovery_lock_password](#enabled-recovery-lock-password)
- [disabled_recovery_lock_password](#disabled-recovery-lock-password)
- [edited_host_name_template](#edited-host-name-template)
- [created_apple_asset_declaration](#created-apple-asset-declaration)
- [edited_disk_encryption_settings](#edited-disk-encryption-settings)
- [edited_apple_asset_declaration](#edited-apple-asset-declaration)
- [deleted_apple_asset_declaration](#deleted-apple-asset-declaration)

### edited_macos_min_version

Generated when the minimum required macOS version or deadline is modified.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the minimum macOS version applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the minimum macOS version applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "minimum_version": The minimum macOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.

#### Example

```json
{
  "team_id": 3,
  "team_name": "Workstations",
  "fleet_id": 3,
  "fleet_name": "Workstations",
  "minimum_version": "13.0.1",
  "deadline": "2023-06-01"
}
```

### edited_ios_min_version

Generated when the minimum required iOS version or deadline is modified.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the minimum iOS version applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the minimum iOS version applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "minimum_version": The minimum iOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.

#### Example

```json
{
  "team_id": 3,
  "team_name": "iPhones",
  "fleet_id": 3,
  "fleet_name": "iPhones",
  "minimum_version": "17.5.1",
  "deadline": "2023-06-01"
}
```

### edited_ipados_min_version

Generated when the minimum required iPadOS version or deadline is modified.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the minimum iPadOS version applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the minimum iPadOS version applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "minimum_version": The minimum iPadOS version required, empty if the requirement was removed.
- "deadline": The deadline by which the minimum version requirement must be applied, empty if the requirement was removed.

#### Example

```json
{
  "team_id": 3,
  "team_name": "iPads",
  "fleet_id": 3,
  "fleet_name": "iPads",
  "minimum_version": "17.5.1",
  "deadline": "2023-06-01"
}
```

### edited_windows_updates

Generated when the Windows OS updates deadline or grace period is modified.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the Windows OS updates settings applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the Windows OS updates settings applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "deadline_days": The number of days before updates are installed, `null` if the requirement was removed.
- "grace_period_days": The number of days after the deadline before the host is forced to restart, `null` if the requirement was removed.

#### Example

```json
{
  "team_id": 3,
  "team_name": "Workstations",
  "fleet_id": 3,
  "fleet_name": "Workstations",
  "deadline_days": 5,
  "grace_period_days": 2
}
```

### enabled_macos_update_new_hosts

Generated when a user turns on updates during macOS Setup Assistant for hosts that automatically enroll (ADE).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the setting applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the setting applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### disabled_macos_update_new_hosts

Generated when a user turns off updates during macOS Setup Assistant for hosts that automatically enroll (ADE).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the setting applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the setting applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### created_macos_profile

Generated when a user adds a new macOS profile to a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the profile.
- "profile_identifier": Identifier of the profile.
- "fleet_id": The ID of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### deleted_macos_profile

Generated when a user deletes a macOS profile from a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "profile_identifier": Identifier of deleted the profile.
- "fleet_id": The ID of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### edited_macos_profile

Generated when a user edits the macOS profiles of a fleet (or no fleet) via the fleetctl CLI, or edits a single macOS profile via the edit profile endpoint.

This activity contains the following fields:
- "profile_name": Name of the edited profile. Only present when a single profile was edited; omitted for fleetctl/GitOps batch edits.
- "profile_identifier": Identifier of the edited profile. Only present when a single profile was edited; omitted for fleetctl/GitOps batch edits.
- "fleet_id": The ID of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "profile_identifier": "com.my.profile",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### enabled_macos_disk_encryption
> **Deprecated:** Use `edited_disk_encryption_settings` instead.

Generated when a user turns on disk encryption for a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that disk encryption applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that disk encryption applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### disabled_macos_disk_encryption
> **Deprecated:** Use `edited_disk_encryption_settings` instead.

Generated when a user turns off disk encryption for a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that disk encryption applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that disk encryption applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### created_windows_profile

Generated when a user adds a new Windows profile to a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the profile.
- "fleet_id": The ID of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### deleted_windows_profile

Generated when a user deletes a Windows profile from a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "fleet_id": The ID of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### edited_windows_profile

Generated when a user edits the Windows profiles of a fleet (or no fleet) via the fleetctl CLI, or edits a single Windows profile via the edit profile endpoint.

This activity contains the following fields:
- "profile_name": Name of the edited profile. Only present when a single profile was edited; omitted for fleetctl/GitOps batch edits.
- "fleet_id": The ID of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### created_declaration_profile

Generated when a user adds a new macOS declaration to a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the declaration.
- "identifier": Identifier of the declaration.
- "fleet_id": The ID of the fleet that the declaration applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the declaration applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### deleted_declaration_profile

Generated when a user removes a macOS declaration from a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the declaration.
- "identifier": Identifier of the declaration.
- "fleet_id": The ID of the fleet that the declaration applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the declaration applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### edited_declaration_profile

Generated when a user edits the macOS declarations of a fleet (or no fleet) via the fleetctl CLI, or edits a single declaration via the edit profile endpoint.

This activity contains the following fields:
- "profile_name": Name of the edited declaration. Only present when a single declaration was edited; omitted for fleetctl/GitOps batch edits.
- "profile_identifier": Identifier of the edited declaration. Only present when a single declaration was edited; omitted for fleetctl/GitOps batch edits.
- "fleet_id": The ID of the fleet that the declarations apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the declarations apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "profile_identifier": "com.my.declaration",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### created_android_profile

Generated when a user adds a new Android profile to a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the profile.
- "fleet_id": The ID of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profile applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### deleted_android_profile

Generated when a user deletes an Android profile from a fleet (or no fleet).

This activity contains the following fields:
- "profile_name": Name of the deleted profile.
- "fleet_id": The ID of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profile applied to, `null` if it applied to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### edited_android_profile

Generated when a user edits the Android profiles of a fleet (or no fleet) via the fleetctl CLI, or edits a single Android profile via the edit profile endpoint.

This activity contains the following fields:
- "profile_name": Name of the edited profile. Only present when a single profile was edited; omitted for fleetctl/GitOps batch edits.
- "fleet_id": The ID of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the profiles apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "profile_name": "Custom settings 1",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
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

### resent_configuration_profile_batch

Generated when a user resends a configuration profile to a batch of hosts.

This activity contains the following fields:
- "profile_name": The name of the configuration profile.
- "profile_uuid": The UUID of the configuration profile.
- "host_count": Number of hosts in the batch.

#### Example

```json
{
  "profile_name": "Passcode requirements",
  "profile_uuid": "a1234567-1234-1234-1234-1234567890ab",
  "host_count": 3
}
```

### installed_opt_in_configuration_profile

Generated when an opt-in configuration profile is installed on a host by the user.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "self_service": Whether the installation was initiated by the end user.
- "profile_name": The name of the configuration profile.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "self_service": true,
  "profile_name": "Passcode requirements"
}
```

### uninstalled_opt_in_configuration_profile

Generated when an opt-in configuration profile is uninstalled on a host by the user.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "self_service": Whether the uninstallation was initiated by the end user.
- "profile_name": The name of the configuration profile.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "self_service": true,
  "profile_name": "Passcode requirements"
}
```

### enabled_recovery_lock_password

Generated when a user turns on Recovery Lock password for a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that Recovery Lock password applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that Recovery Lock password applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### disabled_recovery_lock_password

Generated when a user turns off Recovery Lock password for a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that Recovery Lock password applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that Recovery Lock password applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### edited_host_name_template

Generated when a user edits the host name template for a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the host name template applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the host name template applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "name_template": The host name template, `null` if the template was cleared.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "name_template": "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL"
}
```

### created_apple_asset_declaration

Generated when creating an Apple asset declaration.

This activity contains the following fields:
- "fleet_id": the ID of the fleet the asset belongs to.
- "fleet_name": the name of the fleet the asset belongs to.
- "asset_name": the name of the asset.

#### Example

```json
{
  "fleet_id": 1,
  "fleet_name": "💻 Workstations",
  "asset_name": "My Asset"
}
```

### edited_disk_encryption_settings

Generated when a user edits disk encryption settings for hosts on a fleet (or unassigned hosts).

This activity contains the following fields:
- "fleet_id": The ID of the fleet, `null` if it applies to hosts that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "platform": The platform that disk encryption settings were updated for. Options are "macos", "windows", or "linux".

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "platform": "windows"
}
```

### edited_apple_asset_declaration

Generated when an Apple asset declaration is edited.

This activity contains the following fields:
- "fleet_id": the ID of the fleet the asset belongs to.
- "fleet_name": the name of the fleet the asset belongs to.
- "asset_name": the name of the asset.

#### Example

```json
{
  "fleet_id": 1,
  "fleet_name": "💻 Workstations",
  "asset_name": "My Asset"
}
```

### deleted_apple_asset_declaration

Generated when an Apple asset declaration is deleted.

This activity contains the following fields:
- "fleet_id": the ID of the fleet the asset belongs to.
- "fleet_name": the name of the fleet the asset belongs to.
- "asset_name": the name of the asset.

#### Example

```json
{
  "fleet_id": 1,
  "fleet_name": "💻 Workstations",
  "asset_name": "My Asset"
}
```

## Setup experience

- [changed_macos_setup_assistant](#changed-macos-setup-assistant)
- [deleted_macos_setup_assistant](#deleted-macos-setup-assistant)
- [added_bootstrap_package](#added-bootstrap-package)
- [deleted_bootstrap_package](#deleted-bootstrap-package)
- [added_end_user_agreement](#added-end-user-agreement)
- [deleted_end_user_agreement](#deleted-end-user-agreement)
- [enabled_macos_setup_end_user_auth](#enabled-macos-setup-end-user-auth)
- [disabled_macos_setup_end_user_auth](#disabled-macos-setup-end-user-auth)
- [enabled_apple_business_only_enrollment](#enabled-apple-business-only-enrollment)
- [disabled_apple_business_only_enrollment](#disabled-apple-business-only-enrollment)
- [edited_setup_experience_software](#edited-setup-experience-software)
- [created_setup_experience_script](#created-setup-experience-script)
- [deleted_setup_experience_script](#deleted-setup-experience-script)
- [enabled_managed_local_account](#enabled-managed-local-account)
- [disabled_managed_local_account](#disabled-managed-local-account)
- [canceled_setup_experience](#canceled-setup-experience)
- [edited_account_provisioning](#edited-account-provisioning)

### changed_macos_setup_assistant

Generated when a user sets the macOS setup assistant for a fleet (or no fleet).

This activity contains the following fields:
- "name": Name of the macOS setup assistant file.
- "fleet_id": The ID of the fleet that the setup assistant applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the setup assistant applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "name": "dep_profile.json",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### deleted_macos_setup_assistant

Generated when a user deletes the macOS setup assistant for a fleet (or no fleet).

This activity contains the following fields:
- "name": Name of the deleted macOS setup assistant file.
- "fleet_id": The ID of the fleet that the setup assistant applied to, `null` if it applied to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the setup assistant applied to, `null` if it applied to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "name": "dep_profile.json",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### added_bootstrap_package

Generated when a user adds a new bootstrap package to a fleet (or no fleet).

This activity contains the following fields:
- "package_name": Name of the package.
- "fleet_id": The ID of the fleet that the package applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the package applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "bootstrap_package_name": "bootstrap-package.pkg",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### deleted_bootstrap_package

Generated when a user deletes a bootstrap package from a fleet (or no fleet).

This activity contains the following fields:
- "package_name": Name of the package.
- "fleet_id": The ID of the fleet that the package applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the package applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "package_name": "bootstrap-package.pkg",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### added_end_user_agreement

Generated when a user uploads a custom end user agreement file (EULA for macOS, terms and conditions for Windows).

This activity contains the following fields:
- "platform": the platform of the agreement ("darwin" or "windows").

#### Example

```json
{
  "platform": "windows"
}
```

### deleted_end_user_agreement

Generated when a user deletes a custom end user agreement file (EULA for macOS, terms and conditions for Windows).

This activity contains the following fields:
- "platform": the platform of the agreement ("darwin" or "windows").

#### Example

```json
{
  "platform": "darwin"
}
```

### enabled_macos_setup_end_user_auth

Generated when a user turns on IdP authentication for macOS hosts that automatically enroll to a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that IdP authentication applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that IdP authentication applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### disabled_macos_setup_end_user_auth

Generated when a user turns off IdP authentication for macOS hosts that automatically enroll to a fleet (or no fleet).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that IdP authentication applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that IdP authentication applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### enabled_apple_business_only_enrollment

Generated when a user enables the setting that allows only Apple hosts assigned to Fleet in Apple Business (AB) to turn on MDM.

This activity does not contain any detail fields.

### disabled_apple_business_only_enrollment

Generated when a user disables the setting that allows only Apple hosts assigned to Fleet in Apple Business (AB) to turn on MDM.

This activity does not contain any detail fields.

### edited_setup_experience_software

Generated when a user edits setup experience software.

This activity contains the following fields:
- "platform": the platform of the host ("darwin", "android", "windows", or "linux").
- "fleet_id": the ID of the fleet associated with the setup experience (0 for no fleet).
- "fleet_name": the name of the fleet associated with the setup experience (empty for no fleet).

#### Example

```json
{
  "platform": "darwin",
  "team_id": 1,
  "team_name": "Workstations",
  "fleet_id": 1,
  "fleet_name": "Workstations"
}
```

### created_setup_experience_script

Generated when a script is added to (or replaced in) setup experience.

This activity contains the following fields:
- "fleet_id": the ID of the fleet that the script applies to (`null` for hosts that aren't assigned to a fleet).
- "fleet_name": the name of the fleet that the script applies to (`null` for hosts that aren't assigned to a fleet).
- "script_name": the name of the script that was added.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "script_name": "set-timezones.sh"
}
```

### deleted_setup_experience_script

Generated when a script is removed from setup experience.

This activity contains the following fields:
- "fleet_id": the ID of the fleet that the script applied to (`null` for hosts that aren't assigned to a fleet).
- "fleet_name": the name of the fleet that the script applied to (`null` for hosts that aren't assigned to a fleet).
- "script_name": the name of the script that was removed.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "script_name": "set-timezones.sh"
}
```

### enabled_managed_local_account

Generated when a user turns on create managed local account for a fleet (or unassigned hosts).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that create managed local account applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that create managed local account applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "platform": The platform of the fleet that create managed local account applies to.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "platform": "windows"
}
```

### disabled_managed_local_account

Generated when a user turns off create managed local account for a fleet (or unassigned hosts).

This activity contains the following fields:
- "fleet_id": The ID of the fleet that create managed local account applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that create managed local account applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "platform": The platform of the fleet that create managed local account applies to.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations",
  "platform": "windows"
}
```

### canceled_setup_experience

Generated when macOS setup experience is canceled due to software install failure.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom name that's displayed in the UI. Empty ("") when not set.
- "software_title_id": ID of the software title.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_display_name": "Adobe Acrobat.app",
  "software_title_id": 1234
}
```

### edited_account_provisioning

Generated when settings for account provisioning are edited.

This activity contains the following fields:
- "fleet_id": the ID of the fleet the label belonged to.
- "fleet_name": the name of the fleet the label belonged to.

#### Example

```json
{
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

## Commands

- [ran_custom_mdm_command](#ran-custom-mdm-command)
- [canceled_mdm_command](#canceled-mdm-command)

### ran_custom_mdm_command

Generated when a user runs a custom MDM command via API or the fleetctl CLI.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "host_uuid": UUID of the host.
- "command_uuid": UUID of the MDM command used to install the app.
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

## Integrations

- [enabled_vpp](#enabled-vpp)
- [disabled_vpp](#disabled-vpp)
- [added_google_workspace_integration](#added-google-workspace-integration)
- [edited_google_workspace_integration](#edited-google-workspace-integration)
- [deleted_google_workspace_integration](#deleted-google-workspace-integration)
- [added_microsoft_graph_credential](#added-microsoft-graph-credential)
- [edited_microsoft_graph_credential](#edited-microsoft-graph-credential)
- [deleted_microsoft_graph_credential](#deleted-microsoft-graph-credential)

### enabled_vpp

Generated when VPP features are enabled in Fleet.

This activity contains the following fields:
- "location": Organization unit associated with the VPP content token for the enabled VPP features. Apple previously called this "location." Fleet will rename it to "organization unit" in the next major version.

#### Example

```json
{
  "location": "Acme Inc."
}
```

### disabled_vpp

Generated when VPP features are disabled in Fleet.

This activity contains the following fields:
- "location": Organization unit associated with the VPP content token for the disabled VPP features. Apple previously called this "location." Fleet will rename it to "organization unit" in the next major version.

#### Example

```json
{
  "location": "Acme Inc."
}
```

### added_google_workspace_integration

Generated when a Google Workspace integration is configured to sync IdP host vitals.

This activity contains a `domain` field with the Google Workspace primary domain that was configured.

#### Example

```json
{
  "domain": "example.com"
}
```

### edited_google_workspace_integration

Generated when an existing Google Workspace integration is edited.

This activity contains a `domain` field with the Google Workspace primary domain.

#### Example

```json
{
  "domain": "example.com"
}
```

### deleted_google_workspace_integration

Generated when a Google Workspace integration is removed.

This activity contains a `domain` field with the Google Workspace primary domain that was removed.

#### Example

```json
{
  "domain": "example.com"
}
```

### added_microsoft_graph_credential

Generated when a Microsoft Graph credential is added.

This activity contains the following fields:
- "tenant_id": the Microsoft Entra tenant ID the credential authenticates against.

#### Example

```json
{
  "tenant_id": "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
}
```

### edited_microsoft_graph_credential

Generated when a Microsoft Graph credential is edited.

This activity contains the following fields:
- "tenant_id": the Microsoft Entra tenant ID the credential authenticates against.

#### Example

```json
{
  "tenant_id": "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
}
```

### deleted_microsoft_graph_credential

Generated when a Microsoft Graph credential is deleted.

This activity contains the following fields:
- "tenant_id": the Microsoft Entra tenant ID the credential authenticated against.

#### Example

```json
{
  "tenant_id": "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
}
```

## SCIM

- [scim_user_deprovision_skipped](#scim-user-deprovision-skipped)

### scim_user_deprovision_skipped

Generated when a SCIM user is deactivated or deleted but Fleet cannot determine which Fleet user to deprovision. This can indicate the SCIM user's identifiers were modified before deactivation; verify that no corresponding Fleet account remains active.

This activity contains the following fields:
- "scim_user_id": Unique ID of the SCIM user in Fleet.
- "scim_user_name": The SCIM user's userName at the time of deprovisioning.

#### Example

```json
{
  "scim_user_id": 42,
  "scim_user_name": "nondomain_user"
}
```

## Policies

- [created_policy](#created-policy)
- [edited_policy](#edited-policy)
- [deleted_policy](#deleted-policy)
- [reset_policy](#reset-policy)
- [applied_spec_policy](#applied-spec-policy)
- [ran_automation_webhook](#ran-automation-webhook)
- [failed_automation_webhook](#failed-automation-webhook)
- [ran_automation_ticket](#ran-automation-ticket)
- [failed_automation_ticket](#failed-automation-ticket)
- [ran_automation_calendar_event](#ran-automation-calendar-event)
- [failed_automation_calendar_event](#failed-automation-calendar-event)
- [ran_automation_conditional_access](#ran-automation-conditional-access)
- [failed_automation_conditional_access](#failed-automation-conditional-access)

### created_policy

Generated when creating policies.

This activity contains the following fields:
- "policy_id": the ID of the created policy.
- "policy_name": the name of the created policy.
- "fleet_id": the ID of the fleet the policy belongs to. Use -1 for global policies, 0 for "no fleet" policies.
- "fleet_name": the name of the fleet the policy belongs to. null for global policies and "no fleet" policies.

#### Example

```json
{
  "policy_id": 123,
  "policy_name": "foo",
  "team_id": 1,
  "team_name": "Workstations",
  "fleet_id": 1,
  "fleet_name": "Workstations"
}
```

### edited_policy

Generated when editing policies.

This activity contains the following fields:
- "policy_id": the ID of the edited policy.
- "policy_name": the name of the edited policy.
- "fleet_id": the ID of the fleet the policy belongs to. Use -1 for global policies, 0 for "no fleet" policies.
- "fleet_name": the name of the fleet the policy belongs to. null for global policies and "no fleet" policies.

#### Example

```json
{
  "policy_id": 123,
  "policy_name": "foo",
  "team_id": 1,
  "team_name": "Workstations",
  "fleet_id": 1,
  "fleet_name": "Workstations"
}
```

### deleted_policy

Generated when deleting policies.

This activity contains the following fields:
- "policy_id": the ID of the deleted policy.
- "policy_name": the name of the deleted policy.
- "fleet_id": the ID of the fleet the policy belonged to. Use -1 for global policies, 0 for "no fleet" policies.
- "fleet_name": the name of the fleet the policy belonged to. null for global policies and "no fleet" policies.

#### Example

```json
{
  "policy_id": 123,
  "policy_name": "foo",
  "team_id": 1,
  "team_name": "Workstations",
  "fleet_id": 1,
  "fleet_name": "Workstations"
}
```

### reset_policy

Generated when resetting a policy's results, either for all hosts or for a single host.

This activity contains the following fields:
- "policy_id": the ID of the reset policy.
- "policy_name": the name of the reset policy.
- "fleet_id": the ID of the fleet the policy belongs to. Use -1 for global policies, 0 for "no fleet" policies.
- "fleet_name": the name of the fleet the policy belongs to. null for global policies and "no fleet" policies.
- "host_id": the ID of the host whose result was reset. Only present when the reset was scoped to a single host.
- "host_display_name": the display name of the host whose result was reset. Only present when the reset was scoped to a single host.

#### Example

```json
{
  "policy_id": 123,
  "policy_name": "foo",
  "team_id": 1,
  "team_name": "Workstations",
  "fleet_id": 1,
  "fleet_name": "Workstations",
  "host_id": 42,
  "host_display_name": "Anna's MacBook Pro"
}
```

### applied_spec_policy

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

## Reports

- [created_saved_query](#created-saved-query)
- [edited_saved_query](#edited-saved-query)
- [deleted_saved_query](#deleted-saved-query)
- [deleted_multiple_saved_query](#deleted-multiple-saved-query)
- [applied_spec_saved_query](#applied-spec-saved-query)
- [live_query](#live-query)

### created_saved_query

Generated when creating a new report.

This activity contains the following fields:
- "report_id": the ID of the created report.
- "report_name": the name of the created report.
- "fleet_id": the ID of the fleet the report belongs to.
- "fleet_name": the name of the fleet the report belongs to.

#### Example

```json
{
  "query_id": 123,
  "query_name": "foo",
  "team_id": 1,
  "team_name": "Workstations",
  "report_id": 123,
  "report_name": "foo",
  "fleet_id": 1,
  "fleet_name": "Workstations"
}
```

### edited_saved_query

Generated when editing a saved report.

This activity contains the following fields:
- "report_id": the ID of the report being edited.
- "report_name": the name of the report being edited.
- "fleet_id": the ID of the fleet the report belongs to.
- "fleet_name": the name of the fleet the report belongs to.

#### Example

```json
{
  "query_id": 123,
  "query_name": "foo",
  "team_id": 1,
  "team_name": "Workstations",
  "report_id": 123,
  "report_name": "foo",
  "fleet_id": 1,
  "fleet_name": "Workstations"
}
```

### deleted_saved_query

Generated when deleting a saved report.

This activity contains the following fields:
- "report_name": the name of the report being deleted.
- "fleet_id": the ID of the fleet the report belongs to.
- "fleet_name": the name of the fleet the report belongs to.

#### Example

```json
{
  "query_name": "foo",
  "team_id": 1,
  "team_name": "Workstations",
  "report_name": "foo",
  "fleet_id": 1,
  "fleet_name": "Workstations"
}
```

### deleted_multiple_saved_query

Generated when deleting multiple saved reports.

This activity contains the following fields:
- "report_ids": list of IDs of the deleted saved reports.
- "fleet_id": the ID of the fleet the reports belonged to. -1 for global reports, null for no fleet.
- "fleet_name": the name of the fleet the reports belonged to. null for global or no fleet reports.

#### Example

```json
{
  "query_ids": [1, 42, 100],
  "team_id": 123,
  "team_name": "Workstations",
  "report_ids": [1, 42, 100],
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### applied_spec_saved_query

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

### live_query

Generated when running live queries.

This activity contains the following fields:
- "targets_count": Number of hosts where the live query was targeted to run.
- "query_sql": The SQL query to run on hosts.
- "report_name": Name of the report (this field is not set if this was not a saved report).

#### Example

```json
{
  "targets_count": 5000,
  "query_sql": "SELECT * from osquery_info;",
  "query_name": "foo",
  "report_name": "foo"
}
```

## Scripts

- [ran_script](#ran-script)
- [added_script](#added-script)
- [deleted_script](#deleted-script)
- [edited_script](#edited-script)
- [updated_script](#updated-script)
- [canceled_run_script](#canceled-run-script)
- [ran_script_batch](#ran-script-batch)
- [scheduled_script_batch](#scheduled-script-batch)
- [canceled_script_batch](#canceled-script-batch)

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

### added_script

Generated when a script is added to a fleet (or no fleet).

This activity contains the following fields:
- "script_name": Name of the script.
- "fleet_id": The ID of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### deleted_script

Generated when a script is deleted from a fleet (or no fleet).

This activity contains the following fields:
- "script_name": Name of the script.
- "fleet_id": The ID of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### edited_script

Generated when a user edits the scripts of a fleet (or no fleet) via the fleetctl CLI.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the scripts apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the scripts apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### updated_script

Generated when a script is updated.

This activity contains the following fields:
- "script_name": Name of the script.
- "fleet_id": The ID of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the script applies to, `null` if it applies to devices that are not in a fleet ("Unassigned").

#### Example

```json
{
  "script_name": "set-timezones.sh",
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
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

### ran_script_batch

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

### scheduled_script_batch

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

### canceled_script_batch

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

## Software

- [installed_software](#installed-software)
- [installed_all_self_service_software](#installed-all-self-service-software)
- [notified_end_user_before_patching](#notified-end-user-before-patching)
- [edited_patch_deadline_settings](#edited-patch-deadline-settings)
- [uninstalled_software](#uninstalled-software)
- [added_software](#added-software)
- [edited_software](#edited-software)
- [deleted_software](#deleted-software)
- [added_app_store_app](#added-app-store-app)
- [deleted_app_store_app](#deleted-app-store-app)
- [installed_app_store_app](#installed-app-store-app)
- [edited_app_store_app](#edited-app-store-app)
- [canceled_install_software](#canceled-install-software)
- [canceled_uninstall_software](#canceled-uninstall-software)
- [canceled_install_app_store_app](#canceled-install-app-store-app)

### installed_software

Generated when a Fleet-maintained app or custom package is installed on a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "install_uuid": ID of the software installation.
- "self_service": Whether the installation was initiated by the end user.
- "software_title": Name of the software.
- "software_display_name": Custom name that's displayed in the UI. Empty ("") when not set.
- "software_package": Filename of the installer.
- "status": Status of the software installation.
- "source": Software source type (e.g., "pkg_packages", "sh_packages", "ps1_packages").
- "policy_id": ID of the policy whose failure triggered the installation. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered installation. Null if no associated policy.
- "command_uuid": ID of the in-house app installation.
- "from_setup_experience": Whether the installation was triggered as part of the setup experience.
- "failure_reason": Reason the installation failed before reaching the device (e.g. an unresolvable Fleet variable in the managed app configuration). Only present when "status" is "failed_install" and Fleet failed the install pre-flight; omitted otherwise.
- "skipped_install": Whether the install was skipped because the app was open. This is `true` when the Fleet-maintained app is installed by the patch policy's automation, when the policy had `patch_when_closed`, `notify_before_patching`, or `patch_deadline` enabled at the time of the install. Only present when "status" is "failed_install" and the install was skipped for this reason, omitted otherwise.
- "patch_when_closed": Whether the policy that triggered this install had `patch_when_closed` enabled at the time of the install.
- "notify_before_patching": Whether the policy that triggered this install had `notify_before_patching` enabled at the time of the install. Only present when `true`.
- "patch_deadline": Whether the policy that triggered this install had `patch_deadline` enabled at the time of the install. Only present when `true`.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Falcon.app",
  "software_display_name": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "self_service": true,
  "install_uuid": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
  "status": "pending_install",
  "source": "pkg_packages",
  "policy_id": 1337,
  "policy_name": "Ensure 1Password is installed and up to date",
  "from_setup_experience": false,
  "patch_when_closed": false
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

### notified_end_user_before_patching

Generated when Fleet shows an end user a notification before patching, for patch policies with `notify_before_patching` or `patch_deadline` set to `true`.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "patch_notification_uuid": ID of the notification. The reminder for the same notification reuses this ID.
- "status": Whether notification is displayed or failed.
- "software": Software titles included in the notification. Each entry has `title` (raw name), `title_id`, and `display_name` (set only when an admin picked a per-team override).
- "policy_ids": IDs of the patch policies included in the notification.
- "time_before": Either 1 hour or 5 minutes before patch is forced.
- "install_at": Timestamp at which the apps will be installed if the end user doesn't update them first.
- "script_execution_id": Execution ID of the script run that displayed the notification to the end user.
- "exit_code": Exit code of the notification script. `0` on success; non-zero encodes the failure reason (e.g. `41` for a locked screen).

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "patch_notification_uuid": "9f8c1c1e-0b1a-4f2a-9a3e-6c5d4b3a2f10",
  "software": [
    { "title": "1Password", "title_id": 42, "display_name": "1Password 8" },
    { "title": "Slack", "title_id": 57 }
  ],
  "status": "success",
  "policy_ids": [1337, 1338],
  "time_before": 3600,
  "install_at": "2026-08-06T14:00:00Z",
  "script_execution_id": "c672cccb-fcfa-4424-a25f-dd2d2e3eb3be",
  "exit_code": 0
}
```

### edited_patch_deadline_settings

Generated when the patch deadline days or time are modified for a fleet.

This activity contains the following fields:
- "fleet_id": The ID of the fleet that the patch deadline settings apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").
- "fleet_name": The name of the fleet that the patch deadline settings apply to, `null` if they apply to devices that are not in a fleet ("Unassigned").
- "days": Days of the week when the deadline applies.
- "time": The deadline, formatted as `HH:MM` in the host's local time.

#### Example

```json
{
  "team_id": 3,
  "team_name": "Workstations",
  "fleet_id": 3,
  "fleet_name": "Workstations",
  "days": ["monday", "wednesday", "friday"],
  "time": "11:00"
}
```

### uninstalled_software

Generated when a Fleet-maintained app or custom package is uninstalled on a host.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom name that's displayed in the UI. Empty ("") when not set.
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
  "software_display_name": "Falcon.app",
  "script_execution_id": "ece8d99d-4313-446a-9af2-e152cd1bad1e",
  "self_service": false,
  "status": "uninstalled",
  "source": "pkg_packages"
}
```

### added_software

Generated when a Fleet-maintained app or custom package is added to Fleet.

This activity contains the following fields:
- "software_title": Name of the software.
- "software_package": Filename of the installer.
- "fleet_name": Name of the fleet to which this software was added. `null` if it was added to no fleet.
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
  "team_name": "Workstations",
  "team_id": 123,
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

### edited_software

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
- "pinned_version": The version a Fleet-maintained app is pinned to — a specific version (e.g. `"149.0.7827.54"`) or a caret major-version constraint (e.g. `"^147"`). `null` when the app automatically updates to the latest version, or when the version wasn't changed.

#### Example

```json
{
  "software_title": "Falcon.app",
  "software_package": "FalconSensor-6.44.pkg",
  "team_name": "Workstations",
  "team_id": 123,
  "fleet_name": "Workstations",
  "fleet_id": 123,
  "self_service": true,
  "software_title_id": 2234,
  "software_icon_url": "/api/latest/fleet/software/titles/2234/icon?team_id=123",
  "software_display_name": "Crowdstrike Falcon",
  "pinned_version": "149.0.7827.54",
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

### deleted_software

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
  "team_name": "Workstations",
  "team_id": 123,
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

### added_app_store_app

Generated when an App Store app is added to Fleet.

This activity contains the following fields:
- "software_title": Name of the App Store app.
- "software_title_id": ID of the added software title.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "platform": Platform of the app (`android`, `darwin`, `ios`, or `ipados`).
- "vpp_token": The VPP token the app was added from, at the time of the activity. `null` for Google Play apps.
  - "id": ID of the VPP token.
  - "location": Name of the organization unit in Apple Business for the VPP token.
  - "country_code": Country code of the VPP token (e.g. `us`).
- "self_service": App installation can be initiated by device owner.
- "fleet_name": Name of the fleet to which this App Store app was added, or `null` if it was added to no fleet.
- "fleet_id": ID of the fleet to which this App Store app was added, or `null` if it was added to no fleet.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array.
- "version_name": Name of the admin-created App Store app version. Defaults to "Default version" when the admin didn't set one.
- "configuration": The app's managed configuration, if set. For iOS and iPadOS apps it is in XML format, and for Android Play Store apps it is in JSON format.
- "auto_update_enabled": Whether automatic updates are enabled for this version (iOS and iPadOS only; omitted when unset).
- "auto_update_window_start": Start of the auto-update maintenance window in `HH:MM` host local time (omitted when unset).
- "auto_update_window_end": End of the auto-update maintenance window in `HH:MM` host local time (omitted when unset).

#### Example

```json
{
  "software_title": "Slack",
  "software_title_id": 123,
  "app_store_id": "618783545",
  "platform": "ios",
   "vpp_token": {
    "id": 2,
    "location": "Acme Japan",
    "country_code": "jp"
  },
  "self_service": false,
  "team_name": "Workstations",
  "team_id": 1,
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
  ],
  "version_name": "Production",
  "configuration": "<dict><key>com.slack.workspace</key><string>example.slack.com</string></dict>"
}
```

### deleted_app_store_app

Generated when an App Store app is deleted from Fleet.

This activity contains the following fields:
- "software_title": Name of the App Store app.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "platform": Platform of the app (`android`, `darwin`, `ios`, or `ipados`).
- "fleet_name": Name of the fleet from which this App Store app was deleted, or `null` if it was deleted from no fleet.
- "fleet_id": ID of the fleet from which this App Store app was deleted, or `null` if it was deleted from no fleet.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array.
- "version_name": Name of the admin-created App Store app version that was deleted. Defaults to "Default version" when the admin didn't set one.
- "configuration": The app's managed configuration, if set. For iOS and iPadOS apps it is in XML format, and for Android Play Store apps it is in JSON format.

#### Example

```json
{
  "software_title": "Slack",
  "app_store_id": "618783545",
  "platform": "ios",
  "team_name": "Workstations",
  "team_id": 1,
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
  ],
  "version_name": "Production",
  "configuration": "<dict><key>ServerURL</key><string>https://example.com</string></dict>"
}
```

### installed_app_store_app

Generated when an App Store app is installed on a device.

This activity contains the following fields:
- "host_id": ID of the host on which the app was installed.
- "self_service": App installation was initiated by device owner.
- "host_display_name": Display name of the host.
- "software_title": Name of the App Store app.
- "software_display_name": Custom name that's displayed in the UI. Empty ("") when not set.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "vpp_token": The VPP token the app was added from, at the time of the activity. `null` for Google Play apps, or if the VPP token was deleted before the install finished.
  - "id": ID of the VPP token.
  - "location": Name of the organization unit in Apple Business for the VPP token.
  - "country_code": Country code of the VPP token (e.g. `us`).
- "status": Status of the App Store app installation.
- "command_uuid": UUID of the MDM command used to install the app.
- "policy_id": ID of the policy whose failure triggered the install. Null if no associated policy.
- "policy_name": Name of the policy whose failure triggered the install. Null if no associated policy.
- "host_platform": Platform of the host (e.g., "darwin", "ios", "ipados", "android").
- "from_setup_experience": Whether the app was installed as part of the setup experience.
- "from_configuration_resend": Whether Fleet re-installed the app to deliver an updated managed app configuration or a label change.
- "from_auto_update": Whether the app was installed by an automatic update.
- "failure_reason": Reason the installation failed before reaching the device (e.g. an unresolvable Fleet variable in the managed app configuration). Only present when "status" is "failed_install" and Fleet failed the install pre-flight; omitted otherwise.
- "version_name": Name of the app version that was installed. An app can have more than one version on the same fleet, each with its own settings and managed app configuration.
- "configuration": The managed app configuration that was applied, in XML format for iOS and iPadOS apps and JSON format for Android apps. Null if the version has no managed app configuration.

#### Example

```json
{
  "host_id": 42,
  "self_service": true,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Logic Pro",
  "software_display_name": "Logic Pro",
  "app_store_id": "1234567",
  "vpp_token": {
    "id": 2,
    "location": "Acme Japan",
    "country_code": "jp"
  },
  "command_uuid": "98765432-1234-1234-1234-1234567890ab",
  "status": "installed",
  "policy_id": 123,
  "policy_name": "[Install Software] Logic Pro",
  "host_platform": "darwin",
  "from_auto_update": false,
  "from_setup_experience": false,
  "from_configuration_resend": false,
  "version_name": "Logic Pro",
  "configuration": null
}
```

#### Example (iOS app with a managed app configuration)

```json
{
  "host_id": 57,
  "self_service": false,
  "host_display_name": "Anna's iPhone",
  "software_title": "Zoom Workplace",
  "app_store_id": "546505307",
  "command_uuid": "12345678-90ab-cdef-1234-567890abcdef",
  "policy_id": null,
  "policy_name": null,
  "from_setup_experience": false,
  "from_configuration_resend": true,
  "version_name": "Production",
  "configuration": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>..."
}
```

### edited_app_store_app

Generated when an App Store app is updated in Fleet.

This activity contains the following fields:
- "software_title": Name of the App Store app.
- "software_title_id": ID of the updated app's software title.
- "app_store_id": ID of the app on the Apple App Store or Google Play.
- "platform": Platform of the app (`android`, `darwin`, `ios`, or `ipados`).
- "vpp_token": The VPP token the app was added from, at the time of the activity. `null` for Google Play apps.
  - "id": ID of the VPP token.
  - "location": Name of the organization unit in Apple Business for the VPP token.
  - "country_code": Country code of the VPP token (e.g. `us`).
- "self_service": App installation can be initiated by device owner.
- "fleet_name": Name of the fleet on which this App Store app was updated, or `null` if it was updated on no fleet.
- "fleet_id": ID of the fleet on which this App Store app was updated, or `null` if it was updated on no fleet.
- "labels_include_any": Target hosts that have any label in the array.
- "labels_exclude_any": Target hosts that don't have any label in the array.
- "software_display_name": Display name of the software title.
- "auto_update_enabled": Whether automatic updates are enabled for iOS/iPadOS App Store (VPP) apps.
- "auto_update_window_start": Update window start time (local time of the device) when automatic updates will take place for iOS/iPadOS App Store (VPP) apps, formatted as HH:MM.
- "auto_update_window_end": Update window end time (local time of the device) when automatic updates will take place for iOS/iPadOS App Store (VPP) apps, formatted as HH:MM.
- "version_name": Name of the admin-created App Store app version that was edited. Defaults to "Default version" when the admin didn't set one.
- "configuration": The app's managed configuration, if set. For iOS and iPadOS apps it is in XML format, and for Android Play Store apps it is in JSON format.


#### Example

```json
{
  "software_title": "Slack",
  "software_title_id": 123,
  "app_store_id": "618783545",
  "platform": "ios",
  "vpp_token": {
    "id": 2,
    "location": "Acme Japan",
    "country_code": "jp"
  },
  "self_service": true,
  "team_name": "Workstations",
  "team_id": 1,
  "fleet_name": "Workstations",
  "fleet_id": 1,
  "software_icon_url": "/api/latest/fleet/software/titles/123/icon?team_id=1",
  "labels_include_any": [
    {
      "name": "Engineering",
      "id": 12
    },
    {
      "name": "Product",
      "id": 17
    }
  ],
  "software_display_name": "Slack",
  "auto_update_enabled": true,
  "auto_update_window_start": "22:00",
  "auto_update_window_end": "02:00",
  "version_name": "Production",
  "configuration": "<dict><key>com.slack.workspace</key><string>example.slack.com</string></dict>"
}
```

### canceled_install_software

Generated when upcoming activity `installed_software` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom name that's displayed in the UI. Empty ("") when not set.
- "software_title_id": ID of the software title.
- "from_setup_experience": Whether the install was part of the setup experience.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_display_name": "Adobe Acrobat.app",
  "software_title_id": 12334
}
```

### canceled_uninstall_software

Generated when upcoming activity `uninstalled_software` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom name that's displayed in the UI. Empty ("") when not set.
- "software_title_id": ID of the software title.

#### Example

```json
{
  "host_id": 1,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_display_name": "Adobe Acrobat.app",
  "software_title_id": 12334
}
```

### canceled_install_app_store_app

Generated when upcoming activity `installed_app_store_app` is canceled.

This activity contains the following fields:
- "host_id": ID of the host.
- "host_display_name": Display name of the host.
- "software_title": Name of the software.
- "software_display_name": Custom name that's displayed in the UI. Empty ("") when not set.
- "software_title_id": ID of the software title.
- "from_setup_experience": Whether the install was part of the setup experience.

#### Example

```json
{
  "host_id": 123,
  "host_display_name": "Anna's MacBook Pro",
  "software_title": "Adobe Acrobat.app",
  "software_display_name": "Adobe Acrobat.app",
  "software_title_id": 12334
}
```

## Self-service categories

- [added_self_service_category](#added-self-service-category)
- [edited_self_service_category](#edited-self-service-category)
- [deleted_self_service_category](#deleted-self-service-category)

### added_self_service_category

Generated when a self-service category is added to a fleet.

This activity contains the following fields:
- "self_service_category_name": Name of the self-service category that was added.
- "fleet_name": Name of the fleet the category was added to.
- "fleet_id": ID of the fleet the category was added to.

#### Example

```json
{
  "self_service_category_name": "🛟 Support",
  "fleet_name": "💻 Workstations",
  "fleet_id": 123
}
```

### edited_self_service_category

Generated when a self-service category is renamed on a fleet.

This activity contains the following fields:
- "self_service_category_name": New name of the self-service category.
- "fleet_name": Name of the fleet the category belongs to.
- "fleet_id": ID of the fleet the category belongs to.

#### Example

```json
{
  "self_service_category_name": "🛟 Support utilities",
  "fleet_name": "💻 Workstations",
  "fleet_id": 123
}
```

### deleted_self_service_category

Generated when a self-service category is deleted from a fleet.

This activity contains the following fields:
- "self_service_category_name": Name of the self-service category that was deleted.
- "fleet_name": Name of the fleet the category was deleted from.
- "fleet_id": ID of the fleet the category was deleted from.

#### Example

```json
{
  "self_service_category_name": "🛟 Support",
  "fleet_name": "💻 Workstations",
  "fleet_id": 123
}
```

## Fleets

- [created_team](#created-team)
- [deleted_team](#deleted-team)
- [applied_spec_team](#applied-spec-team)

### created_team

Generated when creating fleets.

This activity contains the following fields:
- "fleet_id": unique ID of the created fleet.
- "fleet_name": the name of the created fleet.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### deleted_team

Generated when deleting fleets.

This activity contains the following fields:
- "fleet_id": unique ID of the deleted fleet.
- "fleet_name": the name of the deleted fleet.

#### Example

```json
{
  "team_id": 123,
  "team_name": "Workstations",
  "fleet_id": 123,
  "fleet_name": "Workstations"
}
```

### applied_spec_team

Generated when applying fleet specs.

This activity contains a field "teams" where each item contains the fleet details with the following fields:
- "id": Unique ID of the fleet.
- "name": Name of the fleet.

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

## Users

- [created_user](#created-user)
- [deleted_user](#deleted-user)
- [changed_user_global_role](#changed-user-global-role)
- [deleted_user_global_role](#deleted-user-global-role)
- [changed_user_team_role](#changed-user-team-role)
- [deleted_user_team_role](#deleted-user-team-role)

### created_user

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

### deleted_user

Generated when a user is deleted. An admin can delete the user, or SCIM can deprovision one that was deleted or deactivated in the IdP. For a SCIM deprovisioning, Fleet is the author of the activity rather than a Fleet user.

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

### changed_user_global_role

Generated when user global roles are changed.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": New global role of the edited user.
- "jit": Whether the role was assigned via just-in-time (JIT) user provisioning (omitted when false).

#### Example

```json
{
  "user_id": 42,
  "user_name": "Foo",
  "user_email": "foo@example.com",
  "role": "Observer"
}
```

### deleted_user_global_role

Generated when user global roles are deleted.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Deleted global role of the edited user.
- "jit": Whether the role was removed via just-in-time (JIT) user provisioning (omitted when false).

#### Example

```json
{
  "user_id": 43,
  "user_name": "Foo",
  "user_email": "foo@example.com",
  "role": "Maintainer"
}
```

### changed_user_team_role

Generated when user fleet roles are changed.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Fleet role set to the edited user.
- "fleet_id": Unique ID of the fleet of the changed role.
- "fleet_name": Name of the fleet of the changed role.
- "jit": Whether the role was assigned via just-in-time (JIT) user provisioning (omitted when false).

#### Example

```json
{
  "user_id": 43,
  "user_name": "Foo",
  "user_email": "foo@example.com",
  "role": "Maintainer",
  "team_id": 5,
  "team_name": "Bar",
  "fleet_id": 5,
  "fleet_name": "Bar"
}
```

### deleted_user_team_role

Generated when user fleet roles are deleted.

This activity contains the following fields:
- "user_id": Unique ID of the edited user in Fleet.
- "user_name": Name of the edited user.
- "user_email": E-mail of the edited user.
- "role": Fleet role deleted from the edited user.
- "fleet_id": Unique ID of the fleet of the deleted role.
- "fleet_name": Name of the fleet of the deleted role.
- "jit": Whether the role was removed via just-in-time (JIT) user provisioning (omitted when false).

#### Example

```json
{
  "user_id": 44,
  "user_name": "Foo",
  "user_email": "foo@example.com",
  "role": "Observer",
  "team_id": 2,
  "team_name": "Zoo",
  "fleet_id": 2,
  "fleet_name": "Zoo"
}
```

## Custom variables

- [created_custom_variable](#created-custom-variable)
- [updated_custom_variable](#updated-custom-variable)
- [deleted_custom_variable](#deleted-custom-variable)

### created_custom_variable

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

### updated_custom_variable

Generated when a custom variable's value is updated.

This activity contains the following fields:
- "custom_variable_name": the name of the custom variable.

#### Example

```json
{
  "custom_variable_name": "SOME_API_KEY"
}
```

### deleted_custom_variable

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

<meta name="title" value="Global audit logs">
<meta name="pageOrderInSection" value="60">
<meta name="description" value="Learn how Fleet logs administrative actions in JSON format."> 
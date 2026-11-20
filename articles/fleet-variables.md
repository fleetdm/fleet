# Variables

_Available in Fleet Premium_

Fleet supports built-in variables (prefixed with `$FLEET_VAR_`) to inject host vitals into [scripts](https://fleetdm.com/guides/scripts), [configuration profiles](https://fleetdm.com/guides/custom-os-settings) or [managed app configurations](https://fleetdm.com/guides/install-app-store-apps#managed-configuration) (iOS, iPadOS, and Android). 

You can also create [custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) (prefixed with `$FLEET_SECRET_`) to define your own key-value pairs. 

To store a different value per host, create [custom host vitals](https://fleetdm.com/guides/custom-host-vitals) (prefixed with `$FLEET_HOST_VITAL_`).

Each variable type works in different places. See [Where variables work](#where-variables-work).

For macOS configuration profiles, you can also use any of Apple's [built-in variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0) in [Automated Certificate Management Environment (ACME)](https://developer.apple.com/documentation/devicemanagement/acmecertificate), [Simple Certificate Enrolment Protocol (SCEP)](https://developer.apple.com/documentation/devicemanagement/scep), or [VPN](https://developer.apple.com/documentation/devicemanagement/vpn) payloads.

When the variable's value changes, Fleet automatically resends configuration profiles and managed app configurations to the hosts they apply to. For scripts, variables are resolved at execution time.

## Built-in variables

| Name | Description |
|---|---|
| <span style="display: inline-block; min-width: 240px;">`$FLEET_VAR_HOST_END_USER_IDP_USERNAME`</span> | Host's IdP username (e.g. "user@example.com"). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME` | Host's IdP full name. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` | Local part of the email (e.g. john from john@example.com). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_GROUPS` | Comma separated IdP groups that host belongs to. When these change, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT` | Host's IdP department. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID` | Host's IdP immutable user ID, ingested via SCIM as `externalId` (e.g. an Entra ID object ID). Unlike the IdP username, this doesn't change when the user is renamed. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_HARDWARE_SERIAL` | Host's hardware serial number. For personal (BYOD) iOS, iPadOS, and Android hosts enrolled with a Managed Apple Account or a work profile, this is the enrollment ID. |
| `$FLEET_VAR_HOST_UUID` | Host's hardware UUID. For personal (BYOD) iOS, iPadOS, and Android hosts enrolled with a Managed Apple Account or a work profile, this is the enrollment ID. For company-owned Android hosts, this is an ID that Fleet generates from the device's brand and serial number. |
| `$FLEET_VAR_HOST_PLATFORM` | Host's platform. Values are `"macos"`, `"ios"`, `"ipados"`, and `"windows"`, and `"android"`. |
| `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>` | Fleet-managed one-time challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the custom SCEP certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>` | Fleet-managed SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` | Fleet-managed ID that's required to automatically renew certificates. The ID must be specified in the Organizational Unit (OU) field in the configuration profile. In some use cases, like Okta, which drops the OU, it can go in the Common Name (CN) instead. |
| `$FLEET_VAR_DIGICERT_PASSWORD_<CA_NAME>` | Fleet-managed password required to decode the base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the DigiCert certificate authority configured in **Settings > Integrations > Certificate authorities**. macOS, iOS, and iPadOS only. |
| `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>` | Fleet-managed base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the DigiCert certificate authority configured in **Settings > Integrations > Certificate authorities**. macOS, iOS, and iPadOS only. |
| `$FLEET_VAR_EJBCA_PASSWORD_<CA_NAME>` | Fleet-managed password required to decode the base64-encoded certificate data issued by a specified EJBCA certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the EJBCA certificate authority configured in **Settings > Integrations > Certificate authorities**. macOS, iOS, and iPadOS only. |
| `$FLEET_VAR_EJBCA_DATA_<CA_NAME>` | Fleet-managed base64-encoded certificate data issued by a specified EJBCA certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the EJBCA certificate authority configured in **Settings > Integrations > Certificate authorities**. macOS, iOS, and iPadOS only. |
| `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID` | ID used for SCEP configuration profile on Windows. It must be included in the `<LocURI>` field. |
| `$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_<CA_NAME>` | Fleet-managed one-time Smallstep challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the Smallstep certificate authority configured in **Settings > Integrations > Certificate authorities**. macOS, iOS, and iPadOS only. |
| `$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_<CA_NAME>` | Fleet-managed Smallstep SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. macOS, iOS, and iPadOS only. |
| `$FLEET_VAR_NDES_SCEP_CHALLENGE` | Fleet-managed one-time NDES challenge password used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_NDES_SCEP_PROXY_URL` | Fleet-managed NDES SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN` | Fleet-managed, per-host registration token for [Platform SSO with Fleet's SSO extension](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet). It's only allowed in the `RegistrationToken` key of a Fleet SSO extension payload. macOS only. |
| `$FLEET_VAR_CONDITIONAL_ACCESS_SCEP_URL` | Fleet's conditional access SCEP endpoint URL, used in the certificate profile for [Okta conditional access](https://fleetdm.com/guides/okta-conditional-access-integration). It must be in the `ServerURL` field, and the profile must also include `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` and a global enroll secret in the `Challenge` field (we recommend a [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles)). Windows only. |



If certificate authority (CA) variables (ex. `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>`) don't exist, GitOps dry runs will succeed but GitOps runs will fail.

> Profiles that use IdP variables will trigger a resend when the IdP user is removed from the host, but will fail sending a new profile due to missing variables, leaving the old one on the device. Once the host has a new IdP user it will be resent again with fresh values.

> The host vital and IdP variables listed above are also accepted in the DigiCert CA fields `certificate_common_name`, `certificate_seat_id`, and `certificate_user_principal_names` in `org_settings.certificate_authorities.digicert[]`, and in the EJBCA CA fields `certificate_common_name`, `username_template`, and `certificate_user_principal_names` in `org_settings.certificate_authorities.ejbca[]` (Apple hosts only). The legacy `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` is also accepted for backwards compatibility. When these values change, or when the CA fields are edited, Fleet reissues the certificate.

> If Fleet can't fill in a variable in an app's managed configuration, Fleet doesn't send the app or its configuration to that host. Learn more in the [App Store apps guide](https://fleetdm.com/guides/install-app-store-apps#variables).

## Where variables work

| Where | Built-in variables (`$FLEET_VAR_`) | Custom variables (`$FLEET_SECRET_`) | Custom host vitals (`$FLEET_HOST_VITAL_`) |
|---|---|---|---|
| [Scripts](https://fleetdm.com/guides/scripts) (macOS, Windows, and Linux), including software install, post-install, and uninstall scripts, and setup experience scripts | ✅ (only `$FLEET_VAR_HOST_*` and `$FLEET_VAR_NDES_SCEP_CHALLENGE`) | ✅ | ✅ |
| Apple (macOS, iOS, and iPadOS) configuration profiles | ✅ (only `$FLEET_VAR_HOST_*` in declarations (`.json`)) | ✅ (not in `PayloadDisplayName`) | ✅ (not in certificate authority payloads) |
| Windows configuration profiles (`.xml`) | ✅ | ✅ | ✅ |
| Android configuration profiles (`.json`) | ✅ (only `$FLEET_VAR_HOST_*`) | ❌ | ✅ |
| iOS and iPadOS [managed app configuration](https://fleetdm.com/guides/install-app-store-apps#managed-configuration) | ✅ (only `$FLEET_VAR_HOST_*`) | ❌ | ❌ |
| Android [managed app configuration](https://fleetdm.com/guides/install-app-store-apps#managed-configuration) | ✅ (only `$FLEET_VAR_HOST_*`) | ❌ | ✅ |
| [Host name templates](https://fleetdm.com/guides/rename-hosts-with-a-naming-template) (macOS, iOS, and iPadOS) | ✅ (only `$FLEET_VAR_HOST_*`) | ✅ | ✅ |
| DigiCert and EJBCA certificate authority fields | ✅ (only `$FLEET_VAR_HOST_*`) | ❌ | ❌ |
| Software pre-install query | ❌ | ❌ | ❌ |

`$FLEET_VAR_HOST_*` are the host and IdP variables: `$FLEET_VAR_HOST_HARDWARE_SERIAL`, `$FLEET_VAR_HOST_UUID`, `$FLEET_VAR_HOST_PLATFORM`, and the `$FLEET_VAR_HOST_END_USER_IDP_*` variables.

Fleet shows an error when you save a script, profile, or configuration that uses a variable where it isn't supported, with two exceptions. Fleet doesn't block custom host vitals in iOS and iPadOS managed app configurations, or custom variables in Android configuration profiles and managed app configurations. In these places, Fleet sends the variable to the host as written, without its value.


<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="authorFullName" value="Marko Lisica">
<meta name="publishedOn" value="2026-04-08">
<meta name="articleTitle" value="Variables">
<meta name="description" value="Use variables to add IdP or host data to scripts, configuration profiles, or managed app configs for device-specific settings.">

# Built-in variables

_Available in Fleet Premium_

Fleet supports built-in variables (prefixed with `$FLEET_VAR_`) to inject host vitals into [scripts](https://fleetdm.com/guides/scripts), [configuration profiles](https://fleetdm.com/guides/custom-os-settings) or managed app configurations ([iOS/iPadOS](https://fleetdm.com/guides/install-app-store-apps#ios-and-ipados-managed-configuration), [Android](https://fleetdm.com/guides/install-app-store-apps#android-managed-configuration)). 

You can also create [custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) (prefixed with `$FLEET_SECRET_`) to define your own key-value pairs. 

To store a different value per host, create [custom host vitals](https://fleetdm.com/guides/custom-host-vitals) (prefixed with `$FLEET_HOST_VITAL_`) and reference them in scripts and configuration profiles. 

For macOS configuration profiles, you can also use any of Apple's [built-in variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0) in [Automated Certificate Management Environment (ACME)](https://developer.apple.com/documentation/devicemanagement/acmecertificate), [Simple Certificate Enrolment Protocol (SCEP)](https://developer.apple.com/documentation/devicemanagement/scep), or [VPN](https://developer.apple.com/documentation/devicemanagement/vpn) payloads.

When the variable's value changes, Fleet automatically resends configuration profiles. For managed app configurations, changes apply on next app install or update. For scripts, variables are resolved at execution time.

Built-in variables:

| Name | Scripts | Configuration profiles | Managed app configuration | Description |
|---|---|---|---|---|
| <span style="display: inline-block; min-width: 240px;">`$FLEET_VAR_HOST_END_USER_IDP_USERNAME`</span> | macOS, Windows, Linux | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's IdP username (e.g. "user@example.com"). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME` | macOS, Windows, Linux | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's IdP full name. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` | macOS, Windows, Linux | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Local part of the email (e.g. john from john@example.com). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_GROUPS` | macOS, Windows, Linux | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Comma separated IdP groups that host belongs to. When these change, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT` | macOS, Windows, Linux | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's IdP department. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_HARDWARE_SERIAL` | macOS, Windows, Linux | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's hardware serial number. Not available for personal (BYOD) Android hosts on Android 12 and later. On user enrolled iOS and iPadOS hosts with Managed Apple Account, it resolves to the enrollment ID. This is deprecated, so use `$FLEET_VAR_HOST_ENROLLMENT_ID` instead. |
| `$FLEET_VAR_HOST_UUID` | macOS, Windows, Linux | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's hardware UUID. Personal (BYOD) hosts and company-owned Android hosts don't have a hardware UUID, so it resolves to a different ID instead. This is deprecated for those hosts. See [Personal hosts and company-owned Android hosts](#personal-hosts-and-company-owned-android-hosts). |
| `$FLEET_VAR_HOST_ENROLLMENT_ID` | None | iOS, iPadOS, Android (including certificate templates) | iOS, iPadOS, and Android | Host's enrollment ID. Only available for personal (BYOD) hosts: user enrolled iOS and iPadOS hosts with Managed Apple Account, and personally owned Android hosts with a work profile on Android 12 and later. This is the same value as the `enrollment_id` returned by the [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) API endpoint. |
| `$FLEET_VAR_HOST_PLATFORM` | macOS, Windows, Linux | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's platform. Values are `"macos"`, `"ios"`, `"ipados"`, and `"windows"`, and `"android"`. |
| `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>` | None | macOS, iOS, iPadOS, Windows | None | Fleet-managed one-time challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the custom SCEP certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>` | None | macOS, iOS, iPadOS, Windows | None | Fleet-managed SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`          | None | None | macOS, iOS, iPadOS, Windows | Fleet-managed ID that's required to automatically renew certificates. The ID must be specified in the Organizational Unit (OU) field in the configuration profile. In some use cases, like Okta, which drops the OU, it can go in the Common Name (CN) instead. |
| `$FLEET_VAR_DIGICERT_PASSWORD_<CA_NAME>` | None | macOS, iOS, iPadOS | None | Fleet-managed password required to decode the base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the DigiCert certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>` | None | macOS, iOS, iPadOS | None | Fleet-managed base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the DigiCert certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_EJBCA_PASSWORD_<CA_NAME>` | None | macOS, iOS, iPadOS | None | Fleet-managed password required to decode the base64-encoded certificate data issued by a specified EJBCA certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the EJBCA certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_EJBCA_DATA_<CA_NAME>` | None | macOS, iOS, iPadOS | None | Fleet-managed base64-encoded certificate data issued by a specified EJBCA certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the EJBCA certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID` | None | Windows | None | ID used for SCEP configuration profile on Windows. It must be included in the `<LocURI>` field. |
| `$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_<CA_NAME>` | None | macOS, iOS, iPadOS | None | Fleet-managed one-time Smallstep challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the Smallstep certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_<CA_NAME>` | None | macOS, iOS, iPadOS | None | Fleet-managed Smallstep SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_NDES_SCEP_CHALLENGE` | None | macOS, iOS, iPadOS, Windows | None | Fleet-managed one-time NDES challenge password used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_NDES_SCEP_PROXY_URL` | None | macOS, iOS, iPadOS, Windows | None | Fleet-managed NDES SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |



If a variable isn't available for a host, the configuration profile or managed app configuration fails for that host, and Fleet shows an error explaining which variable it couldn't populate.

### Personal hosts and company-owned Android hosts

On these hosts, `$FLEET_VAR_HOST_HARDWARE_SERIAL` and `$FLEET_VAR_HOST_UUID` don't match the `hardware_serial` and `uuid` returned by the [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) API endpoint. They resolve to the same values as before Fleet 5, so existing configuration profiles, managed app configurations, and certificate templates keep working. Using them on these hosts is deprecated and will fail in a future major version.

| Host | `$FLEET_VAR_HOST_HARDWARE_SERIAL` | `$FLEET_VAR_HOST_UUID` | Use instead |
|---|---|---|---|
| Personal iOS and iPadOS | Enrollment ID | Enrollment ID | `$FLEET_VAR_HOST_ENROLLMENT_ID` |
| Personal Android, Android 12 and later | Not available | Enrollment ID | `$FLEET_VAR_HOST_ENROLLMENT_ID` |
| Personal Android, Android 11 and earlier | Hardware serial | ID generated by Fleet | `$FLEET_VAR_HOST_HARDWARE_SERIAL` |
| Company-owned Android | Hardware serial | ID generated by Fleet | `$FLEET_VAR_HOST_HARDWARE_SERIAL` |

If certificate authority (CA) variables (ex. `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>`) don't exist, GitOps dry runs will succeed but GitOps runs will fail.

> Profiles that use IdP variables will trigger a resend when the IdP user is removed from the host, but will fail sending a new profile due to missing variables, leaving the old one on the device. Once the host has a new IdP user it will be resent again with fresh values.

> The host vital and IdP variables listed above are also accepted in the DigiCert CA fields `certificate_common_name`, `certificate_seat_id`, and `certificate_user_principal_names` in `org_settings.certificate_authorities.digicert[]` (Apple hosts only). The legacy `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` is also accepted for backwards compatibility. When these values change, or when the CA fields are edited, Fleet reissues the certificate.


<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="authorFullName" value="Marko Lisica">
<meta name="publishedOn" value="2026-04-08">
<meta name="articleTitle" value="Built-in variables">
<meta name="description" value="Use variables to add IdP or host data to scripts, configuration profiles, or managed app configs for device-specific settings.">

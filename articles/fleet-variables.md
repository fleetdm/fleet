# Built-in variables

_Available in Fleet Premium_

Fleet supports built-in variables (prefixed with `$FLEET_VAR_`) to inject host vitals into [configuration profiles](https://fleetdm.com/guides/custom-os-settings) or managed app configurations ([iOS/iPadOS](https://fleetdm.com/guides/install-app-store-apps#ios-and-ipados-managed-configuration), [Android](https://fleetdm.com/guides/install-app-store-apps#android-managed-configuration)). 

You can also create [custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) (prefixed with `$FLEET_SECRET_`) to define your own key-value pairs. 

For macOS configuration profiles, you can also use any of Apple's [built-in variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0) in [Automated Certificate Management Environment (ACME)](https://developer.apple.com/documentation/devicemanagement/acmecertificate), [Simple Certificate Enrolment Protocol (SCEP)](https://developer.apple.com/documentation/devicemanagement/scep), or [VPN](https://developer.apple.com/documentation/devicemanagement/vpn) payloads.

When the variable's value changes, Fleet automatically resends configuration profiles. For managed app configurations, changes apply on next app install or update.

Built-in variables:

| Name | Configuration profiles | Managed app configuration | Description |
|---|---|---|---|
| <span style="display: inline-block; min-width: 240px;">`$FLEET_VAR_NDES_SCEP_CHALLENGE`</span> | macOS, iOS, iPadOS | None | Fleet-managed one-time NDES challenge password used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_NDES_SCEP_PROXY_URL` | macOS, iOS, iPadOS | None | Fleet-managed NDES SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's IdP username (e.g. "user@example.com"). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME` | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's IdP full name. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Local part of the email (e.g. john from john@example.com). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_GROUPS` | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Comma separated IdP groups that host belongs to. When these change, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT` | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's IdP department. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_HARDWARE_SERIAL` | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's hardware serial number. Not available for user-enrolled iOS and iPadOS hosts with Managed Apple Account. |
| `$FLEET_VAR_HOST_UUID` | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's hardware UUID, or Enrollment ID for user-enrolled iOS and iPadOS hosts. |
| `$FLEET_VAR_HOST_PLATFORM` | macOS, iOS, iPadOS, Windows, Android | iOS, iPadOS, and Android | Host's platform. Values are `"macos"`, `"ios"`, `"ipados"`, `"windows"`, and `"android"`. |
| `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>` | macOS, iOS, iPadOS, Windows | None | Fleet-managed one-time challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the custom SCEP certificate authority configured in **Settings > Integrations > Certificate enrollment**. |
| `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>` | macOS, iOS, iPadOS, Windows | None | Fleet-managed SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`                | macOS, iOS, iPadOS, Windows | Fleet-managed ID that's required to automatically renew certificates. The ID must be specified in the Organizational Unit (OU) field in the configuration profile. |
| `$FLEET_VAR_DIGICERT_PASSWORD_<CA_NAME>` | macOS, iOS, iPadOS | None | Fleet-managed password required to decode the base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the DigiCert certificate authority configured in **Settings > Integrations > Certificate enrollment**. |
| `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>` | macOS, iOS, iPadOS | None | Fleet-managed base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the DigiCert certificate authority configured in **Settings > Integrations > Certificate enrollment**. |
| `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID` | Windows | None | ID used for SCEP configuration profile on Windows. It must be included in the `<LocURI>` field. |
| `$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_<CA_NAME>` | macOS, iOS, iPadOS | None | Fleet-managed one-time Smallstep challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the Smallstep certificate authority configured in **Settings > Integrations > Certificate enrollment**. |
| `$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_<CA_NAME>` | macOS, iOS, iPadOS | None | Fleet-managed Smallstep SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |



If certificate authority (CA) variables (ex. `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>`) don't exist, GitOps dry runs will succeed but GitOps runs will fail.

> Profiles that use IdP variables will trigger a resend when the IdP user is removed from the host, but will fail sending a new profile due to missing variables, leaving the old one on the device. Once the host has a new IdP user it will be resent again with fresh values.


<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="authorFullName" value="Marko Lisica">
<meta name="publishedOn" value="2026-04-08">
<meta name="articleTitle" value="Built-in variables">
<meta name="description" value="Use variables to add IdP or host data to configuration profiles or managed app configs for device-specific settings.">

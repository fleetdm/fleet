# Built-in variables

_Available in Fleet Premium_

Fleet supports built-in variables (prefixed with `$FLEET_VAR_`) to inject host vitals into [configuration profiles](https://fleetdm.com/guides/custom-os-settings) or [iOS/iPadOS managed app configurations](https://fleetdm.com/guides/install-app-store-apps#ios-and-ipados-managed-configuration). 

You can also create [custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) (prefixed with `$FLEET_SECRET_`) to define your own key-value pairs. 

For macOS configuration profiles, you can also use any of Apple's [built-in variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0) in [Automated Certificate Management Environment (ACME)](https://developer.apple.com/documentation/devicemanagement/acmecertificate), [Simple Certificate Enrolment Protocol (SCEP)](https://developer.apple.com/documentation/devicemanagement/scep), or [VPN](https://developer.apple.com/documentation/devicemanagement/vpn) payloads.

When the variable's value changes, Fleet automatically resends configuration profiles. For managed app configurations, changes apply on next app install or update.

Built-in variables:

| Name | Configuration profiles | Managed app configuration | Description |
|---|---|---|---|
| <span style="display: inline-block; min-width: 240px;">`$FLEET_VAR_NDES_SCEP_CHALLENGE`</span> | macOS, iOS, iPadOS | None | Fleet-managed one-time NDES challenge password used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_NDES_SCEP_PROXY_URL` | macOS, iOS, iPadOS | None | Fleet-managed NDES SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` | macOS, iOS, iPadOS, Windows | iOS and iPadOS | Host's IdP username (e.g. "user@example.com"). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME` | macOS, iOS, iPadOS, Windows | iOS and iPadOS | Host's IdP full name. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` | macOS, iOS, iPadOS, Windows | iOS and iPadOS | Local part of the email (e.g. john from john@example.com). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_GROUPS` | macOS, iOS, iPadOS, Windows | iOS and iPadOS | Comma separated IdP groups that host belongs to. When these change, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT` | macOS, iOS, iPadOS, Windows | iOS and iPadOS | Host's IdP department. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_HARDWARE_SERIAL` | macOS, iOS, iPadOS, Windows | iOS and iPadOS | Host's hardware serial number. Not available for user enrolled iOS and iPadOS hosts with Managed Apple Account. |
| `$FLEET_VAR_HOST_UUID` | macOS, iOS, iPadOS, Windows | iOS and iPadOS | Host's hardware UUID, or Enrollment ID for user enrolled iOS and iPadOS hosts. |
| `$FLEET_VAR_HOST_PLATFORM` | macOS, iOS, iPadOS, Windows | iOS and iPadOS | Host's platform. Values are `"macos"`, `"ios"`, `"ipados"`, and `"windows"`. |
| `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>` | macOS, iOS, iPadOS, Windows | None | Fleet-managed one-time challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the custom SCEP certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>` | macOS, iOS, iPadOS, Windows | None | Fleet-managed SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_SCEP_RENEWAL_ID` | macOS, iOS, iPadOS, Windows | None | Fleet-managed ID that's required to automatically renew Smallstep, Microsoft NDES, and custom SCEP certificates. The ID must be specified in the Organizational Unit (OU) field in the configuration profile. |
| `$FLEET_VAR_DIGICERT_PASSWORD_<CA_NAME>` | macOS, iOS, iPadOS | None | Fleet-managed password required to decode the base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the DigiCert certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>` | macOS, iOS, iPadOS | None | Fleet-managed base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the DigiCert certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID` | Windows | None | ID used for SCEP configuration profile on Windows. It must be included in the `<LocURI>` field. |
| `$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_<CA_NAME>` | macOS, iOS, iPadOS | None | Fleet-managed one-time Smallstep challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the Smallstep certificate authority configured in **Settings > Integrations > Certificate authorities**. |
| `$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_<CA_NAME>` | macOS, iOS, iPadOS | None | Fleet-managed Smallstep SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |



If certificate authority (CA) variables (ex. `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>`) don't exist, GitOps dry runs will succeed but GitOps runs will fail.


<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="authorFullName" value="Marko Lisica">
<meta name="publishedOn" value="2026-04-08">
<meta name="articleTitle" value="Built-in variables">
<meta name="description" value="You can use variables to incorporate IdP or host vitals into a configuration profile or managed app configuration for deploying device-specific settings.">
# Variables

You can use variables to incorporate IdP or host vitals into a configuration profile or managed app configuration for deploying device-specific settings.

For macOS configuration profiles, you can use any of Apple's [built-in variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0) in [Automated Certificate Management Environment (ACME)](https://developer.apple.com/documentation/devicemanagement/acmecertificate), [Simple Certificate Enrolment Protocol (SCEP)](https://developer.apple.com/documentation/devicemanagement/scep), or [VPN](https://developer.apple.com/documentation/devicemanagement/vpn) payloads.

Fleet also supports adding [GitHub](https://docs.github.com/en/actions/learn-github-actions/variables#defining-environment-variables-for-a-single-workflow) or [GitLab](https://docs.gitlab.com/ci/variables/) environment variables in your configuration profiles and managed app configuration. Use `$ENV_VARIABLE` format.

When the variable's value changes, configuration profiles are automatically resent. Managed app configurtaion is applied on next install or update.

In Fleet Premium, you can use reserved variables beginning with `$FLEET_VAR_`. Fleet will populate these variables when profiles are sent to hosts. Supported variables are:

| Name | Platforms | Description |
| ---- | --------- | ----------- |
| <span style="display: inline-block; min-width: 240px;">`$FLEET_VAR_NDES_SCEP_CHALLENGE`</span> | macOS, iOS, iPadOS | Fleet-managed one-time NDES challenge password used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_NDES_SCEP_PROXY_URL`                   | macOS, iOS, iPadOS | Fleet-managed NDES SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`            | macOS, iOS, iPadOS, Windows | Host's IdP username (e.g. "user@example.com"). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME`           | macOS, iOS, iPadOS, Windows | Host's IdP full name. When this changes, Fleet will automatically resend the profile. |`            | macOS, iOS, iPadOS | Host's IdP username. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` | macOS, iOS, iPadOS, Windows | Local part of the email (e.g. john from john@example.com). When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_GROUPS`              | macOS, iOS, iPadOS, Windows | Comma separated IdP groups that host belongs to. When these change, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT`          | macOS, iOS, iPadOS, Windows | Host's IdP department. When this changes, Fleet will automatically resend the profile. |
| `$FLEET_VAR_HOST_UUID`                             | macOS, iOS, iPadOS, Windows | Host's hardware UUID. |
| `$FLEET_VAR_HOST_HARDWARE_SERIAL`                  | macOS, iOS, iPadOS, Windows | Host's hardware serial number. |
| `$FLEET_VAR_HOST_PLATFORM`                         | macOS, iOS, iPadOS, Windows | Host's platform. Values are `"macos"`, `"ios"`, `"ipados"`, and `"windows"`. |
| `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>`       | macOS, iOS, iPadOS, Windows | Fleet-managed one-time challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the certificate authority configured in [custom_scep_proxy](#custom-scep-proxy). |
| `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>`       | macOS, iOS, iPadOS, Windows | Fleet-managed SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |
| `$FLEET_VAR_SCEP_RENEWAL_ID`       | macOS, iOS, iPadOS, Windows | Fleet-managed ID that's required to automatically renew Smallstep, Microsoft NDES, and custom SCEP certificates. The ID must be specified in the Organizational Unit (OU) field in the configuration profile. |
| `$FLEET_VAR_DIGICERT_PASSWORD_<CA_NAME>`           | macOS, iOS, iPadOS | Fleet-managed password required to decode the base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the certificate authority configured in [digicert](#digicert). |
| `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>`               | macOS, iOS, iPadOS | Fleet-managed base64-encoded certificate data issued by a specified DigiCert certificate authority during PKCS12 profile deployment. `<CA_NAME>` should be replaced with name of the certificate authority configured in [digicert](#digicert). |
| `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID`               | Windows | ID used for SCEP configuration profile on Windows. It must be included in the `<LocURI>` field.|
| `$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_<CA_NAME>`       | macOS, iOS, iPadOS | Fleet-managed one-time Smallstep challenge password used during SCEP certificate configuration profile deployment. `<CA_NAME>` should be replaced with name of the certificate authority configured in [custom_scep_proxy](#custom-scep-proxy). |
| `$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_<CA_NAME>`       | macOS, iOS, iPadOS | Fleet-managed Smallstep SCEP proxy endpoint URL used during SCEP certificate configuration profile deployment. |

The dollar sign (`$`) can be escaped so it's not considered a variable by using a backslash (e.g. `\$100`). Additionally, `MY${variable}HERE` syntax can be used to put strings around the variable.

In XML, certain characters (`&`, `<`, `>`, `"`, `'`) must be escaped because they have special meanings in the markup language. GitHub and GitLab environment variables, as well as Fleet's reserved variables, will be automatically escaped when used in a `.mobileconfig` configuration profile. For example, `&` will become `&amp;`.

If certificate authority (CA) variables (ex. `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>`) don't exist, GitOps dry runs will succeed but GitOps runs will fail.

To hide variable values in the API and UI, you can use Fleet's [custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles#gitops).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="authorFullName" value="Marko Lisica">
<meta name="publishedOn" value="2026-04-08">
<meta name="articleTitle" value="Variables">
<meta name="description" value="You can use variables to incorporate IdP or host vitals into a configuration profile or managed app configuration for deploying device-specific settings.">
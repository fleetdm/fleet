# Entra conditional access integration

Fleet v4.69.0 integrates with Entra ID to provide Microsoft "Conditional Access" for macOS.
Fleet can now connect to Microsoft Entra ID and block end users from logging into third-party apps if they're failing any Fleet policies (non-compliant).

> This feature is only available on Fleet Cloud and currently supports macOS.

For more information about this feature see https://learn.microsoft.com/en-us/intune/intune-service/protect/device-compliance-partners.

### Configure Fleet as compliance partner in Intune

The steps to configure Fleet as "Compliance partner" for macOS devices can be found here: https://learn.microsoft.com/en-us/intune/intune-service/protect/device-compliance-partners. The steps are executed in the Intune portal (https://intune.microsoft.com).

After this is done, the "Fleet partner" will be shown with a "Pending activation" status.

![Conditional access pending activation](../website/assets/images/compliance-partner-pending-activation.png)

## Setup integration in Fleet

Now we need to connect and provision Fleet to operate on your Entra ID tenant (activate partner).

To connect Fleet to your Entra account you need your "Microsoft Entra tenant ID", which can be found in https://entra.microsoft.com. You can follow the steps in https://learn.microsoft.com/en-us/entra/fundamentals/how-to-find-tenant to get your tenant ID.

Once you have your tenant ID, go to Fleet: `Settings` > `Integrations` > `Conditional access` and enter the tenant ID.

![Conditional access setup](../website/assets/images/conditional-access-setup.png)

After clicking `Save` you will be redirected to https://login.microsoftonline.com to consent to the permissions for Fleet's multi-tenant application.
After consenting you will be redirected back to Fleet (to `/settings/integrations/conditional-access`).

The next step is to enable and configure the integration on your teams.

## Configure devices in Fleet

The following steps need to be configured on the Fleet teams you want to enable Microsoft "Conditional Access".

### Automatic install software for Company Portal.app

To enroll macOS devices to Entra for Conditional Access you will need to configure Fleet to automatically install the "Company Portal" macOS application.

The Company Portal macOS application can be downloaded from https://go.microsoft.com/fwlink/?linkid=853070.

To configure automatic installation on your macOS devices you go to `Software` > `Select the team` > `Add software` > `Custom package`. Upload the `CompanyPortal-Installer.pkg` and check the `Automatic install` option.

!['Company Portal.app' automatic install](../website/assets/images/company-portal-automatic.png)

### Configure profile

For Entra's Conditional Access feature we need to deploy a Platform SSO extension for Company Portal.
The extension must be deployed via configuration profiles. For more information see https://learn.microsoft.com/en-us/intune/intune-service/configuration/platform-sso-macos#step-3---deploy-the-company-portal-app-for-macos.

Add the following configuration profile to your teams in `Controls` > `OS settings` > `Custom settings` > `+ Add profile`.

`company-portal-single-signon-extension.mobileconfig`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>ExtensionData</key>
			<dict>
				<key>Enable_SSO_On_All_ManagedApps</key>
				<integer>1</integer>
			</dict>
			<key>ExtensionIdentifier</key>
			<string>com.microsoft.CompanyPortalMac.ssoextension</string>
			<key>Hosts</key>
			<array/>
			<key>PayloadDisplayName</key>
			<string>Company Portal Single Sign-On Extension</string>
			<key>PayloadIdentifier</key>
			<string>com.apple.extensiblesso.F82C8673-439F-4751-B562-42517A5FD990</string>
			<key>PayloadType</key>
			<string>com.apple.extensiblesso</string>
			<key>PayloadUUID</key>
			<string>F82C8673-439F-4751-B562-42517A5FD990</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>TeamIdentifier</key>
			<string>UBF8T346G9</string>
			<key>Type</key>
			<string>Redirect</string>
			<key>URLs</key>
			<array>
				<string>https://login.microsoftonline.com</string>
				<string>https://login.microsoft.com</string>
				<string>https://sts.windows.net</string>
				<string>https://login.partner.microsoftonline.cn</string>
				<string>https://login.chinacloudapi.cn</string>
				<string>https://login.microsoftonline.us</string>
				<string>https://login-us.microsoftonline.com</string>
			</array>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>Company Portal Single Sign-On Extension</string>
	<key>PayloadIdentifier</key>
	<string>com.fleetdm.D4DB9649-BA4A-4FA1-AA4F-D1CF606308B1</string>
	<key>PayloadOrganization</key>
	<string></string>
	<key>PayloadRemovalDisallowed</key>
	<true/>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>com.fleetdm.D4DB9649-BA4A-4FA1-AA4F-D1CF606308B1</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
```

### Label for "Company portal.app installed and configured"

We will need to create a dynamic label to determine which macOS devices have "Company Portal" installed and have the SSO extension installed (this is the configuration profile from the previous step).
We will use this label to trigger users to log in to Entra via Company Portal.

Go to `Hosts` > `Filter by platform or label` > `Add label +` > `Dynamic`.

- Name: `Company Portal installed and configured`
- Description: `Company Portal is installed on the host and configured to log in to Entra using the Platform SSO extension (which must be deployed via configuration profiles).`
- Query:
  ```sql
  SELECT 1 WHERE EXISTS (
    SELECT 1 FROM apps WHERE bundle_identifier = 'com.microsoft.CompanyPortalMac'
  ) AND EXISTS (
    SELECT 1 FROM managed_policies WHERE value = 'com.microsoft.CompanyPortalMac.ssoextension'
  );
  ```
- Platform: `macOS`

### Policy and script to trigger users to log in to Entra

We will create a policy and an associated script to trigger users to log in to Entra on their macOS devices.

#### Create policy

Go to `Policies` > `Select team` > `Add policy`.
- Query:
  ```sql
  -- Checks if the user has logged in to Entra on this host (Fleet requires the Device ID and User Principal Name to be able to mark devices as compliant/non-compliant).
  SELECT 1 FROM (SELECT common_name AS device_id FROM certificates WHERE issuer LIKE '/DC=net+DC=windows+CN=MS-Organization-Access+OU%' LIMIT 1)
  CROSS JOIN (SELECT label as user_principal_name FROM keychain_items WHERE account = 'com.microsoft.workplacejoin.registeredUserPrincipalName' LIMIT 1);
  ```
- Name: `Company Portal sign in`.
- Description: `This policy checks that the user has signed to Entra on the host using the Company Portal application (using the flow for Conditional access).`
- Resolve: `The script associated with this policy will open the Company Portal application in the correct mode for Conditional access.`
- Target: `macOS`
- Select `Custom` > `Include any` > `Company Portal installed and configured` (this is the label we configured in the previous step)

#### Create script

We will need the following script to trigger users to log in to Entra:
`user-enroll-entra-company-portal.sh`:
```bash
#!/bin/bash

# Company Portal has to be started with the following arguments to work in "Conditional access" mode.
open "/Applications/Company Portal.app" --args -r
```

To upload the script go to: `Controls` > `Select team` > `Scripts` > `Upload`.

#### Associate script to policy

Go to `Policies` > `Select team` > `Manage automations v` > `Scripts`.
Check `Company Portal sign in` and select `user-enroll-entra-company-portal.sh`.

## Configure Fleet policies for Conditional Access

The final step is to configure Fleet policies that will determine whether a device is marked as "compliant" or "not compliant" on Entra.

Go to `Policies` > `Select team` > `Automations` > `Conditional access`.
1. Make sure the feature is enabled for the team.
2. Check the policies you want for Conditional access.

IMPORTANT: If a device is not MDM-enrolled to Fleet then it will be marked as "not compliant".

### Disabling "Conditional Access" on a team

If you need all your hosts on a team to be marked as "Compliant" (e.g. to unblock access to a resource) go to `Policies` > `Select team` > `Automations` > `Conditional access`, uncheck all policies and hit `Save`. The hosts will be marked as "Compliant" the next time they check in with policy results (within one hour, or by refetching manually).

To disable the "Conditional Access" feature on a team go to `Policies` > `Select team` > `Automations` > `Conditional access` > `Disable`.
Once disabled, hosts will not be reporting compliance status to Entra anymore.

## GitOps

Here's the full configuration that you can apply via GitOps.
> It is only including the necessary keys for this integration.

`default.yml`:
```yml
labels:
- description: Company Portal is installed on the host and configured to log in to
    Entra using the Platform SSO extension (which must be deployed via configuration
    profiles).
  label_membership_type: dynamic
  name: Company Portal installed and configured
  platform: darwin
  query: |-
    SELECT 1 WHERE EXISTS (
      SELECT 1 FROM apps WHERE bundle_identifier = 'com.microsoft.CompanyPortalMac'
    ) AND EXISTS (
      SELECT 1 FROM managed_policies WHERE value = 'com.microsoft.CompanyPortalMac.ssoextension'
    );
org_settings:
  integrations:
    conditional_access_enabled: true # enables setting for "No team"
```

`teams/team-name.yml` (should be the same for `teams/no-team.yml` with the `team_settings` removed):
```yml
team_settings:
  integrations:
    conditional_access_enabled: true
controls:
  macos_settings:
    custom_settings:
    - path: ../lib/team-name/profiles/company-portal-single-signon-extension.mobileconfig
policies:
- calendar_events_enabled: false
  conditional_access_enabled: true
  critical: false
  description: Example description for compliance policy 2
  name: Compliance check policy 2
  platform: darwin
  query: SELECT * FROM osquery_info WHERE start_time < 0;
  resolution: Resolution steps for this policy
- calendar_events_enabled: false
  conditional_access_enabled: false
  critical: false
  description: Policy triggers automatic install of Company Portal on each host that's
    missing this software.
  install_software:
    hash_sha256: 931db4af2fe6320a1bfb6776fae75b6f7280a947203a5a622b2cae00e8f6b6e6
      # Company Portal (CompanyPortal-Installer.pkg) version 5.2504.0
  name: '[Install software] Company Portal (pkg)'
  platform: darwin
  query: SELECT 1 FROM apps WHERE bundle_identifier = 'com.microsoft.CompanyPortalMac';
  resolution:
- calendar_events_enabled: false
  conditional_access_enabled: false
  critical: false
  description: This policy checks that the user has signed to Entra on the host using
    the Company Portal application (using the flow for Conditional access).
  labels_include_any:
  - Company Portal installed and configured
  name: Company Portal sign in
  platform: darwin
  query: |-
    -- Checks if the user has logged in to Entra on this host (Fleet requires the Device ID and User Principal Name to be able to mark devices as compliant/non-compliant).
    SELECT 1 FROM (SELECT common_name AS device_id FROM certificates WHERE issuer LIKE '/DC=net+DC=windows+CN=MS-Organization-Access+OU%' LIMIT 1)
    CROSS JOIN (SELECT label as user_principal_name FROM keychain_items WHERE account = 'com.microsoft.workplacejoin.registeredUserPrincipalName' LIMIT 1);
  resolution: The script associated with this policy will open the Company Portal
    application in the correct mode for Conditional access.
  run_script:
    path: ../lib/team-name/scripts/user-enroll-entra-company-portal.sh
software:
  packages:
  - hash_sha256: 931db4af2fe6320a1bfb6776fae75b6f7280a947203a5a622b2cae00e8f6b6e6
      # Company Portal (CompanyPortal-Installer.pkg) version 5.2504.0
    install_script:
      path: ../lib/team-name/scripts/company-portal-darwin-install
    uninstall_script:
      path: ../lib/team-name/scripts/company-portal-darwin-uninstall
```

`lib/team-name/scripts/user-enroll-entra-company-portal.sh`: See [Create script](#create-script).
`lib/team-name/profiles/company-portal-single-signon-extension.mobileconfig`: See [Configure profile](#configure-profile).

<meta name="articleTitle" value="Entra conditional access integration">
<meta name="authorFullName" value="Lucas Manuel Rodriguez">
<meta name="authorGitHubUsername" value="lucasmrod">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-05-28">
<meta name="description" value="Entra conditional access integration">
# Entra Conditional Access integration

Fleet v4.69.0.
macOS only.

## Setup integration in Fleet

Setup screens and steps.

## Configure devices for integration

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

Name: `Company Portal installed and configured`
Description: `Company Portal is installed on the host and configured to log in to Entra using the Platform SSO extension (which must be deployed via configuration profiles).`
Query:
```sql
SELECT 1 WHERE EXISTS (
	SELECT 1 FROM apps WHERE bundle_identifier = 'com.microsoft.CompanyPortalMac'
) AND EXISTS (
	SELECT 1 FROM managed_policies WHERE value = 'com.microsoft.CompanyPortalMac.ssoextension'
);
```
Platform: `macOS`

### Policy and script to trigger users to log in to Entra

We will create a policy and an associated script to trigger users to log in to Entra on their macOS devices.

#### Create policy

Go to `Policies` > `Select team` > `Add policy`.
Query:
```sql
-- Checks if the user has logged in to Entra on this host (Fleet requires the Device ID and User Principal Name to be able to mark devices as compliant/non-compliant).
SELECT 1 FROM (SELECT common_name AS device_id FROM certificates WHERE issuer LIKE '/DC=net+DC=windows+CN=MS-Organization-Access+OU%' LIMIT 1)
CROSS JOIN (SELECT label as user_principal_name FROM keychain_items WHERE account = 'com.microsoft.workplacejoin.registeredUserPrincipalName' LIMIT 1);
```
Name: `Company Portal sign in`.
Description: `This policy checks that the user has signed to Entra on the host using the Company Portal application (using the flow for Conditional access).`
Resolve: `The script associated with this policy will open the Company Portal application in the correct mode for Conditional access.`
Target: `macOS`
Select `Custom` > `Include any` > `Company Portal installed and configured` (this is the label we configured in the previous step)

#### Create script

We will need the following script to trigger users to log in to Entra:
`user-enroll-entra-company-portal.sh`:
```bash
#!/bin/bash

open "/Applications/Company Portal.app" --args -r
```

To upload the script go to: `Controls` > `Select team` > `Scripts` > `Upload`.

#### Associate script to policy

Go to `Policies` > `Select team` > `Manage automations v` > `Scripts`.
1. Make sure the feature is enabled for the team.
2. Check `Company Portal sign in` and select `user-enroll-entra-company-portal.sh`.

## Configure Fleet policies for Conditional Access

The final step is to configure Fleet policies to determine whether a device is marked as Compliant or not-Compliant on Entra.

Go to `Policies` > `Select team` > `Automations` > 
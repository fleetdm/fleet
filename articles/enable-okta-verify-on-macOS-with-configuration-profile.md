# Enable Okta Verify on macOS using configuration profile

## Introduction

This guide will show you how to install [Okta Verify](https://help.okta.com/en-us/content/topics/mobile/okta-verify-overview.htm) on your macOS hosts and set them as managed by issuing a SCEP certificate via a configuration profile [managed through Fleet](https://fleetdm.com/guides/custom-os-settings).

By following these steps, you can automate the deployment of Okta Verify across your devices. This will allow you to enforce multifactor authentication policies, improve device security, and manage user access seamlessly.

## Prerequisites

* MDM enabled and configured

## Step-by-step instructions

### **Step 1: Install Okta Verify on your hosts**

Okta Verify can be installed:

* As a Volume Purchasing Program (VPP) application, follow [these steps to install VPP apps](https://fleetdm.com/guides/install-vpp-apps-on-macos-using-fleet).
* As a *.pkg *file download the [installer from Okta](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/ov-install-options-macos.htm) and [deploy the installer using Fleet](https://fleetdm.com/guides/deploy-security-agents).

After installing Okta Verify on the host, the device will be registered in Okta.

### **Step 2: Issue a SCEP certificate for management attestation**

The next step to ensure Okta detects the device as managed is to issue a SCEP certificate.

* Follow the instructions on the Okta documentation to [configure a certificate authority](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/configure-ca-main.htm) using a **static** SCEP challenge.
* In your text editor, copy and paste the following configuration profile and edit the relevant values:
    * `[REPLACE_WITH_CHALLENGE] `with the SCEP challenge you generated in the previous step.
    * `[REPLACE_WITH_URL]`with the URL to your SCEP server.
    * Adjust the `CN `value according to your organization's needs. You can use any of the [profile variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0) to uniquely identify your device. In the example `%ComputerName%` `managementAttestation` `%HardwareUUID%,` the certificate Common Name (CN) will contain both the computer name and the hardware UUID.

```xml

<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Inc//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadIdentifier</key>
	<string>Ignored</string>
	<key>PayloadUUID</key>
	<string>Ignored</string>
	<key>PayloadDisplayName</key>
	<string>SCEP device attestation</string>
	<key>PayloadContent</key>
	<array>
  	<dict>
    	<key>PayloadContent</key>
    	<dict>
      	<key>Key Type</key>
      	<string>RSA</string>
      	<key>Challenge</key>
      	<string>[REPLACE_WITH_CHALLENGE]</string>
      	<key>Key Usage</key>
      	<integer>1</integer>
      	<key>Keysize</key>
      	<integer>2048</integer>
      	<key>URL</key>
  	<string>[REPLACE_WITH_URL]</string>
  	<key>AllowAllAppsAccess</key>
  	<true />
  	<key>KeyIsExtractable</key>
  	<false />
      	<key>Subject</key>
      	<array>
        	<array>
          	<array>
            	<string>O</string>
            	<string>Fleet</string>
          	</array>
        	</array>
        	<array>
          	<array>
            	<string>CN</string>
            	<string>%ComputerName% managementAttestation %HardwareUUID%</string>
          	</array>
        	</array>
      	</array>
    	</dict>
    	<key>PayloadIdentifier</key>
    	<string>com.apple.security.scep.C2D94E67-4F1A-4A3C-8142-7523A8D35713</string>
    	<key>PayloadType</key>
    	<string>com.apple.security.scep</string>
    	<key>PayloadUUID</key>
    	<string>632289FA-C3E0-481A-A417-BF40012FB729</string>
    	<key>PayloadVersion</key>
    	<integer>1</integer>
  	</dict>
	</array>
  </dict>
</plist>

```

> Make sure to use `.mobileconfig` as the file extension

* Enforce the configuration profile on your hosts. You can follow [this guide on enforcing custom OS settings in Fleet](https://fleetdm.com/guides/custom-os-settings).
* You can optionally verify the issued certificate by opening Keychain Access on the device or by running a [live query](https://fleetdm.com/guides/get-current-telemetry-from-your-devices-with-live-queries):

```sql
SELECT * FROM certificates where common_name like '%managementAttestation%';
```

### **Step 3: Configure device management in Okta**

With Okta Verify installed and an attestation certificate in place, all left is to configure Okta and the device for device management, useful links from the Okta documentation are:

* [Managed devices](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/managed-main.htm)
* [Enable and configure Okta Verify](https://help.okta.com/en-us/content/topics/mobile/okta-verify-overview.htm)

Make sure the device is properly set up in Okta and that the user has used Okta FastPass at least once to see it as managed on the Okta dashboard.

## Conclusion

This guide covered how to install Okta Verify on your macOS hosts, issue a SCEP certificate for management attestation, and configure device management in Okta. By automating this process through Fleet, you can enforce multi-factor authentication, improve device security, and ensure that devices accessing your organization’s resources are properly managed.

For more detailed information on managing devices and using Okta Verify, explore the Okta documentation and Fleet’s guides to optimize your device management strategy further.

See Fleet's [documentation](https://fleetdm.com/docs/using-fleet) and additional [guides](https://fleetdm.com/guides) for more details on advanced setups, software features, and vulnerability detection.

<meta name="articleTitle" value="Enable Okta Verify on macOS using configuration profile">
<meta name="authorFullName" value="Roberto Dip">
<meta name="authorGitHubUsername" value="roperzh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-09-23">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-security-agents-1600x900@2x.png">
<meta name="description" value="This guide will walk you through enabling Okta verify on macOS hosts using a configuration profile.">

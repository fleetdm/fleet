# Sysadmin diaries: restoring `fleetd`

![Sysadmin diaries: restoring fleetd](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

As a sysadmin, unexpected challenges are part of the job. In our last diary installment, we discussed the methods of [device enrollment](https://fleetdm.com/guides/sysadmin-diaries-device-enrollment). Today, we tackle a new challenge: a surly employee has deleted the `fleetd` files from their device. What happens next? Can we restore the `fleetd` agent using Mobile Device Management (MDM) commands? In this post, we’ll explore various methods to tackle this situation and ensure your fleet of devices remains secure and compliant.


### What is `fleetd` and why it matters

`Fleetd` is a suite of agents Fleet provides to collect and manage information about your devices. It includes osquery, Orbit, Fleet Desktop, and the `fleetd` Chrome extension. These tools help you maintain visibility and control over your device fleet.


### Scenario: the surly employee deletion

Imagine a disgruntled employee deleting the `fleetd` files from their device. This disruptive act can hinder your ability to manage the device and potentially compromise security. Fortunately, you can reinstall the `fleetd` agent and restore order with the right MDM commands. It's important to note that ADE (Automated Device Enrollment) enrollment ensures we can maintain control of the laptop and still send MDM commands to the host, such as remote lock or wipe.


### Solutions and commands

There are several approaches to reinstall the `fleetd` agent using MDM commands:


#### 1. Resending the `fleetd` configuration profile

One potential solution is to resend the `fleetd` configuration profile. The new feature for [resending profiles](https://fleetdm.com/docs/rest-api/rest-api#resend-hosts-configuration-profile) makes this easy to accomplish through the MDM interface.


#### 2. Wipe the device

A more extreme method is wiping the device, which performs an Erase All Contents and Settings (EACS). This wipes and resets the laptop by erasing the user-data volume, returning the device to an "out-of-box" experience. This process avoids reinstalling macOS, making it a quick and efficient solution but probably an aggressive action.


#### 3. Sending the install command

By default, the install profile is not sent after the first enrollment. However, you can manually send a command to reinstall `fleetd`. Here is the XML command for macOS:

```xml
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">

<plist version="1.0">
  <dict>
    <key>Command</key>
    <dict>
      <key>ManifestURL</key>
      <string>https://download.fleetdm.com/fleetd-base-manifest.plist</string>
      <key>RequestType</key>
      <string>InstallEnterpriseApplication</string>
    </dict>
    <key>CommandUUID</key>
    <string>adc1bc23-abec-4499-b57f-c8755c7ffe3c</string>
  </dict>
</plist>
```

To run this command, use the following `fleetctl` command:

```sh
fleetctl mdm run-command --hosts=HOST_IDENTIFIER --payload=path/to/file.xml
```

For Windows, the process involves two steps. First, [add the profile](https://fleetdm.com/docs/using-fleet/mdm-custom-os-settings) using gitops or the UI:

```xml
<Add>
	<CmdID>addCommandUUID</CmdID>
	<Item>
		<Target>
		<LocURI>./Device/Vendor/MSFT/EnterpriseDesktopAppManagement/MSI/%7BA427C0AA-E2D5-40DF-ACE8-0D726A6BE096%7D/DownloadInstall</LocURI>
		</Target>
	</Item>
</Add>
```

Then, execute the command using `fleetctl`:

```xml
<Exec>
	<CmdID>execCommandUUID</CmdID>
	<Item>
		<Target>
			<LocURI>./Device/Vendor/MSFT/EnterpriseDesktopAppManagement/MSI/%7BA427C0AA-E2D5-40DF-ACE8-0D726A6BE096%7D/DownloadInstall</LocURI>
		</Target>
		<Data>
			<MsiInstallJob id="{A427C0AA-E2D5-40DF-ACE8-0D726A6BE096}">
			<Product Version="1.0.0.0">
				<Download>
					<ContentURLList>
						<ContentURL>https://download.fleetdm.com/fleetd-base.msi</ContentURL>
					</ContentURLList>
				</Download>
				<Validation>
                	<FileHash>9F89C57D1B34800480B38BD96186106EB6418A82B137A0D56694BF6FFA4DDF1A</FileHash>
				</Validation>
				<Enforcement>
					<CommandLine>/quiet FLEET_URL="REPLACE_WITH_FLEET_URL_HERE" FLEET_SECRET="REPLACE_WITH_FLEET_SECRET_HERE"</CommandLine>
					<TimeOut>10</TimeOut>
					<RetryCount>1</RetryCount>
					<RetryInterval>5</RetryInterval>
				</Enforcement>
			</Product>
			</MsiInstallJob>
		</Data>
		<Meta>
			<Type xmlns="syncml:metinf">text/plain</Type>
			<Format xmlns="syncml:metinf">xml</Format>
		</Meta>
	</Item>
</Exec>

```


### Success story and experiment results

Recently, we conducted an experiment to test these methods. After executing the commands, we observed the device coming back online, confirming the effectiveness of these solutions. This successful experiment highlights the practicality of using MDM commands to restore the `fleetd` agent.


### Conclusion

Dealing with the deletion of `fleetd` files by a surly employee can be a challenge. However, using MDM commands to resend configuration profiles, utilize the EACS, or manually send the install command can efficiently restore functionality and ensure device security. Documenting these processes further strengthens your device management capabilities and prepares you for any future disruptions.





<meta name="articleTitle" value="Sysadmin diaries: restoring fleetd">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-06-14">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
<meta name="description" value="In this sysadmin diary, we explore restoring fleetd deleted by a surly employee.">

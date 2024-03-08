# Enroll hosts

## Introduction

Fleet gathers information from an [osquery](https://github.com/osquery/osquery) agent installed on each of your hosts. The recommended way to install osquery is using fleetd.

You can enroll macOS, Windows or Linux hosts via the [CLI](#cli) or [UI](#ui). To learn how to enroll Chromebooks, see [Enroll Chromebooks](#enroll-chromebooks).

### Supported osquery versions

Fleet supports the [latest version of osquery](https://github.com/osquery/osquery/tags). 

## CLI

> You must have `fleetctl` installed. [Learn how to install `fleetctl`](https://fleetdm.com/fleetctl-preview).

The `fleetctl package` command is used to generate a fleetd installer.

The `--type` flag is used to specify installer type:
- macOS: .pkg
- Windows: .msi
- Linux: .deb or .rpm
  
A `--fleet-url` (Fleet instance URL) and `--enroll-secret` (Fleet enrollment secret) must be specified in order to communicate with Fleet instance.

#### Example

Generate macOS installer (.pkg)

```json
fleetctl package --type pkg --fleet-url=example.fleetinstance.com --enroll-secret=85O6XRG8'!l~P&zWt_'f&$QK(sM8_D4x
```

Tip: To see all options for `fleetctl package` command, run `fleetctl package -h` in your Terminal.

## UI

To generate an installer in Fleet UI:

1. Go to the **Hosts** page, and select **Add hosts**.
2. Select the tab for your desired platform (e.g. macOS).
3. A CLI command with all necessary flags will be generated. Copy and run the command with [fleetctl](https://fleetdm.com/docs/using-fleet/fleetctl-cli) installed.

### Generate installer to enroll host to a specific team

With hosts segmented into teams, you can apply unique queries and give users access to only the hosts in specific teams. [Learn more about teams](https://fleetdm.com/docs/using-fleet/segment-hosts).

To generate an installer that enrolls to a specific team: from the **Hosts** page, select the desired team from the menu at the top of the screen, then follow the instructions above for generating an installer. The team's enroll secret will be included in the generated command.

### Enroll multiple hosts

If you're managing an enterprise environment with multiple hosts, you likely have an enterprise deployment tool like [Munki](https://www.munki.org/munki/), [Jamf Pro](https://www.jamf.com/products/jamf-pro/), [Chef](https://www.chef.io/), [Ansible](https://www.ansible.com/), or [Puppet](https://puppet.com/) to deliver software to your hosts.
You can use your software management tool of choice to distribute a fleetd installer generated via the instructions above.

### Fleet Desktop

[Fleet Desktop](./Fleet-desktop.md) is a menu bar icon available on macOS, Windows, and Linux that gives your end users visibility into the security posture of their machine.

You can include Fleet Desktop in the fleetd installer by including `--fleet-desktop` in the `fleetctl package` command.

## Enroll Chromebooks

> The fleetd Chrome browser extension is supported on ChromeOS operating systems that are managed using [Google Admin](https://admin.google.com). It is not intended for non-ChromeOS hosts with the Chrome browser installed.

### Overview
Google Admin uses organizational units (OUs) to organize devices and users.

One limitation in Google Admin is that extensions can only be configured at the user level, meaning that a user with a MacBook running Chrome, for example, will also get the fleetd Chrome extension.

When deployed on OSs other than ChromeOS, the fleetd Chrome extension will not perform any operation and will not appear in the Chrome toolbar. 
However, it will appear in the "Manage Extensions" page of Chrome.
Fleet admins who are comfortable with this situation can skip step 2 below.

To install the fleetd Chrome extension on Google Admin, there are two steps:

1. Create an OU for all users who have Chromebooks and force-install the fleetd Chrome extension for those users
2. Create an OU for all non-Chromebook devices and block the fleetd Chrome extension on this OU

> More complex setups may be necessary, depending on the organization's needs, but the basic principle remains the same.

### Step 1: OU for Chromebook users
Create an [organizational unit](https://support.google.com/a/answer/182537?hl=en) where the extension should be installed. [Add all the relevant users](https://support.google.com/a/answer/182449?hl=en) to this OU.

In the Google Admin console:

1. In the navigation menu, visit **Devices > Chrome > Apps & Extensions > Users & browsers**.
2. Select the relevant OU where you want the fleetd Chrome extension to be installed.
3. In the bottom right, select the **+** button and select **Add Chrome app or extension by ID**.
4. Go to your Fleet instance and select **Hosts > Add Hosts** and select **ChromeOS** in the popup modal.
5. Enter the **Extension ID**, **Installation URL**, and **Policy for extensions** using the data provided in the modal.
6. Under **Installation Policy**, select **Force install**, and under **Update URL**, select **Installation URL** (see above).

> For the fleetd Chrome extension to have full access to Chrome data, it must be force-installed by enterprise policy as per above

### Step 2: OU to block non-Chromebook devices
Create an [organizational unit](https://support.google.com/a/answer/182537?hl=en) to house devices where the extension should not be installed. [Add all the relevant devices](https://support.google.com/chrome/a/answer/2978876?hl=en) to this OU.

In the Google Admin console:

1. In the navigation menu, select **Devices > Chrome > Managed Browsers**.
2. Select the relevant OU where you want the fleetd Chrome extension to be blocked.
3. In the bottom right, select the **+** button and select **Add Chrome app or extension by ID**.
4. Go to your Fleet instance and select **Hosts > Add Hosts** and select **ChromeOS** in the popup modal.
5. Enter the **Extension ID** and **Installation URL** using the data provided in the modal.
6. Under **Installation Policy**, select **Block**.

### Unenroll

How to unenroll a host from Fleet:

1. Determine if your host has MDM features turned on by looking at the **MDM status** on the host's **Host details** page. 

2. For macOS hosts with MDM turned on, select **Actions > Turn off MDM** to turn MDM off. Instructions for turning off MDM on Windows hosts coming soon.

3. Determine the platform of the host you're trying to unenroll and follow the instructions to uninstall the fleetd agent:

- macOS: Run the [script here](https://github.com/fleetdm/fleet/tree/main/orbit/tools/cleanup/cleanup_macos.sh) 
- Windows: On the Windows device, select **Start > Settings > Apps > Apps & features**. Find "Fleet osquery", select **Uninstall**.
- Linux (Ubuntu): With the APT package manager installed, run `sudo apt remove fleet-osquery -y`.
- Linux (CentOS): Run `sudo rpm -e fleet-osquery-X.Y.Z.x86_64`.

4. Select **Actions > Delete** to delete the host from Fleet.

## Advanced


- [Signing fleetd installer](#signing-fleetd-installer)
- [Grant full disk access to osquery on macOS](#grant-full-disk-access-to-osquery-on-macos) 
- [Using mTLS](#using-mtls)
- [Specifying update channels](#specifying-update-channels)
- [Testing osquery queries locally](#testing-osquery-queries-locally)
- [Finding fleetd logs](#finding-fleetd-logs)
- [Using system keystore for enroll secret](#using-system-keystore-for-enroll-secret)
- [Generating Windows installers using local WiX toolset](#generating-windows-installers-using-local-wix-toolset)
- [Experimental features](#experimental-features)

### Signing fleetd installers

  >**Note:** Currently, the `fleetctl package` command does not support signing Windows fleetd installers. Windows installers can be signed after building.

The `fleetctl package` command supports signing and notarizing macOS osquery installers via the
`--sign-identity` and `--notarize` flags.

Check out the example below:

```sh
  AC_USERNAME=appleid@example.com AC_PASSWORD=app-specific-password fleetctl package --type pkg --sign-identity=[PATH TO SIGN IDENTITY] --notarize --fleet-url=[YOUR FLEET URL] --enroll-secret=[YOUR ENROLLMENT SECRET]
```

The above command must be run on a macOS device, as the notarizing and signing of macOS fleetd installers can only be done on macOS devices.

Also, remember to replace both `AC_USERNAME` and `AC_PASSWORD` environment variables with your Apple ID and a valid [app-specific](https://support.apple.com/en-ca/HT204397) password, respectively. Some organizations (notably those with Apple Enterprise Developer Accounts) may also need to specify `AC_TEAM_ID`. This value can be found on the [Apple Developer "Membership" page](https://developer.apple.com/account/#!/membership) under "Team ID."

### Grant full disk access to osquery on macOS

MacOS does not allow applications to access all system files by default. 

If you are using an MDM solution or Fleet's MDM features, one of which is required to deploy these profiles, you can deploy a "Privacy Preferences Policy Control" policy to grant fleetd or osquery that level of access. 

This is required to query for files located in protected paths as well as to use event
tables that require access to the [EndpointSecurity API](https://developer.apple.com/documentation/endpointsecurity#overview), such as *es_process_events*.

#### Creating the configuration profile

##### Obtaining identifiers

If you use plain osquery, instructions are [available here](https://osquery.readthedocs.io/en/stable/deployment/process-auditing/).

On a system with osquery installed via the Fleet osquery installer (fleetd), obtain the
`CodeRequirement` of fleetd by running:

```sh
codesign -dr - /opt/orbit/bin/orbit/macos/stable/orbit
```

The output should be similar or identical to:

```sh
Executable=/opt/orbit/bin/orbit/macos/edge/orbit
designated => identifier "com.fleetdm.orbit" and anchor apple generic and certificate 1[field.1.2.840.113635.100.6.2.6] /* exists */ and certificate leaf[field.1.2.840.113635.100.6.1.13] /* exists */ and certificate leaf[subject.OU] = "8VBZ3948LU"
```

Note down the **executable path** and the entire **identifier**.

Osqueryd will inherit the privileges from Orbit and does not need explicit permissions.

##### Creating the profile

Depending on your MDM, this might be possible in the UI or require a custom profile. If your MDM has a feature to configure *Policy Preferences*, follow these steps:

1. Configure the identifier type to “path.”
2. Paste the full path to Orbit as the identifier.
3. Paste the full code signing identifier into the Code requirement field. 
4. Allow “Access all files.” Access to Downloads, Documents, etc., is inherited from this.

If your MDM solution does not have built-in support for privacy preferences profiles, you can use
[PPPC-Utility](https://github.com/jamf/PPPC-Utility) to create a profile with those values, then upload it to
your MDM as a custom profile.

##### Test the profile
Link the profile to a test group that contains at least one Mac.
Once the computer has received the profile, which you can verify by looking at *Profiles* in *System
Preferences*, run this query from Fleet:

```sql
SELECT * FROM file WHERE path LIKE '/Users/%/Downloads/%%';
```

If this query returns files, the profile was applied, as **Downloads** is a
protected location. You can now enjoy the benefits of osquery on all system files and start
using the **es_process_events** table!

If this query does not return data, you can look at operating system logs to confirm whether or not full disk
access has been applied.

See the last hour of logs related to TCC permissions with this command:

`log show --predicate 'subsystem == "com.apple.TCC"' --info --last 1h`

You can then look for `orbit` or `osquery` to narrow down results.

### Using mTLS

`Applies only to Fleet Premium`

Fleetd supports using TLS client certificates for authentication to the Fleet server and [TUF](https://theupdateframework.io/) server.

When generating the packages, use the following flags:
```sh
fleetctl package \
  [...]
  --fleet-tls-client-certificate=fleet-client.crt \
  --fleet-tls-client-key=fleet-client.key \
  --update-tls-client-certificate=update-client.crt \
  --update-tls-client-key=update-client.key \
  [...]
```
The certificates must be in PEM format.

The client certificates can also be pushed to existing installations by placing them in the following locations:
- For macOS and Linux:
  - `/opt/orbit/fleet_client.crt`
  - `/opt/orbit/fleet_client.key`
  - `/opt/orbit/update_client.crt`
  - `/opt/orbit/update_client.key`
- For Windows:
  - `C:\Program Files\Orbit\fleet_client.crt`
  - `C:\Program Files\Orbit\fleet_client.key`
  - `C:\Program Files\Orbit\update_client.crt`
  - `C:\Program Files\Orbit\update_client.key`

If using Fleet Desktop, you may need to specify an alternative host for the "My device" URL (in the Fleet tray icon).
Such alternative host should not require client certificates on the TLS connection.
```sh
fleetctl package
  [...]
  --fleet-desktop \
  --fleet-desktop-alternative-browser-host=fleet-desktop.example.com \
  [...]
```
If this setting is not used, you will need to configure client TLS certificates on devices' browsers.

### Specifying update channels

Fleetd uses the concept of "update channels" to determine the version of it's components: Orbit, Fleet Desktop, osquery.

Configure update channels for these components with the `--orbit-channel`, `--desktop-channel` and `--osqueryd-channel` flags when running the `fleetctl package command`.

| Channel | Versions |
| ------- | -------- |
| `4`     | 4.x.x    |
| `4.6`   | 4.6.x    |
| `4.6.0` | 4.6.0    |

Additionally, `stable` and `edge` are special channel names. The `stable` channel will provide the most recent osquery version that Fleet deems to be stable. 

When a new version of osquery is released, it's added to the `edge` channel for beta testing. Fleet then provides input to the osquery TSC based on testing. After the version is declared stable by the osquery TSC, Fleet will promote the version to `stable` ASAP.

### Testing osquery queries locally

Fleet comes packaged with `osqueryi` which is a tool for testing osquery queries locally.

With fleetd installed on your host, run `orbit osqueryi` or `orbit shell` to open the `osqueryi`.

### Finding fleetd logs

Fleetd will send stdout/stderr logs to the following directories:

  - macOS: `/private/var/log/orbit/orbit.std{out|err}.log`.
  - Windows: `C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log` (the log file is rotated).
  - Linux: Orbit and osqueryd stdout/stderr output is sent to syslog (`/var/log/syslog` on Debian systems and `/var/log/messages` on CentOS).

If the `logger_path` agent configuration is set to `filesystem`, fleetd will send osquery's "result" and "status" logs to the following directories:
  - Windows: C:\Program Files\Orbit\osquery_log
  - macOS: /opt/orbit/osquery_log
  - Linux: /opt/orbit/osquery_log

### Using system keystore for enroll secret

On macOS and Windows, fleetd will add the enroll secret to the system keystore (Keychain on macOS, Credential Manager on Windows) on launch. Subsequent launches will retrieve the enroll secret from the keystore.

System keystore access can be disabled via `--disable-keystore` flag for the `fleetctl package` command. On macOS, subsequent installations of fleetd must be signed by the same organization as the original installation to access the enroll secret in the keychain.

>**Note:** The keychain is not used on macOS when the enroll secret is provided via MDM profile. Keychain support when passing the enroll secret via MDM profile is coming soon.

### Generating Windows installers using local WiX toolset

`Applies only to Fleet Premium`

When creating a fleetd installer for Windows hosts (**.msi**) on a Windows or macOS machine, you can tell `fleetctl package` to
use local installations of the 3 WiX v3 binaries used by this command (`heat.exe`, `candle.exe`, and
`light.exe`) instead of those in a pre-configured container, which is the default behavior. To do
so:
  1. Install the WiX v3 binaries. To install, you can download them
     [here](https://github.com/wixtoolset/wix3/releases/download/wix3112rtm/wix311-binaries.zip), then unzip the downloaded file.
  2. Find the absolute filepath of the directory containing your local WiX v3 binaries. This will be wherever you saved the unzipped package contents.
  3. Run `fleetctl package`, and pass the absolute path above as the string argument to the
     `--local-wix-dir` flag. For example:
     ```
      fleetctl package --type msi --fleet-url=[YOUR FLEET URL] --enroll-secret=[YOUR ENROLLMENT SECRET] --local-wix-dir "\Users\me\AppData\Local\Temp\wix311-binaries"
     ```
     If the provided path doesn't contain all 3 binaries, the command will fail.

>**Note:** Creating a fleetd agent for Windows (.msi) on macOS also requires Wine. To install Wine see the script [here](https://github.com/fleetdm/fleet/blob/fleet-v4.44.0/scripts/macos-install-wine.sh).

### Experimental features

> Any features listed here are not recommended for use in production environments

**Using `fleetd` without enrolling Orbit**

*Only available in fleetd v1.15.1 on Linux and macOS*

It is possible to generate a fleetd package that does not connect to Fleet by omitting the `--fleet-url` and `--enroll-secret` flags when building a package.

This can be useful in situations where you would like to test using `fleetd` to manage osquery updates while still managing osquery command-line flags and extensions locally 
but can result in a large volume of error logs. In fleetd v1.15.1, we added an experimental feature to reduce log chatter in this scenario.
 
Applying the environmental variable `"FLEETD_SILENCE_ENROLL_ERROR"=1` on a host will silence fleetd enrollment errors if a `--fleet-url` is not present.
This variable is read at launch and will require a restart of the Orbit service if it is not set before installing `fleetd` v1.15.1.

<meta name="pageOrderInSection" value="500">
<meta name="description" value="Learn how to generate installers and enroll hosts to Fleet using fleetd.">
<meta name="navSection" value="The basics">

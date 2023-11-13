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

### Including Fleet Desktop

> Fleet Desktop requires a Fleet version of 4.12.0 and above. To check your Fleet version, select the avatar on the right side of the top bar and select **My account**. Your Fleet version is displayed below the **Get API token** button.

Hosts without Fleet Desktop currently installed require a new installer to be generated and run on the target host. To include Fleet desktop in your installer, select the **Include Fleet Desktop** checkbox when generating an installer via the instructions above.

Alternatively, you can generate an installer that includes Fleet Desktop in `fleetctl package` by appending the `--fleet-desktop` flag.

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

> Currently, the Chrome extension can only be installed across the entire organization. The work to enable installation for sub-groups is tracked in https://github.com/fleetdm/fleet/issues/13353. 

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

## Grant full disk access to osquery on macOS

macOS does not allow applications to access all system files by default. If you are using MDM, which
is required to deploy these profiles, you
can deploy a "Privacy Preferences Policy Control" policy to grant fleetd or osquery that level of
access. This is necessary to query for files located in protected paths as well as to use event
tables that require access to the [EndpointSecurity
API](https://developer.apple.com/documentation/endpointsecurity#overview), such as *es_process_events*.

### Creating the configuration profile
#### Obtaining identifiers
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

> **NOTE:** Depending on the version of `fleetctl` used to package and install Orbit, as well as the update channel you've specified, the executable path may differ.
> Fleetctl versions <= 4.13.2 would install Orbit to `/var/lib/orbit` instead of `/opt/orbit`.

Note down the **executable path** and the entire **identifier**.

Osqueryd will inherit the privileges from Orbit and does not need explicit permissions.

#### Creating the profile
Depending on your MDM, this might be possible in the UI or require a custom profile. If your MDM has a feature to configure *Policy Preferences*, follow these steps:

1. Configure the identifier type to “path.”
2. Paste the full path to Orbit as the identifier.
3. Paste the full code signing identifier into the Code requirement field. 
4. Allow “Access all files.” Access to Downloads, Documents, etc., is inherited from this.

If your MDM does not have built-in support for privacy preferences profiles, you can use
[PPPC-Utility](https://github.com/jamf/PPPC-Utility) to create a profile with those values, then upload it to
your MDM as a custom profile.

#### Test the profile
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

## Advanced

- [Signing fleetd installer](#signing-fleetd-installer)
- [Generating Windows installers using local WiX toolset](#generating-windows-installers-using-local-wix-toolset)
- [fleetd configuration options](#fleetd-configuration-options)
- [Enroll hosts with plain osquery](#enroll-hosts-with-plain-osquery)

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

### Generating Windows installers using local WiX toolset

`Applies only to Fleet Premium`

When creating a fleetd installer for Windows hosts (**.msi**) on a Windows machine, you can tell `fleetctl package` to
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

### Fleetd configuration options

The following command-line flags to `fleetctl package` allow you to configure an osquery installer further to communicate with a specific Fleet instance.

| Flag                       | Options                                                                                                                                 |
| -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| --type                     | **Required** - Type of package to build.<br> Options: `pkg`(macOS),`msi`(Windows), `deb`(Debian based Linux), `rpm`(RHEL, CentOS, etc.) |
| --fleet-desktop            | Include Fleet Desktop.                                                                                                                  |
| --enroll-secret            | Enroll secret for authenticating to Fleet server                                                                                        |
| --fleet-url                | URL (`host:port`) of Fleet server                                                                                                       |
| --fleet-certificate        | Path to server certificate bundle                                                                                                       |
| --identifier               | Identifier for package product (default: `com.fleetdm.orbit`)                                                                           |
| --version                  | Version for package product (default: `0.0.3`)                                                                                          |
| --insecure                 | Disable TLS certificate verification (default: `false`)                                                                                 |
| --service                  | Install osquery with a persistence service (launchd, systemd, etc.) (default: `true`)                                                   |
| --sign-identity            | Identity to use for macOS codesigning                                                                                                   |
| --notarize                 | Whether to notarize macOS packages (default: `false`)                                                                                   |
| --disable-updates          | Disable auto updates on the generated package (default: false)                                                                          |
| --osqueryd-channel         | Update channel of osqueryd to use (default: `stable`)                                                                                   |
| --orbit-channel            | Update channel of Orbit to use (default: `stable`)                                                                                      |
| --desktop-channel          | Update channel of desktop to use (default: `stable`)                                                                                    |
| --update-url               | URL for update server (default: `https://tuf.fleetctl.com`)                                                                             |
| --update-roots             | Root key JSON metadata for update server (from fleetctl updates roots)                                                                  |
| --use-system-configuration | Try to read --fleet-url and --enroll-secret using configuration in the host (currently only macOS profiles are supported)               |
| --enable-scripts           | Enable script execution (default: `false`)                                                                                              |
| --debug                    | Enable debug logging (default: `false`)                                                                                                 |
| --verbose                  | Log detailed information when building the package (default: false)                                                                     |
| --local-wix-dir            | Use local installations of the 3 WiX v3 binaries this command uses (`heat.exe`, `candle.exe`, and `light.exe`) instead of installations in a pre-configered Docker Hub (only available on Windows w/ WiX v3)                                                    |
| --help, -h                 | show help (default: `false`)                                                                                                            |

Fleet supports other methods for adding your hosts to Fleet, such as the plain osquery
binaries or [Kolide Osquery
Launcher](https://github.com/kolide/launcher/blob/master/docs/launcher.md#connecting-to-fleet).

### Enroll hosts with plain osquery

Osquery's [TLS API](http://osquery.readthedocs.io/en/stable/deployment/remote/) plugin lets you use
the native osqueryd binaries to connect to Fleet. Learn more [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Enroll-hosts-with-plain-osquery.md).


<meta name="pageOrderInSection" value="500">
<meta name="description" value="Learn how to generate installers and enroll hosts in your Fleet instance using fleetd or osquery.">
<meta name="navSection" value="The basics">

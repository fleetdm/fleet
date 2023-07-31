# fleetdm

## Table of Contents

1. [Description](#description)
1. [Setup - The basics of getting started with fleetdm](#setup)
    * [Setup requirements](#setup-requirements)
    * [Beginning with fleetdm](#beginning-with-fleetdm)
1. [Usage - Configuration options and additional functionality](#usage)
    * [Defining profiles for a device](#defining-profiles-for-a-device)
    * [Releasing a device from await configuration](#releasing-a-device-from-await-configuration)
3. [Limitations - OS compatibility, etc.](#limitations)
4. [Development - Guide for contributing to the module](#development)

## Description

Manage MDM settings for macOS devices using [Fleet](https://fleetdm.com)

## Setup

### Setup Requirements

This module requires to add `fleetdm` as a reporter in your `report` settings,
this helps Fleet understand when your Puppet run is finished and assign the
device to a team with the necessary profiles.

For example, in your server configuration:

```
reports = http,fleetdm
```

To communicate with the Fleet server, you also need to provide your server URL
and a token as Hiera values:

```yaml
---
fleetdm::host: https://example.com
fleetdm::token: my_token 
```

Note: for the token, we recommend using an [API-only user][1], with a GitOps role.

### Beginning with fleetdm

## Usage

### Defining profiles for a device

The `examples/` folder in this repo contain some examples. Generally, you can
define profiles using the custom resource type `fleetdm::profile`:


```pp
node default {
  fleetdm::profile { 'com.apple.universalaccess':
    template => template('fleetdm/profile-template.mobileconfig.erb'),
    group    => 'workstations',
  }
}
```

The `group` parameter is used to create/match profiles with teams in
Fleet. In the example above, all devices will be assigned to a team named
`workstations`.

You can use this feature along with the `ensure` param to create teams that
**don't** contain specific profiles, for example given the following manifest:

```pp
node default {
  fleetdm::profile { 'com.apple.universalaccess':
    template => template('fleetdm/profile-template.mobileconfig.erb'),
    group    => 'workstations',
  }

  if $facts['architecture'] == 'x86_64' {
      fleetdm::profile { 'my.arm.only.profile':
        ensure => absent,
        template => template('fleetdm/my-arm-only-profile.mobileconfig.erb'),
        group    => 'amd64',
      }
  } else {
      fleetdm::profile { 'my.arm.only.profile':
        template => template('fleetdm/my-arm-only-profile.mobileconfig.erb'),
        group    => 'workstations',
      }
  }
}
```

Assuming you have devices with both architectures checking in, you'll end up
with the following two teams in Fleet:

- `workstations`: with two profiles, `com.apple.universalaccess` and `my.arm.only.profile`
- `workstations - amd64`: with only one profile, `com.apple.universalaccess`

### Sending a custom MDM Command

You can use the `fleetdm::command_xml` function to send any custom MDM command to the device:

```pp
$host_uuid = $facts['system_profiler']['hardware_uuid']
$command_uuid = generate('/usr/bin/uuidgen').strip

$xml_data = "<?xml version='1.0' encoding='UTF-8'?>
<!DOCTYPE plist PUBLIC '-//Apple//DTD PLIST 1.0//EN' 'http://www.apple.com/DTDs/PropertyList-1.0.dtd'>
<plist version='1.0'>
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>EnableRemoteDesktop</string>
    </dict>
    <key>CommandUUID</key>
    <string>${command_uuid}</string>
</dict>
</plist>"

$response = fleetdm::command_xml($host_uuid, $xml_data)
$err = $response['error']

if $err != '' {
  notify { "Error sending MDM command: ${err}": }
}
```

### Releasing a device from await configuration

If your DEP profile had `await_device_configured` set to `true`, you can use the `fleetdm::release_device` function to release the device:

```
$host_uuid = $facts['system_profiler']['hardware_uuid']
$response = fleetdm::release_device($host_uuid)
$err = $response['error']

if $err != '' {
  notify { "error releasing device: ${err}": }
}
```

## Limitations

At the moment, this module only works for macOS devices.

## Development

Information about how to contribute can be found in the [`CONTRIBUTING.md` file](https://github.com/fleetdm/fleet/blob/main/ee/tools/puppet/fleetdm/CONTRIBUTING.md).

[1]: https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user

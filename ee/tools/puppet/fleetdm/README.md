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

### Releasing a device from await configuration

If your DEP profile had `await_device_configured` set to `true`, you can use the `fleetdm::release_device` function to release the device:

```
$host_uuid = $facts['system_profiler']['hardware_uuid']
$response = fleetdm::release_device($host_uuid)
$err = $response['error']

if $err {
  notify { "error releasing device: ${err}": }
}
```

## Limitations

At the moment, this module only works for macOS devices.

## Development

To trigger a puppet run locally:

```
puppet apply --debug --test --modulepath="$(pwd)/.." --reports=fleetdm  --hiera_config hiera.yaml examples/multiple-teams.pp
```

To lint/fix Puppet (`.pp`) files, use:

```
pdk bundle exec puppet-lint --fix .
```

To lint/fix Ruby (`.rb`) files, use:

```
pdk bundle exec rubocop -A
```

[1]: https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user

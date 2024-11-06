# Puppet module

_Available in Fleet Premium_

Use [Fleet's Puppet module](https://forge.puppet.com/modules/fleetdm/fleetdm/readme) to automatically install custom configuration profiles on your macOS hosts based on host attributes you define in Puppet.

The module also includes functions for releasing a macOS host from [Await Configuration](https://developer.apple.com/documentation/devicemanagement/release_device_from_await_configuration) and sending any custom MDM commands.

## Setup

To set up the Puppet module, we will do the following steps:

1. Install the Puppet module
2. Configure Puppet to talk to Fleet using Hiera
3. Set Fleet as a reporter

### Step 1: install the Puppet module

Install [Fleet's Puppet module](https://forge.puppet.com/modules/fleetdm/fleetdm/readme). For more instructions on how to install Puppet modules, check out the Puppet docs [here](https://www.puppet.com/docs/puppet/8/modules_installing.html).

### Step 2: configure Puppet to talk to Fleet using Heira

1. In Fleet, create an API-only user with the GitOps role. Instructions for creating an API-only user are
   [here](https://fleetdm.com/guides/fleetctl#create-api-only-user). 

2. Set the `fleetdm::token` and `fleetdm::host` values to the API token of your API-only user and
   your Fleet server's URL, respectively. Here's an example of the Hiera YAML:

```yaml
fleetdm::host: https://fleet.example.com
fleetdm::token: your-api-token 
```

Puppet docs on configuring Hiera are [here](https://www.puppet.com/docs/puppet/6/hiera_config_yaml_5.html).

If you have staging and production Puppet environments, you can optionally set different values for each environment. This allows you to have your staging and production environments that talk to separate staging and production Fleet servers.

### Step 3: set Fleet as a reporter

In your Puppet configuration, set `http:fleetdm` as the value for `reports`. Here's an example of the Puppet configuration:

```
reports = http,fleetdm
```

Puppet configuration reference docs are [here](https://www.puppet.com/docs/puppet/7/configuration#reports).

## Install configuration profiles

Using the Puppet module you can define the set of configuration profiles for each host (Puppet node) and Fleet will create a team with these profiles and assign the host to that team.

When a host is assigned to a team in Fleet, all configuration profiles for that team are installed on the host.

As an example, let's install one configuration profile on all hosts. Here's what your Puppet code will look like:

```pp
node default {
  fleetdm::profile { 'com.apple.payload.identifier':
    template => template('example-profile.mobileconfig'),
    group    => 'MacOS workstations',
  }
}
```

This will create a team called "MacOS workstations" with the `example-profile.mobileconfig` configuration profile and assign all hosts to this team.

Use the `group` parameter to define the team name in Fleet.

As another example, let's assign one configuration profile to all hosts and another configuration profile to only my M1 hosts. Here's what your Puppet code will look like:

```pp
node default {
  fleetdm::profile { 'com.apple.payload.identifier-1':
    template => template('example-profile.mobileconfig'),
    group    => 'MacOS workstations',
  }

  if $facts['architecture'] == 'intel' {
      fleetdm::profile { 'com.apple.payload.identifier-2':
        ensure => absent,
        template => template('m1-only.mobileconfig'),
        group    => 'Intel',
      }
  } else {
      fleetdm::profile { 'com.apple.example-2':
        template => template('com.apple.payload.identifier-2'),
        group    => 'MacOS workstations',
      }
  }
}
```

This will create two teams in Fleet: 

1. "MacOS workstations" with two configuration profiles: `example-profile.mobileconfig` and `m1-only.mobileconfig`.
2. "MacOS workstations - Intel" with one configuration profile: `example-profile.mobileconfig`.

Set the `ensure` parameter to `absent` to create teams that exclude specific profiles.

For more examples check out the `examples/` folder in Fleet's GitHub repository [here](https://github.com/fleetdm/fleet/tree/main/ee/tools/puppet/fleetdm/examples).

> Note that all teams created by Puppet inherit the bootstrap package, macOS Setup Assistant settings, and end user authentication settings from "No team." Learn more about these [here](https://fleetdm.com/guides/macos-setup-experience). In addition all teams automatically enable disk encryption. Learn more about disk encryption [here](https://fleetdm.com/guides/enforce-disk-encryption).

## Release host

If you set `enable_release_device_manually` to `true` in your [macOS setup experience](https://fleetdm.com/docs/rest-api/rest-api#configure-setup-experience), you can use the `fleetdm::release_device` function to release the host from the Setup Assistant. 

Here's what your Puppet code, with error handling, will look like:

```pp
$host_uuid = $facts['system_profiler']['hardware_uuid']
$response = fleetdm::release_device($host_uuid)
$err = $response['error']

if $err != '' {
  notify { "error releasing device: ${err}": }
}
```

## Custom commands

You can use the `fleetdm::command_xml` function to send any custom MDM command to a host.

Here's what your Puppet code, with error handling, will look like:

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

The above example includes the XML payload for the `EnableRemoteDesktop` MDM command. Learn more about creating the payload for other custom commands [here](https://fleetdm.com/guides/mdm-commands).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-05-24">
<meta name="articleTitle" value="Puppet module">
<meta name="description" value="Learn how to use Fleet's Puppet module to automatically assign custom configuration profiles on your macOS hosts.">

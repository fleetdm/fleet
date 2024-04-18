# GitOps

Fleet can be managed using configuration files (YAML) with GitOps workflow. To learn how to setup GitOps workflow see [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

> Old workflow with YAML configuration files is documented [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Configuration-files.md).  `fleetctl apply` can be still used for imports and backwards compatibility.

On this page, you can learn how to craft configuration files.

## Default configuration

The `default.yml` file defines queries, policies, controls, and agent options for all hosts. If you're using Fleet Premium, this file will define queries and policies that run for "All teams". Controls and agent options will be defined for hosts assigned to "No team." 

Queries, policies, configuration profiles, scripts, and agent options can be referenced from `lib/` folder. Learn more about it in the [Library section](#library-lib).


### Agent options

The `agent_options` key controls the settings applied to the agent on all your hosts. These settings are applied when each host checks in.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to find all possible options.

### Features

...

----

## Team configuration

The `team/{team_name}.yml` file updates controls, queries, policies, and agent options for hosts assigned to the specified team. Below the example file, you can find each option explained.

Queries, policies, configuration profiles, scripts and agent options can be referenced from `lib/` folder. Learn more about it in the [Library section](https://#library-lib).

### Agent options

...

### Controls

...

----

## Library

Folder for policies, queries, configuration profiles, scripts, and agent options. Configuration files from library can be referenced in top level keys in the `default.yml` file and the files in the `teams/` folder.

### Policies

The `lib/{name}.policies.yml` files define set of policies that can be referenced in a default and team configurations.

```yaml
- name: Windows - Enable BitLocker
  platform: windows
  description: "This policy checks if BitLocker (disk encryption) is enabled on the C: volume."
  resolution: As an IT admin, turn on disk encryption in Fleet.
  query: SELECT * FROM bitlocker_info WHERE drive_letter='C:' AND protection_status = 1;
- name: Windows - Disable guest account
  platform: windows
  description: This policy checks if the guest account is disabled. The Guest account allows unauthenticated network users to gain access to the system.
  resolution: "As an IT admin, deploy a Windows profile with the Accounts_EnableGuestAccountStatus option documented here: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-localpoliciessecurityoptions#accounts_enableguestaccountstatus"
  query: SELECT 1 FROM mdm_bridge where mdm_command_input = "<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/LocalPoliciesSecurityOptions/Accounts_EnableGuestAccountStatus</LocURI></Target></Item></Get></SyncBody>" and CAST(mdm_command_output AS INT) = 0;
- name: Windows - Require 10 character password
  platform: windows
  description: This policy checks if the end user is required to enter a password, with at least 10 characters, to unlock the host.
  resolution: "As an IT admin, deploy a Windows profile with the DevicePasswordEnabled and MinDevicePasswordLength option documented here: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-devicelock"
  query: SELECT 1 FROM mdm_bridge where mdm_command_input = "<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/DeviceLock/DevicePasswordEnabled</LocURI></Target></Item></Get></SyncBody>" and CAST(mdm_command_output AS INT) = 0;
- name: Windows - Enable screen saver after 20 minutes
  platform: windows
  description: This policy checks if maximum amount of time (in minutes) the device is allowed to sit idle before the screen is locked. End users can select any value less than the specified maximum.
  resolution: "As an IT admin, to deploy a Windows profile with the MaxInactivityTimeDeviceLock option documented here: https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-devicelock#maxinactivitytimedevicelock"
  query: SELECT 1 FROM mdm_bridge where mdm_command_input = "<SyncBody><Get><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Result/DeviceLock/MaxInactivityTimeDeviceLock</LocURI></Target></Item></Get></SyncBody>" and CAST(mdm_command_output AS INT) <= 20;
```

### Queries

The `lib/{name}.queries.yml` files define a set of policies that can be referenced in a default and team configurations.

```yaml
name: Collect USB devices
  description: Collects the USB devices that are currently connected to macOS and Linux hosts.
  query: SELECT model, vendor FROM usb_devices;
  interval: 300 # 5 minutes
  observer_can_run: true
  automations_enabled: false
```

### Agent options

The `lib/agent-options.yml` defines agent options. See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to find all possible options.

```yaml
command_line_flags:
config:
  decorators:
    load:
      - SELECT uuid AS host_uuid FROM system_info;
      - SELECT hostname AS hostname FROM system_info;
  options:
    disable_distributed: false
    distributed_interval: 10
    distributed_plugin: tls
    distributed_tls_max_attempts: 3
    logger_tls_endpoint: /api/v1/osquery/log
    pack_delimiter: /
```

### Configuration profiles

The `lib/`folder can be used to add configuration profiles that can be referenced in a default and team configurations. You can add macOS profiles (.json), declaration (DDM) profiles (.json) and Windows profiles (.xml)

### Scripts

The `lib/`folder can be used to add scripts that can be referenced in a default and team configurations. You can add shell scripts (.sh) for macOS and Linux and PowerShell scripts (.ps1) for Windows hosts.

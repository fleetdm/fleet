# GitOps

Fleet can be managed using configuration files (YAML) with GitOps workflow. To learn how to set up GitOps workflow see [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

> The workflow with YAML configuration files is documented [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Configuration-files.md).  `fleetctl apply` can be still used for imports and backwards compatibility.

This page lists the options available in configuration files.

## Agent options

The `agent_options` key controls the settings applied to the agent on all your hosts. These settings are applied when each host checks in.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to find all possible options.

## Queries

The `lib/{name}.queries.yml` files control saved queries in Fleet.

- Optional
- Array of dictionaries
- Config format:  
  ```yaml
  - name: Collect failed login attempts
    description: Lists the users at least one failed login attempt and timestamp of failed login. Number of failed login attempts reset to zero after a user successfully logs in.
    query: SELECT users.username, account_policy_data.failed_login_count, account_policy_data.failed_login_timestamp FROM users INNER JOIN account_policy_data using (uid) WHERE account_policy_data.failed_login_count > 0;
    interval: 300 # 5 minutes
    observer_can_run: false
    automations_enabled: false
    platform: darwin,linux,windows
  - name: Collect USB devices
    description: Collects the USB devices that are currently connected to macOS and Linux hosts.
    query: SELECT model, vendor FROM usb_devices;
    interval: 300
    observer_can_run: true
    automations_enabled: false
  ``` 

## Policies

The `lib/{name}.poliicies.yml` files control policies in Fleet.

- Optional
- Array of dictionaries
- Config format:
  ```yaml
  - name: macOS - Enable FileVault
    platform: darwin
    description: This policy checks if FileVault (disk encryption) is enabled.
    resolution: As an IT admin, turn on disk encryption in Fleet.
    query: SELECT 1 FROM filevault_status WHERE status = 'FileVault is On.';
  - name: macOS - Disable guest account
    platform: darwin
    description: This policy checks if the guest account is disabled.
    resolution: An an IT admin, deploy a macOS, login window profile with the DisableGuestAccount option set to true.
    query: SELECT 1 FROM managed_policies WHERE domain='com.apple.loginwindow' AND username = '' AND name='DisableGuestAccount' AND CAST(value AS INT) = 1;
  ```

## Controls

The `controls` section defines device management settings (OS updates, configuration profiles, disk encryption, OS updates, and scripts). This is the top-level key, and all options referenced below are under this one.


### Mobile device management (MDM) options

#### mdm.apple​_bm​_default​_team

_Available in Fleet Premium_

Set the name of the default team to use with Apple Business Manager. macOS hosts will be added to this team when they’re first unboxed.

- Optional
- Default value: `""`
- Config format:
  ```yaml
  mdm:
    apple_bm_default_team: "Workstations"
  ```

#### mdm.windows​_enabled​_and​_configured

Turns on or off Windows MDM.

- Optional
- Default value: `false`
- Config format:
  ```yaml
  mdm:
    windows_enabled_and_configured: true
  ```

#### mdm.macos​_updates

_Available in Fleet Premium_

Configures OS update enforcement for macOS hosts.

##### mdm.macos​_updates.minimum_version

macOS version that the end-user must update to.

- Requires `mdm.macos_updates.deadline` to be set  
- Default value: `""`
- Config format:
  ```yaml
  mdm:
    macos_updates:
      minimum_version: "14.3.0"
  ```

##### mdm.macos​_updates.deadline

A deadline in the form of YYYY-MM-DD. The exact deadline time is 04:00:00 (UTC-8).

- Requires `mdm.macos_updates.minimum_version` to be set  
- Default value: `""`
- Config format:
  ```yaml
  mdm:
    macos_updates:
      deadline: "2022-01-01"
  ```

#### mdm.windows​_updates

_Available in Fleet Premium_

Configures OS update enforcement for Windows hosts.

##### mdm.windows​_updates.deadline

A deadline in days.

- Optional
- Default value: `""`
- Config format:
  ```yaml
  mdm:
    windows_updates:
      deadline_days: "5"
  ```

##### mdm.windows​_updates.grace_period

A grace period in days.

- Optional
- Default value: `""`
- Config format:
  ```yaml
  mdm:
    windows_updates:
      deadline_days: "2"
  ```

#### mdm.macos_settings

Configures OS settings for macOS hosts.

##### mdm.macos​_settings.custom​_settings

Defines configuration profile files to apply to macOS hosts.

- Optional
- Default value: none
- Config format:
  ```yaml
  mdm:
    macos_settings:
      custom_settings:
        - path: '/path/to/profile1.mobileconfig'
          labels:
            - Label name 1
          - path: '/path/to/profile2.mobileconfig'
  ```
  The `labels` key is optional.

##### mdm.macos​_settings.enable​_disk​_encryption

_Available in Fleet Premium_

Enables disk encryption on macOS and Windows hosts.

- Optional
- Default value: `false`
- Config format:
  ```yaml
  mdm:
    macos_settings:
      enable_disk_encryption: true
  ```
  
## Organization settings

The `org_settings` section defines organization-wide settings (Fleet features, organization
information, server settings, SSO settings, webhook settings, and integrations). This is the top-level key, and all options referenced below are under this one.


See "[Organization settings](https://fleetdm.com/docs/configuration/configuration-files#organization-settings)" to find all possible options.


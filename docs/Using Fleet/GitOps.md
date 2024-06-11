# GitOps

Use Fleet's best practice GitOps workflow to manage your computers as code.

This page lists the available in configuration options.

To learn how to set up GitOps workflow see [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

The [`fleetctl apply`]((https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Configuration-files.md)) format is maintained for imports and backwards compatibility GitOps.

## Policies

The `lib/{name}.policies.yml` files control policies in Fleet.

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

## Agent options

The `agent_options` key controls the settings applied to the agent on all your hosts. These settings are applied when each host checks in.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to find all possible options.

## Default

TODO

## Teams

### Controls

The `controls` section defines device management settings (OS updates, configuration profiles, disk encryption, OS updates, and scripts). This is the top-level key, and all options referenced below are under this one.

#### macos_settings.custom_settings

Configures OS settings for macOS hosts.

Defines configuration profile files to apply to macOS hosts.

- Optional
- Default value: none
- Config format:
  ```yaml
  macos_settings:
    custom_settings:
      - path: '/path/to/profile1.mobileconfig'
        labels:
          - Label name 1
        - path: '/path/to/profile2.mobileconfig'
  ```
  The `labels` key is optional.

#### windows_settings.custom_settings

Configures OS settings for Windows hosts.

Defines configuration profile files to apply to Window hosts.

- Optional
- Default value: none
- Config format:
  ```yaml
  windows_settings:
    custom_settings:
      - path: '/path/to/profile1.xml'
        labels:
          - Label name 1
        - path: '/path/to/profile2.xml'
  ```
  The `labels` key is optional.

##### enable​_disk​_encryption

_Available in Fleet Premium_

Enables disk encryption on macOS and Windows hosts.

- Optional
- Default value: `false`
- Config format:
  ```yaml
  macos_settings:
    enable_disk_encryption: true
  ```

#### macos​_updates

_Available in Fleet Premium_

Configures OS update enforcement for macOS hosts.

- Requires `mdm.macos_updates.deadline` to be set  
- Default value: `""`
- Config format:
  ```yaml
  macos_updates:
    minimum_version: "14.3.0"
    deadline: "2022-01-01"
  ```

#### windows​_updates

_Available in Fleet Premium_

Configures OS update enforcement for Windows hosts.

A deadline in days.

- Optional
- Default value: `""`
- Config format:
  ```yaml
  mdm:
    windows_updates:
      deadline_days: "5"
      grace_period_days: "2"
  ```


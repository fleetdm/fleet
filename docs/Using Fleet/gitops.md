# GitOps

Use Fleet's best practice GitOps workflow to manage your computers as code.

This page lists the available in configuration options.

To learn how to set up GitOps workflow see [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

The [`fleetctl apply`]((https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Configuration-files.md)) format is maintained for imports and backwards compatibility GitOps.

## Policies

Polcies can be specified inline in your `default.yml` file or `teams/team-name.yml` files. They can also be specified in separate files in your `lib/` folder.

### Options

For possible options, see the parameters for the [Add policy API endpoint](../REST%20API/rest-api.md#add-policices).

### Example

#### Inline
  
`default.yml` or `teams/team-name.yml`

```yaml
polcies:
  - name: macOS - Enable FileVault
    description: This policy checks if FileVault (disk encryption) is enabled.
    resolution: As an IT admin, turn on disk encryption in Fleet.
    query: SELECT 1 FROM filevault_status WHERE status = 'FileVault is On.';
    platform: darwin
    critical: false
    calendar_event_enabled: false
```

#### Separate file
 
`lib/policies-name.policies.yml`

```yaml
- name: macOS - Enable FileVault
  description: This policy checks if FileVault (disk encryption) is enabled.
  resolution: As an IT admin, turn on disk encryption in Fleet.
  query: SELECT 1 FROM filevault_status WHERE status = 'FileVault is On.';
  platform: darwin
  critical: false
  calendar_event_enabled: false
- name: macOS - Disable guest account
  description: This policy checks if the guest account is disabled.
  resolution: An an IT admin, deploy a macOS, login window profile with the DisableGuestAccount option set to true.
  query: SELECT 1 FROM managed_policies WHERE domain='com.apple.loginwindow' AND username = '' AND name='DisableGuestAccount' AND CAST(value AS INT) = 1;
  platform: darwin
  critical: false
  calendar_event_enabled: false
```

`default.yml` or `teams/team-name.yml`

```yaml
policies:
  - path: `path-to/lib/policies-name.policies.yml`
```

## Queries

Queries can be specified inline in your `default.yml` file or `teams/team-name.yml` files. They can also be specified in separate files in your `lib/` folder.

Note that the `team_id` option isn't supported in GitOps.

### Options

For possible options, see the parameter for the parameters of the [Create query API endpoint](../REST%20API/rest-api.md#create-query).

### Example

#### Inline
  
`default.yml` or `teams/team-name.yml`

```yaml
queries:
  - name: Collect failed login attempts
    description: Lists the users at least one failed login attempt and timestamp of failed login. Number of failed login attempts reset to zero after a user successfully logs in.
    query: SELECT users.username, account_policy_data.failed_login_count, account_policy_data.failed_login_timestamp FROM users INNER JOIN account_policy_data using (uid) WHERE account_policy_data.failed_login_count > 0;
    platform: darwin,linux,windows
    interval: 300
    observer_can_run: false
    automations_enabled: false
```

#### Separate file
 
`lib/queries-name.queries.yml`

```yaml
- name: Collect failed login attempts
  description: Lists the users at least one failed login attempt and timestamp of failed login. Number of failed login attempts reset to zero after a user successfully logs in.
  query: SELECT users.username, account_policy_data.failed_login_count, account_policy_data.failed_login_timestamp FROM users INNER JOIN account_policy_data using (uid) WHERE account_policy_data.failed_login_count > 0;
  platform: darwin,linux,windows
  interval: 300
  observer_can_run: false
  automations_enabled: false
- name: Collect USB devices
  description: Collects the USB devices that are currently connected to macOS and Linux hosts.
  query: SELECT model, vendor FROM usb_devices;
  platform: darwin,linux
  interval: 300
  observer_can_run: true
  automations_enabled: false
```

`default.yml` or `teams/team-name.yml`

```yaml
queries:
  - path: `path-to/lib/queries-name.queries.yml`
```

## Agent options

Agent options can be specified inline in your `default.yml` file or `teams/team-name.yml` files. They can also be specified in separate files in your `lib/` folder.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to find all possible options.

### Example

#### Inline
  
`default.yml` or `teams/team-name.yml`

```yaml
agent_options:
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
      logger_tls_endpoint: /api/osquery/log
      logger_tls_period: 10
      pack_delimiter: /
```

#### Separate file
 
`lib/agent-options.yml`

```yaml
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
    logger_tls_endpoint: /api/osquery/log
    logger_tls_period: 10
    pack_delimiter: /
```

`default.yml` or `teams/team-name.yml`

```yaml
queries:
  - path: `path-to/lib/agent-options.yml`
```

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


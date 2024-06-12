# GitOps

Use Fleet's best practice GitOps workflow to manage your computers as code.

This page lists the available in configuration options.

To learn how to set up GitOps workflow see [Fleet GitOps repo](https://github.com/fleetdm/fleet-gitops).

The [`fleetctl apply`]((https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Configuration-files.md)) format is maintained for imports and backwards compatibility GitOps.

## File structure

- `default.yml`- file where you define the queries, policies, controls, and agent options for all hosts. If you're using Fleet Premium, this file updates queries and policies that run on all hosts ("All teams"). Controls and agent options are defined for hosts on "No team."
- `teams/` - folder where you define your teams in Fleet. These `teams/team-name.yml` files define the controls, queries, policies, and agent options for hosts assigned to the specified team. Teams are available in Fleet Premium.
- `lib/` - folder where you define policies, queries, configuration profiles, scripts, and agent options. These files can be referenced in top level keys in the `default.yml` file and the files in the `teams/` folder.

The following files are responsible for running the GitHub action. Most users don't need to edit these file.
- `.github/workflows/workflow.yml` - the GitHub workflow file that applies the latest configuration to Fleet.
- `gitops.sh` - the bash script that applies the latest configuration to Fleet. This script is used in the GitHub action file.
- `.github/gitops-action/action.yml` - the GitHub action that runs `gitops.sh`. This action is used in the GitHub workflow file. It can also be used in other workflows.

## Configuration options

- [`policies`](#policies)
- [`queries`](#queries)
- [`agent_options`](#agent_options)
- [`controls`](#controls)
- [`org_settings` and `team_settings`](#org_settings-and-team_settings)

The following are the required keys in the `default.yml` file:

```yaml
policies:
queries:
agent_options:
controls:
org_settings:
```

The follow are the required keys in any `teams/team-name.yml` file:

```yaml
policies:
queries:
agent_options:
controls:
team_settings:
```

### `policies`

Polcies can be specified inline in your `default.yml` file or `teams/team-name.yml` files. They can also be specified in separate files in your `lib/` folder.

#### Options

For possible options, see the parameters for the [Add policy API endpoint](../REST%20API/rest-api.md#add-policices).

#### Example

##### Inline
  
`default.yml` or `teams/team-name.yml`

```yaml
policies:
  - name: macOS - Enable FileVault
    description: This policy checks if FileVault (disk encryption) is enabled.
    resolution: As an IT admin, turn on disk encryption in Fleet.
    query: SELECT 1 FROM filevault_status WHERE status = 'FileVault is On.';
    platform: darwin
    critical: false
    calendar_event_enabled: false
```

##### Separate file
 
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

### `queries`

Queries can be specified inline in your `default.yml` file or `teams/team-name.yml` files. They can also be specified in separate files in your `lib/` folder.

Note that the `team_id` option isn't supported in GitOps.

#### Options

For possible options, see the parameter for the parameters of the [Create query API endpoint](../REST%20API/rest-api.md#create-query).

#### Example

##### Inline
  
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

##### Separate file
 
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

### `agent_options`

Agent options can be specified inline in your `default.yml` file or `teams/team-name.yml` files. They can also be specified in separate files in your `lib/` folder.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" to find all possible options.

#### Example

##### Inline
  
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

##### Separate file
 
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

### `controls`

TODO

### `org_settings` and `team_settings`

TODO

## Environment variables

TODO

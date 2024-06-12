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

#### `features`

The `features` section of the configuration YAML lets you define what predefined queries are sent to the hosts and later on processed by Fleet for different functionalities.
- `additional_queries` adds extra host details. This information will be updated at the same time as other host details and is returned by the API when host objects are returned (default: empty).
- `enable_host_users` specifies whether or not Fleet collects user data from hosts (default: `true`).
- `enable_software_inventory` specifies whether or not Fleet collects softwre inventory from hosts (default: `true`).

Example:

```yaml
org_settings:
  features:
    additional_queries:
      time: SELECT * FROM time
      macs: SELECT mac FROM interface_details
    enable_host_users: true
    enable_software_inventory: true
```

#### `fleet_desktop`

Direct end users to a custom URL when they select **Transparency** in the Fleet Desktop dropdown (default: [https://fleetdm.com/transparency](https://fleetdm.com/transparency)).

Can only be configure for all teams (`org_settings`).

Example:

```yaml
org_settings:
  fleet_desktop:
    transparency_url: "https://example.org/transparency"
```

#### `host_expiry_settings`

The `host_expiry_settings` section lets you define if and when hosts should be automatically deleted from Fleet if they have not checked in.
- `host_expiry_enabled` (default: `false`)
- `host_expiry_window` if a host has not communicated with Fleet in the specified number of days, it will be removed. Must be > `0` when host expiry is enabled (default: `0`).

Example:

```yaml
org_settings:
  host_expiry_settings:
  	host_expiry_enabled: true
    host_expiry_window: 10
```

#### `org_info`

- `name` is the name of your organization (default: empty)
- `logo_url` is a public URL of the logo for your organization (default: Fleet logo).
- `org_logo_url_light_background` is a public URL of the logo for your organization that can be used with light backgrounds (default: Fleet logo).
- `contact_url` is a URL that appears in error messages presented to end users (default: https://fleetdm.com/company/contact)

Can only be configure for all teams (`org_settings`).

Example:

```yaml
org_settings:
  org_info:
    org_name: Fleet
    org_logo_url: https://example.com/logo.png
    org_logo_url_light_background: https://example.com/logo-light.png
    contact_url: https://fleetdm.com/company/contact
```

#### `server_settings`

- `enable_analytics` specifies whether or not to enable Fleet's [usage statistics](../Using%20Fleet/Usage-statistics.md) (default: `true`).
- `live_query_disabled` disables the ability to run live queries (ad hoc queries executed via the UI or fleetctl) (default: `false`).
- `query_reports_disabled` disables query reports and deletes existing repors (default: `false`).
- `scripts_disabled` blocks access to run scripts. Scripts may still be added in the UI and CLI (defaul: `false`).
- `server_url` is the base URL of the Fleet instance (default: provided during Fleet setup)

Can only be configure for all teams (`org_settings`).

Example:

  ```yaml
org_settings:
  server_settings:
    enable_analytics: true
    live_query_disabled: false
    query_reports_disabled: false
    scripts_disabled: false
    server_url: https://instance.fleet.com
  ```

TODO is SMTP down from new fleetctl-apply.md doc

## Environment variables

TODO

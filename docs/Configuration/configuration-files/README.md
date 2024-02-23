# Configuration files

- [Queries](#queries)
- [Labels](#labels)
- [Enroll secrets](#enroll-secrets)
  - [Multiple enroll secrets](#multiple-enroll-secrets)
  - [Rotating enroll secrets](#rotating-enroll-secrets)
- [Teams](#teams)
  - [Team agent options](#team-agent-options)
  - [Team enroll secrets](#team-enroll-secrets)
  - [Mobile device management settings for teams](#mobile-device-management-mdm-settings-for-teams)
- [Organization settings](#organization-settings)

Fleet can be managed with configuration files (YAML syntax) and the fleetctl command line tool. This page tells you how to write these configuration files.

Changes are applied to Fleet when the configuration file is applied using fleetctl. Check out the [fleetctl documentation](https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-to-configure-fleet) to learn how to apply configuration files.

## Queries

The `queries` YAML file controls queries in Fleet.

You can define one or more queries in the same file with `---`.

The following example file includes several queries:

```yaml
---
apiVersion: v1
kind: query
spec:
  name: osquery_info
  description: A heartbeat counter that reports general performance (CPU, memory) and version.
  query: select i.*, p.resident_size, p.user_time, p.system_time, time.minutes as counter from osquery_info i, processes p, time where p.pid = i.pid;
  team: ""
  interval: 3600 # 1 hour
  observer_can_run: true
  automations_enabled: true
  discard_data: false
---
apiVersion: v1 
kind: query 
spec: 
  name: Get serial number of a laptop 
  description: Returns the serial number of a laptop, which can be useful for asset tracking.
  query: SELECT hardware_serial FROM system_info; 
  team: Workstations
  interval: 0
  observer_can_run: true
  discard_data: false
--- 
apiVersion: v1 
kind: query 
spec: 
  name: Get recently added or removed USB drives 
  description: Report event publisher health and track event counters. 
  query: |-
    SELECT action, DATETIME(time, 'unixepoch') AS datetime, vendor, mounts.path 
    FROM disk_events 
    LEFT JOIN mounts 
      ON mounts.device = disk_events.device
    ;
  team: Workstations (Canary)
  interval: 86400 # 24 hours
  observer_can_run: false
  min_osquery_version: 5.4.0
  platform: darwin,windows
  automations_enabled: true
  logging: differential
  discard_data: true
```

Continued edits and applications to this file will update the queries.

If you want to change the name of a query, you must first create a new query with the new name and then delete the query with the old name.

## Labels

The following file describes the labels which hosts should be automatically grouped into. The label resource should include the actual SQL query so that the label is self-contained:

```yaml
apiVersion: v1
kind: label
spec:
  name: slack_not_running
  query: >
    SELECT * FROM system_info
    WHERE NOT EXISTS (
      SELECT *
      FROM processes
      WHERE name LIKE "%Slack%"
    );
```

Labels can also be "manually managed". When defining the label, reference hosts
by hostname:

```yaml
apiVersion: v1
kind: label
spec:
  name: Manually Managed Example
  label_membership_type: manual
  hosts:
    - hostname1
    - hostname2
    - hostname3
```

## Enroll secrets

The following file shows how to configure enroll secrets. Enroll secrets are valid until you delete them.

```yaml
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
    - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
    - secret: YBh0n4pvRplKyWiowv9bf3zp6BBOJ13O
```

Osquery provides the enroll secret only during the enrollment process. Once a host is enrolled, the node key it receives remains valid for authentication independent from the enroll secret.

Currently enrolled hosts do not necessarily need enroll secrets updated, as the existing enrollment will continue to be valid as long as the host is not deleted from Fleet and the osquery store on the host remains valid. Any newly enrolling hosts must have the new secret.

Deploying a new enroll secret cannot be done centrally from Fleet.

> Enroll secrets must be alphanumeric and should not contain special characters. 

### Multiple enroll secrets

Fleet allows the abiility to maintain multiple enroll secrets. Some organizations have internal goals  around rotating secrets. Having multiple secrets allows some of them to work at the same time the rotation is happening.
Another reason you might want to use multiple enroll secrets is to use a certain [team enroll secret](#team-enroll-secrets) to auto-enroll hosts into a specific [team](https://fleetdm.com/docs/using-fleet/teams) (Fleet Premium).

### Rotating enroll secrets

Rotating enroll secrets follows this process:

1. Add a new secret.
2. Transition existing clients to the new secret. Note that existing clients may not need to be
   updated, as the enroll secret is not used by already enrolled clients.
3. Remove the old secret.

To do this with `fleetctl` (assuming the existing secret is `oldsecret` and the new secret is `newsecret`):

Begin by retrieving the existing secret configuration:

```sh
$ fleetctl get enroll_secret
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - created_at: "2021-11-17T00:39:50Z"
    secret: oldsecret
```

Apply the new configuration with both secrets:

```sh
$ echo '
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - created_at: "2021-11-17T00:39:50Z"
    secret: oldsecret
  - secret: newsecret
' > secrets.yml
$ fleetctl apply -f secrets.yml
```

Now transition clients to using only the new secret. When the transition is completed, remove the
old secret:

```sh
$ echo '
---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - secret: newsecret
' > secrets.yml
$ fleetctl apply -f secrets.yml
```

At this point, the old secret will no longer be accepted for new enrollments and the rotation is
complete.

A similar process may be followed for rotating team-specific enroll secrets. For teams, the secrets
are managed in the team yaml.

## Teams

**Applies only to Fleet Premium**.

The `team` YAML file controls a team in Fleet.

You can define one or more teams in the same file with `---`.

The following example file includes one team:

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineering
    features:
      enable_host_users: false
      enable_software_inventory: true
      additional_queries:
        time: SELECT * FROM time
        macs: SELECT mac FROM interface_details
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
          logger_tls_endpoint: /api/v1/osquery/log
          logger_tls_period: 10
          pack_delimiter: /
      overrides: {}
      command_line_flags: {}
    secrets:
      - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
      - secret: JZ/C/Z7ucq22dt/zjx2kEuDBN0iLjqfz
    host_expiry_settings: 
      host_expiry_enabled: true 
      host_expiry_window: 14
    mdm:
      macos_updates:
        minimum_version: "12.3.1"
        deadline: "2022-01-04"
      macos_settings:
        custom_settings:
          - path: '/path/to/profile1.mobileconfig'
            labels:
              - Label name 1
          - path: '/path/to/profile2.mobileconfig'
          - path: '/path/to/profile3.mobileconfig'
            labels:
              - Label name 2
              - Label name 3
        enable_disk_encryption: true
      windows_settings:
        custom_settings:
          - path: '/path/to/profile4.xml'
            labels:
              - Label name 4
          - path: '/path/to/profile5.xml'
    scripts:
        - path/to/script1.sh
        - path/to/script2.sh
```

### Team agent options

The team agent options specify options that only apply to this team. When team-specific agent options have been specified, the agent options specified at the organization level are ignored for this team.

The documentation for this section is identical to the [Agent options](#agent-options) documentation for the organization settings, except that the YAML section where it is set must be as follows. (Note the `kind: team` key and the location of the `agent_options` key under `team` must have a `name` key to identify the team to configure.)

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineering
    agent_options:
      # the team-specific options go here
```

### Team secrets

The `secrets` section provides the list of enroll secrets that will be valid for this team. When a new team is created via `fleetctl apply`, an enroll secret is automatically generated for it. If the section is missing, the existing secrets are left unmodified. Otherwise, they are replaced with this list of secrets for this team.

- Optional setting (array of dictionaries)
- Default value: none (empty)
- Config file format:
  ```yaml
  team:
    name: Client Platform Engineering
    secrets:
      - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
      - secret: JZ/C/Z7ucq22dt/zjx2kEuDBN0iLjqfz
  ```

### Modify an existing team

You can modify an existing team by applying a new team configuration file with the same `name` as an existing team. The new team configuration will completely replace the previous configuration. In order to avoid overiding existing settings, we reccomend retreiving the existing configuration and modifying it.

Retrieve the team configuration and output to a YAML file:

```sh
% fleetctl get teams --name Workstations --yaml > workstation_config.yml
```
After updating the generated YAML, apply the changes:

```sh
% fleetctl apply -f workstation_config.yml
```

Depending on your Fleet version, you may see `unsupported key` errors for the following keys when applying the new team configuration:

```text
id
user_count
host_count
integrations
webhook_settings
description
agent_options
created_at
user_count
host_count
integrations
webhook_settings
```

You can bypass these errors by removing the key from your YAML or adding the `--force` flag. This flag will apply the changes without validation and should be used with caution.

### Mobile device management (MDM) settings for teams

The `mdm` section of this configuration YAML lets you control MDM settings for each team in Fleet.

To specify Team MDM configuration, as opposed to [Organization-wide MDM configuration](#mobile-device-management-mdm-settings), follow the below YAML format. Note the `kind: team` field, as well as the  `name` and `mdm` fields under `team`.

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineering
    mdm:
      # the team-specific mdm options go here
```

### Team scripts

List of saved scripts that can be run on hosts that are part of the team.

- Default value: none
- Config file format:
  ```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineering
    scripts:
      - path/to/script1.sh
      - path/to/script2.sh
  ```

## Organization settings

The `config` YAML file controls Fleet's organization settings and MDM features for hosts assigned to "No team."

The following example file shows the default organization settings:

```yaml
apiVersion: v1
kind: config
spec:
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
    overrides: {}
    command_line_flags: {}
  features:
    enable_host_users: true
    enable_software_inventory: true
  fleet_desktop:
    transparency_url: https://fleetdm.com/transparency
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  integrations:
    jira: null
    zendesk: null
  org_info:
    org_logo_url: ""
    org_logo_url_light_background: ""
    contact_url: ""
    org_name: Fleet
  server_settings:
    deferred_save_host: false
    enable_analytics: true
    live_query_disabled: false
    query_reports_disabled: false
    scripts_disabled: false
    server_url: ""
  smtp_settings:
    authentication_method: authmethod_plain
    authentication_type: authtype_username_password
    domain: ""
    enable_smtp: false
    enable_ssl_tls: true
    enable_start_tls: true
    password: ""
    port: 587
    sender_address: ""
    server: ""
    user_name: ""
    verify_ssl_certs: true
  sso_settings:
    enable_jit_provisioning: false
    enable_sso: false
    enable_sso_idp_login: false
    entity_id: ""
    idp_image_url: ""
    idp_name: ""
    issuer_uri: ""
    metadata: ""
    metadata_url: ""
  vulnerabilities:
    databases_path: "/tmp/vulndbs"
    periodicity: 1h
    cpe_database_url: ""
    cpe_translations_url: ""
    cve_feed_prefix_url: ""
    current_instance_checks: "auto"
    disable_data_sync: false
    recent_vulnerability_max_age: 30d
    disable_win_os_vulnerabilities: false
  webhook_settings:
    failing_policies_webhook:
      destination_url: ""
      enable_failing_policies_webhook: false
      host_batch_size: 0
      policy_ids: null
    host_status_webhook:
      days_count: 0
      destination_url: ""
      enable_host_status_webhook: false
      host_percentage: 0
    interval: 24h
    vulnerabilities_webhook:
      destination_url: ""
      enable_vulnerabilities_webhook: false
      host_batch_size: 0
  mdm:
    apple_bm_default_team: ""
    windows_enabled_and_configured: false
    macos_updates:
      minimum_version: ""
      deadline: ""
    macos_settings:
      custom_settings:
        - path: '/path/to/profile1.mobileconfig'
          labels:
            - Label name 1
        - path: '/path/to/profile2.mobileconfig'
        - path: '/path/to/profile3.mobileconfig'
          labels:
            - Label name 2
            - Label name 3
      enable_disk_encryption: true
    windows_settings:
      custom_settings:
        - path: '/path/to/profile4.xml'
          labels:
            - Label name 4
        - path: '/path/to/profile5.xml'
```

### Settings

All possible settings are organized below by section.

Each section's key must be one level below the `spec` key, indented with spaces (not `<tab>` characters) as required by the YAML format.

For example, when adding the `host_expiry_settings.host_expiry_enabled` setting, you'd specify the `host_expiry_settings` section one level below the `spec` key:

```yaml
apiVersion: v1
kind: config
spec:
  host_expiry_settings:
    host_expiry_enabled: true
```

#### Features

The `features` section of the configuration YAML lets you define what predefined queries are sent to the hosts and later on processed by Fleet for different functionalities.

> Note: this section used to be named `host_settings`, but was renamed in Fleet v4.20.0,
> `host_settings` is still supported for backwards compatibility.

##### features.additional_queries

This is the additional information to collect from hosts along with the host details. This information will be updated at the same time as other host details and is returned by the API when host objects are returned. Users must take care to keep the data returned by these queries small in order to mitigate potential performance impacts on the Fleet server.

- Optional setting (dictionary of key-value strings)
- Default value: none (empty)
- Config file format:
  ```yaml
  features:
    additional_queries:
      time: SELECT * FROM time
      macs: SELECT mac FROM interface_details
  ```
- Deprecated config file format:
  ```yaml
  host_settings:
    additional_queries:
      time: SELECT * FROM time
      macs: SELECT mac FROM interface_details
  ```

##### features.enable_host_users

Whether or not Fleet sends the query needed to gather user-related data from hosts.

- Optional setting (boolean)
- Default value: `true`
- Config file format:
  ```yaml
  features:
  	enable_host_users: false
  ```
- Deprecated config file format:
  ```yaml
  host_settings:
  	enable_host_users: false
  ```

##### features.enable_software_inventory

Whether or not Fleet sends the query needed to gather the list of software installed on hosts, along with other metadata.

- Optional setting (boolean)
- Default value: `true`
- Config file format:
  ```yaml
  features:
  	enable_software_inventory: false
  ```
- Deprecated config file format:
  ```yaml
  host_settings:
  	enable_software_inventory: false
  ```

##### features.detail_query_overrides

This feature can be used to override "detail queries" hardcoded in Fleet.

> IMPORTANT: This feature should only be used when debugging issues with Fleet's hardcoded queries.
Use with caution as this may break Fleet ingestion of hosts data.

- Optional setting (dictionary of key-value strings)
- Default value: none (empty)
- Config file format:
  ```yaml
  features:
    detail_query_overrides:
      # null disables the "users" query from running on hosts.
      users: null
      # "" disables the "disk_encryption_linux" query from running on hosts.
      disk_encryption_linux: ""
      # this replaces the hardcoded "mdm" detail query.
      mdm: "SELECT enrolled, server_url, installed_from_dep, payload_identifier FROM mdm;"
  ```

#### Fleet Desktop

For more information about Fleet Desktop, see [Fleet Desktop's documentation](https://fleetdm.com/docs/using-fleet/fleet-desktop).

##### fleet_desktop.transparency_url

**Available in Fleet Premium**. Direct users of Fleet Desktop to a custom transparency URL page.

- Optional setting (string)
- Default value: Fleet's default transparency URL ("[https://fleetdm.com/transparency](https://fleetdm.com/transparency)")
- Config file format:
  ```yaml
  fleet_desktop:
    transparency_url: "https://example.org/transparency"
  ```

#### Host expiry settings

The `host_expiry_settings` section lets you define if and when hosts should be removed from Fleet if they have not checked in. Once a host has been removed from Fleet, it will need to re-enroll with a valid `enroll_secret` to connect to your Fleet instance.

##### host_expiry_settings.host_expiry_enabled

Whether offline hosts' expiration is enabled. If `host_expiry_enabled` is set to `true`, Fleet allows automatic cleanup of hosts that have not communicated with Fleet in some number of days.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  host_expiry_settings:
  	host_expiry_enabled: true
  ```

##### host_expiry_settings.host_expiry_window

If a host has not communicated with Fleet in the specified number of days, it will be removed.

- Optional setting (integer)
- Default value: `0` (must be > 0 when enabling host expiry)
- Config file format:
  ```yaml
  host_expiry_settings:
  	host_expiry_window: 10
  ```

#### Integrations

For more information about integrations and Fleet automations in general, see the [Automations documentation](https://fleetdm.com/docs/using-fleet/automations). Only one automation can be enabled for a given automation type (e.g., for failing policies, only one of the webhooks, the Jira integration, or the Zendesk automation can be enabled).

It's recommended to use the Fleet UI to configure integrations since secret credentials (in the form of an API token) must be provided. See the [Automations documentation](https://fleetdm.com/docs/using-fleet/automations) for the UI configuration steps.

#### Organization information

##### org_info.org_name

The name of the organization.

- Required setting (string)
- Default value: none (provided during Fleet setup)
- Config file format:
  ```yaml
  org_info:
  	org_name: Fleet
  ```

##### org_info.org_logo_url

The URL of the logo of the organization.

This logo is displayed in the top bar and other areas of Fleet that have dark backgrounds.

- Optional setting (string)
- Default value: none (uses Fleet's logo)
- Config file format:
  ```yaml
  org_info:
    org_logo_url: https://example.com/logo.png
  ```

##### org_info.org_logo_url_light_background

The URL of a logo of the organization that can be used with light backgrounds.

> Note: this URL is currently only used for the dialogs displayed during MDM migration

- Optional setting (string)
- Default value: none (uses Fleet's logo)
- Config file format:
  ```yaml
  org_info:
    org_logo_url_light_background: https://example.com/logo-light.png
  ```

##### org_info.contact_url

A URL that can be used by end users to contact the organization.

- Optional setting (string)
- Default value: https://fleetdm.com/company/contact
- Config file format:
  ```yaml
  org_info:
    contact_url: https://example.com/contact-us
  ```

#### Server settings

##### server_settings.debug_host_ids

There's a lot of information coming from hosts, but it's sometimes useful to see exactly what a host is returning in order
to debug different scenarios.

For example, let's say the hosts with ids 342 and 98 are not behaving as you expect in Fleet. You can enable verbose
logging with the following configuration:

```yaml
---
apiVersion: v1
kind: config
spec:
  server_settings:
    debug_host_ids:
      - 342
      - 98
```

Once you have collected the logs, you can easily disable the debug logging by applying the following configuration:

```yaml
---
apiVersion: v1
kind: config
spec:
  server_settings:
    debug_host_ids: []
```

> **Warning:** This will potentially log a lot of data. Some of that data might be private. Please verify it before posting it
in a public channel or a GitHub issue.

- Optional setting (array of integers)
- Default value: empty
- Config file format:
  ```yaml
  server_settings:
    debug_host_ids:
      - 342
      - 98
  ```

##### server_settings.deferred_save_host

Whether saving host-related information is done synchronously in the HTTP handler of the host's request, or asynchronously. This can provide better performance in deployments with many hosts. Note that this is an **experimental feature**.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  server_settings:
    deferred_save_host: true
  ```

##### server_settings.enable_analytics

If sending usage analytics is enabled or not.

- Optional setting (boolean)
- Default value: `true`
- Config file format:
  ```yaml
  server_settings:
    enable_analytics: false
  ```

##### server_settings.live_query_disabled

If the live query feature is disabled or not.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  server_settings:
    live_query_disabled: true
  ```

##### server_settings.query_reports_disabled

Whether the query reports feature is disabled.
If this setting is changed from `false` to `true`, then all stored query results will be deleted (this process can take up to one hour).

Query reports are cached results of scheduled queries stored in Fleet (up to 1000).

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  server_settings:
    query_reports_disabled: true
  ```

##### server_settings.scripts_disabled

Whether the scripts feature is disabled.

If this setting is changed from `false` to `true`, then users will not be able to execute scripts on
hosts. Scripts can still be added or modified in Fleet.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  server_settings:
    scripts_disabled: true
  ```


##### server_settings.server_url

The base URL of the fleet server, including the scheme (e.g. "https://").

- Required setting (string)
- Default value: none (provided during Fleet setup)
- Config file format:
  ```yaml
  server_settings:
    server_url: https://fleet.example.org:8080
  ```

#### SMTP settings

It's recommended to use the Fleet UI to configure SMTP since a secret password must be provided. Navigate to **Settings -> Organization settings -> SMTP Options** to proceed with this configuration.

#### SSO settings

For additional information on SSO configuration, including just-in-time (JIT) user provisioning, creating SSO users in Fleet, and identity providers configuration, see [Configuring single sign-on (SSO)](https://fleetdm.com/docs/deploying/configuration#configuring-single-sign-on-sso).

##### sso_settings.enable_jit_provisioning

**Available in Fleet Premium**. Enables [just-in-time user provisioning](https://fleetdm.com/docs/deploy/single-sign-on-sso#just-in-time-jit-user-provisioning).

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  sso_settings:
    enable_jit_provisioning: true
  ```

##### sso_settings.enable_jit_role_sync

> This setting is now deprecated and will be removed soon.
> For more information on how SSO login and role syncing works see [customization of user roles](../../Deploying/Configuration.md#customization-of-user-roles)

##### sso_settings.enable_sso

Configures if single sign-on is enabled.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  sso_settings:
    enable_sso: true
  ```

##### sso_settings.enable_sso_idp_login

Allow single sign-on login initiated by identity provider.

- Optional setting (boolean)
- Default value: `false`
- Config file format:
  ```yaml
  sso_settings:
    enable_sso_idp_login: true
  ```

##### sso_settings.entity_id

The required entity ID is a Uniform Resource Identifier (URI) that you use to identify Fleet when configuring the identity provider. It must exactly match the Entity ID field used in identity provider configuration.

- Required setting if SSO is enabled, must have at least 5 characters (string)
- Default value: ""
- Config file format:
  ```yaml
  sso_settings:
    entity_id: "https://example.com"
  ```

##### sso_settings.idp_image_url

An optional link to an image such as a logo for the identity provider.

- Optional setting (string)
- Default value: ""
- Config file format:
  ```yaml
  sso_settings:
    idp_image_url: "https://example.com/logo"
  ```

##### sso_settings.idp_name

A required human-friendly name for the identity provider that will provide single sign-on authentication.

- Required setting if SSO is enabled (string)
- Default value: ""
- Config file format:
  ```yaml
  sso_settings:
    idp_name: "SimpleSAML"
  ```

##### sso_settings.metadata

Metadata (in XML format) provided by the identity provider.

- Optional setting, either `metadata` or `metadata_url` must be set if SSO is enabled, but not both (string).
- Default value: "".
- Config file format:
  ```yaml
  sso_settings:
    metadata: "<md:EntityDescriptor entityID="https://idp.example.org/SAML2"> ... /md:EntityDescriptor>"
  ```

##### sso_settings.metadata_url

A URL that references the identity provider metadata.

- Optional setting, either `metadata` or `metadata_url` must be set if SSO is enabled, but not both (string).
- Default value: "".
- Config file format:
  ```yaml
  sso_settings:
    metadata_url: https://idp.example.org/idp-meta.xml
  ```

#### Vulnerability settings

##### vulnerabilities.databases_path

Path to a directory on the local filesystem (accessible to the Fleet server) where the various vulnerability databases will be stored.

- Optional setting, must be set to enable vulnerability detection (string).
- Default value: "/tmp/vulndb".
- Config file format:
  ```yaml
  vulnerabilities:
    databases_path: "/path/to/dir"
  ```

#### Webhook settings

For more information about webhooks and Fleet automations in general, see the [Automations documentation](https://fleetdm.com/docs/using-fleet/automations).

##### webhook_settings.interval

The interval at which to check for webhook conditions. This value currently configures both the host status and failing policies webhooks, but not the recent vulnerabilities webhook. (See the [Recent vulnerabilities section](#recent-vulnerabilities) for details.)

- Optional setting (time duration as a string)
- Default value: `24h`
- Config file format:
  ```yaml
  webhook_settings:
    interval: "12h"
  ```
  > **Note:** Fleet's webhook notifications about failing policies default to a 24h time interval based upon the initial start time of the policy. To adjust policy automation intervals, set the interval to a longer period and manually trigger automations using fleetctl. You'll see small differences over time as well based on how long it takes to run, other jobs that are queued up at the same time, etc.
  
  
##### Failing policies webhook

The following options allow the configuration of a webhook that will be triggered if selected policies are not passing for some hosts.

###### webhook_settings.failing_policies_webhook.destination_url

The URL to `POST` to when the condition for the webhook triggers.

- Optional setting, required if webhook is enabled (string).
- Default value: "".
- Config file format:
  ```yaml
  webhook_settings:
    failing_policies_webhook:
      destination_url: "https://example.org/webhook_handler"
  ```

###### webhook_settings.failing_policies_webhook.enable_failing_policies_webhook

Defines whether to enable the failing policies webhook. Note that currently, if the failing policies webhook and the `osquery.enable_async_host_processing` options are set, some failing policies webhooks could be missing. Some transitions from succeeding to failing or vice-versa could happen without triggering a webhook request.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```yaml
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
  ```

###### webhook_settings.failing_policies_webhook.host_batch_size

Maximum number of hosts to batch on `POST` requests. A value of `0`, the default, means no batching. All hosts failing a policy will be sent on one `POST` request.

- Optional setting (integer).
- Default value: `0`.
- Config file format:
  ```yaml
  webhook_settings:
    failing_policies_webhook:
      host_batch_size: 100
  ```

###### webhook_settings.failing_policies_webhook.policy_ids

The IDs of the policies for which the webhook will be enabled.

- Optional setting (array of integers).
- Default value: empty.
- Config file format:
  ```yaml
  webhook_settings:
    failing_policies_webhook:
      policy_ids:
        - 1
        - 2
        - 3
  ```

##### Host status webhook

The following options allow the configuration of a webhook that will be triggered if the specified percentage of hosts are offline for the specified amount of time.

###### webhook_settings.host_status_webhook.days_count

Number of days that hosts need to be offline to count as part of the percentage.

- Optional setting, required if webhook is enabled (integer).
- Default value: `0`.
- Config file format:
  ```yaml
  webhook_settings:
    host_status_webhook:
      days_count: 5
  ```

###### webhook_settings.host_status_webhook.destination_url

The URL to `POST` to when the condition for the webhook triggers.

- Optional setting, required if webhook is enabled (string).
- Default value: "".
- Config file format:
  ```yaml
  webhook_settings:
    host_status_webhook:
      destination_url: "https://example.org/webhook_handler"
  ```

###### webhook_settings.host_status_webhook.enable_host_status_webhook

Defines whether the webhook check for host status will run or not.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```yaml
  webhook_settings:
    host_status_webhook:
      enable_host_status_webhook: true
  ```

###### webhook_settings.host_status_webhook.host_percentage

The percentage of hosts that need to be offline to trigger the webhook.

- Optional setting, required if webhook is enabled (float).
- Default value: `0`.
- Config file format:
  ```yaml
  webhook_settings:
    host_status_webhook:
      host_percentage: 10
  ```

##### Vulnerabilities webhook

The following options allow the configuration of a webhook that will be triggered if recently published vulnerabilities are detected and there are affected hosts. A vulnerability is considered recent if it has been published in the last 30 days (based on the National Vulnerability Database, NVD).

Note that the recent vulnerabilities webhook is not checked at `webhook_settings.interval` like other webhooks. It is checked as part of the vulnerability processing and runs at the `vulnerabilities.periodicity` interval specified in the [fleet configuration](https://fleetdm.com/docs/deploying/configuration#periodicity).

###### webhook_settings.vulnerabilities_webhook.destination_url

The URL to `POST` to when the condition for the webhook triggers.

- Optional setting, required if webhook is enabled (string).
- Default value: "".
- Config file format:
  ```yaml
  webhook_settings:
    vulnerabilities_webhook:
      destination_url: "https://example.org/webhook_handler"
  ```

###### webhook_settings.vulnerabilities_webhook.enable_vulnerabilities_webhook

Defines whether to enable the vulnerabilities webhook.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```yaml
  webhook_settings:
    vulnerabilities_webhook:
      enable_vulnerabilities_webhook: true
  ```

###### webhook_settings.vulnerabilities_webhook.host_batch_size

Maximum number of hosts to batch on `POST` requests. A value of `0`, the default, means no batching. All hosts affected will be sent on one `POST` request.

- Optional setting (integer).
- Default value: `0`.
- Config file format:
  ```yaml
  webhook_settings:
    vulnerabilities_webhook:
      host_batch_size: 100
  ```

#### Agent options

The `agent_options` key controls the settings applied to the agent on all your hosts. These settings are applied when each host checks in.

See "[Agent configuration](https://fleetdm.com/docs/configuration/agent-configuration)" for more information.

#### Mobile device management (MDM) settings

The `mdm` section of the configuration YAML lets you control MDM settings in Fleet.

##### mdm.apple_bm_default_team

**Applies only to Fleet Premium**.

Set name of default team to use with Apple Business Manager.

- Default value: ""
- Config file format:
  ```yaml
  mdm:
    apple_bm_default_team: "Workstations"
  ```

##### mdm.windows_enabled_and_configured

Enables or disables Windows MDM support.

- Default value: false
- Config file format:
  ```yaml
  mdm:
    windows_enabled_and_configured: true
  ```

##### mdm.macos_updates

**Applies only to Fleet Premium**.

The following options allow configuring OS updates for macOS hosts.

##### mdm.macos_updates.minimum_version

Requires `mdm.macos_updates.deadline` to be set.

- Default value: ""
- Config file format:
  ```yaml
  mdm:
    macos_updates:
      minimum_version: "12.1.1"
  ```

##### mdm.macos_updates.deadline

A deadline in the form of `YYYY-MM-DD`. The exact deadline time is at 04:00:00 (UTC-8).

Requires `mdm.macos_updates.minimum_version` to be set.

- Default value: ""
- Config file format:
  ```yaml
  mdm:
    macos_updates:
      deadline: "2022-01-01"
  ```

##### mdm.windows_updates

**Applies only to Fleet Premium**.

The following options allow configuring OS updates for Windows hosts.

##### mdm.windows_updates.deadline

A deadline in days.

- Default value: ""
- Config file format:
  ```yaml
  mdm:
    windows_updates:
      deadline_days: "5"
  ```

##### mdm.windows_updates.grace_period

A grace period in days.

- Default value: ""
- Config file format:
  ```yaml
  mdm:
    windows_updates:
      grace_period_days: "2"
  ```

##### mdm.macos_settings

The following settings are macOS-specific settings for Fleet's MDM solution.

##### mdm.macos_settings.custom_settings

List of configuration profile files to apply to all hosts.

If you're using Fleet Premium, these profiles apply to all hosts assigned to no team.

> If you want to add profiles to all macOS hosts on a specific team in Fleet, use the `team` YAML document. Learn how to create one [here](#teams).

- Default value: none
- Config file format:
  ```yaml
  mdm:
    macos_settings:
      custom_settings:
        - path: '/path/to/profile1.mobileconfig'
          labels:
            - Label name 1
        - path: '/path/to/profile2.mobileconfig'
        - path: '/path/to/profile3.mobileconfig'
          labels:
            - Label name 2
            - Label name 3
  ```

##### mdm.macos_settings.enable_disk_encryption

**Applies only to Fleet Premium**.

Enforce disk encryption and disk encryption key escrow on all hosts.

If you're using Fleet Premium, this enforces disk encryption on all hosts assigned to no team.

> If you want to enforce disk encryption on all macOS hosts on a specific team in Fleet, use the `team` YAML document. Learn how to create one [here](#teams).

- Default value: false
- Config file format:
  ```yaml
  mdm:
    macos_settings:
      enable_disk_encryption: true
  ```

##### mdm.windows_settings

The following settings are Windows-specific settings for Fleet's MDM solution.

##### mdm.windows_settings.custom_settings

List of configuration profile files to apply to all hosts.

If you're using Fleet Premium, these profiles apply to all hosts assigned to no team.

> If you want to add profiles to all Windows hosts on a specific team in Fleet, use the `team` YAML document. Learn how to create one [here](#teams).

- Default value: none
- Config file format:
  ```yaml
  mdm:
    windows_settings:
      custom_settings:
        - path: '/path/to/profile1.xml'
          labels:
            - Label name 1
        - path: '/path/to/profile2.xml'
  ```

#### Scripts 

List of saved scripts that can be run on all hosts.

> If you want to add scripts to hosts on a specific team in Fleet, use the `team` YAML document. Learn how to create one [here](#teams).

- Default value: none
- Config file format:
  ```yaml
  scripts:
    - path/to/script1.sh
    - path/to/script2.sh
  ```

#### Advanced configuration

> **Note:** More settings are included in the [contributor documentation](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Configuration-for-contributors.md). It's possible, although not recommended, to configure these settings in the YAML configuration file.

<meta name="description" value="Learn how to use configuration files and the fleetctl command line tool to configure Fleet.">

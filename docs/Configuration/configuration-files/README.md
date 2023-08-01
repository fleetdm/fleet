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

Osquery provides the enroll secret only during the enrollment process. Once a host is enrolled, the node key it receives remains valid for authentication independent from the enroll secret.

Currently enrolled hosts do not necessarily need enroll secrets updated, as the existing enrollment will continue to be valid as long as the host is not deleted from Fleet and the osquery store on the host remains valid. Any newly enrolling hosts must have the new secret.

Deploying a new enroll secret cannot be done centrally from Fleet.

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

```
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

```
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

```
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
    mdm:
      macos_updates:
        minimum_version: "12.3.1"
        deadline: "2022-01-04"
      macos_settings:
        custom_settings:
          - path/to/profile1.mobileconfig
          - path/to/profile2.mobileconfig
        enable_disk_encryption: true
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

The `secrets` section provides the list of enroll secrets that will be valid for this team. If the section is missing, the existing secrets are left unmodified. Otherwise, they are replaced with this list of secrets for this team.

- Optional setting (array of dictionaries)
- Default value: none (empty)
- Config file format:
  ```
  team:
    name: Client Platform Engineering
    secrets:
      - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
      - secret: JZ/C/Z7ucq22dt/zjx2kEuDBN0iLjqfz
  ```

### Modify an existing team

You can modify an existing team by applying a new team configuration file with the same `name` as an existing team. The new team configuration will completely replace the previous configuration. In order to avoid overiding existing settings, we reccomend retreiving the existing configuration and modifying it.

Retrieve the team configuration and output to a YAML file:

```console
% fleetctl get teams --name Workstations --yaml > workstation_config.yml
```
After updating the generated YAML, apply the changes:

```console
% fleetctl apply -f workstation_config.yml
```

Depending on your Fleet version, you may see `unsupported key` errors for the following keys when applying the new team configuration:

```
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
    org_name: Fleet
  server_settings:
    deferred_save_host: false
    enable_analytics: true
    live_query_disabled: false
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
        - path/to/profile1.mobileconfig
        - path/to/profile2.mobileconfig
      enable_disk_encryption: true
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
      # null allows to disable the "users" query from running on hosts.
      users: null
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

- Optional setting (string)
- Default value: none (uses Fleet's logo)
- Config file format:
  ```yaml
  org_info:
  	org_logo_url: https://example.com/logo.png
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
  ```
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

**Available in Fleet Premium**. Enables [just-in-time user provisioning](https://fleetdm.com/docs/deploying/configuration#just-in-time-jit-user-provisioning).

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

See the [osquery documentation](https://osquery.readthedocs.io/en/stable/installation/cli-flags/#configuration-control-flags) for the available options. This document shows all examples in command line flag format. Remove the dashed lines (`--`) for Fleet to successfully update the setting. For example, use `distributed_interval` instead of `--distributed_interval`.

Agent options are validated using the latest version of osquery.

When updating agent options, you may see an error similar to this:

```
[...] unsupported key provided: "logger_plugin"
If you’re not using the latest osquery, use the fleetctl apply --force command to override validation.
```

This error indicates that you're providing a config option that isn't valid in the current version of osquery, typically because you're setting a command line flag through the configuration key. This has always been unsupported through the config plugin, but osquery has recently become more opinionated and Fleet now validates the configuration to make sure there aren't errors in the osquery agent.

If you are not using the latest version of osquery, you can create a config YAML file and apply it with `fleetctl` using the `--force` flag to override the validation:

```fleetctl apply --force -f config.yaml```

You can verify that your agent options are valid by using [the fleetctl apply command](https://fleetdm.com/docs/using-fleet/fleetctl-cli#fleetctl-apply) with the `--dry-run` flag. This will report any error and do nothing if the configuration was valid. If you don't use the latest version of osquery, you can override validation using the `--force` flag. This will update agent options even if they are invalid.

Existing options will be overwritten by the application of this file.

##### `command_line_flags` option

> This feature requires [Fleetd, the Fleet agent manager](https://fleetdm.com/announcements/introducing-orbit-your-fleet-agent-manager).

The `command_line_flags` key inside of `agent_options` allows you to remotely manage the osquery command line flags. These command line flags are options that typically require osquery to restart for them to take effect. But with Fleetd, you can use the `command_line_flags` key to take care of that. Fleetd will write these to the flagfile on the host and pass it to osquery.

To see the full list of these osquery command line flags, please run `osquery` with the `--help` switch.

> YAML `command_line_flags` are not additive and will replace any osquery command line flags in the CLI.

Just like the other `agent_options` above, remove the dashed lines (`--`) for Fleet to successfully update them.

Here is an example of using the `command_line_flags` key:

```yaml
apiVersion: v1
kind: config
spec:
  agent_options:
    command_line_flags: # requires Fleet's osquery installer
      verbose: true
      disable_watchdog: false
      logger_path: /path/to/logger
```

Note that the `command_line_flags` key does not support the `overrides` key, which is documented below.

You can verify that these flags have taken effect on the hosts by running a query against the `osquery_flags` table.

> If you revoked an old enroll secret, this feature won't update for hosts that were added to Fleet using this old enroll secret. This is because Fleetd uses the enroll secret to receive new flags from Fleet. For these hosts, all existing features will work as expected.

For further documentation on how to rotate enroll secrets, please see [this guide](#rotating-enroll-secrets).

If you prefer to deploy a new package with the updated enroll secret:

1. Check which hosts need a new enroll secret by running the following query: `SELECT * FROM orbit_info WHERE enrolled = false`.

> The hosts that don't have Fleetd installed will return an error because the `orbit_info` table doesn't exist. You can safely ignore these errors.

2. In Fleet, head to the Hosts page and select **Add hosts** to find the fleetctl package command with an active enroll secret.

3. Copy and run the fleetctl package command to create a new package. Distribute this package to the hosts that returned results in step 1.

4. Done!



> In order for these options to be applied to your hosts, the `osquery` agent must be configured to use the `tls` config plugin and pointed to the correct endpoint. If you are using Fleetd to enroll your hosts, this is done automatically.

```
"--config_plugin=tls",
"--config_tls_endpoint=" + path.Join(prefix, "/api/v1/osquery/config")
```

```yaml
apiVersion: v1
kind: config
spec:
  agent_options:
```

##### `extensions` option

> This feature requires [Fleetd, the Fleet agent manager](https://fleetdm.com/announcements/introducing-orbit-your-fleet-agent-manager), along with a custom TUF auto-update server.

The `extensions` key inside of `agent_options` allows you to remotely manage and deploy osquery extensions. Just like other `agent_options` the `extensions` key can be applied either to a team specific one or the global one.


This is best illustrated with an example. Here is an example of using the `extensions` key:

```yaml
apiVersion: v1
kind: config
spec:
  agent_options:
    extensions: # requires Fleet's osquery installer
      hello_world:
        channel: 'stable'
        platform: 'macos'
```

In the above example, we are configuring our `hello_world` extension. We do this by creating a `hello_world` subkey under `extensions`, and then specifying the `channel` and `platform` keys for that extension.

Next, you will need to make sure to push the binary file of our `hello_world` extension as a target on your TUF server. This step needs to follow these conventions:

* The binary file of the extension must have the same name as the extension, followed by the `.ext`. In the above case, the filename should be `hello_world.ext`
* The target name for the TUF server must be named as `extensions/<extension_name>`. For the above example, this would be `extensions/hello_world`
* `platform` is one of `macos`, `linux`, or `windows`

If you are using `fleetctl` to manage your TUF server, these same conventions apply. You can run the following command to add a new target:

```bash
fleetctl updates add --path /path/to/local/TUF/repo --target /path/to/extensions/binary/hello_world.ext --name extensions/hello_world --platform macos --version 0.1
```

After successfully configuring the agent options, and pushing the extension as a target on your TUF server, Fleetd will periodically check with the TUF server for updates to these extensions.

If you are using a self-hosted TUF server, you must also manage all of Fleetd's versions, including osquery, Fleet Desktop and osquery extensions.

Fleet recommends deploying extensions created with osquery-go or natively with C++, instead of Python. Extensions written in Python require the user to compile it into a single packaged binary along with all the dependencies.


##### Example Agent options YAML

```yaml
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        distributed_interval: 3
        distributed_tls_max_attempts: 3
        logger_tls_endpoint: /api/osquery/log
        logger_tls_period: 10
      decorators:
        load:
          - "SELECT version FROM osquery_info"
          - "SELECT uuid AS host_uuid FROM system_info"
        always:
          - "SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1"
        interval:
          3600: "SELECT total_seconds AS uptime FROM uptime"
    overrides:
      # Note configs in overrides take precedence over the default config defined
      # under the config key above. Be aware that these overrides are NOT merged
      # with the top-level configuration!! This means that settings values defined
      # on the top-level config.options section will not be propagated to the platform
      # override sections. So for example, the config.options.distributed_interval value
      # will be discarded on a platform override section, and only the section value
      # for distributed_interval will be used. If the given setting is not specified
      # in the override section, its default value will be enforced.
      # Going back to the example, if the override section is windows,
      # overrides.platforms.windows.distributed_interval will have to be set again to 5
      # for this setting to be enforced as expected, otherwise the setting will get
      # its default value (60 in the case of distributed_interval).
      # Hosts receive overrides based on the platform returned by `SELECT platform FROM os_version`.
      # In this example, the base config would be used for Windows and CentOS hosts,
      # while Mac and Ubuntu hosts would receive their respective overrides.
      platforms:
        darwin:
          options:
            distributed_interval: 10
            distributed_tls_max_attempts: 10
            logger_tls_endpoint: /api/osquery/log
            logger_tls_period: 300
            disable_tables: chrome_extensions
            docker_socket: /var/run/docker.sock
          file_paths:
            users:
              - /Users/%/Library/%%
              - /Users/%/Documents/%%
            etc:
              - /etc/%%

        ubuntu:
          options:
            distributed_interval: 10
            distributed_tls_max_attempts: 3
            logger_tls_endpoint: /api/osquery/log
            logger_tls_period: 60
            schedule_timeout: 60
            docker_socket: /etc/run/docker.sock
          file_paths:
            homes:
              - /root/.ssh/%%
              - /home/%/.ssh/%%
            etc:
              - /etc/%%
            tmp:
              - /tmp/%%
          exclude_paths:
            homes:
              - /home/not_to_monitor/.ssh/%%
            tmp:
              - /tmp/too_many_events/
          decorators:
            load:
              - "SELECT * FROM cpuid"
              - "SELECT * FROM docker_info"
            interval:
              3600: "SELECT total_seconds AS uptime FROM uptime"
  host_expiry_settings:
    # ...
```
##### agent_options.config

The config key sets the osqueryd configuration options for your agents. In a plain osquery deployment, these would typically be set in `osquery.conf`. Each key below represents a corresponding key in the osquery documentation.

For detailed information on osquery configuration options, check out the [osquery configuration docs](https://osquery.readthedocs.io/en/stable/deployment/configuration/).

```yaml
agent_options:
    config:
      options: ~
      decorators: ~
      yara: ~
    overrides: ~

```

###### agent_options.config.options

In the options key, you can set your osqueryd options and feature flags.

Any command line only flags must be set using the `command_line_flags` key for Orbit agents, or by modifying the osquery flags on your hosts if you're using plain osquery.

To see a full list of flags, broken down by the method you can use to set them (configuration options vs command line flags), you can run `osqueryd --help` on a plain osquery agent. For Orbit agents, run `sudo orbit osqueryd --help`. The options will be shown there in command line format as `--key value`. In `yaml` format, that would become `key: value`.

```yaml
agent_options:
    config:
      options:
        distributed_interval: 3
        distributed_tls_max_attempts: 3
        logger_tls_endpoint: /api/osquery/log
        logger_tls_period: 10
      decorators: ~
```
###### agent_options.config.decorators

In the decorators key, you can specify queries to include additional information in your osquery results logs.

Use `load` for details you want to update values when the configuration loads, `always` to update every time a scheduled query is run, and `interval` if you want to update on a schedule.

```yaml
agent_options:
    config:
      options: ~
      decorators:
        load:
          - "SELECT version FROM osquery_info"
          - "SELECT uuid AS host_uuid FROM system_info"
        always:
          - "SELECT user AS username FROM logged_in_users WHERE user <> '' ORDER BY time LIMIT 1"
        interval:
          3600: "SELECT total_seconds AS uptime FROM uptime"
```

##### agent_options.config.yara

You can use Fleet to configure the `yara` and `yara_events` osquery tables. Fore more information on YARA configuration and continuous monitoring using the `yara_events` table, check out the [YARA-based scanning with osquery section](https://osquery.readthedocs.io/en/stable/deployment/yara/) of the osquery documentation.

The following is an example Fleet configuration file with YARA configuration. The values are taken from an example config supplied in the above link to the osquery documentation.

```yaml
---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      # ...
      yara:
        file_paths:
          system_binaries:
          - sig_group_1
          tmp:
          - sig_group_1
          - sig_group_2
        signatures:
          sig_group_1:
          - /Users/wxs/sigs/foo.sig
          - /Users/wxs/sigs/bar.sig
          sig_group_2:
          - /Users/wxs/sigs/baz.sig
    overrides: {}
```

##### agent_options.overrides

The `overrides` key allows you to segment hosts, by their platform, and supply these groups with unique osquery configuration options. When you choose to use the overrides option for a specific platform, all options specified in the default configuration will be ignored for that platform.

In the example file below, all Darwin and Ubuntu hosts will **only** receive the options specified in their respective overrides sections.

> IMPORTANT: If a given option is not specified in a platform override section, its default value will be enforced.

```yaml
agent_options:
  config:
    options:
      distributed_interval: 3
      distributed_tls_max_attempts: 3
      logger_tls_period: 10
  overrides:
    # Note configs in overrides take precedence over the default config defined
    # under the config key above. Hosts receive overrides based on the platform
    # returned by `SELECT platform FROM os_version`. In this example, the base
    # config would be used for Windows and CentOS hosts, while Mac and Ubuntu
    # hosts would receive their respective overrides. Note, these overrides are
    # NOT merged with the top level configuration.
    platforms:
      darwin:
        options:
          distributed_interval: 10
          distributed_tls_max_attempts: 10
          logger_tls_endpoint: /api/osquery/log
          logger_tls_period: 300
          disable_tables: chrome_extensions
          docker_socket: /var/run/docker.sock
        file_paths:
          users:
            - /Users/%/Library/%%
            - /Users/%/Documents/%%
          etc:
            - /etc/%%
```
##### agent_options.auto_table_construction

You can use Fleet to query local SQLite databases as tables. For more information on creating ATC configuration from a SQLite database, check out the [Automatic Table Construction section](https://osquery.readthedocs.io/en/stable/deployment/configuration/#automatic-table-construction) of the osquery documentation.

If you already know what your ATC configuration needs to look like, you can add it to an options config file:

```yaml
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        # ...
    overrides:
      platforms:
        darwin:
          auto_table_construction:
            tcc_system_entries:
              query: "SELECT service, client, allowed, prompt_count, last_modified FROM access"
              path: "/Library/Application Support/com.apple.TCC/TCC.db"
              columns:
                - "service"
                - "client"
                - "allowed"
                - "prompt_count"
                - "last_modified"
```


##### agent_options.command_line_flags

> Requires Fleet v4.22.0 or later and Orbit v1.3.0 or later**

In the `command_line_flags` key, you can update the osquery flags of your Orbit enrolled agents.

```yaml
agent_options:
  config:
  overrides:
  command_line_flags:
    enable_file_events: true
```

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

> Windows MDM features are not ready for production and are currently in development. These features are disabled by default.

Enables or disables Windows MDM support.

- Default value: false
- Config file format:
  ```yaml
  mdm:
    windows_enabled_and_configured: true
  ```

##### mdm.macos_updates

**Applies only to Fleet Premium**.

The following options allow configuring the behavior of Nudge for macOS hosts that belong to no team and are enrolled into Fleet's MDM.

##### mdm.macos_updates.minimum_version

Hosts that belong to no team and are enrolled into Fleet's MDM will be nudged until their macOS is at or above this version.

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

Hosts that belong to no team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past.

Requires `mdm.macos_updates.minimum_version` to be set.

- Default value: ""
- Config file format:
  ```yaml
  mdm:
    macos_updates:
      deadline: "2022-01-01"
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
        - path/to/profile1.mobileconfig
        - path/to/profile2.mobileconfig
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

#### Advanced configuration

> **Note:** More settings are included in the [contributor documentation](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Configuration-for-contributors.md). It's possible, although not recommended, to configure these settings in the YAML configuration file.

<meta name="description" value="Learn how to use configuration files and the fleetctl command line tool to configure Fleet.">

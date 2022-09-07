# Configuration files

- [Queries](#queries)
- [Packs](#packs)
- [Labels](#labels)
- [Enroll secrets](#enroll-secrets)
- [Teams](#teams)
- [Organization settings](#organization-settings)

Entities in Fleet, such as queries, packs, labels, agent options, and enroll secrets, can be managed with configuration files in YAML syntax.

This page contains links to examples that can help you understand the configuration options for your Fleet YAML file(s).

Examples in this directory are presented in two forms:
- [`single-file-configuration.yml`](./single-file-configuration.yml) presents multiple YAML documents in one file. One file is often easier to manage than several. Group related objects into a single file whenever it makes sense.
- The `multi-file-configuration` directory presents multiple YAML documents in separate files. They are in the following structure:

```
├─ packs
├   └─ osquery-monitoring.yml
├─ agent-options.yml
├─ enroll-secrets.yml
├─ labels.yml
├─ queries.yml
```

## Using YAML files in Fleet

A Fleet configuration is defined using one or more declarative "messages" in YAML syntax. Each message can live in it's own file or multiple in one file, each separated by `---`. Each file/message contains a few required top-level keys:

- `apiVersion` - the API version of the file/request
- `spec` - the "data" of the request
- `kind ` - the type of file/object (i.e.: pack, query, config)

The file may optionally also include some `metadata` for more complex data types (i.e.: packs).

When you reason about how to manage these config files, consider following the [General Config Tips](https://kubernetes.io/docs/concepts/configuration/overview/#general-config-tips) published by the Kubernetes project. Some of the especially relevant tips are included here as well:

- When defining configurations, specify the latest stable API version.
- Configuration files should be stored in version control before being pushed to the cluster. This allows quick roll-back of a configuration if needed. It also aids with cluster re-creation and restoration if necessary.
- Don’t specify default values unnecessarily – simple and minimal configs will reduce errors.

## Queries

For especially long or complex queries, you may want to define one query in one file. Continued edits and applications to this file will update the query as long as the `metadata.name` does not change. If you want to change the name of a query, you must first create a new query with the new name and then delete the query with the old name. Make sure the old query name is not defined in any packs before deleting it or an error will occur.

```yaml
apiVersion: v1
kind: query
spec:
  name: docker_processes
  description: The docker containers processes that are running on a system.
  query: SELECT * FROM docker_container_processes;
```

To define multiple queries in a file, concatenate multiple `query` resources together in a single file with `---`. For example, consider a file that you might store at `queries/osquery_monitoring.yml`:

```yaml
---
apiVersion: v1
kind: query
spec:
  name: osquery_schedule
  description: Report performance stats for each file in the query schedule.
  query: select name, interval, executions, output_size, wall_time, (user_time/executions) as avg_user_time, (system_time/executions) as avg_system_time, average_memory, last_executed from osquery_schedule;
---
apiVersion: v1
kind: query
spec:
  name: osquery_info
  description: A heartbeat counter that reports general performance (CPU, memory) and version.
  query: select i.*, p.resident_size, p.user_time, p.system_time, time.minutes as counter from osquery_info i, processes p, time where p.pid = i.pid;
---
apiVersion: v1
kind: query
spec:
  name: osquery_events
  description: Report event publisher health and track event counters.
  query: select name, publisher, type, subscriptions, events, active from osquery_events;
```

## Packs

To define query packs (packs), reference queries defined elsewhere by name. This is why the "name" of a query is so important. You can define many of these packs in many files.

```yaml
apiVersion: v1
kind: pack
spec:
  name: osquery_monitoring
  disabled: false
  targets:
    labels:
      - All Hosts
  queries:
    - query: osquery_version
      name: osquery_version_differential
      interval: 7200
    - query: osquery_version
      name: osquery_version_snapshot
      interval: 7200
      snapshot: true
    - query: osquery_schedule
      interval: 7200
      removed: false
    - query: osquery_events
      interval: 86400
      removed: false
    - query: osquery_info
      interval: 600
      removed: false
```

The `targets` field allows you to specify the `labels` field. With the `labels` field, the hosts that become members of the specified labels, upon enrolling to Fleet, will automatically become targets of the given pack.

### Labels

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

The following file shows how to configure enroll secrets.

```yaml
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
    - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
    - secret: YBh0n4pvRplKyWiowv9bf3zp6BBOJ13O
```

## Teams

_Applies only to Fleet Premium_.

The following is an example configuration file for a Team.

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
          logger_plugin: tls
          logger_tls_endpoint: /api/v1/osquery/log
          logger_tls_period: 10
          pack_delimiter: /
      overrides: {}
    secrets:
      - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
      - secret: JZ/C/Z7ucq22dt/zjx2kEuDBN0iLjqfz
```

### Team settings

#### Team agent options

The team agent options specifies options that apply only to this team. When team-specific agent options have been specified, the agent options specified at the organization level are ignored for this team.

The documentation for this section is identical to the [Agent options](#agent-options) documentation for the organization settings, except that the yaml section where it is set must be as follows (note the `kind: team` key, and the location of the `agent_options` key under `team`, which must have a `name` key to identify the team to configure):

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineering
    agent_options:
      # the team-specific options go here
```

#### Secrets

The `secrets` section provides the list of enroll secrets that will be valid for this team. If the section is missing, the existing secrets are left unmodified, otherwise they are replaced with this list of secrets for this team.

- Optional setting (array of dictionaries).
- Default value: none (empty).
- Config file format:
  ```
  team:
    name: Client Platform Engineering
    secrets:
      - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
      - secret: JZ/C/Z7ucq22dt/zjx2kEuDBN0iLjqfz
  ```

## Organization settings

The `config` YAML file controls Fleet's organization settings.

The following example file, shows the default organization settings.

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
        logger_plugin: tls
        logger_tls_endpoint: /api/osquery/log
        logger_tls_period: 10
        pack_delimiter: /
    overrides: {}
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
  vulnerability_settings:
    databases_path: ""
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
```

### Settings

All possible settings are organized below by section.

Each section's key must be one level below the `spec` key, indented with spaces (not `<tab>` charaters) as required by the YAML format.

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

Additional information to collect from hosts along with the host details. This information will be updated at the same time as other host details and is returned by the API when host objects are returned. Users must take care to keep the data returned by these queries small in order to mitigate potential performance impacts on the Fleet server.

- Optional setting (dictionary of key-value strings).
- Default value: none (empty).
- Config file format:
  ```
  features:
  	additional_queries:
      time: SELECT * FROM time
      macs: SELECT mac FROM interface_details
  ```
- Deprecated config file format:
  ```
  host_settings:
  	additional_queries:
      time: SELECT * FROM time
      macs: SELECT mac FROM interface_details
  ```

##### features.enable_host_users

Whether or not Fleet sends the query needed to gather user-related data from hosts.

- Optional setting (boolean).
- Default value: `true`.
- Config file format:
  ```
  features:
  	enable_host_users: false
  ```
- Deprecated config file format:
  ```
  host_settings:
  	enable_host_users: false
  ```

##### features.enable_software_inventory

Whether or not Fleet sends the query needed to gather the list of software installed on hosts, along with other metadata.

- Optional setting (boolean).
- Default value: `true`.
- Config file format:
  ```
  features:
  	enable_software_inventory: false
  ```
- Deprecated config file format:
  ```
  host_settings:
  	enable_software_inventory: false
  ```

#### Fleet Desktop Settings

For more information about Fleet Desktop, see [Fleet Desktop's documentation](../../Using-Fleet/Fleet-desktop.md).

##### fleet_desktop.transparency_url

_Available in Fleet Premium_. Sets a custom transparency URL page to take users of Fleet Desktop to.

- Optional setting (string).
- Default value: Fleet's default transparency URL ("https://fleetdm.com/transparency").
- Config file format:
  ```
  fleet_desktop:
    transparency_url: "https://example.org/transparency"
  ```

#### Host Expiry Settings

The `host_expiry` section lets you define if and when hosts should be removed from Fleet if they have not checked in. Once a host has been removed from Fleet, it will need to re-enroll with a valid `enroll_secret` to connect to your Fleet instance.

##### host_expiry_settings.host_expiry_enabled

Whether offline hosts' expiration is enabled. If `host_expiry_enabled` is set to `true`, Fleet allows automatic cleanup of hosts that have not communicated with Fleet in some number of days.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  host_expiry_settings:
  	host_expiry_enabled: true
  ```

##### host_expiry_settings.host_expiry_window

If a host has not communicated with Fleet in the specified number of days, it will be removed.

- Optional setting (integer).
- Default value: `0` (must be > 0 when enabling host expiry).
- Config file format:
  ```
  host_expiry_settings:
  	host_expiry_window: 10
  ```

#### Integrations

For more information about integrations and Fleet automations in general, see the [Automations documentation](../../Using-Fleet/Automations.md). Only one automation can be enabled for a given automation type (e.g. for Failing Policies, only one of the webhook, the Jira integration or the Zendesk automation can be enabled).

It is recommended to use the Fleet UI to configure integrations, as secret credentials (in the form of an API token) must be provided. See the [Automations documentation](../../Using-Fleet/Automations.md) for the UI configuration steps.

#### Organization Information

##### org_info.org_name

The name of the organization.

- Required setting (string).
- Default value: none (provided during Fleet setup).
- Config file format:
  ```
  org_info:
  	org_name: Fleet
  ```

##### org_info.org_logo_url

The URL of the logo of the organization.

- Optional setting (string).
- Default value: none (uses Fleet's logo).
- Config file format:
  ```
  org_info:
  	org_logo_url: https://example.com/logo.png
  ```

#### Server Settings

##### server_settings.debug_host_ids

There's a lot of information coming from hosts, but it's sometimes useful to see exactly what a host is returning in order
to debug different scenarios.

So for example, let's say the hosts with ids 342 and 98 are not behaving as you expect in Fleet, you can enable verbose
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

> **Warning:** This will potentially log a lot of data. Some of that data might be private. Please verify it before posting it.
in a public channel or a GitHub issue.

- Optional setting (array of integers).
- Default value: empty.
- Config file format:
  ```
  server_settings:
    debug_host_ids:
      - 342
      - 98
  ```

##### server_settings.deferred_save_host

Whether saving host-related information is done synchronously in the HTTP handler of the host's request, or asynchronously. This can provide better performance in deployments with many hosts. Note that this is an **experimental feature**.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  server_settings:
    deferred_save_host: true
  ```

##### server_settings.enable_analytics

If sending usage analytics is enabled or not.

- Optional setting (boolean).
- Default value: `true`.
- Config file format:
  ```
  server_settings:
    enable_analytics: false
  ```

##### server_settings.live_query_disabled

If the live query feature is disabled or not.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  server_settings:
    live_query_disabled: true
  ```

##### server_settings.server_url

The base URL of the fleet server, including the scheme (e.g. "https://").

- Required setting (string).
- Default value: none (provided during Fleet setup).
- Config file format:
  ```
  server_settings:
    server_url: https://fleet.example.org:8080
  ```

#### SMTP Settings

It is recommended to use the Fleet UI to configure SMTP, as a secret password must be provided. Navigate to **Settings -> Organization settings -> SMTP Options** to proceed with this configuration.

#### SSO Settings

For additional information on SSO configuration, including just-in-time (JIT) user provisioning, creating SSO users in Fleet and identity providers configuration, see [Configuring single sign-on (SSO)](../../Deploying/Configuration.md#configuring-single-sign-on-sso).

##### sso_settings.enable_jit_provisioning

_Available in Fleet Premium_. Enables [just-in-time user provisioning](../../Deploying/Configuration.md#just-in-time-jit-user-provisioning).

- Optional setting (boolean).
- Default value: `false`
- Config file format:
  ```
  sso_settings:
    enable_jit_provisioning: true
  ```

##### sso_settings.enable_sso

Configures if single sign-on is enabled.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  sso_settings:
    enable_sso: true
  ```

##### sso_settings.enable_sso_idp_login

Allow single sign-on login initiated by identity provider.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  sso_settings:
    enable_sso_idp_login: true
  ```

##### sso_settings.entity_id

The required entity ID is a Uniform Resource Identifier (URI) that you use to identify Fleet when configuring the identity provider. It must exactly match the Entity ID field used in identity provider configuration.

- Required setting if SSO is enabled, must have at least 5 characters (string).
- Default value: "".
- Config file format:
  ```
  sso_settings:
    entity_id: "https://example.com"
  ```

##### sso_settings.idp_image_url

An optional link to an image such as a logo for the identity provider.

- Optional setting (string).
- Default value: "".
- Config file format:
  ```
  sso_settings:
    idp_image_url: "https://example.com/logo"
  ```

##### sso_settings.idp_name

A required human-friendly name for the identity provider that will provide single sign-on authentication.

- Required setting if SSO is enabled (string).
- Default value: "".
- Config file format:
  ```
  sso_settings:
    idp_name: "SimpleSAML"
  ```

##### sso_settings.issuer_uri

The issuer URI supplied by the identity provider.

- Optional setting (string).
- Default value: "".
- Config file format:
  ```
  sso_settings:
    issuer_uri: "https://example.com/saml2/sso-service"
  ```

##### sso_settings.metadata

Metadata (in XML format) provided by the identity provider.

- Optional setting, either `metadata` or `metadata_url` must be set if SSO is enabled, but not both (string).
- Default value: "".
- Config file format:
  ```
  sso_settings:
    metadata: "<md:EntityDescriptor entityID="https://idp.example.org/SAML2"> ... /md:EntityDescriptor>"
  ```

##### sso_settings.metadata_url

A URL that references the identity provider metadata.

- Optional setting, either `metadata` or `metadata_url` must be set if SSO is enabled, but not both (string).
- Default value: "".
- Config file format:
  ```
  sso_settings:
    metadata_url: https://idp.example.org/idp-meta.xml
  ```

#### Vulnerability Settings

##### vulnerability_settings.databases_path

Path to a directory on the local filesystem (accessible to the Fleet server) where the various vulnerability databases will be stored.

- Optional setting, must be set to enable vulnerability detection (string).
- Default value: "".
- Config file format:
  ```
  vulnerability_settings:
    databases_path: "/path/to/dir"
  ```

#### Webhook Settings

For more information about webhooks and Fleet automations in general, see the [Automations documentation](../../Using-Fleet/Automations.md).

##### webhook_settings.interval

The interval at which to check for webhook conditions. This value currently configures both the host status and failing policies webhooks (not the recent vulnerabilities webhook, see the [Recent vulnerabilities section](#recent-vulnerabilities) for details).

- Optional setting (time duration as a string).
- Default value: `24h`.
- Config file format:
  ```
  webhook_settings:
    interval: "12h"
  ```

##### Failing Policies

The following options allow the configuration of a webhook that will be triggered if selected policies are not passing for some hosts.

###### webhook_settings.failing_policies_webhook.destination_url

The URL to `POST` to when the condition for the webhook triggers.

- Optional setting, required if webhook is enabled (string).
- Default value: "".
- Config file format:
  ```
  webhook_settings:
    failing_policies_webhook:
      destination_url: "https://example.org/webhook_handler"
  ```

###### webhook_settings.failing_policies_webhook.enable_failing_policies_webhook

Defines whether to enable the failing policies webhook. Note that currently, if the failing policies webhook *and* the `osquery.enable_async_host_processing` options are set, some failing policies webhooks could be missing (some transitions from succeeding to failing or vice-versa could happen without triggering a webhook request).

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
  ```

###### webhook_settings.failing_policies_webhook.host_batch_size

Maximum number of hosts to batch on `POST` requests. A value of `0`, the default, means no batching, all hosts failing a policy will be sent on one `POST` request.

- Optional setting (integer).
- Default value: `0`.
- Config file format:
  ```
  webhook_settings:
    failing_policies_webhook:
      host_batch_size: 100
  ```

###### webhook_settings.failing_policies_webhook.policy_ids

The IDs of the policies for which the webhook will be enabled.

- Optional setting (array of integers).
- Default value: empty.
- Config file format:
  ```
  webhook_settings:
    failing_policies_webhook:
      policy_ids:
        - 1
        - 2
        - 3
  ```

##### Host Status

The following options allow the configuration of a webhook that will be triggered if the specified percentage of hosts are offline for the specified amount of time.

###### webhook_settings.host_status_webhook.days_count

Number of days that hosts need to be offline for to count as part of the percentage.

- Optional setting, required if webhook is enabled (integer).
- Default value: `0`.
- Config file format:
  ```
  webhook_settings:
    host_status_webhook:
      days_count: 5
  ```

###### webhook_settings.host_status_webhook.destination_url

The URL to `POST` to when the condition for the webhook triggers.

- Optional setting, required if webhook is enabled (string).
- Default value: "".
- Config file format:
  ```
  webhook_settings:
    host_status_webhook:
      destination_url: "https://example.org/webhook_handler"
  ```

###### webhook_settings.host_status_webhook.enable_host_status_webhook

Defines whether the webhook check for host status will run or not.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  webhook_settings:
    host_status_webhook:
      enable_host_status_webhook: true
  ```

###### webhook_settings.host_status_webhook.host_percentage

The percentage of hosts that need to be offline to trigger the webhook.

- Optional setting, required if webhook is enabled (float).
- Default value: `0`.
- Config file format:
  ```
  webhook_settings:
    host_status_webhook:
      host_percentage: 10
  ```

##### Recent vulnerabilities

The following options allow the configuration of a webhook that will be triggered if recently published vulnerabilities are detected and there are affected hosts. A vulnerability is considered recent if it has been published in the last 30 days (based on the National Vulnerability Database, NVD).

Note that the recent vulnerabilities webhook is not checked at `webhook_settings.interval` like other webhooks - it is checked as part of the vulnerability processing and runs at the `vulnerabilities.periodicity` interval specified in the [fleet configuration](../../Deploying/Configuration.md#periodicity).

###### webhook_settings.vulnerabilities_webhook.destination_url

The URL to `POST` to when the condition for the webhook triggers.

- Optional setting, required if webhook is enabled (string).
- Default value: "".
- Config file format:
  ```
  webhook_settings:
    vulnerabilities_webhook:
      destination_url: "https://example.org/webhook_handler"
  ```

###### webhook_settings.vulnerabilities_webhook.enable_vulnerabilities_webhook

Defines whether to enable the vulnerabilities webhook.

- Optional setting (boolean).
- Default value: `false`.
- Config file format:
  ```
  webhook_settings:
    vulnerabilities_webhook:
      enable_vulnerabilities_webhook: true
  ```

###### webhook_settings.vulnerabilities_webhook.host_batch_size

Maximum number of hosts to batch on `POST` requests. A value of `0`, the default, means no batching, all hosts affected will be sent on one `POST` request.

- Optional setting (integer).
- Default value: `0`.
- Config file format:
  ```
  webhook_settings:
    vulnerabilities_webhook:
      host_batch_size: 100
  ```

#### Agent options

The `agent_options` key describes options returned to osqueryd when it checks for configuration. See the [osquery documentation](https://osquery.readthedocs.io/en/stable/deployment/configuration/#options) for the available options. Existing options will be over-written by the application of this file.

> In Fleet v4.0.0, "osquery options" are renamed to "agent options" and are now configured using the organization settings (config) configuration file. [Check out out the Fleet v3 documentation](https://github.com/fleetdm/fleet/blob/3.13.0/docs/1-Using-Fleet/2-fleetctl-CLI.md#update-osquery-options) if you're using an older version of Fleet.

##### Overrides option

The `overrides` key allows you to segment hosts, by their platform, and supply these groups with unique osquery configuration options. When you choose to use the overrides option for a specific platform, all options specified in the default configuration will be ignored for that platform.

In the example file below, all Darwin and Ubuntu hosts will only receive the options specified in their respective overrides sections.

```yaml
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        distributed_interval: 3
        distributed_tls_max_attempts: 3
        logger_plugin: tls
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
            logger_plugin: tls
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
            logger_plugin: tls
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

##### Auto table construction

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

##### YARA configuration

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

#### Advanced configuration

> **Note:** More settings are included in the [contributor documentation](../../Contributing/Configuration-for-contributors.md). It's possible, although not recommended, to configure these settings in the YAML configuration file.

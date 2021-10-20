# Configuration files

- [Queries](#queries)
- [Packs](#packs)
- [Labels](#labels)
- [Enroll secrets](#enroll-secrets)
- [Teams](#teams)
- [Organization settings](#organization-settings)

Entities in Fleet, such as queries, packs, labels, agent options, and enroll secrets, can be managed with configuration files in yaml syntax.

This page contains links to examples that can help you understand the configuration options for your Fleet yaml file(s).

Examples in this directory are presented in two forms:
- [`single-file-configuration.yml`](./single-file-configuration.yml) presents multiple yaml documents in one file. One file is often easier to manage than several. Group related objects into a single file whenever it makes sense.
- The `multi-file-configuration` directory presents multiple yaml documents in separate files. They are in the following structure:

```
├─ packs
├   └─ osquery-monitoring.yml
├─ agent-options.yml
├─ enroll-secrets.yml
├─ labels.yml
├─ queries.yml
```

## Using yaml files in Fleet

A Fleet configuration is defined using one or more declarative "messages" in yaml syntax. Each message can live in it's own file or multiple in one file, each separated by `---`. Each file/message contains a few required top-level keys:

- `apiVersion` - the API version of the file/request
- `spec` - the "data" of the request
- `kind ` - the type of file/object (i.e.: pack, query, config)

The file may optionally also include some `metadata` for more complex data types (i.e.: packs).

When you reason about how to manage these config files, consider following the [General Config Tips](https://kubernetes.io/docs/concepts/configuration/overview/#general-config-tips) published by the Kubernetes project. Some of the especially relevant tips are included here as well:

- When defining configurations, specify the latest stable API version.
- Configuration files should be stored in version control before being pushed to the cluster. This allows quick roll-back of a configuration if needed. It also aids with cluster re-creation and restoration if necessary.
- Don’t specify default values unnecessarily – simple and minimal configs will reduce errors.

### Queries

For especially long or complex queries, you may want to define one query in one file. Continued edits and applications to this file will update the query as long as the `metadata.name` does not change. If you want to change the name of a query, you must first create a new query with the new name and then delete the query with the old name. Make sure the old query name is not defined in any packs before deleting it or an error will occur.

```yaml
apiVersion: v1
kind: query
spec:
  name: docker_processes
  description: The docker containers processes that are running on a system.
  query: select * from docker_container_processes;
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

### Packs

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

#### Moving queries and packs from one Fleet environment to another

When managing multiple Fleet environments, you may want to move queries and/or packs from one "exporter" environment to a another "importer" environment.

1. Navigate to `~/.fleet/config` to find the context names for your "exporter" and "importer" environment. For the purpose of these instructions we will use the context names `exporter` and `importer` respectively.
2. Run the command `fleetctl get queries --yaml --context exporter > queries.yaml && fleetctl apply -f queries.yml --context importer`. This will import all the queries from your exporter Fleet instance into your importer Fleet instance. _Note, this will also write a list of all queries in yaml syntax to a file names `queries.yml`._
3. Run the command `fleetctl get packs --yaml --context exporter > packs.yaml && fleetctl apply -f packs.yml --context importer`. This will import all the packs from your exporter Fleet instance into your importer Fleet instance. _Note, this will also write a list of all packs in yaml syntax to a file names `packs.yml`._

### Labels

The following file describes the labels which hosts should be automatically grouped into. The label resource should include the actual SQL query so that the label is self-contained:

```yaml
apiVersion: v1
kind: label
spec:
  name: slack_not_running
  query: >
    SELECT * from system_info
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

### Enroll secrets

The following file shows how to configure enroll secrets.

```yaml
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
    - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
    - secret: YBh0n4pvRplKyWiowv9bf3zp6BBOJ13O
```

### Teams

`Applies only to Fleet Premium`

The following is an example configuration file for a Team.

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Client Platform Engineerin
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
### Organization settings

The following file describes organization settings applied to the Fleet server.

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
        distributed_tls_read_endpoint: /api/v1/osquery/distributed/read
        distributed_tls_write_endpoint: /api/v1/osquery/distributed/write
        logger_plugin: tls
        logger_tls_endpoint: /api/v1/osquery/log
        logger_tls_period: 10
        pack_delimiter: /
    overrides: {}
  host_expiry_settings:
    host_expiry_enabled: true
    host_expiry_window: 10
  host_settings:
    # "additional" information to collect from hosts along with the host
    # details. This information will be updated at the same time as other host
    # details and is returned by the API when host objects are returned. Users
    # must take care to keep the data returned by these queries small in
    # order to mitigate potential performance impacts on the Fleet server.
    additional_queries:
      time: select * from time
      macs: select mac from interface_details
  org_info:
    org_logo_url: "https://example.org/logo.png"
    org_name: Example Org
  server_settings:
    server_url: https://fleet.example.org:8080
  smtp_settings:
    authentication_method: authmethod_plain
    authentication_type: authtype_username_password
    domain: example.org
    enable_smtp: true
    enable_ssl_tls: true
    enable_start_tls: true
    password: supersekretsmtppass
    port: 587
    sender_address: fleet@example.org
    server: mail.example.org
    user_name: test_user
    verify_ssl_certs: true
  sso_settings:
    enable_sso: false
    entity_id: 1234567890
    idp_image_url: https://idp.example.org/logo.png
    idp_name: IDP Vendor 1
    issuer_uri: https://idp.example.org/SAML2/SSO/POST
    metadata: "<md:EntityDescriptor entityID="https://idp.example.org/SAML2"> ... /md:EntityDescriptor>"
    metadata_url: https://idp.example.org/idp-meta.xml
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
        logger_tls_endpoint: /api/v1/osquery/log
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
            logger_tls_endpoint: /api/v1/osquery/log
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
            logger_tls_endpoint: /api/v1/osquery/log
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

#### Auto table construction

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
              query: "select service, client, allowed, prompt_count, last_modified from access"
              path: "/Library/Application Support/com.apple.TCC/TCC.db"
              columns:
                - "service"
                - "client"
                - "allowed"
                - "prompt_count"
                - "last_modified"
```

#### YARA configuration

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

#### SMTP authentication

**Warning:** Be careful not to store your SMTP credentials in source control. It is recommended to set the password through the web UI or `fleetctl` and then remove the line from the checked in version. Fleet will leave the password as-is if the field is missing from the applied configuration.

The following options are available when configuring SMTP authentication:

- `smtp_settings.authentication_type`
  - `authtype_none` - use this if your SMTP server is open
  - `authtype_username_password` - use this if your SMTP server requires authentication with a username and password
- `smtp_settings.authentication_method` - required with authentication type `authtype_username_password`
  - `authmethod_cram_md5`
  - `authmethod_login`
  - `authmethod_plain`

#### Webhooks

##### Host Status

The following options allow the configuration of a webhook that will be triggered if the specified percentage of hosts 
are offline for the specified amount of time.

- `webhook_settings.interval`: the interval at which to check for webhook conditions. Default: 24h
- `webhook_settings.host_status_webhook.enable_host_status_webhook`: true or false. Defines whether the check for host status will run or not.
- `webhook_settings.host_status_webhook.destination_url`: the URL to POST to when the condition for the webhook triggers.
- `webhook_settings.host_status_webhook.host_percentage`: the percentage of hosts that need to be offline  
- `webhook_settings.host_status_webhook.days_count`: amount of days that hosts need to be offline for to count as part of the percentage.

#### Debug host

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

WARNING: this will log potentially a lot of data. Some of that data might be private, please verify it before posting it
in a public channel or a Github issue.
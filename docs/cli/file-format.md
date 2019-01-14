# Configuration File Format

A Fleet configuration is defined using one or more declarative "messages" in yaml syntax. Each message can live in it's own file or multiple in one file, each separated by `---`. Each file/message contains a few required top-level keys:

- `apiVersion` - the API version of the file/request
- `spec` - the "data" of the request
- `kind ` - the type of file/object (i.e.: pack, query, config)

The file may optionally also include some `metadata` for more complex data types (i.e.: packs).

When you reason about how to manage these config files, consider following the [General Config Tips](https://kubernetes.io/docs/concepts/configuration/overview/#general-config-tips) published by the Kubernetes project. Some of the especially relevant tips are included here as well:

- When defining configurations, specify the latest stable API version.
- Configuration files should be stored in version control before being pushed to the cluster. This allows quick roll-back of a configuration if needed. It also aids with cluster re-creation and restoration if necessary.
- Group related objects into a single file whenever it makes sense. One file is often easier to manage than several. See the [config-single-file.yml](../../examples/config-single-file.yml) file as an example of this syntax.
- Don’t specify default values unnecessarily – simple and minimal configs will reduce errors.

All of these files can be concatenated together into [one file](../../examples/config-single-file.yml) (seperated by `---`), or they can be in [individual files with a directory structure](../../examples/config-many-files) like the following:

```
|-- config.yml
|-- labels.yml
|-- packs
|   `-- osquery-monitoring.yml
`-- queries.yml
```

## Osquery Queries

For especially long or complex queries, you may want to define one query in one file. Continued edits and applications to this file will update the query as long as the `metadata.name` does not change. If you want to change the name of a query, you must first create a new query with the new name and then delete the query with the old name. Make sure the old query name is not defined in any packs before deleting it or an error will occur.

```yaml
apiVersion: v1
kind: query
spec:
  name: docker_processes
  description: The docker containers processes that are running on a system.
  query: select * from docker_container_processes;
  support:
    osquery: 2.9.0
    platforms:
      - linux
      - darwin
```

To define multiple queries in a file, concatenate multiple `query` resources together in a single file with `---`. For example, consider a file that you might store at `queries/osquery_monitoring.yml`:

```yaml
apiVersion: v1
kind: query
spec:
  name: osquery_version
  description: The version of the Launcher and Osquery process
  query: select launcher.version, osquery.version from kolide_launcher_info launcher, osquery_info osquery;
  support:
    launcher: 0.3.0
    osquery: 2.9.0
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

## Query Packs

To define query packs, reference queries defined elsewhere by name. This is why the "name" of a query is so important. You can define many of these packs in many files.

```yaml
apiVersion: v1
kind: pack
spec:
  name: osquery_monitoring
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

## Host Labels

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

## Osquery Configuration Options

The following file describes configuration options passed to the osquery instance. All other configuration data will be over-written by the application of this file.

```yaml
apiVersion: v1
kind: options
spec:
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
```

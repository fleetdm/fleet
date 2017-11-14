CLI Documentation
=================

Kolide Fleet provides a server which allows you to manage and orchestrate an osquery deployment across of a set of workstations and servers. For certain use-cases, it makes sense to maintain the configuration and data of an osquery deployment in source-controlled files. It is also desireable to be able to manage these files with a familiar command-line tool. To facilitate this, we are working on an experimental CLI called `fleetctl`.

### Warning: In Progress

This CLI is largely just a proposal and large sections (if not most) of this do not work. The objective user-experience is documented here so that contributors working on this feature can share documentation with the community to gather feedback.

## Inspiration

Inspiration for the `fleetctl` command-line experience as well as the file format has been principally derived from the [Kubernetes](https://kubernetes.io/) container orchestration tool. This is for a few reasons:

- Format Familiarity: At Kolide, we love Kubernetes and we think it is the future of production infrastructure management. We believe that many of the people that use this interface to manage Fleet will also be Kubernetes operators. By using a familiar command-line interface and file format, the cognitive overhead can be reduced since the operator is already familiar with how these tools work and behave.
- Established Best Practices: Kubernetes deployments can easily become very complex. Because of this, Kubernetes operators have an established set of best practices that they often follow when writing and maintaining config files. Some of these best practices and tips are documented on the [official Kubernetes website](https://kubernetes.io/docs/concepts/configuration/overview/#general-config-tips) and some are documented by [the community](https://www.mirantis.com/blog/introduction-to-yaml-creating-a-kubernetes-deployment/). Since the file format and workflow is so similar, we can make re-use these best practices when managing Fleet configurations as well.

## `fleetctl` - The CLI

The `fleetctl` tool is heavily inspired by the [`kubectl`](https://kubernetes.io/docs/user-guide/kubectl-overview/) tool. If you are familiar with `kubectl`, this will all feel very familiar to you. If not, some further explanation would likely be helpful.

Fleet exposes the aspects of an osquery deployment as a set of "objects". Objects may be a query, a pack, a set of configuration options, etc. The documentaiton for [Declarative Management of Kubernetes Objects Using Configuration Files](https://kubernetes.io/docs/tutorials/object-management-kubectl/declarative-object-management-configuration/) says the following about the object lifecycle:

> Objects can be created, updated, and deleted by storing multiple object configuration files in a directory and using `kubectl apply` to recursively create and update those objects as needed.

Similarly, Fleet objects can be created, updated, and deleted by storing multiple object configuration files in a directory and using `fleetctl apply` to recursively create and update those objects as needed.

### Help Output

```
$ fleetctl --help
fleetctl controls an instance of the Kolide Fleet osquery fleet manager.

Find more information at https://kolide.com/fleet

  Usage:
    fleetctl [command] [flags]


  Commands:
    fleetctl query    - run a query across your fleet

    fleetctl apply    - apply a set of osquery configurations
    fleetctl edit     - edit your complete configuration in an ephemeral editor
    fleetctl config   - modify how and which Fleet server to connect to

    fleetctl help     - get help on how to define an intent type
    fleetctl version  - print full version information
```

### Workflow

```bash
# Make sure you're currently using the current server (in this case: staging)
fleetctl config set-context staging

# Edit the config file (or files) for your Fleet instance (or one of them) and apply the file
vim fleet-staging.yml
fleetctl apply -f ./fleet-staging.yml

# Commit the changes to an upstream source tree
git add fleet-staging.yml
git commit -m "new changes to staging fleet instance"
git push
```

Alternatively, you can specify the context as a flag for easy use in parallel scripts or instances where you may have many Fleet environments:

```bash
# Edit your Fleet config file
vim fleet.yml

# First apply the configuration to your staging environment for testing
fleetctl apply -f ./fleet.yml --context=staging

# Apply the configuration to both staging and production at the same time
fleetctl apply -f ./fleet.yml --context=staging,production
```

## Configuration File Format

A Fleet configuration is defined using one or more declarative "messages" using yaml syntax. Each message can live in it's own file or you can miltiple in one file, each seperated by `---`. Each file/message contains a few required top-level keys:

- `apiVersion` - the API version of the file/request
- `spec` - the "data" of the request
- `kind ` - the type of file/object (i.e.: pack, query, config)

The file may optionally also include some `metadata` for more complex data types (i.e.: packs).

When you reason about how to manage these config files, consider following the [General Config Tips](https://kubernetes.io/docs/concepts/configuration/overview/#general-config-tips) published by the Kubernetes project. Some of the especially relevant tips are included here as well:

- When defining configurations, specify the latest stable API version.
- Configuration files should be stored in version control before being pushed to the cluster. This allows quick roll-back of a configuration if needed. It also aids with cluster re-creation and restoration if necessary.
- Group related objects into a single file whenever it makes sense. One file is often easier to manage than several. See the [config-single-file.yml](../../examples/config-single-file.yml) file as an example of this syntax.
- Don’t specify default values unnecessarily – simple and minimal configs will reduce errors.

All of these files can be concatonated together into [one file](../../examples/config-single-file.yml) (seperated by `---`), or they can be in [individual files with a directory structure](../../examples/config-many-files) like the following:

```
|-- config.yml
|-- decorators.yml
|-- fim.yml
|-- labels.yml
|-- packs
|   `-- osquery-monitoring.yml
`-- queries
    |-- processes.yml
    `-- queries.yml
```

### Osquery Configuration Options

The following file describes configuration options to pass to the osquery instance. All other configuration data will be over-written by the application of this file.

```yaml
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryOptions
spec:
  config:
    - distributed_interval: 3
    - distributed_tls_max_attempts: 3
    - logger_plugin: tls
    - logger_tls_endpoint: /api/v1/osquery/log
    - logger_tls_period: 10
```

### Osquery Logging Decorators

The following file describes logging decorators that should be applied on osquery instances. All other decorator data will be over-written by the application of this file.

```yaml
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryDecorators
spec:
  decorators:
    - name: hostname
      query: select hostname from system_info;
      type: interval
      interval: 10
    - name: uuid
      query: select uuid from osquery_info;
      type: load
    - name: instance_id
      query: select instance_id from osquery_info;
      type: load
```

### File Integrity Monitoring

The following file describes the configuration for osqueryd's file integrity monitoring system. All other FIM configuration will be over-written by the application of this file.

```yaml
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryFIM
spec:
  fim:
    interval: 500
    groups:
      - name: etc
        paths:
          - /etc/%%
      - name: users
        paths:
          - /Users/%/Library/%%
          - /Users/%/Documents/%%
```

### Host Labels

The following file describes the labels which hosts should be automatically grouped into. All other label data will be over-written by the application of this file.

```yaml
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryLabels
spec:
  labels:
    - name: all_hosts
      query: select 1;
    - name: macs
      query: select 1 from os_version where platform = "darwin";
    - name: ubuntu
      query: select 1 from os_version where platform = "ubuntu";
    - name: centos
      query: select 1 from os_version where platform = "centos";
    - name: windows
      query: select 1 from os_version where platform = "windows";
    - name: pending_updates
      query: SELECT value from plist where path = "/Library/Preferences/ManagedInstalls.plist" and key = "PendingUpdateCount" and value > "0";
      platforms:
        - darwin
    - name: slack_not_running
      query: >
        SELECT * from system_info
        WHERE NOT EXISTS (
          SELECT *
          FROM processes
          WHERE name LIKE "%Slack%"
        );
```

### Osquery Queries

For especially long or complex queries, you may want to define one query in one file. Continued edits and applications to this file will update the query as long as the `metadata.name` does not change. If you want to change the name of a query, you must first create a new query with the new name and then delete the query with the old name. Make sure the old query name is not defined in any packs before deleting it or an error will occur.

```yaml
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryQuery
metadata:
  - name: processes
spec:
  query: select * from processes;
```

For many cases, however, defining many queries in a single file is sufficient. This syntax is easy to use, but can make some operations like renaming and deleting queries more arduous.

```yaml
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryQueries
spec:
  queries:
    - name: launcher_version
      query: select version from kolide_launcher_info;
    - name: osquery_version
      query: select version from osquery_info;
    - name: osquery_schedule
      query: select name, interval, executions, output_size, wall_time, (user_time/executions) as avg_user_time, (system_time/executions) as avg_system_time, average_memory, last_executed from osquery_schedule;
      description: Report performance publisher health and track event counters.
    - name: osquery_events
      query: select name, publisher, type, subscriptions, events, active from osquery_events;
      description: Report event publisher health and track event counters.
    - name: osquery_info
      query: select i.*, p.resident_size, p.user_time, p.system_time, time.minutes as counter from osquery_info i, processes p, time where p.pid = i.pid;
      description: A heartbeat counter that reports general performance (CPU, memory) and version.
```

### Query Packs

To define query packs, you reference queries defined elsewhere by name. This is why the "name" of a query is so important. You can define many of these packs in many files.

```yaml
apiVersion: k8s.kolide.com/v1alpha1
kind: OsqueryPack
metadata:
  name: osquery_monitoring
spec:
  targets:
    labels:
      - all_hosts
  queries:
    - name: osquery_schedule
      interval: 7200
      removed: false
      description: Report performance publisher health and track event counters.
    - name: osquery_events
      interval: 86400
      removed: false
      description: Report event publisher health and track event counters.
    - name: oquery_info
      interval: 600
      removed: false
      description: A heartbeat counter that reports general performance (CPU, memory) and version.
```

# fleetctl CLI
- [Setting Up Fleet via the CLI](#setting-up-fleet-via-the-cli)
  - [Running Fleet](#running-fleet)
  - [`fleetctl config`](#fleetctl-config)
  - [`fleetctl setup`](#fleetctl-setup)
  - [Connecting a host](#connecting-a-host)
  - [Query hosts](#query-hosts)
  - [Update osquery options](#update-osquery-options)
- [Logging in to an existing Fleet instance](#logging-in-to-an-existing-Fleet-instance)
  - [Logging in with SAML (SSO) authentication](#logging-in-with-SAML-SSO-authentication)
- [Using fleetctl for configuration](#using-fleetctl-for-configuration)
  - [Convert osquery JSON](#convert-osquery-json)
  - [Osquery queries](#osquery-queries)
  - [Query packs](#query-packs)
    - [Moving queries and packs from one Fleet environment to another](#moving-queries-and-packs-from-one-fleet-environment-to-another)
  - [Host labels](#host-labels)
  - [Osquery configuration options](#osquery-configuration-options)
  - [Auto table construction](#auto-table-construction)
  - [Fleet configuration options](#fleet-configuration-options)
  - [Enroll secrets](#enroll-secrets)
- [File carving with Fleet](#file-carving-with-fleet)
  - [Configuration](#configuration)
  - [Usage](#usage)
  - [Troubleshooting](#troubleshooting)

## Setting up Fleet via the CLI

This document walks through setting up and configuring Fleet via the CLI. If you already have a running fleet instance, skip ahead to [Logging In To An Existing Fleet Instance](#logging-in-to-an-existing-fleet-instance) to configure the `fleetctl` CLI.

This guide illustrates:

- A minimal CLI workflow for managing an osquery fleet
- The set of API interactions that are required if you want to perform remote, automated management of a Fleet instance

### Running Fleet

For the sake of this tutorial, I will be using the local development Docker Compose infrastructure to run Fleet locally. This is documented in some detail in the [developer documentation](../3-Contribution/1-Building-Fleet.md#development-infrastructure), but the following are the minimal set of commands that you can run from the root of the repository (assuming that you have a working Go/JavaScript toolchain installed along with Docker Compose):

```
docker-compose up -d
make deps
make generate
make
./build/fleet prepare db
./build/fleet serve --auth_jwt_key="insecure"
```

The `fleet serve` command will be the long running command that runs the Fleet server.

### `fleetctl config`

At this point, the MySQL database doesn't have any users in it. Because of this, Fleet is exposing a one-time setup endpoint. Before we can hit that endpoint (by running `fleetctl setup`), we have to first configure the local `fleetctl` context.

Now, since our Fleet instance is local in this tutorial, we didn't get a valid TLS certificate, so we need to run the following to configure our Fleet context:

```
fleetctl config set --address https://localhost:8080 --tls-skip-verify
[+] Set the address config key to "https://localhost:8080" in the "default" context
[+] Set the tls-skip-verify config key to "true" in the "default" context
```

Now, if you were connecting to a Fleet instance for real, you wouldn't want to skip TLS certificate verification, so you might run something like:

```
fleetctl config set --address https://fleet.corp.example.com
[+] Set the address config key to "https://fleet.corp.example.com" in the "default" context
```

### `fleetctl setup`

Now that we've configured our local CLI context, lets go ahead and create our admin account:

```
fleetctl setup --email mike@arpaia.co
Password:
[+] Fleet setup successful and context configured!
```

It's possible to specify the password via the `--password` flag or the `$PASSWORD` environment variable, but be cautious of the security implications of such an action. For local use, the interactive mode above is the most secure.

### Connecting a host

For the sake of this tutorial, I'm going to be using Kolide's osquery launcher to start osquery locally and connect it to Fleet. To learn more about connecting osquery to Fleet, see the [Adding Hosts to Fleet](../2-Deployment/3-Adding-hosts.md) documentation.

To get your osquery enroll secret, run the following:

```
fleetctl get enroll-secret
E7P6zs9D0mvY7ct08weZ7xvLtQfGYrdC
```

You need to use this secret to connect a host. If you're running Fleet locally, you'd run:

```
launcher \
  --hostname localhost:8080 \
  --enroll_secret E7P6zs9D0mvY7ct08weZ7xvLtQfGYrdC \
  --root_directory=$(mktemp -d) \
  --insecure
```

### Query hosts

To run a simple query against all hosts, you might run something like the following:

```
fleetctl query --query 'select * from osquery_info;' --labels='All Hosts' > results.json
⠂  100% responded (100% online) | 1/1 targeted hosts (1/1 online)
^C
```

When the query is done (or you have enough results), CTRL-C and look at the `results.json` file:

```json
{
  "host": "marpaia",
  "rows": [
    {
      "build_distro": "10.13",
      "build_platform": "darwin",
      "config_hash": "d7cafcd183cc50c686b4c128263bd4eace5d89e1",
      "config_valid": "1",
      "extensions": "active",
      "host_hostname": "marpaia",
      "instance_id": "37840766-7182-4a68-a204-c7f577bd71e1",
      "pid": "22984",
      "start_time": "1527031727",
      "uuid": "B312055D-9209-5C89-9DDB-987299518FF7",
      "version": "3.2.3",
      "watcher": "-1"
    }
  ]
}
```

### Update osquery options

By default, each osquery node will check in with Fleet every 10 seconds. Let's say, for testing, you want to increase this to every 2 seconds. If this is the first time you've ever modified osquery options, let's download them locally:

```
fleetctl get options > options.yaml
```

The `options.yaml` file will look something like this:

```yaml
apiVersion: v1
kind: options
spec:
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
```

Let's edit the file so that the `distributed_interval` option is 2 instead of 10. Save the file and run:

```
fleetctl apply -f ./options.yaml
```

Now run a live query again. You should notice results coming back more quickly.

## Logging in to an existing Fleet instance

If you have an existing Fleet instance (version 2.0.0 or above), then simply run `fleetctl login` (after configuring your local CLI context):

```
fleetctl config set --address https://fleet.corp.example.com
[+] Set the address config key to "https://fleet.corp.example.com" in the "default" context

fleetctl login
Log in using the standard Fleet credentials.
Email: mike@arpaia.co
Password:
[+] Fleet login successful and context configured!
```

Once your local context is configured, you can use the above `fleetctl` normally. See `fleetctl --help` for more information.

### Logging in with SAML (SSO) authentication

Users that authenticate to Fleet via SSO should retrieve their API token from the UI and set it manually in their `fleetctl` configuration (instead of logging in via `fleetctl login`).

1. Go to the "Account Settings" page in Fleet (https://fleet.corp.example.com/settings). Click the "Get API Token" button to bring up a modal with the API token.

2. Set the API token in the `~/.fleet/config` file. The file should look like the following:

```
contexts:
  default:
    address: https://fleet.corp.example.com
    email: example@example.com
    token: your_token_here
```

Note the token can also be set with `fleetctl config set --token`, but this may leak the token into a user's shell history.

## Using fleetctl for configuration

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

All of these files can be concatenated together into [one file](../../examples/config-single-file.yml) (separated by `---`), or they can be in [individual files with a directory structure](../../examples/config-many-files) like the following:

```
|-- config.yml
|-- labels.yml
|-- packs
|   `-- osquery-monitoring.yml
`-- queries.yml
```

### Convert osquery JSON

`fleetctl` includes easy tooling to convert osquery pack JSON into the
`fleetctl` format. Use `fleetctl convert` with a path to the pack file:

```
fleetctl convert -f test.json
---
apiVersion: v1
kind: pack
spec:
  name: test
  queries:
  - description: "this is a test query"
    interval: 10
    name: processes
    query: processes
    removed: false
  targets:
    labels: null
---
apiVersion: v1
kind: query
spec:
  name: processes
  query: select * from processes
```

### Osquery queries

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
apiVersion: v1
kind: query
spec:
  name: osquery_version
  description: The version of the Launcher and Osquery process
  query: select launcher.version, osquery.version from kolide_launcher_info launcher, osquery_info osquery;
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

### Query packs

To define query packs, reference queries defined elsewhere by name. This is why the "name" of a query is so important. You can define many of these packs in many files.

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

#### Moving queries and packs from one Fleet environment to another

When managing multiple Fleet environments, you may want to move queries and/or packs from one "exporter" environment to a another "importer" environment.

1. Navigate to `~/.fleet/config` to find the context names for your "exporter" and "importer" environment. For the purpose of these instructions we will use the context names `exporter` and `importer` respectively.
2. Run the command `fleetctl get queries --yaml --context exporter > queries.yaml && fleetctl apply -f queries.yml --context importer`. This will import all the queries from your exporter Fleet instance into your importer Fleet instance. *Note, this will also write a list of all queries in yaml syntax to a file names `queries.yml`.*
3. Run the command `fleetctl get packs --yaml --context exporter > packs.yaml && fleetctl apply -f packs.yml --context importer`. This will import all the packs from your exporter Fleet instance into your importer Fleet instance. *Note, this will also write a list of all packs in yaml syntax to a file names `packs.yml`.*

### Host labels

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


### Osquery configuration options

The following file describes options returned to osqueryd when it checks for configuration. See the [osquery documentation](https://osquery.readthedocs.io/en/stable/deployment/configuration/#options) for the available options. Existing options will be over-written by the application of this file.

#### Overrides option

The overrides option allows you to segment hosts, by their platform, and supply these groups with unique osquery configuration options. When you choose to use the overrides option for a specific platform, all options specified in the default configuration will be ignored for that platform. 

In the example file below, all Darwin and Ubuntu hosts will only receive the options specified in their respective overrides sections.

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

### Auto table construction

You can use Fleet to query local SQLite databases as tables. For more information on creating ATC configuration from a SQLite database, see the [Osquery Automatic Table Construction documentation](https://osquery.readthedocs.io/en/stable/deployment/configuration/#automatic-table-construction)

If you already know what your ATC configuration needs to look like, you can add it to an options config file:

```yaml
apiVersion: v1
kind: options
spec:
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

### Fleet configuration options
The following file describes configuration options applied to the Fleet server.

```yaml
apiVersion: v1
kind: config
spec:
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
    kolide_server_url: https://fleet.example.org:8080
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

### Enroll secrets

The following file shows how to configure enroll secrets. Note that secrets can be changed or made inactive, but not deleted. Hosts may not enroll with inactive secrets.

The name of the enroll secret used to authenticate is stored with the host and is included with API results.

```yaml
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
  - active: true
    name: default
    secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
  - active: true
    name: new_one
    secret: reallyworks
  - active: false
    name: inactive_secret
    secret: thissecretwontwork!
```

## File carving with Fleet

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from osquery agents, returning the full contents to Fleet.

File carving data can be either stored in Fleet's database or to an external S3 bucket. For information on how to configure the latter, consult the [configuration docs](https://github.com/fleetdm/fleet/blob/master/docs/2-Deployment/2-Configuration.md#s3-file-carving-backend).

### Configuration

Given a working flagfile for connecting osquery agents to Fleet, add the following flags to enable carving:

```
--disable_carver=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=2000000
```

The default flagfile provided in the "Add New Host" dialog also includes this configuration.

#### Carver block size

The `carver_block_size` flag should be configured in osquery. 2MB (`2000000`) is a good starting value.

The configured value must be less than the value of `max_allowed_packet` in the MySQL connection, allowing for some overhead. The default for MySQL 5.7 is 4MB and for MySQL 8 it is 64MB.

In case S3 is used as the storage backend, this value must be instead set to be at least 5MB due to the [constraints of S3's multipart uploads](https://docs.aws.amazon.com/AmazonS3/latest/dev/qfacts.html).

Using a smaller value for `carver_block_size` will lead to more HTTP requests during the carving process, resulting in longer carve times and higher load on the Fleet server. If the value is too high, HTTP requests may run long enough to cause server timeouts.

#### Compression

Compression of the carve contents can be enabled with the `carver_compression` flag in osquery. When used, the carve results will be compressed with [Zstandard](https://facebook.github.io/zstd/) compression.

### Usage

File carves are initiated with osquery queries. Issue a query to the `carves` table, providing `carve = 1` along with the desired path(s) as constraints.

For example, to extract the `/etc/hosts` file on a host with hostname `mac-workstation`:

```
fleetctl query --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path = "/etc/hosts"'
```

The standard osquery file globbing syntax is also supported to carve entire directories or more:
```
fleetctl query --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path LIKE "/etc/%%"'
```

#### Retrieving carves

List the non-expired (see below) carves with `fleetctl get carves`. Note that carves will not be available through this command until osquery checks in to the Fleet server with the first of the carve contents. This can take some time from initiation of the carve.

To also retrieve expired carves, use `fleetctl get carves --expired`.

Contents of carves are returned as .tar archives, and compressed if that option is configured.

To download the contents of a carve with ID 3, use

```
fleetctl get carve --outfile carve.tar 3
```

It can also be useful to pipe the results directly into the tar command for unarchiving:

```
fleetctl get carve --stdout 3 | tar -x
```

#### Expiration

Carve contents remain available for 24 hours after the first data is provided from the osquery client. After this time, the carve contents are cleaned from the database and the carve is marked as "expired".

The same is not true if S3 is used as the storage backend. In that scenario, it is suggested to setup a [bucket lifecycle configuration](https://docs.aws.amazon.com/AmazonS3/latest/dev/object-lifecycle-mgmt.html) to avoid retaining data in excess. Fleet, in an "eventual consistent" manner (i.e. by periodically performing comparisons), will keep the metadata relative to the files carves in sync with what it is actually available in the bucket.

### Troubleshooting

#### Check carve status in osquery

Osquery can report on the status of carves through queries to the `carves` table.

The details provided by

```
fleetctl query --labels 'All Hosts' --query 'SELECT * FROM carves'
```

can be helpful to debug carving problems.

#### Ensure  `carver_block_size` is set appropriately

This value must be less than the `max_allowed_packet` setting in MySQL. If it is too large, MySQL will reject the writes.

The value must be small enough that HTTP requests do not time out.
